# Projet final — Plateforme météo en Go

**Module** : Golang — M2 Dev Manager Full Stack
**Format** : équipes de **3 ou 4**
**Calendrier** : lancement J4, travail en autonomie sur l'intersession, soutenance J5.
**Pondération** : **50 %** de la note du module.
**Stack imposée** : Go (stdlib en priorité) · PostgreSQL · Git/GitHub · Docker (au moins pour la BDD).

---

## 1. Contexte

Vous avez construit en cours :
- un parseur JSON+XML d'un dataset météo statique (J2) ;
- une API REST CRUD en Go pur sur ce dataset (J3) ;
- les bases de la concurrence et de la persistance (J4).

Le projet final assemble tout ça **dans une vraie application météo connectée au monde réel** : vous allez ingérer des données depuis des API publiques, les unifier, les stocker dans une base PostgreSQL, les exposer via une API REST, et bâtir par-dessus des fonctions d'analyse qui détectent les **événements météo sensibles** (tempêtes, canicules, vagues de froid, etc.).

C'est un sujet proche de ce que vous verrez en production. Architectes d'API météo, équipes BI, plateformes IoT : ils résolvent exactement ce problème.

---

## 2. Périmètre obligatoire — les 5 piliers

Les 5 piliers ci-dessous **doivent tous être présents** pour que le projet soit acceptable. C'est le minimum vital. Tout ce qui s'y ajoute est bonus (cf. carte blanche §5).

###  Pilier 1 — Ingestion d'API publiques météo

