package api

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"letsgo/internal/config"
	"letsgo/internal/detector"
	"letsgo/internal/storage"
)

type Server struct {
	db       *sql.DB
	stations *storage.StationRepo
	obs      *storage.ObservationRepo
	events   *storage.EventRepo
	detector *detector.Engine
	cfg      config.Config
}

func NewServer(db *sql.DB, cfg config.Config) *Server {
	s := &Server{
		db:       db,
		stations: storage.NewStationRepo(db),
		obs:      storage.NewObservationRepo(db),
		events:   storage.NewEventRepo(db),
		cfg:      cfg,
	}
	s.detector = detector.NewEngine(s.obs, s.events, cfg)
	return s
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /ready", s.handleReady)

	mux.HandleFunc("GET /stations", s.handleListStations)
	mux.HandleFunc("GET /stations/{id}", s.handleGetStation)
	mux.HandleFunc("GET /stations/{id}/observations", s.handleStationObservations)
	mux.HandleFunc("GET /stations/{id}/events", s.handleStationEvents)

	mux.HandleFunc("GET /observations/aggregate", s.handleObservationsAggregate)

	mux.HandleFunc("GET /events", s.handleListEvents)
	mux.HandleFunc("GET /events/{id}", s.handleGetEvent)
	mux.HandleFunc("GET /events/stats", s.handleEventStats)

	mux.HandleFunc("POST /detect", s.handleDetect)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "" || r.URL.Path != "/" {
			writeError(w, http.StatusNotFound, "not_found", "route not found")
		}
	})

	return withMiddleware(slog.Default(), mux)
}

func withMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
		if r.Method != "" {
			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"duration", time.Since(start).String(),
			)
		}
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "message": message})
}
