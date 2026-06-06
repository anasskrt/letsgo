package api

import (
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "db_unavailable", "database not reachable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleListStations(w http.ResponseWriter, r *http.Request) {
	country := r.URL.Query().Get("country")
	bboxStr := r.URL.Query().Get("bbox")

	var bbox []float64
	if bboxStr != "" {
		parts := strings.Split(bboxStr, ",")
		if len(parts) == 4 {
			for _, p := range parts {
				v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
				if err != nil {
					writeError(w, http.StatusBadRequest, "invalid_bbox", "bbox must be lat1,lon1,lat2,lon2")
					return
				}
				bbox = append(bbox, v)
			}
		}
	}

	stations, err := s.stations.List(country, bbox)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stations)
}

func (s *Server) handleGetStation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "station id must be an integer")
		return
	}

	station, err := s.stations.GetByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "station not found")
		return
	}
	writeJSON(w, http.StatusOK, station)
}
