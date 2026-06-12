package openmeteo

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// logger silencieux pour ne pas polluer la sortie des tests
func testClient(baseURL string) *Client {
	c := NewClient(100, slog.New(slog.NewTextHandler(io.Discard, nil)))
	c.baseURL = baseURL
	c.retryWait = time.Millisecond // pour ne pas attendre des secondes
	return c
}

var paris = City{Name: "Paris", Country: "FR", Lat: 48.85, Lon: 2.35}

// le serveur renvoie deux 500 puis un 200 : le client doit reussir grace
// aux tentatives (retry + backoff).
func TestFetchRetrySur5xx(t *testing.T) {
	appels := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appels++
		if appels < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(`{"latitude":48.85,"longitude":2.35,"hourly":{"time":["2026-05-01T00:00"]}}`))
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	data, err := c.FetchArchive(context.Background(), paris, "2026-05-01", "2026-05-02")
	if err != nil {
		t.Fatalf("ne devrait pas echouer, err=%v", err)
	}
	if appels != 3 {
		t.Errorf("attendu 3 appels (2 echecs + 1 ok), obtenu %d", appels)
	}
	if len(data.Hourly.Time) != 1 {
		t.Errorf("donnees mal decodees: %+v", data.Hourly)
	}
}

// un 404 ne doit pas etre retente : on abandonne tout de suite.
func TestFetchPasDeRetrySur4xx(t *testing.T) {
	appels := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appels++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	_, err := c.FetchArchive(context.Background(), paris, "2026-05-01", "2026-05-02")
	if err == nil {
		t.Fatal("un 404 devrait renvoyer une erreur")
	}
	if appels != 1 {
		t.Errorf("un 4xx ne doit pas etre retente, obtenu %d appels", appels)
	}
}

// si le serveur renvoie toujours 500, on doit abandonner apres maxRetries.
func TestFetchAbandonApresMaxRetries(t *testing.T) {
	appels := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appels++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := testClient(srv.URL)
	_, err := c.FetchArchive(context.Background(), paris, "2026-05-01", "2026-05-02")
	if err == nil {
		t.Fatal("devrait abandonner et renvoyer une erreur")
	}
	if appels != maxRetries {
		t.Errorf("attendu %d tentatives, obtenu %d", maxRetries, appels)
	}
}
