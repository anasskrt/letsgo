package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type OpenMeteoResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Hourly    struct {
		Time           []string  `json:"time"`
		Temperature2m  []float64 `json:"temperature_2m"`
		WindSpeed10m   []float64 `json:"wind_speed_10m"`
		Precipitation  []float64 `json:"precipitation"`
	} `json:"hourly"`
}

type Station struct {
	Name      string
	Latitude  float64
	Longitude float64
}

func main() {
	stations := []Station{
		{Name: "Paris", Latitude: 48.8567, Longitude: 2.3522},
		{Name: "London", Latitude: 51.5074, Longitude: -0.1278},
		{Name: "Berlin", Latitude: 52.5200, Longitude: 13.4050},
		{Name: "Madrid", Latitude: 40.4168, Longitude: -3.7038},
		{Name: "Rome", Latitude: 41.9028, Longitude: 12.4964},
		{Name: "Vienna", Latitude: 48.2082, Longitude: 16.3738},
		{Name: "Warsaw", Latitude: 52.2297, Longitude: 21.0122},
		{Name: "Stockholm", Latitude: 59.3293, Longitude: 18.0686},
		{Name: "Oslo", Latitude: 59.9139, Longitude: 10.7522},
		{Name: "Athens", Latitude: 37.9838, Longitude: 23.7275},
		{Name: "Lisbon", Latitude: 38.7223, Longitude: -9.1393},
		{Name: "Brussels", Latitude: 50.8503, Longitude: 4.3517},
		{Name: "Zurich", Latitude: 47.3769, Longitude: 8.5417},
		{Name: "Prague", Latitude: 50.0755, Longitude: 14.4378},
		{Name: "Budapest", Latitude: 47.4979, Longitude: 19.0402},
	}

	client := &http.Client{Timeout: 30 * time.Second}

	for _, s := range stations {
		url := fmt.Sprintf(
			"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&hourly=temperature_2m,wind_speed_10m,precipitation&past_days=30",
			s.Latitude, s.Longitude,
		)

		resp, err := client.Get(url)
		if err != nil {
			log.Printf("ERROR fetching %s: %v", s.Name, err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("ERROR reading response for %s: %v", s.Name, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("ERROR %s returned status %d: %s", s.Name, resp.StatusCode, string(body))
			continue
		}

		var data OpenMeteoResponse
		if err := json.Unmarshal(body, &data); err != nil {
			log.Printf("ERROR parsing JSON for %s: %v", s.Name, err)
			continue
		}

		fmt.Printf("--- %s (%.4f, %.4f) ---\n", s.Name, data.Latitude, data.Longitude)
		obs := len(data.Hourly.Time)
		if obs > 0 {
			fmt.Printf("  Observations: %d\n", obs)
			fmt.Printf("  First: %s  T=%.1f°C  Wind=%.1fkm/h  Precip=%.1fmm\n",
				data.Hourly.Time[0],
				data.Hourly.Temperature2m[0],
				data.Hourly.WindSpeed10m[0],
				data.Hourly.Precipitation[0],
			)
			last := obs - 1
			fmt.Printf("  Last:  %s  T=%.1f°C  Wind=%.1fkm/h  Precip=%.1fmm\n",
				data.Hourly.Time[last],
				data.Hourly.Temperature2m[last],
				data.Hourly.WindSpeed10m[last],
				data.Hourly.Precipitation[last],
			)
		}
		fmt.Println()
	}

	fmt.Println("Ingestion complete.")
	os.Exit(0)
}
