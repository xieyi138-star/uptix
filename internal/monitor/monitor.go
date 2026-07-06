package monitor

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/uptix/uptix/internal/db"
	"github.com/uptix/uptix/internal/models"
)

type Monitor struct {
	db *db.DB
}

func New(database *db.DB) *Monitor {
	return &Monitor{db: database}
}

// Run starts the monitoring loop. It checks all monitors at their configured intervals.
func (m *Monitor) Run(ctx context.Context) {
	log.Info().Msg("monitor engine started")
	ticker := time.NewTicker(10 * time.Second) // master tick
	defer ticker.Stop()

	lastCheck := make(map[int64]time.Time)

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("monitor engine stopped")
			return
		case <-ticker.C:
			monitors, err := m.db.ListMonitors()
			if err != nil {
				log.Error().Err(err).Msg("failed to list monitors")
				continue
			}
			now := time.Now()
			for _, mon := range monitors {
				if last, ok := lastCheck[mon.ID]; ok {
					if now.Sub(last) < time.Duration(mon.IntervalSecs)*time.Second {
						continue
					}
				}
				lastCheck[mon.ID] = now
				go m.checkOne(mon)
			}
		}
	}
}

func (m *Monitor) checkOne(mon models.Monitor) {
	result := m.performCheck(mon)

	m.db.RecordCheck(&result)
	m.db.UpdateMonitorStatus(mon.ID, result.Status, result.ResponseTimeMs)

	// Recalculate 30-day uptime
	checks, _ := m.db.GetRecentChecks(mon.ID, 2880) // ~30 days at 15-min intervals
	up := 0
	for _, c := range checks {
		if c.Status == "up" {
			up++
		}
	}
	if len(checks) > 0 {
		pct := float64(up) / float64(len(checks)) * 100
		m.db.UpdateMonitorUptime(mon.ID, pct)
	}

	// Auto-create incident on state change from up → down
	if result.Status == "down" && mon.Status == "up" {
		m.db.CreateIncident(&models.Incident{
			Title:       mon.Name + " is down",
			Description: result.ErrorMessage,
			Status:      "investigating",
			Severity:    "major",
			MonitorID:   &mon.ID,
		})
		log.Warn().Str("monitor", mon.Name).Msg("auto-created incident: monitor went down")
	}

	// Auto-resolve when back up
	if result.Status == "up" && mon.Status == "down" {
		// Find the open incident for this monitor and resolve it
		incidents, _ := m.db.ListIncidents(true)
		for _, inc := range incidents {
			if inc.MonitorID != nil && *inc.MonitorID == mon.ID {
				m.db.ResolveIncident(inc.ID, "Service automatically recovered. "+mon.Name+" is back up.")
				log.Info().Str("monitor", mon.Name).Int64("incident", inc.ID).Msg("auto-resolved incident")
				break
			}
		}
	}

	log.Debug().Str("name", mon.Name).Str("url", mon.URL).Str("status", result.Status).Int64("ms", result.ResponseTimeMs).Msg("check complete")
}

func (m *Monitor) performCheck(mon models.Monitor) models.CheckResult {
	result := models.CheckResult{
		MonitorID: mon.ID,
		CheckedAt: time.Now(),
	}

	switch mon.Type {
	case "http", "https":
		return m.checkHTTP(mon, result)
	case "tcp":
		return m.checkTCP(mon, result)
	case "dns":
		return m.checkDNS(mon, result)
	case "ssl":
		return m.checkSSL(mon, result)
	default:
		return m.checkHTTP(mon, result)
	}
}

func (m *Monitor) checkHTTP(mon models.Monitor, result models.CheckResult) models.CheckResult {
	timeout := time.Duration(mon.TimeoutSecs) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	start := time.Now()
	resp, err := client.Get(mon.URL)
	result.ResponseTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Status = "down"
		result.ErrorMessage = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Status = "up"
	} else if resp.StatusCode >= 500 {
		result.Status = "down"
		result.ErrorMessage = "HTTP " + http.StatusText(resp.StatusCode)
	} else {
		result.Status = "degraded"
		result.ErrorMessage = "HTTP " + http.StatusText(resp.StatusCode)
	}
	return result
}

func (m *Monitor) checkTCP(mon models.Monitor, result models.CheckResult) models.CheckResult {
	timeout := time.Duration(mon.TimeoutSecs) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", mon.URL, timeout)
	result.ResponseTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Status = "down"
		result.ErrorMessage = err.Error()
		return result
	}
	conn.Close()
	result.Status = "up"
	return result
}

func (m *Monitor) checkDNS(mon models.Monitor, result models.CheckResult) models.CheckResult {
	timeout := time.Duration(mon.TimeoutSecs) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	start := time.Now()
	_, err := net.LookupHost(mon.URL)
	result.ResponseTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Status = "down"
		result.ErrorMessage = err.Error()
		return result
	}
	result.Status = "up"
	return result
}

func (m *Monitor) checkSSL(mon models.Monitor, result models.CheckResult) models.CheckResult {
	timeout := time.Duration(mon.TimeoutSecs) * time.Second
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	start := time.Now()
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", mon.URL, &tls.Config{
		InsecureSkipVerify: false,
	})
	result.ResponseTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Status = "down"
		result.ErrorMessage = err.Error()
		return result
	}
	defer conn.Close()

	// Check expiry
	for _, cert := range conn.ConnectionState().PeerCertificates {
		if time.Until(cert.NotAfter) < 7*24*time.Hour {
			result.Status = "degraded"
			result.ErrorMessage = "SSL certificate expires in less than 7 days"
			return result
		}
	}
	result.Status = "up"
	return result
}
