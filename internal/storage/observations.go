package storage

import (
	"database/sql"
	"fmt"
	"time"

	"letsgo/internal/model"
)

type ObservationRepo struct {
	db *sql.DB
}

func (r *ObservationRepo) DB() *sql.DB { return r.db }

func NewObservationRepo(db *sql.DB) *ObservationRepo {
	return &ObservationRepo{db: db}
}

func (r *ObservationRepo) BulkInsert(obs []model.Observation) error {
	if len(obs) == 0 {
		return nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO observations (station_id, timestamp, temperature_2m, wind_speed_10m, wind_gusts_10m, precipitation, pressure_msl, relative_humidity_2m, cloud_cover, weather_code, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (station_id, timestamp) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	for _, o := range obs {
		_, err := stmt.Exec(o.StationID, o.Timestamp, o.Temperature2m, o.WindSpeed10m, o.WindGusts10m, o.Precipitation, o.PressureMsl, o.RelativeHumidity, o.CloudCover, o.WeatherCode, o.Source)
		if err != nil {
			return fmt.Errorf("insert obs: %w", err)
		}
	}
	return tx.Commit()
}

func (r *ObservationRepo) ListByStation(stationID int, from, to time.Time, limit, offset int) ([]model.Observation, error) {
	query := `SELECT id, station_id, timestamp, temperature_2m, wind_speed_10m, wind_gusts_10m, precipitation, pressure_msl, relative_humidity_2m, cloud_cover, weather_code, source
		FROM observations WHERE station_id = $1`
	args := []any{stationID}
	argIdx := 2

	if !from.IsZero() {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIdx)
		args = append(args, from)
		argIdx++
	}
	if !to.IsZero() {
		query += fmt.Sprintf(" AND timestamp <= $%d", argIdx)
		args = append(args, to)
		argIdx++
	}
	query += " ORDER BY timestamp ASC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, offset)
		argIdx++
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list observations: %w", err)
	}
	defer rows.Close()

	var result []model.Observation
	for rows.Next() {
		var o model.Observation
		if err := rows.Scan(&o.ID, &o.StationID, &o.Timestamp, &o.Temperature2m, &o.WindSpeed10m, &o.WindGusts10m, &o.Precipitation, &o.PressureMsl, &o.RelativeHumidity, &o.CloudCover, &o.WeatherCode, &o.Source); err != nil {
			return nil, fmt.Errorf("scan obs: %w", err)
		}
		result = append(result, o)
	}
	return result, nil
}

func (r *ObservationRepo) AggregateDaily(stationID int) ([]model.AggregatedDaily, error) {
	query := `
		SELECT
			DATE(timestamp) AS day,
			AVG(temperature_2m) AS avg_temp,
			MAX(temperature_2m) AS max_temp,
			MIN(temperature_2m) AS min_temp,
			SUM(precipitation) AS total_precip,
			AVG(wind_speed_10m) AS avg_wind,
			MAX(wind_gusts_10m) AS max_gust
		FROM observations
		WHERE station_id = $1
		GROUP BY DATE(timestamp)
		ORDER BY day
	`
	rows, err := r.db.Query(query, stationID)
	if err != nil {
		return nil, fmt.Errorf("aggregate daily: %w", err)
	}
	defer rows.Close()

	var result []model.AggregatedDaily
	for rows.Next() {
		var a model.AggregatedDaily
		if err := rows.Scan(&a.Day, &a.AvgTemp, &a.MaxTemp, &a.MinTemp, &a.TotalPrecip, &a.AvgWind, &a.MaxGust); err != nil {
			return nil, fmt.Errorf("scan daily: %w", err)
		}
		result = append(result, a)
	}
	return result, nil
}
