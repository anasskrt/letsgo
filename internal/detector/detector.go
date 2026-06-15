package detector

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	"letsgo/internal/config"
	"letsgo/internal/model"
	"letsgo/internal/storage"
)

type Engine struct {
	stations *storage.StationRepo
	obs      *storage.ObservationRepo
	events   *storage.EventRepo
	cfg      config.Config
}

func NewEngine(obs *storage.ObservationRepo, events *storage.EventRepo, cfg config.Config) *Engine {
	return &Engine{
		stations: storage.NewStationRepo(obs.DB()),
		obs:      obs,
		events:   events,
		cfg:      cfg,
	}
}

func (e *Engine) Run() (int, error) {
	slog.Info("starting event detection")
	stations, err := e.stations.List("", nil)
	if err != nil {
		return 0, fmt.Errorf("list stations: %w", err)
	}

	total := 0
	for _, s := range stations {
		count, err := e.detectForStation(s.ID)
		if err != nil {
			slog.Error("detection error", "station_id", s.ID, "error", err)
			continue
		}
		total += count
	}
	slog.Info("event detection complete", "events_detected", total)
	return total, nil
}

func (e *Engine) detectForStation(stationID int) (int, error) {
	total := 0
	now := time.Now().UTC()

	from := now.Add(-72 * time.Hour)
	obs, err := e.obs.ListByStation(stationID, from, now, 0, 0)
	if err != nil {
		return 0, err
	}

	total += e.detectStorms(stationID, obs)

	daily, err := e.obs.AggregateDaily(stationID)
	if err != nil {
		return 0, err
	}

	total += e.detectHeatwaves(stationID, daily)
	total += e.detectColdwaves(stationID, daily)
	total += e.detectFloods(stationID, obs)

	return total, nil
}

func (e *Engine) detectStorms(stationID int, obs []model.Observation) int {
	if len(obs) == 0 {
		return 0
	}

	type event struct {
		start time.Time
		end   time.Time
		max   float64
	}
	var candidates []event
	var current *event

	for _, o := range obs {
		gust := o.WindGusts10m
		if gust == 0 {
			gust = o.WindSpeed10m * 1.5
		}

		if gust >= e.cfg.StormWindThreshold {
			if current == nil {
				current = &event{start: o.Timestamp, end: o.Timestamp, max: gust}
			} else {
				current.end = o.Timestamp
				if gust > current.max {
					current.max = gust
				}
			}
		} else {
			if current != nil {
				duration := current.end.Sub(current.start).Hours()
				if duration >= float64(e.cfg.StormMinHours) {
					candidates = append(candidates, *current)
				}
				current = nil
			}
		}
	}
	if current != nil {
		duration := current.end.Sub(current.start).Hours()
		if duration >= float64(e.cfg.StormMinHours) {
			candidates = append(candidates, *current)
		}
	}

	count := 0
	for _, c := range candidates {
		severity := "moderate"
		if c.max >= 100 {
			severity = "severe"
		}
		evt := &model.Event{
			Type:      model.EventStorm,
			StationID: stationID,
			StartedAt: c.start,
			EndedAt:   c.end,
			Severity:  severity,
			Metadata: map[string]any{
				"max_wind_gust":  math.Round(c.max*10) / 10,
				"duration_hours": math.Round(c.end.Sub(c.start).Hours()*10) / 10,
			},
		}
		if err := e.events.Insert(evt); err != nil {
			slog.Error("insert storm event", "error", err)
			continue
		}
		count++
	}
	return count
}

func (e *Engine) detectHeatwaves(stationID int, daily []model.AggregatedDaily) int {
	return e.detectConsecutiveTemp(stationID, daily, e.cfg.HeatwaveTempMin, 0, e.cfg.HeatwaveMinDays, "max", model.EventHeatwave)
}

func (e *Engine) detectColdwaves(stationID int, daily []model.AggregatedDaily) int {
	return e.detectConsecutiveTemp(stationID, daily, 0, e.cfg.ColdwaveTempMax, e.cfg.ColdwaveMinDays, "min", model.EventColdwave)
}

func (e *Engine) detectConsecutiveTemp(stationID int, daily []model.AggregatedDaily, above, below float64, minDays int, field string, eventType model.EventType) int {
	type segment struct {
		start   time.Time
		end     time.Time
		maxTemp float64
		minTemp float64
		avgTemp float64
		count   int
	}
	var segments []segment
	var cur *segment

	for _, d := range daily {
		cond := false
		switch field {
		case "max":
			cond = d.MaxTemp >= above
		case "min":
			cond = d.MinTemp <= below
		}

		if cond {
			if cur == nil {
				cur = &segment{start: d.Day, end: d.Day, maxTemp: d.MaxTemp, minTemp: d.MinTemp, avgTemp: d.AvgTemp, count: 1}
			} else {
				cur.end = d.Day
				if d.MaxTemp > cur.maxTemp {
					cur.maxTemp = d.MaxTemp
				}
				if d.MinTemp < cur.minTemp {
					cur.minTemp = d.MinTemp
				}
				cur.avgTemp = (cur.avgTemp*float64(cur.count) + d.AvgTemp) / float64(cur.count+1)
				cur.count++
			}
		} else {
			if cur != nil && cur.count >= minDays {
				segments = append(segments, *cur)
			}
			cur = nil
		}
	}
	if cur != nil && cur.count >= minDays {
		segments = append(segments, *cur)
	}

	count := 0
	for _, s := range segments {
		severity := "moderate"
		if eventType == model.EventHeatwave && s.maxTemp >= 40 {
			severity = "severe"
		}
		if eventType == model.EventColdwave && s.minTemp <= -10 {
			severity = "severe"
		}
		evt := &model.Event{
			Type:      eventType,
			StationID: stationID,
			StartedAt: s.start,
			EndedAt:   s.end,
			Severity:  severity,
			Metadata: map[string]any{
				"max_temp_c": math.Round(s.maxTemp*10) / 10,
				"min_temp_c": math.Round(s.minTemp*10) / 10,
				"avg_temp_c": math.Round(s.avgTemp*10) / 10,
				"days":       s.count,
			},
		}
		if err := e.events.Insert(evt); err != nil {
			slog.Error("insert temp event", "error", err)
			continue
		}
		count++
	}
	return count
}

func (e *Engine) detectFloods(stationID int, obs []model.Observation) int {
	if len(obs) == 0 {
		return 0
	}

	count := 0
	for i := range obs {
		windowEnd := obs[i].Timestamp
		windowStart := windowEnd.Add(-time.Duration(e.cfg.FloodWindowHours) * time.Hour)
		sum := 0.0
		for j := i; j >= 0 && (obs[j].Timestamp.After(windowStart) || obs[j].Timestamp.Equal(windowStart)); j-- {
			sum += obs[j].Precipitation
		}
		if sum >= e.cfg.FloodPrecipThreshold {
			evt := &model.Event{
				Type:      model.EventFlood,
				StationID: stationID,
				StartedAt: windowStart,
				EndedAt:   windowEnd,
				Severity:  "moderate",
				Metadata: map[string]any{
					"precipitation_24h_mm": math.Round(sum*10) / 10,
				},
			}
			if err := e.events.Insert(evt); err != nil {
				slog.Error("insert flood event", "error", err)
				continue
			}
			count++
		}
	}
	return count
}
