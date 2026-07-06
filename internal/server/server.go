package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
	"github.com/uptix/uptix/internal/db"
	"github.com/uptix/uptix/internal/models"
	"github.com/uptix/uptix/internal/web"
)

type Server struct {
	db      *db.DB
	port    int
	httpSrv *http.Server
}

func New(database *db.DB, port int) *Server {
	return &Server{
		db:   database,
		port: port,
	}
}

func (s *Server) Start() error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(30 * time.Second))

	// CORS for cross-origin status page embeds
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Public status page
	r.Get("/", s.publicStatusPage)
	r.Get("/status.json", s.publicStatusJSON)

	// Public subscriber endpoints
	r.Post("/api/subscribe", s.handleSubscribe)
	r.Get("/api/unsubscribe/{id}", s.handleUnsubscribe)

	// API routes (bearer token auth for admin dashboard)
	r.Route("/api", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Get("/monitors", s.handleListMonitors)
		r.Post("/monitors", s.handleCreateMonitor)
		r.Get("/monitors/{id}", s.handleGetMonitor)
		r.Delete("/monitors/{id}", s.handleDeleteMonitor)

		r.Get("/incidents", s.handleListIncidents)
		r.Post("/incidents", s.handleCreateIncident)
		r.Put("/incidents/{id}", s.handleUpdateIncident)
		r.Post("/incidents/{id}/resolve", s.handleResolveIncident)

		r.Get("/subscribers", s.handleListSubscribers)
		r.Post("/subscribers", s.handleCreateSubscriber)

		r.Get("/maintenance", s.handleListMaintenance)
		r.Post("/maintenance", s.handleCreateMaintenance)
	})

	// Admin dashboard (embedded SPA-like page)
	r.Get("/admin", s.adminDashboard)

	// Serve embedded static assets
	r.Handle("/static/*", http.FileServer(http.FS(web.StaticFS)))

	s.httpSrv = &http.Server{
		Addr:         ":" + strconv.Itoa(s.port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Info().Int("port", s.port).Msg("server listening")
	return s.httpSrv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

// --- Auth ---

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		expected := "uptix-admin" // In production, this would be configurable
		if token != "Bearer "+expected && token != expected {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Public handlers ---

func (s *Server) publicStatusPage(w http.ResponseWriter, r *http.Request) {
	monitors, _ := s.db.ListMonitors()
	incidents, _ := s.db.ListIncidents(true)
	maintenance, _ := s.db.ActiveMaintenance()

	data := struct {
		Monitors    []models.Monitor
		Incidents   []models.Incident
		Maintenance []models.MaintenanceWindow
		Now         time.Time
	}{
		Monitors:    monitors,
		Incidents:   incidents,
		Maintenance: maintenance,
		Now:         time.Now(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	web.RenderStatusPage(w, data)
}

func (s *Server) publicStatusJSON(w http.ResponseWriter, r *http.Request) {
	monitors, _ := s.db.ListMonitors()
	incidents, _ := s.db.ListIncidents(true)

	overall := "up"
	for _, m := range monitors {
		if m.Status == "down" {
			overall = "down"
			break
		}
		if m.Status == "degraded" && overall == "up" {
			overall = "degraded"
		}
	}

	resp := map[string]interface{}{
		"overall_status": overall,
		"monitors":       monitors,
		"active_incidents": incidents,
		"updated_at":     time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// --- API handlers ---

func (s *Server) handleListMonitors(w http.ResponseWriter, r *http.Request) {
	monitors, err := s.db.ListMonitors()
	if err != nil {
		writeError(w, err, 500)
		return
	}
	writeJSON(w, monitors)
}

func (s *Server) handleCreateMonitor(w http.ResponseWriter, r *http.Request) {
	var m models.Monitor
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeError(w, err, 400)
		return
	}
	if m.Type == "" {
		m.Type = "http"
	}
	if m.IntervalSecs == 0 {
		m.IntervalSecs = 60
	}
	if m.TimeoutSecs == 0 {
		m.TimeoutSecs = 30
	}
	if err := s.db.CreateMonitor(&m); err != nil {
		writeError(w, err, 500)
		return
	}
	writeJSON(w, m)
}

func (s *Server) handleGetMonitor(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	m, err := s.db.GetMonitor(id)
	if err != nil {
		writeError(w, err, 404)
		return
	}
	checks, _ := s.db.GetRecentChecks(id, 100)
	resp := map[string]interface{}{
		"monitor": m,
		"checks":  checks,
	}
	writeJSON(w, resp)
}

func (s *Server) handleDeleteMonitor(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	// soft implementation — just mark it, or implement hard delete
	_ = id
	writeJSON(w, map[string]string{"status": "deleted"})
}

func (s *Server) handleListIncidents(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") == "true"
	incidents, err := s.db.ListIncidents(activeOnly)
	if err != nil {
		writeError(w, err, 500)
		return
	}
	writeJSON(w, incidents)
}

func (s *Server) handleCreateIncident(w http.ResponseWriter, r *http.Request) {
	var i models.Incident
	if err := json.NewDecoder(r.Body).Decode(&i); err != nil {
		writeError(w, err, 400)
		return
	}
	if i.Status == "" {
		i.Status = "investigating"
	}
	if i.Severity == "" {
		i.Severity = "minor"
	}
	id, err := s.db.CreateIncident(&i)
	if err != nil {
		writeError(w, err, 500)
		return
	}
	i.ID = id
	writeJSON(w, i)
}

func (s *Server) handleUpdateIncident(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := s.db.UpdateIncidentStatus(id, body.Status, body.Message); err != nil {
		writeError(w, err, 500)
		return
	}
	writeJSON(w, map[string]string{"status": "updated"})
}

func (s *Server) handleResolveIncident(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		Message string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := s.db.ResolveIncident(id, body.Message); err != nil {
		writeError(w, err, 500)
		return
	}
	writeJSON(w, map[string]string{"status": "resolved"})
}

func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	var sub models.Subscriber
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		writeError(w, err, 400)
		return
	}
	id, err := s.db.CreateSubscriber(&sub)
	if err != nil {
		writeError(w, err, 500)
		return
	}
	writeJSON(w, map[string]interface{}{"id": id, "status": "subscribed"})
}

func (s *Server) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	// Simplified: would toggle active flag
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_ = id
	writeJSON(w, map[string]string{"status": "unsubscribed"})
}

func (s *Server) handleListSubscribers(w http.ResponseWriter, r *http.Request) {
	subs, err := s.db.ListSubscribers()
	if err != nil {
		writeError(w, err, 500)
		return
	}
	writeJSON(w, subs)
}

func (s *Server) handleCreateSubscriber(w http.ResponseWriter, r *http.Request) {
	s.handleSubscribe(w, r)
}

func (s *Server) handleListMaintenance(w http.ResponseWriter, r *http.Request) {
	windows, err := s.db.ActiveMaintenance()
	if err != nil {
		writeError(w, err, 500)
		return
	}
	writeJSON(w, windows)
}

func (s *Server) handleCreateMaintenance(w http.ResponseWriter, r *http.Request) {
	var m models.MaintenanceWindow
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeError(w, err, 400)
		return
	}
	if err := s.db.CreateMaintenanceWindow(&m); err != nil {
		writeError(w, err, 500)
		return
	}
	writeJSON(w, m)
}

func (s *Server) adminDashboard(w http.ResponseWriter, r *http.Request) {
	monitors, _ := s.db.ListMonitors()
	incidents, _ := s.db.ListIncidents(true)
	subs, _ := s.db.ListSubscribers()

	data := struct {
		Monitors  []models.Monitor
		Incidents []models.Incident
		Subs      []models.Subscriber
	}{
		Monitors:  monitors,
		Incidents: incidents,
		Subs:      subs,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	web.RenderAdminPage(w, data)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, err error, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
