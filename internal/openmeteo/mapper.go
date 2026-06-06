package openmeteo

import (
	"time"

	"letsgo/internal/model"
)

func ToStation(name, countryCode string, src ForecastResponse) model.Station {
	return model.Station{
		Name:        name,
		CountryCode: countryCode,
		Latitude:    src.Latitude,
		Longitude:   src.Longitude,
		Elevation:   src.Elevation,
		Source:      "open-meteo",
	}
}

func ToObservations(stationID int, src ForecastResponse) []model.Observation {
	n := len(src.Hourly.Time)
	if n == 0 {
		return nil
	}
	obs := make([]model.Observation, 0, n)
	for i := 0; i < n; i++ {
		ts, err := time.Parse("2006-01-02T15:04", src.Hourly.Time[i])
		if err != nil {
			continue
		}
		o := model.Observation{
			StationID:    stationID,
			Timestamp:    ts,
			Temperature2m: safeFloat(src.Hourly.Temperature2m, i),
			WindSpeed10m:  safeFloat(src.Hourly.WindSpeed10m, i),
			Precipitation: safeFloat(src.Hourly.Precipitation, i),
			PressureMsl:   safeFloat(src.Hourly.PressureMsl, i),
			RelativeHumidity: safeFloat(src.Hourly.RelativeHumidity2m, i),
			CloudCover:    safeFloat(src.Hourly.CloudCover, i),
			WeatherCode:   safeInt(src.Hourly.WeatherCode, i),
			Source:        "open-meteo",
		}
		o.WindGusts10m = safeFloat(src.Hourly.WindGusts10m, i)
		obs = append(obs, o)
	}
	return obs
}

func safeFloat(s []float64, i int) float64 {
	if i < len(s) {
		return s[i]
	}
	return 0
}

func safeInt(s []int, i int) int {
	if i < len(s) {
		return s[i]
	}
	return 0
}

func celsiusToFahrenheit(c float64) float64 {
	return c*9/5 + 32
}

func fahrenheitToCelsius(f float64) float64 {
	return (f - 32) * 5 / 9
}

func kmhToMph(kmh float64) float64 {
	return kmh * 0.621371
}

func mphToKmh(mph float64) float64 {
	return mph / 0.621371
}

func hpaToPa(hpa float64) float64 {
	return hpa * 100
}

func paToHpa(pa float64) float64 {
	return pa / 100
}
