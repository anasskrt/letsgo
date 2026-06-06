package openmeteo

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	rateLimit  time.Duration
	maxRetries int
	lastCall   time.Time
}

func NewClient(baseURL string, rateLimit time.Duration, maxRetries int) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		rateLimit:  rateLimit,
		maxRetries: maxRetries,
	}
}

func (c *Client) FetchForecast(lat, lon float64, pastDays int) (*ForecastResponse, error) {
	url := fmt.Sprintf(
		"%s/forecast?latitude=%.4f&longitude=%.4f&hourly=temperature_2m,wind_speed_10m,wind_gusts_10m,precipitation,pressure_msl,relative_humidity_2m,cloud_cover,weather_code&past_days=%d&timezone=UTC",
		c.baseURL, lat, lon, pastDays,
	)

	c.throttle()

	var lastErr error
	for attempt := range c.maxRetries {
		if attempt > 0 {
			backoff := time.Duration(1<<attempt) * time.Second
			slog.Info("retrying", "url", url, "attempt", attempt+1, "backoff", backoff)
			time.Sleep(backoff)
		}

		resp, err := c.httpClient.Get(url)
		if err != nil {
			lastErr = fmt.Errorf("http call: %w", err)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read body: %w", readErr)
			continue
		}

		if resp.StatusCode >= 500 && attempt < c.maxRetries-1 {
			lastErr = fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
		}

		var data ForecastResponse
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("json decode: %w", err)
		}
		return &data, nil
	}
	return nil, fmt.Errorf("all %d attempts failed: %w", c.maxRetries, lastErr)
}

func (c *Client) throttle() {
	if c.rateLimit <= 0 {
		return
	}
	elapsed := time.Since(c.lastCall)
	if elapsed < c.rateLimit {
		time.Sleep(c.rateLimit - elapsed)
	}
	c.lastCall = time.Now()
}
