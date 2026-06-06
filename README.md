# LetsGo — Plateforme météo

Application Go de collecte, stockage et analyse de données météo avec détection d'événements.

## Architecture

```
cmd/
├── api/          # Serveur API REST
└── ingester/     # Ingestion de données Open-Meteo
internal/
├── model/        # Types métier neutres
├── openmeteo/    # Client Open-Meteo + mapping
├── storage/      # Repository pattern (PostgreSQL)
├── api/          # Handlers HTTP
├── detector/     # Moteur de détection d'événements
└── config/       # Configuration (variables d'environnement)
migrations/       # SQL versionné
```

## Lancement

```bash
# Démarrer PostgreSQL
docker compose up -d postgres

# Ingérer les données (20 stations, 30 jours)
go run ./cmd/ingester

# Lancer le serveur API
go run ./cmd/api
```

Ou tout d'un coup :
```bash
docker compose up --build
```

## Routes API

### Stations
| Méthode | Route | Description |
|---------|-------|-------------|
| GET | `/stations` | Liste toutes les stations |
| GET | `/stations?country=FR` | Filtre par pays |
| GET | `/stations?bbox=lat1,lon1,lat2,lon2` | Filtre géographique |
| GET | `/stations/{id}` | Détail d'une station |
| GET | `/stations/{id}/observations?from=2026-06-01&to=2026-06-10` | Observations avec intervalle |
| GET | `/stations/{id}/observations?limit=100&offset=0` | Pagination |
| GET | `/stations/{id}/events` | Événements d'une station |

### Observations
| Méthode | Route | Description |
|---------|-------|-------------|
| GET | `/observations/aggregate?period=daily&station_id=X` | Agrégation journalière |

### Événements
| Méthode | Route | Description |
|---------|-------|-------------|
| GET | `/events` | Liste tous les événements |
| GET | `/events?type=storm&from=...&to=...` | Filtres |
| GET | `/events/{id}` | Détail d'un événement |
| GET | `/events/stats?type=storm&country=FR` | Statistiques agrégées |
| POST | `/detect` | Déclencher la détection |

### Health
| Méthode | Route | Description |
|---------|-------|-------------|
| GET | `/health` | Liveness |
| GET | `/ready` | Readiness (vérifie BDD) |

## Événements détectés

| Type | Déclencheur |
|------|-------------|
| **Tempête** | Rafales > 80 km/h pendant ≥ 3h consécutives |
| **Canicule** | T° max ≥ 35 °C pendant ≥ 3 jours |
| **Vague de froid** | T° max ≤ 0 °C pendant ≥ 3 jours |
| **Inondation** | Précipitations > 50 mm en 24h |

Seuils configurables via variables d'environnement (cf. `.env.example`).

## Configuration

Copier `.env.example` vers `.env` et ajuster.

## Tests

```bash
go test ./... -v -cover
```
