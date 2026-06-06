package model

import "time"

type Station struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	CountryCode  string    `json:"country_code"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	Elevation    float64   `json:"elevation"`
	Source       string    `json:"source"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Observation struct {
	ID              int       `json:"id"`
	StationID       int       `json:"station_id"`
	Timestamp       time.Time `json:"timestamp"`
	Temperature2m   float64   `json:"temperature_2m"`
	WindSpeed10m    float64   `json:"wind_speed_10m"`
	WindGusts10m    float64   `json:"wind_gusts_10m"`
	Precipitation   float64   `json:"precipitation"`
	PressureMsl     float64   `json:"pressure_msl"`
	RelativeHumidity float64  `json:"relative_humidity_2m"`
	CloudCover      float64   `json:"cloud_cover"`
	WeatherCode     int       `json:"weather_code"`
	Source          string    `json:"source"`
}

type AggregatedDaily struct {
	Day        string  `json:"day"`
	AvgTemp    float64 `json:"avg_temp"`
	MaxTemp    float64 `json:"max_temp"`
	MinTemp    float64 `json:"min_temp"`
	TotalPrecip float64 `json:"total_precip"`
	AvgWind    float64 `json:"avg_wind"`
	MaxGust    float64 `json:"max_gust"`
}

type EventType string

const (
	EventStorm     EventType = "storm"
	EventHeatwave  EventType = "heatwave"
	EventColdwave  EventType = "coldwave"
	EventFlood     EventType = "flood"
)

type Event struct {
	ID        int            `json:"id"`
	Type      EventType      `json:"type"`
	StationID int            `json:"station_id"`
	StartedAt time.Time      `json:"started_at"`
	EndedAt   time.Time      `json:"ended_at"`
	Severity  string         `json:"severity"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
}

type Source struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	BaseURL  string    `json:"base_url"`
	IsActive bool      `json:"is_active"`
}
