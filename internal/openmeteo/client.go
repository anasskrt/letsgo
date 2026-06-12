// Package openmeteo gere les appels a l'API publique Open-Meteo
// (https://open-meteo.com). C'est la partie "ingestion" du projet :
// elle recupere les donnees brutes, gere les erreurs reseau et renvoie
// la reponse decodee. La conversion vers le modele interne se fait
// ailleurs (partie modele unifie de l'equipe).
package openmeteo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// City est une station qu'on veut suivre. La liste est codee en dur, on
// n'a pas besoin de plus pour le projet.
type City struct {
	Name    string
	Country string
	Lat     float64
	Lon     float64
}

// Cities : une vingtaine de villes europeennes pour avoir de la matiere a
// analyser (le sujet demande au moins 10 stations).
var Cities = []City{
	{"Paris", "FR", 48.85, 2.35},
	{"Marseille", "FR", 43.30, 5.37},
	{"Lyon", "FR", 45.76, 4.84},
	{"Toulouse", "FR", 43.60, 1.44},
	{"Lille", "FR", 50.63, 3.06},
	{"Bordeaux", "FR", 44.84, -0.58},
	{"Strasbourg", "FR", 48.58, 7.75},
	{"Nice", "FR", 43.70, 7.27},
	{"Brest", "FR", 48.39, -4.49},
	{"London", "GB", 51.51, -0.13},
	{"Madrid", "ES", 40.42, -3.70},
	{"Barcelona", "ES", 41.39, 2.17},
	{"Berlin", "DE", 52.52, 13.40},
	{"Munich", "DE", 48.14, 11.58},
	{"Rome", "IT", 41.90, 12.50},
	{"Milan", "IT", 45.46, 9.19},
	{"Amsterdam", "NL", 52.37, 4.90},
	{"Brussels", "BE", 50.85, 4.35},
	{"Lisbon", "PT", 38.72, -9.14},
	{"Vienna", "AT", 48.21, 16.37},
}

// ArchiveResponse est la reponse brute de l'API archive d'Open-Meteo.
// Elle est exportee pour que la partie "modele unifie" puisse la mapper.
type ArchiveResponse struct {
	Latitude  float64    `json:"latitude"`
	Longitude float64    `json:"longitude"`
	Elevation float64    `json:"elevation"`
	Hourly    HourlyData `json:"hourly"`
}

// HourlyData contient les tableaux de mesures horaires.
type HourlyData struct {
	Time          []string  `json:"time"`
	Temperature   []float64 `json:"temperature_2m"`
	Humidity      []float64 `json:"relative_humidity_2m"`
	Pressure      []float64 `json:"surface_pressure"`
	Precipitation []float64 `json:"precipitation"`
	WindSpeed     []float64 `json:"wind_speed_10m"`
	WindGust      []float64 `json:"wind_gusts_10m"`
	WindDir       []float64 `json:"wind_direction_10m"`
}

// Client appelle l'API Open-Meteo avec un throttling et un timeout.
type Client struct {
	http      *http.Client
	limiter   *rateLimiter
	log       *slog.Logger
	baseURL   string        // surchargeable pour les tests
	retryWait time.Duration // temps de base entre deux tentatives
}

const defaultBaseURL = "https://archive-api.open-meteo.com/v1/archive"

// maxRetries : nombre de tentatives avant d'abandonner proprement.
const maxRetries = 4

// NewClient cree un client. reqPerSec limite le nombre d'appels par seconde
// pour respecter les rate limits du fournisseur (~10 000 req/jour en gratuit).
func NewClient(reqPerSec int, log *slog.Logger) *Client {
	return &Client{
		http:      &http.Client{Timeout: 15 * time.Second},
		limiter:   newRateLimiter(reqPerSec),
		log:       log,
		baseURL:   defaultBaseURL,
		retryWait: time.Second,
	}
}

// FetchArchive recupere l'historique horaire d'une station entre deux dates
// (format AAAA-MM-JJ). Gere les retry avec backoff exponentiel sur les
// erreurs 5xx et les erreurs reseau, et abandonne apres maxRetries.
func (c *Client) FetchArchive(ctx context.Context, city City, start, end string) (ArchiveResponse, error) {
	params := url.Values{}
	params.Set("latitude", fmt.Sprintf("%f", city.Lat))
	params.Set("longitude", fmt.Sprintf("%f", city.Lon))
	params.Set("start_date", start)
	params.Set("end_date", end)
	params.Set("hourly", "temperature_2m,relative_humidity_2m,surface_pressure,precipitation,wind_speed_10m,wind_gusts_10m,wind_direction_10m")
	params.Set("timezone", "UTC")
	fullURL := c.baseURL + "?" + params.Encode()

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// throttling : on attend un jeton avant chaque appel
		c.limiter.wait()

		resp, err := c.doRequest(ctx, fullURL)
		if err != nil {
			// erreur reseau (timeout, dns...) : on retente
			lastErr = err
			c.log.Warn("erreur reseau, nouvelle tentative",
				"ville", city.Name, "attempt", attempt, "err", err.Error())
			c.backoff(attempt)
			continue
		}

		if resp.StatusCode >= 500 {
			// erreur serveur : on retente avec backoff exponentiel
			lastErr = fmt.Errorf("code serveur %d", resp.StatusCode)
			resp.Body.Close()
			c.log.Warn("erreur 5xx, nouvelle tentative",
				"ville", city.Name, "attempt", attempt, "status", resp.StatusCode)
			c.backoff(attempt)
			continue
		}

		if resp.StatusCode >= 400 {
			// erreur client (mauvaise requete) : inutile de retenter
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return ArchiveResponse{}, fmt.Errorf("code client %d: %s", resp.StatusCode, string(body))
		}

		// reponse OK : on decode le JSON
		var data ArchiveResponse
		err = json.NewDecoder(resp.Body).Decode(&data)
		resp.Body.Close()
		if err != nil {
			return ArchiveResponse{}, fmt.Errorf("decodage json: %w", err)
		}
		return data, nil
	}

	return ArchiveResponse{}, fmt.Errorf("abandon apres %d tentatives: %w", maxRetries, lastErr)
}

func (c *Client) doRequest(ctx context.Context, fullURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	return c.http.Do(req)
}

// backoff attend de plus en plus longtemps entre les tentatives
// (par defaut 1s, 2s, 4s...).
func (c *Client) backoff(attempt int) {
	wait := c.retryWait * time.Duration(1<<(attempt-1))
	time.Sleep(wait)
}
