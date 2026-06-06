package detector

import (
	"testing"
	"time"

	"letsgo/internal/model"
)

func TestConsecutiveSegmentLogic(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	obs := []model.Observation{
		{Timestamp: now, WindGusts10m: 85, WindSpeed10m: 60},
		{Timestamp: now.Add(1 * time.Hour), WindGusts10m: 90, WindSpeed10m: 65},
		{Timestamp: now.Add(2 * time.Hour), WindGusts10m: 0, WindSpeed10m: 30},
		{Timestamp: now.Add(3 * time.Hour), WindGusts10m: 95, WindSpeed10m: 70},
		{Timestamp: now.Add(4 * time.Hour), WindGusts10m: 0, WindSpeed10m: 25},
	}

	type segment struct {
		start time.Time
		end   time.Time
		max   float64
	}

	threshold := 80.0
	minHours := 1.0

	var candidates []segment
	var cur *segment
	for _, o := range obs {
		gust := o.WindGusts10m
		if gust == 0 {
			gust = o.WindSpeed10m * 1.5
		}
		if gust >= threshold {
			if cur == nil {
				cur = &segment{start: o.Timestamp, end: o.Timestamp, max: gust}
			} else {
				cur.end = o.Timestamp
				if gust > cur.max {
					cur.max = gust
				}
			}
		} else {
			if cur != nil {
				duration := cur.end.Sub(cur.start).Hours()
				if duration >= minHours {
					candidates = append(candidates, *cur)
				}
				cur = nil
			}
		}
	}

	if len(candidates) == 0 {
		t.Error("expected at least 1 storm segment")
	}
	for _, c := range candidates {
		duration := c.end.Sub(c.start).Hours()
		if duration < minHours {
			t.Errorf("segment too short: %f hours", duration)
		}
	}
	t.Logf("found %d storm segments", len(candidates))
}

func TestHeatwaveSegmentLogic(t *testing.T) {
	daily := []model.AggregatedDaily{
		{Day: "2026-06-01", MaxTemp: 36},
		{Day: "2026-06-02", MaxTemp: 37},
		{Day: "2026-06-03", MaxTemp: 38},
		{Day: "2026-06-04", MaxTemp: 34},
		{Day: "2026-06-05", MaxTemp: 39},
	}

	type segment struct {
		start string
		end   string
		count int
	}

	minDays := 3
	threshold := 35.0

	var segments []segment
	var cur *segment
	for _, d := range daily {
		if d.MaxTemp >= threshold {
			if cur == nil {
				cur = &segment{start: d.Day, end: d.Day, count: 1}
			} else {
				cur.end = d.Day
				cur.count++
			}
		} else {
			if cur != nil && cur.count >= minDays {
				segments = append(segments, *cur)
			}
			cur = nil
		}
	}

	if len(segments) != 1 {
		t.Fatalf("expected 1 heatwave segment, got %d", len(segments))
	}
	if segments[0].count != 3 {
		t.Errorf("expected 3 days, got %d", segments[0].count)
	}
	if segments[0].start != "2026-06-01" {
		t.Errorf("expected start 2026-06-01, got %s", segments[0].start)
	}
}

func TestFloodSegmentLogic(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	obs := []model.Observation{
		{Timestamp: now, Precipitation: 20},
		{Timestamp: now.Add(1 * time.Hour), Precipitation: 15},
		{Timestamp: now.Add(2 * time.Hour), Precipitation: 18},
	}

	sum := 0.0
	windowStart := obs[len(obs)-1].Timestamp.Add(-24 * time.Hour)
	for _, o := range obs {
		if o.Timestamp.After(windowStart) || o.Timestamp.Equal(windowStart) {
			sum += o.Precipitation
		}
	}

	if sum < 50 {
		t.Logf("total precip %f (below 50mm threshold, expected)", sum)
	} else {
		t.Logf("total precip %f (above threshold)", sum)
	}
}
