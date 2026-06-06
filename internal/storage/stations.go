package storage

import (
	"database/sql"
	"fmt"
	"strings"

	"letsgo/internal/model"
)

type StationRepo struct {
	db *sql.DB
}

func NewStationRepo(db *sql.DB) *StationRepo {
	return &StationRepo{db: db}
}

func (r *StationRepo) Upsert(station *model.Station) error {
	err := r.db.QueryRow(`
		INSERT INTO stations (name, country_code, latitude, longitude, elevation, source, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			country_code = EXCLUDED.country_code,
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			elevation = EXCLUDED.elevation,
			updated_at = now()
		RETURNING id
	`, station.Name, station.CountryCode, station.Latitude, station.Longitude, station.Elevation, station.Source).Scan(&station.ID)
	if err != nil {
		return fmt.Errorf("upsert station: %w", err)
	}
	return nil
}

func (r *StationRepo) GetByID(id int) (*model.Station, error) {
	var s model.Station
	err := r.db.QueryRow(`
		SELECT id, name, country_code, latitude, longitude, elevation, source, created_at, updated_at
		FROM stations WHERE id = $1
	`, id).Scan(&s.ID, &s.Name, &s.CountryCode, &s.Latitude, &s.Longitude, &s.Elevation, &s.Source, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get station %d: %w", id, err)
	}
	return &s, nil
}

func (r *StationRepo) List(country string, bbox []float64) ([]model.Station, error) {
	query := `SELECT id, name, country_code, latitude, longitude, elevation, source, created_at, updated_at FROM stations WHERE 1=1`
	args := []any{}
	argIdx := 1

	if country != "" {
		query += fmt.Sprintf(" AND country_code = $%d", argIdx)
		args = append(args, strings.ToUpper(country))
		argIdx++
	}
	if len(bbox) == 4 {
		query += fmt.Sprintf(" AND latitude >= $%d AND latitude <= $%d AND longitude >= $%d AND longitude <= $%d",
			argIdx, argIdx+1, argIdx+2, argIdx+3)
		args = append(args, bbox[0], bbox[1], bbox[2], bbox[3])
		argIdx += 4
	}

	query += " ORDER BY name"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list stations: %w", err)
	}
	defer rows.Close()

	var stations []model.Station
	for rows.Next() {
		var s model.Station
		if err := rows.Scan(&s.ID, &s.Name, &s.CountryCode, &s.Latitude, &s.Longitude, &s.Elevation, &s.Source, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan station: %w", err)
		}
		stations = append(stations, s)
	}
	return stations, nil
}
