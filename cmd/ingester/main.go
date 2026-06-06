package main

import (
	"log/slog"
	"os"
	"sync"
	"time"

	"letsgo/internal/config"
	"letsgo/internal/openmeteo"
	"letsgo/internal/storage"
)

type rawStation struct {
	Name        string
	CountryCode string
	Latitude    float64
	Longitude   float64
}

var stations = []rawStation{
	{Name: "Paris", CountryCode: "FR", Latitude: 48.8567, Longitude: 2.3522},
	{Name: "London", CountryCode: "GB", Latitude: 51.5074, Longitude: -0.1278},
	{Name: "Berlin", CountryCode: "DE", Latitude: 52.5200, Longitude: 13.4050},
	{Name: "Madrid", CountryCode: "ES", Latitude: 40.4168, Longitude: -3.7038},
	{Name: "Rome", CountryCode: "IT", Latitude: 41.9028, Longitude: 12.4964},
	{Name: "Vienna", CountryCode: "AT", Latitude: 48.2082, Longitude: 16.3738},
	{Name: "Warsaw", CountryCode: "PL", Latitude: 52.2297, Longitude: 21.0122},
	{Name: "Stockholm", CountryCode: "SE", Latitude: 59.3293, Longitude: 18.0686},
	{Name: "Oslo", CountryCode: "NO", Latitude: 59.9139, Longitude: 10.7522},
	{Name: "Athens", CountryCode: "GR", Latitude: 37.9838, Longitude: 23.7275},
	{Name: "Lisbon", CountryCode: "PT", Latitude: 38.7223, Longitude: -9.1393},
	{Name: "Brussels", CountryCode: "BE", Latitude: 50.8503, Longitude: 4.3517},
	{Name: "Zurich", CountryCode: "CH", Latitude: 47.3769, Longitude: 8.5417},
	{Name: "Prague", CountryCode: "CZ", Latitude: 50.0755, Longitude: 14.4378},
	{Name: "Budapest", CountryCode: "HU", Latitude: 47.4979, Longitude: 19.0402},
	{Name: "Helsinki", CountryCode: "FI", Latitude: 60.1699, Longitude: 24.9384},
	{Name: "Dublin", CountryCode: "IE", Latitude: 53.3498, Longitude: -6.2603},
	{Name: "Copenhagen", CountryCode: "DK", Latitude: 55.6761, Longitude: 12.5683},
	{Name: "Amsterdam", CountryCode: "NL", Latitude: 52.3676, Longitude: 4.9041},
	{Name: "Lyon", CountryCode: "FR", Latitude: 45.7640, Longitude: 4.8357},
}

type result struct {
	Name string
	Err  error
}

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	db, err := storage.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := storage.RunMigrations(db, "migrations"); err != nil {
		slog.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	client := openmeteo.NewClient(cfg.OpenMeteoBaseURL, cfg.OpenMeteoRate, cfg.MaxRetries)
	stationRepo := storage.NewStationRepo(db)
	obsRepo := storage.NewObservationRepo(db)

	start := time.Now()
	numWorkers := 4
	stationCh := make(chan rawStation, len(stations))
	resultCh := make(chan result, len(stations))

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for rs := range stationCh {
				slog.Info("fetching station", "name", rs.Name)
				resp, err := client.FetchForecast(rs.Latitude, rs.Longitude, cfg.PastDays)
				if err != nil {
					slog.Error("fetch failed", "name", rs.Name, "error", err)
					resultCh <- result{Name: rs.Name, Err: err}
					continue
				}

				station := openmeteo.ToStation(rs.Name, rs.CountryCode, *resp)
				if err := stationRepo.Upsert(&station); err != nil {
					slog.Error("station upsert failed", "name", rs.Name, "error", err)
					resultCh <- result{Name: rs.Name, Err: err}
					continue
				}

				obs := openmeteo.ToObservations(station.ID, *resp)
				slog.Info("observations mapped", "station", rs.Name, "count", len(obs))

				if err := obsRepo.BulkInsert(obs); err != nil {
					slog.Error("observation insert failed", "name", rs.Name, "error", err)
					resultCh <- result{Name: rs.Name, Err: err}
					continue
				}

				resultCh <- result{Name: rs.Name, Err: nil}
			}
		}()
	}

	go func() {
		for _, rs := range stations {
			stationCh <- rs
		}
		close(stationCh)
	}()

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	totalCalls := 0
	totalErrors := 0
	for res := range resultCh {
		totalCalls++
		if res.Err != nil {
			totalErrors++
		}
	}

	elapsed := time.Since(start)
	slog.Info("ingestion complete",
		"duration", elapsed.String(),
		"stations", len(stations),
		"calls", totalCalls,
		"errors", totalErrors,
	)
}
