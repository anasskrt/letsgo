// Commande ingester : recupere l'historique meteo de plusieurs stations
// depuis Open-Meteo, en parallele, et affiche un resume de l'ingestion.
// C'est la partie "ingestion d'API publiques" du projet.
package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/anasskerchaoui/letsgo/internal/openmeteo"
)

// resultat d'une station ingeree (envoye par les workers).
type result struct {
	city     openmeteo.City
	hours    int
	duration time.Duration
	err      error
}

func main() {
	// logs structures en JSON (log/slog, demande par le sujet)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// configuration via variables d'environnement (12-factor)
	days := envInt("INGEST_DAYS", 35)
	workers := envInt("INGEST_WORKERS", 4)
	rate := envInt("INGEST_RATE", 5)

	// fenetre temporelle : on prend 'days' jours d'historique. On s'arrete
	// quelques jours avant aujourd'hui car l'archive Open-Meteo a un petit
	// decalage avant d'etre complete.
	end := time.Now().UTC().AddDate(0, 0, -5)
	start := end.AddDate(0, 0, -days)
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	logger.Info("debut de l'ingestion",
		"stations", len(openmeteo.Cities),
		"workers", workers,
		"du", startStr,
		"au", endStr)

	client := openmeteo.NewClient(rate, logger)
	ctx := context.Background()

	jobs := make(chan openmeteo.City)
	results := make(chan result)

	// on lance un pool de workers qui traitent les stations en parallele
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for city := range jobs {
				started := time.Now()
				data, err := client.FetchArchive(ctx, city, startStr, endStr)
				results <- result{
					city:     city,
					hours:    len(data.Hourly.Time),
					duration: time.Since(started),
					err:      err,
				}
			}
		}()
	}

	// on envoie toutes les villes dans le channel des jobs
	go func() {
		for _, city := range openmeteo.Cities {
			jobs <- city
		}
		close(jobs)
	}()

	// quand tous les workers ont fini, on ferme le channel des resultats
	go func() {
		wg.Wait()
		close(results)
	}()

	// on collecte les resultats et on agrege les statistiques
	globalStart := time.Now()
	var nbOk, nbErr, totalObs int
	for r := range results {
		if r.err != nil {
			nbErr++
			logger.Error("echec station",
				"ville", r.city.Name,
				"err", r.err.Error())
			continue
		}
		nbOk++
		totalObs += r.hours
		logger.Info("station recuperee",
			"ville", r.city.Name,
			"pays", r.city.Country,
			"observations", r.hours,
			"duree_ms", r.duration.Milliseconds())
	}

	// resume final de l'ingestion (nb appels, erreurs, duree)
	logger.Info("ingestion terminee",
		"stations_ok", nbOk,
		"stations_erreur", nbErr,
		"appels_total", len(openmeteo.Cities),
		"observations_total", totalObs,
		"duree_totale_s", int(time.Since(globalStart).Seconds()))

	if nbErr > 0 {
		os.Exit(1)
	}
}

// envInt lit une variable d'environnement entiere, avec une valeur par defaut.
func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
