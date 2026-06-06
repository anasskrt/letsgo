package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL string
	ServerPort  string

	OpenMeteoBaseURL string
	OpenMeteoRate    time.Duration
	PastDays         int
	MaxRetries       int

	StormWindThreshold   float64
	StormMinHours        int
	HeatwaveTempMin      float64
	HeatwaveMinDays      int
	ColdwaveTempMax      float64
	ColdwaveMinDays      int
	FloodPrecipThreshold float64
	FloodWindowHours     int
}

func Load() Config {
	return Config{
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://weather:weather@localhost:5432/weather?sslmode=disable"),
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		OpenMeteoBaseURL: getEnv("OPENMETEO_BASE_URL", "https://api.open-meteo.com/v1"),
		OpenMeteoRate:    getDurationEnv("OPENMETEO_RATE_MS", 500),
		PastDays:         getIntEnv("PAST_DAYS", 30),
		MaxRetries:       getIntEnv("MAX_RETRIES", 3),
		StormWindThreshold:   getFloatEnv("STORM_WIND_THRESHOLD", 80),
		StormMinHours:        getIntEnv("STORM_MIN_HOURS", 3),
		HeatwaveTempMin:      getFloatEnv("HEATWAVE_TEMP_MIN", 35),
		HeatwaveMinDays:      getIntEnv("HEATWAVE_MIN_DAYS", 3),
		ColdwaveTempMax:      getFloatEnv("COLDWAVE_TEMP_MAX", 0),
		ColdwaveMinDays:      getIntEnv("COLDWAVE_MIN_DAYS", 3),
		FloodPrecipThreshold: getFloatEnv("FLOOD_PRECIP_THRESHOLD", 50),
		FloodWindowHours:     getIntEnv("FLOOD_WINDOW_HOURS", 24),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getFloatEnv(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getDurationEnv(key string, fallbackMs int) time.Duration {
	if v := os.Getenv(key); v != "" {
		if ms, err := strconv.Atoi(v); err == nil {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return time.Duration(fallbackMs) * time.Millisecond
}
