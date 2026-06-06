package api

import (
	"net/http"
	"strconv"
	"time"

	"letsgo/internal/model"
)

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	eventType := q.Get("type")

	var from, to time.Time
	if f := q.Get("from"); f != "" {
		var err error
		from, err = time.Parse("2006-01-02", f)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_from", "from must be YYYY-MM-DD")
			return
		}
	}
	if t := q.Get("to"); t != "" {
		var err error
		to, err = time.Parse("2006-01-02", t)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_to", "to must be YYYY-MM-DD")
			return
		}
	}

	events, err := s.events.List(eventType, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	if events == nil {
		events = []model.Event{}
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleGetEvent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "event id must be an integer")
		return
	}

	event, err := s.events.GetByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "event not found")
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (s *Server) handleStationEvents(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "station id must be an integer")
		return
	}

	events, err := s.events.ListByStation(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleEventStats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	eventType := q.Get("type")
	country := q.Get("country")

	stats, err := s.events.Stats(eventType, country)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleDetect(w http.ResponseWriter, r *http.Request) {
	count, err := s.detector.Run()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "detect_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"detected": count})
}
