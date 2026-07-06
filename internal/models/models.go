package models

import "time"

// Monitor represents a single service/endpoint being monitored.
type Monitor struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	Type           string    `json:"type"` // http, tcp, dns, ssl, cron
	IntervalSecs   int       `json:"interval_secs"`
	TimeoutSecs    int       `json:"timeout_secs"`
	Status         string    `json:"status"` // up, down, degraded
	LastCheckedAt  time.Time `json:"last_checked_at"`
	ResponseTimeMs int64     `json:"response_time_ms"`
	UptimePercent  float64   `json:"uptime_percent"` // 30-day rolling
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CheckResult is a single monitoring check outcome.
type CheckResult struct {
	ID             int64     `json:"id"`
	MonitorID      int64     `json:"monitor_id"`
	Status         string    `json:"status"` // up, down
	StatusCode     int       `json:"status_code"`
	ResponseTimeMs int64     `json:"response_time_ms"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
}

// Incident represents an ongoing or resolved service disruption.
type Incident struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // investigating, identified, monitoring, resolved
	Severity    string    `json:"severity"` // minor, major, critical
	MonitorID   *int64    `json:"monitor_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// IncidentUpdate is a single status update within an incident timeline.
type IncidentUpdate struct {
	ID         int64     `json:"id"`
	IncidentID int64     `json:"incident_id"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}

// Subscriber is an email/webhook recipient for incident notifications.
type Subscriber struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email,omitempty"`
	WebhookURL string   `json:"webhook_url,omitempty"`
	Type      string    `json:"type"` // email, slack, discord, webhook
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// MaintenanceWindow is a scheduled maintenance period.
type MaintenanceWindow struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
	Recurring string    `json:"recurring,omitempty"` // cron expression for recurring windows
	CreatedAt time.Time `json:"created_at"`
}