Récupérer des données réelles depuis au moins **une API publique météo**. Source recommandée : **Open-Meteo** (gratuit, sans clé d'API, doc claire — [open-meteo.com](https://open-meteo.com/)).

À couvrir :
- Lecture de **plusieurs stations** (au minimum 10, idéalement 20-30 pour avoir matière à détecter des événements).
- Récupération sur une **fenêtre temporelle suffisante** (au minimum 30 jours d'historique).
- **Gestion d'erreurs réseau** : timeouts, codes 4xx/5xx du fournisseur, retry avec backoff exponentiel sur les 5xx, abandon propre après N tentatives.
- **Respect des rate limits** du fournisseur (souvent ~10 000 req/jour gratuit). Throttling explicite côté ingester.
- Logs structurés de l'ingestion (combien d'appels, combien d'erreurs, durée).

Le binaire d'ingestion doit être **séparable du serveur API** (typiquement `cmd/ingester/` et `cmd/api/`).

###  Pilier 2 — Modèle unifié

Quel que soit le fournisseur d'origine, vos données doivent transiter vers **un modèle interne neutre** identique à celui que vous avez écrit au TP J2.

À couvrir :
- Types Go neutres (`Station`, `Observation`, `Wind`, `AirQuality`, etc.) sans tags JSON/XML ni détails d'un fournisseur particulier.
- **Une fonction de mapping par source** (`openMeteoToStation`, `meteoFranceToStation`, …) clairement isolée.
- Conversions d'unités si nécessaire (°F→°C, mph→km/h, hPa↔Pa).
- Tests unitaires sur le mapping — au moins un test par fonction de conversion.

Si vous ajoutez une 2ème source (cf. carte blanche), le modèle interne ne doit pas bouger.

###  Pilier 3 — Stockage en BDD PostgreSQL

Persistance dans PostgreSQL. Pas de SQLite, pas de fichier JSON, pas de cache mémoire seul.

À couvrir :
- **Schéma** : au minimum les tables `stations`, `observations`, `sources`, `events` (cf. pilier 5). Clés primaires, clés étrangères, index sur les colonnes interrogées (`station_id`, `timestamp`, `country_code`).
- **Migrations versionnées** : un dossier `migrations/` avec des fichiers SQL numérotés (`0001_init.sql`, etc.). On doit pouvoir remettre la BDD à zéro avec une commande.
- **Repository pattern** : une couche `storage/` qui isole les requêtes SQL du reste du code. Aucun `database/sql` ne fuit dans les handlers HTTP.
- Driver : `database/sql` + `lib/pq` ou `jackc/pgx` (au choix). Pas d'ORM (GORM est tentant, mais pour l'apprentissage on s'en passe).
- Docker Compose qui démarre Postgres + l'application, ou au minimum un docker-compose pour Postgres seul.

### Pilier 4 — API REST sur les données stockées

Reprendre les 6 routes du TP J3 (`GET /stations`, `GET /stations/{id}`, etc.) mais **lues depuis la BDD**, pas depuis une map en mémoire. Ajouter au moins **3 routes spécifiques projet** parmi :

- `GET /stations?country=FR` — filtre par code pays
- `GET /stations?bbox=lat1,lon1,lat2,lon2` — filtre par bbox géographique
- `GET /stations/{id}/observations?from=2026-04-01&to=2026-05-01` — intervalle temporel
- `GET /stations/{id}/observations?limit=100&offset=0` — pagination
- `GET /observations/aggregate?period=daily&station_id=X` — agrégation journalière (température moyenne, max, min)

Toutes les routes doivent renvoyer des codes statut HTTP corrects (200/201/204/400/404/409/500), un format d'erreur JSON normalisé avec code interne, et un `Content-Type: application/json`.

###  Pilier 5 — Détection d'événements météo sensibles

C'est le cœur métier du projet. Vous devez **détecter, persister et exposer** des événements météo "remarquables" à partir des observations stockées.

**Au minimum 3 types d'événements** parmi :

| Événement | Définition (suggestion, ajustable) |
|---|---|
| **Tempête** | Rafales > 80 km/h pendant au moins 3 heures consécutives sur une station |
| **Canicule** | Température max ≥ 35 °C pendant au moins 3 jours consécutifs |
| **Vague de froid** | Température max ≤ 0 °C pendant au moins 3 jours consécutifs |
| **Inondation potentielle** | Précipitations > 50 mm en 24 h sur une station |
| **Sécheresse** | Précipitations cumulées < 5 mm sur 30 jours consécutifs |
| **Pic de pollution** | PM2.5 > 50 µg/m³ pendant au moins 6 heures |

Le détecteur tourne en arrière-plan (ou en batch périodique) et écrit dans la table `events` : `id`, `type`, `station_id`, `started_at`, `ended_at`, `severity`, `metadata JSONB`.

**Routes API associées** (minimum) :

- `GET /events` — liste tous les événements (avec filtres `?type=storm&from=...&to=...`)
- `GET /events/{id}` — détail d'un événement
- `GET /stations/{id}/events` — événements liés à une station
- `GET /events/stats?type=storm&country=FR` — comptage agrégé

Les seuils de détection doivent être **configurables** (via variables d'environnement ou fichier `.yaml`), pas hardcodés.

---

## 3. Contraintes techniques

| Élément | Imposé |
|---|---|
| Langage | Go 1.22+ |
| Serveur HTTP | `net/http` stdlib. **Pas de Gin/Echo/Fiber.** |
| Base de données | PostgreSQL 14+, en Docker pour la dev |
| Migrations | SQL versionné, exécutable avec une commande (golang-migrate accepté) |
| JSON / XML | `encoding/json` et `encoding/xml` stdlib |
| Concurrence | goroutines + channels (et pas un seul goroutine, on attend du parallélisme réel sur l'ingester) |
| Tests | `testing` stdlib + `httptest` pour les handlers. Au moins **15 % de couverture** sur le code métier (mapping, détection événements, repository). |
| Logs | `log/slog` (stdlib Go 1.21+), logs structurés JSON |
| Config | Variables d'environnement (12-factor). `.env.example` dans le repo, jamais de secret commit. |
| Docker | `Dockerfile` qui build le binaire en multi-stage + `docker-compose.yml` qui démarre Postgres + l'API |
| Repo | Git public ou privé partagé avec moi, `.gitignore` propre, commits réguliers (pas un seul commit "final" à la fin) |
| Doc | `README.md` à la racine : architecture, comment lancer, les routes, exemples Postman, captures d'écran de la démo |

---



## 4. Carte blanche — idées d'évolution

Une fois les 5 piliers obligatoires terminés, vous avez **carte blanche** pour améliorer. Les idées ci-dessous sont des suggestions — **chacune doit être validée avec moi avant d'être implémentée**, pour qu'on évite que vous partiez sur 3 chantiers qui ne vous serviront pas pour la note.

### Niveau 1 — Améliorations natives (bonus 1-2 pts)

- **Multi-source** : ajouter une 2ème API publique (OpenWeatherMap, AccuWeather, Météo France). Démontre la robustesse du modèle unifié.
- **Cache HTTP** : middleware qui cache les réponses GET pendant N secondes pour réduire la charge BDD. Headers `Cache-Control` / `ETag`.
- **Documentation OpenAPI/Swagger** : générer un swagger.yaml depuis le code et le servir sur `/docs`.
- **Tests d'intégration** : un binaire qui démarre Postgres en testcontainer, ingère un mini dataset, et vérifie que les events sont bien détectés.

### Niveau 2 — Industrialisation (bonus 2-3 pts)

- **Middleware logging structuré** : chaque requête HTTP loguée en JSON avec request-id, latency, status.
- **Middleware recovery** : intercepter les panics et renvoyer un 500 propre.
- **Graceful shutdown** : `signal.NotifyContext` + `Server.Shutdown(ctx)`. L'app se ferme proprement sur Ctrl+C.
- **Health checks** : `GET /health` (liveness) + `GET /ready` (readiness, vérifie la BDD).
- **Métriques Prometheus** : compteurs de requêtes par route, latency histogram, exposés sur `/metrics`.
- **CI GitHub Actions** : workflow qui lance `go build`, `go test`, `golangci-lint` sur chaque push.

### Niveau 3 — Frontend / visualisation (bonus 1 pts)

- **Dashboard HTML/JS simple** : une page statique servie sur `/` qui consomme votre API et affiche la liste des événements + une carte (Leaflet).
- **Frontend React/Vue séparé** : projet indépendant qui consomme votre API. Si l'équipe a un dev front, c'est l'occasion.
- **Notifications WebSocket** : quand un nouvel événement est détecté, push live aux clients connectés.
- **Carte interactive des événements** : Leaflet ou Mapbox, pins par événement, popup avec détails.

### À NE PAS faire (pour vous éviter d'y perdre du temps)

-  Migrer vers Gin/Echo en cours de projet — vous perdez l'apprentissage stdlib.
-  Utiliser un ORM (GORM, sqlboiler) — vous perdez la maîtrise SQL.
-  Implémenter un message broker maison (Kafka-like) — hors scope.
-  Faire du Go générique sur tous les types — c'est de la sur-ingénierie sur un projet de cette taille.
-  Refaire le frontend du tableau de bord PowerBI de l'école — on est sur du back.

---

## 5. Calendrier et jalons

### J4 — Lancement officiel (en cours)

**Matin** : présentation du sujet, formation des équipes, choix de l'API publique, brouillon du schéma BDD au tableau.
**Après-midi** : setup repo, `docker-compose` Postgres, premier `cmd/ingester` qui appelle Open-Meteo et affiche une station.

**À avoir avant la fin de J4** :
- Repo Git créé, équipe ajoutée
- `docker-compose` qui démarre Postgres
- Un `cmd/ingester/main.go` qui fait au moins UN appel HTTP réussi vers Open-Meteo

### Intersession (autonomie)

Travail à votre rythme. **Je suis disponible sur Teams/LinkedIn** pour des questions pendant cette période — je réponds dans la journée sur des horaires normaux.

**Jalons indicatifs** (planifiez les vôtres) :

| Quand | Objectif |
|---|---|
| Fin de semaine 1 | Pilier 1 et 2 (ingestion + modèle unifié) opérationnels, 10 stations + 30 jours en BDD |
| Mi-semaine 2 | Pilier 3 (storage) terminé, pilier 4 (API) en cours, 3 routes au moins |
| Fin de semaine 2 | Pilier 4 complet, pilier 5 (events) au moins 2 types détectés |
| Veille de soutenance | Tout en place, démo répétée, slides prêtes |

### J5 — Soutenance

**30 - 40  minutes par équipe** :
- 10 - 20 min de présentation (slides + démo live)
- 10 - 20 min de Q&A


---

## 6. Livrables attendus

À fournir **au plus tard la veille de la soutenance, 18h00** :

1. **Lien vers le repo Git** avec accès "Reporter" ou "Read" pour moi.
2. **Le repo contient à la racine** :
   - `README.md` complet (architecture, comment lancer, routes, captures d'écran)
   - `docker-compose.yml` qui démarre tout
   - `.env.example` (jamais de `.env` versionné !)
   - `migrations/` avec les fichiers SQL
   - Collection Postman à jour si l'API a évolué
3. **Slides de soutenance** (`.pdf` ou `.pptx`), 10-15 slides max.

---

## 7. Barème (sur 20)

Conforme à la grille critériée du module.

| Critère | Points |
|---|---|
| **Pertinence métier** — projet répond au sujet, fonctions de détection cohérentes, exemples parlants en démo | **5** |
| **Architecture & justification choix techniques** — séparation des responsabilités, schéma BDD propre, choix défendus en soutenance | **4** |
| **Qualité d'implémentation** — code idiomatique Go, gestion d'erreurs, tests présents et qui passent, pas de bugs grossiers | **4** |
| **Industrialisation** — Docker, migrations, CI si présent, documentation, déploiement local en une commande | **4** |
| **Soutenance** — clarté de la démo, qualité du discours, réponses aux questions, respect du timing | **3** |

**Bonus carte blanche** (cf. §4) : jusqu'à **+3 pts**, plafonnés à 20/20.

**Note collective** sauf cas manifeste de free-riding (silence complet d'un membre en soutenance, zéro commit, pas de présence aux points équipe). Dans ce cas-là, j'ajuste individuellement avec preuve.

