package openmeteo

import (
	"testing"
)

func TestToStation(t *testing.T) {
	resp := ForecastResponse{
		Latitude:  48.8567,
		Longitude: 2.3522,
		Elevation: 35.0,
	}
	s := ToStation("Paris", "FR", resp)
	if s.Name != "Paris" {
		t.Errorf("expected Paris, got %s", s.Name)
	}
	if s.Latitude != 48.8567 {
		t.Errorf("expected 48.8567, got %f", s.Latitude)
	}
	if s.Source != "open-meteo" {
		t.Errorf("expected open-meteo, got %s", s.Source)
	}
}

func TestToObservations(t *testing.T) {
	resp := ForecastResponse{
		Latitude:  48.8567,
		Longitude: 2.3522,
		Hourly: HourlyResponse{
			Time:          []string{"2026-06-01T00:00", "2026-06-01T01:00"},
			Temperature2m: []float64{20.5, 21.0},
			WindSpeed10m:  []float64{10.0, 12.0},
			WindGusts10m:  []float64{15.0, 18.0},
			Precipitation: []float64{0.0, 0.5},
		},
	}
	obs := ToObservations(1, resp)
	if len(obs) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(obs))
	}
	if obs[0].Temperature2m != 20.5 {
		t.Errorf("expected 20.5, got %f", obs[0].Temperature2m)
	}
	if obs[0].StationID != 1 {
		t.Errorf("expected station 1, got %d", obs[0].StationID)
	}
	if obs[1].WindSpeed10m != 12.0 {
		t.Errorf("expected 12.0, got %f", obs[1].WindSpeed10m)
	}
}

func TestConversions(t *testing.T) {
	if got := celsiusToFahrenheit(0); got != 32 {
		t.Errorf("0°C -> °F: expected 32, got %f", got)
	}
	if got := celsiusToFahrenheit(100); got != 212 {
		t.Errorf("100°C -> °F: expected 212, got %f", got)
	}
	if got := fahrenheitToCelsius(32); got != 0 {
		t.Errorf("32°F -> °C: expected 0, got %f", got)
	}
	if got := kmhToMph(100); got < 62.13 || got > 62.14 {
		t.Errorf("100 km/h -> mph: expected ~62.14, got %f", got)
	}
	if got := mphToKmh(62.1371); got < 99.99 || got > 100.01 {
		t.Errorf("62.14 mph -> km/h: expected ~100, got %f", got)
	}
	if got := hpaToPa(1013.25); got != 101325 {
		t.Errorf("1013.25 hPa -> Pa: expected 101325, got %f", got)
	}
	if got := paToHpa(101325); got != 1013.25 {
		t.Errorf("101325 Pa -> hPa: expected 1013.25, got %f", got)
	}
}
