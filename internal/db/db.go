package db

import (
	"database/sql"
	"fmt"

	"github.com/uptix/uptix/internal/models"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func New(path, driver string) (*DB, error) {
	if driver == "" {
		driver = "sqlite"
	}
	conn, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	conn.SetMaxOpenConns(1) // SQLite single-writer
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(0)
	return &DB{conn: conn}, nil
}

func (d *DB) Close() error { return d.conn.Close() }

func (d *DB) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS monitors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		type TEXT NOT NULL DEFAULT 'http',
		interval_secs INTEGER NOT NULL DEFAULT 60,
		timeout_secs INTEGER NOT NULL DEFAULT 30,
		status TEXT NOT NULL DEFAULT 'up',
		last_checked_at TIMESTAMP,
		response_time_ms INTEGER DEFAULT 0,
		uptime_percent REAL DEFAULT 100.0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS check_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		monitor_id INTEGER NOT NULL REFERENCES monitors(id),
		status TEXT NOT NULL,
		status_code INTEGER DEFAULT 0,
		response_time_ms INTEGER DEFAULT 0,
		error_message TEXT DEFAULT '',
		checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS incidents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		description TEXT DEFAULT '',
		status TEXT NOT NULL DEFAULT 'investigating',
		severity TEXT NOT NULL DEFAULT 'minor',
		monitor_id INTEGER REFERENCES monitors(id),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		resolved_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS incident_updates (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		incident_id INTEGER NOT NULL REFERENCES incidents(id),
		status TEXT NOT NULL,
		message TEXT DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS subscribers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT DEFAULT '',
		webhook_url TEXT DEFAULT '',
		type TEXT NOT NULL DEFAULT 'email',
		active INTEGER NOT NULL DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS maintenance_windows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		start_at TIMESTAMP NOT NULL,
		end_at TIMESTAMP NOT NULL,
		recurring TEXT DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_check_results_monitor ON check_results(monitor_id, checked_at);
	CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
	`
	_, err := d.conn.Exec(schema)
	return err
}

// --- Monitor CRUD ---

func (d *DB) ListMonitors() ([]models.Monitor, error) {
	rows, err := d.conn.Query(`SELECT id, name, url, type, interval_secs, timeout_secs, status, last_checked_at, response_time_ms, uptime_percent, created_at, updated_at FROM monitors ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []models.Monitor
	for rows.Next() {
		var m models.Monitor
		var lastChecked sql.NullTime
		err := rows.Scan(&m.ID, &m.Name, &m.URL, &m.Type, &m.IntervalSecs, &m.TimeoutSecs, &m.Status, &lastChecked, &m.ResponseTimeMs, &m.UptimePercent, &m.CreatedAt, &m.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if lastChecked.Valid {
			m.LastCheckedAt = lastChecked.Time
		}
		monitors = append(monitors, m)
	}
	return monitors, nil
}

func (d *DB) GetMonitor(id int64) (*models.Monitor, error) {
	m := &models.Monitor{}
	var lastChecked sql.NullTime
	err := d.conn.QueryRow(`SELECT id, name, url, type, interval_secs, timeout_secs, status, last_checked_at, response_time_ms, uptime_percent, created_at, updated_at FROM monitors WHERE id=?`, id).Scan(&m.ID, &m.Name, &m.URL, &m.Type, &m.IntervalSecs, &m.TimeoutSecs, &m.Status, &lastChecked, &m.ResponseTimeMs, &m.UptimePercent, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if lastChecked.Valid {
		m.LastCheckedAt = lastChecked.Time
	}
	return m, nil
}

func (d *DB) CreateMonitor(m *models.Monitor) error {
	result, err := d.conn.Exec(`INSERT INTO monitors (name, url, type, interval_secs, timeout_secs) VALUES (?, ?, ?, ?, ?)`, m.Name, m.URL, m.Type, m.IntervalSecs, m.TimeoutSecs)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	m.ID = id
	return nil
}

func (d *DB) UpdateMonitorStatus(id int64, status string, responseTimeMs int64) error {
	_, err := d.conn.Exec(`UPDATE monitors SET status=?, last_checked_at=CURRENT_TIMESTAMP, response_time_ms=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, status, responseTimeMs, id)
	return err
}

func (d *DB) UpdateMonitorUptime(id int64, pct float64) error {
	_, err := d.conn.Exec(`UPDATE monitors SET uptime_percent=? WHERE id=?`, pct, id)
	return err
}

func (d *DB) RecordCheck(r *models.CheckResult) error {
	_, err := d.conn.Exec(`INSERT INTO check_results (monitor_id, status, status_code, response_time_ms, error_message) VALUES (?, ?, ?, ?, ?)`, r.MonitorID, r.Status, r.StatusCode, r.ResponseTimeMs, r.ErrorMessage)
	return err
}

func (d *DB) GetRecentChecks(monitorID int64, limit int) ([]models.CheckResult, error) {
	rows, err := d.conn.Query(`SELECT id, monitor_id, status, status_code, response_time_ms, error_message, checked_at FROM check_results WHERE monitor_id=? ORDER BY checked_at DESC LIMIT ?`, monitorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []models.CheckResult
	for rows.Next() {
		var r models.CheckResult
		rows.Scan(&r.ID, &r.MonitorID, &r.Status, &r.StatusCode, &r.ResponseTimeMs, &r.ErrorMessage, &r.CheckedAt)
		results = append(results, r)
	}
	return results, nil
}

// --- Incident CRUD ---

func (d *DB) ListIncidents(activeOnly bool) ([]models.Incident, error) {
	query := `SELECT id, title, description, status, severity, monitor_id, created_at, updated_at, resolved_at FROM incidents`
	if activeOnly {
		query += ` WHERE status != 'resolved'`
	}
	query += ` ORDER BY created_at DESC LIMIT 50`
	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIncidents(rows)
}

func (d *DB) CreateIncident(i *models.Incident) (int64, error) {
	result, err := d.conn.Exec(`INSERT INTO incidents (title, description, status, severity, monitor_id) VALUES (?, ?, ?, ?, ?)`, i.Title, i.Description, i.Status, i.Severity, i.MonitorID)
	if err != nil {
		return 0, err
	}
	id, _ := result.LastInsertId()

	// Auto-create first update
	d.conn.Exec(`INSERT INTO incident_updates (incident_id, status, message) VALUES (?, ?, ?)`, id, i.Status, i.Description)
	return id, nil
}

func (d *DB) UpdateIncidentStatus(id int64, status, message string) error {
	_, err := d.conn.Exec(`UPDATE incidents SET status=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, status, id)
	if err != nil {
		return err
	}
	_, err = d.conn.Exec(`INSERT INTO incident_updates (incident_id, status, message) VALUES (?, ?, ?)`, id, status, message)
	return err
}

func (d *DB) ResolveIncident(id int64, message string) error {
	_, err := d.conn.Exec(`UPDATE incidents SET status='resolved', resolved_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, id)
	if err != nil {
		return err
	}
	_, err = d.conn.Exec(`INSERT INTO incident_updates (incident_id, status, message) VALUES (?, 'resolved', ?)`, id, message)
	return err
}

func (d *DB) GetIncidentUpdates(incidentID int64) ([]models.IncidentUpdate, error) {
	rows, err := d.conn.Query(`SELECT id, incident_id, status, message, created_at FROM incident_updates WHERE incident_id=? ORDER BY created_at ASC`, incidentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var updates []models.IncidentUpdate
	for rows.Next() {
		var u models.IncidentUpdate
		rows.Scan(&u.ID, &u.IncidentID, &u.Status, &u.Message, &u.CreatedAt)
		updates = append(updates, u)
	}
	return updates, nil
}

// --- Subscriber CRUD ---

func (d *DB) ListSubscribers() ([]models.Subscriber, error) {
	rows, err := d.conn.Query(`SELECT id, email, webhook_url, type, active, created_at FROM subscribers WHERE active=1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []models.Subscriber
	for rows.Next() {
		var s models.Subscriber
		rows.Scan(&s.ID, &s.Email, &s.WebhookURL, &s.Type, &s.Active, &s.CreatedAt)
		subs = append(subs, s)
	}
	return subs, nil
}

func (d *DB) CreateSubscriber(s *models.Subscriber) (int64, error) {
	result, err := d.conn.Exec(`INSERT INTO subscribers (email, webhook_url, type) VALUES (?, ?, ?)`, s.Email, s.WebhookURL, s.Type)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// --- Maintenance Windows ---

func (d *DB) ActiveMaintenance() ([]models.MaintenanceWindow, error) {
	rows, err := d.conn.Query(`SELECT id, title, start_at, end_at, recurring, created_at FROM maintenance_windows WHERE end_at > CURRENT_TIMESTAMP ORDER BY start_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var windows []models.MaintenanceWindow
	for rows.Next() {
		var m models.MaintenanceWindow
		rows.Scan(&m.ID, &m.Title, &m.StartAt, &m.EndAt, &m.Recurring, &m.CreatedAt)
		windows = append(windows, m)
	}
	return windows, nil
}

func (d *DB) CreateMaintenanceWindow(m *models.MaintenanceWindow) error {
	_, err := d.conn.Exec(`INSERT INTO maintenance_windows (title, start_at, end_at, recurring) VALUES (?, ?, ?, ?)`, m.Title, m.StartAt, m.EndAt, m.Recurring)
	return err
}

func scanIncidents(rows *sql.Rows) ([]models.Incident, error) {
	var incidents []models.Incident
	for rows.Next() {
		var i models.Incident
		var monitorID sql.NullInt64
		var resolvedAt sql.NullTime
		rows.Scan(&i.ID, &i.Title, &i.Description, &i.Status, &i.Severity, &monitorID, &i.CreatedAt, &i.UpdatedAt, &resolvedAt)
		if monitorID.Valid {
			i.MonitorID = &monitorID.Int64
		}
		if resolvedAt.Valid {
			i.ResolvedAt = &resolvedAt.Time
		}
		incidents = append(incidents, i)
	}
	return incidents, nil
}
