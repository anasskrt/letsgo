package api

import (
	"net/http"
	"strconv"
	"time"
)

func (s *Server) handleStationObservations(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "station id must be an integer")
		return
	}

	q := r.URL.Query()
	var from, to time.Time
	if f := q.Get("from"); f != "" {
		from, err = time.Parse("2006-01-02", f)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_from", "from must be YYYY-MM-DD")
			return
		}
	}
	if t := q.Get("to"); t != "" {
		to, err = time.Parse("2006-01-02", t)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_to", "to must be YYYY-MM-DD")
			return
		}
	}

	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	obs, err := s.obs.ListByStation(id, from, to, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, obs)
}

func (s *Server) handleObservationsAggregate(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	period := q.Get("period")
	stationID, err := strconv.Atoi(q.Get("station_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_station_id", "station_id must be an integer")
		return
	}

	if period != "daily" {
		writeError(w, http.StatusBadRequest, "invalid_period", "only daily aggregation is supported")
		return
	}

	result, err := s.obs.AggregateDaily(stationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
