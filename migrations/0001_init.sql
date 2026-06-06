CREATE TABLE IF NOT EXISTS sources (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    base_url    TEXT NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS stations (
    id           SERIAL PRIMARY KEY,
    name         TEXT NOT NULL,
    country_code TEXT NOT NULL DEFAULT '',
    latitude     DOUBLE PRECISION NOT NULL,
    longitude    DOUBLE PRECISION NOT NULL,
    elevation    DOUBLE PRECISION NOT NULL DEFAULT 0,
    source       TEXT NOT NULL DEFAULT 'open-meteo',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_stations_country ON stations (country_code);
CREATE INDEX IF NOT EXISTS idx_stations_coords ON stations (latitude, longitude);

CREATE TABLE IF NOT EXISTS observations (
    id               SERIAL PRIMARY KEY,
    station_id       INTEGER NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    timestamp        TIMESTAMPTZ NOT NULL,
    temperature_2m   DOUBLE PRECISION,
    wind_speed_10m   DOUBLE PRECISION,
    wind_gusts_10m   DOUBLE PRECISION,
    precipitation    DOUBLE PRECISION,
    pressure_msl     DOUBLE PRECISION,
    relative_humidity_2m DOUBLE PRECISION,
    cloud_cover      DOUBLE PRECISION,
    weather_code     INTEGER,
    source           TEXT NOT NULL DEFAULT 'open-meteo'
);

CREATE INDEX IF NOT EXISTS idx_observations_station ON observations (station_id);
CREATE INDEX IF NOT EXISTS idx_observations_ts ON observations (timestamp);
CREATE UNIQUE INDEX IF NOT EXISTS idx_observations_station_ts ON observations (station_id, timestamp);

CREATE TABLE IF NOT EXISTS events (
    id          SERIAL PRIMARY KEY,
    type        TEXT NOT NULL,
    station_id  INTEGER NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    started_at  TIMESTAMPTZ NOT NULL,
    ended_at    TIMESTAMPTZ NOT NULL,
    severity    TEXT NOT NULL DEFAULT 'moderate',
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_events_type ON events (type);
CREATE INDEX IF NOT EXISTS idx_events_station ON events (station_id);
CREATE INDEX IF NOT EXISTS idx_events_period ON events (started_at, ended_at);
