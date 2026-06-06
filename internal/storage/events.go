package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"letsgo/internal/model"
)

type EventRepo struct {
	db *sql.DB
}

func NewEventRepo(db *sql.DB) *EventRepo {
	return &EventRepo{db: db}
}

func (r *EventRepo) Insert(e *model.Event) error {
	meta, _ := json.Marshal(e.Metadata)
	err := r.db.QueryRow(`
		INSERT INTO events (type, station_id, started_at, ended_at, severity, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, e.Type, e.StationID, e.StartedAt, e.EndedAt, e.Severity, meta).Scan(&e.ID, &e.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}

func (r *EventRepo) List(eventType string, from, to time.Time) ([]model.Event, error) {
	query := `SELECT id, type, station_id, started_at, ended_at, severity, metadata, created_at FROM events WHERE 1=1`
	args := []any{}
	argIdx := 1

	if eventType != "" {
		query += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, eventType)
		argIdx++
	}
	if !from.IsZero() {
		query += fmt.Sprintf(" AND started_at >= $%d", argIdx)
		args = append(args, from)
		argIdx++
	}
	if !to.IsZero() {
		query += fmt.Sprintf(" AND ended_at <= $%d", argIdx)
		args = append(args, to)
		argIdx++
	}

	query += " ORDER BY started_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

func (r *EventRepo) GetByID(id int) (*model.Event, error) {
	var e model.Event
	var meta []byte
	err := r.db.QueryRow(`
		SELECT id, type, station_id, started_at, ended_at, severity, metadata, created_at
		FROM events WHERE id = $1
	`, id).Scan(&e.ID, &e.Type, &e.StationID, &e.StartedAt, &e.EndedAt, &e.Severity, &meta, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get event %d: %w", id, err)
	}
	json.Unmarshal(meta, &e.Metadata)
	return &e, nil
}

func (r *EventRepo) ListByStation(stationID int) ([]model.Event, error) {
	rows, err := r.db.Query(`
		SELECT id, type, station_id, started_at, ended_at, severity, metadata, created_at
		FROM events WHERE station_id = $1 ORDER BY started_at DESC
	`, stationID)
	if err != nil {
		return nil, fmt.Errorf("list station events: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (r *EventRepo) Stats(eventType, country string) (map[string]int, error) {
	query := `SELECT e.type, COUNT(*) FROM events e`
	args := []any{}
	argIdx := 1
	whereClauses := []string{}

	if eventType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("e.type = $%d", argIdx))
		args = append(args, eventType)
		argIdx++
	}
	if country != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("EXISTS (SELECT 1 FROM stations s WHERE s.id = e.station_id AND s.country_code = $%d)", argIdx))
		args = append(args, strings.ToUpper(country))
		argIdx++
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " GROUP BY e.type"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("event stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var t string
		var count int
		if err := rows.Scan(&t, &count); err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
		stats[t] = count
	}
	return stats, nil
}

func scanEvents(rows *sql.Rows) ([]model.Event, error) {
	events := []model.Event{}
	for rows.Next() {
		var e model.Event
		var meta []byte
		if err := rows.Scan(&e.ID, &e.Type, &e.StationID, &e.StartedAt, &e.EndedAt, &e.Severity, &meta, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		json.Unmarshal(meta, &e.Metadata)
		events = append(events, e)
	}
	return events, nil
}
