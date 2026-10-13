package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// weatherTestServer serves the given JSON body through the real
// fetchOpenMeteoWeather HTTP+JSON decoding path by pointing
// openMeteoArchiveURL at an httptest server (audit finding 6: tests must
// exercise actual decoding, not a constructed Go struct).
func weatherTestServer(t *testing.T, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)

	origURL := openMeteoArchiveURL
	openMeteoArchiveURL = srv.URL
	t.Cleanup(func() { openMeteoArchiveURL = origURL })
}

func weatherTestRange() (time.Time, time.Time) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	return start, start.Add(24 * time.Hour)
}

func mustWeatherDay(t *testing.T, w domain.Weather, date string) domain.WeatherDay {
	t.Helper()
	day, ok := w[date]
	if !ok {
		t.Fatalf("expected day %s to be present, got %v", date, w)
	}
	return day
}

func TestFetchWeatherFullCompleteDay(t *testing.T) {
	weatherTestServer(t, `{"daily":{
		"time":["2026-07-01","2026-07-02"],
		"weather_code":[0,61],
		"temperature_2m_max":[31.5,22.0],
		"temperature_2m_min":[24.0,18.5]
	}}`)
	start, end := weatherTestRange()
	w := fetchOpenMeteoWeather(35.0, 135.0, start, end)
	if len(w) != 2 {
		t.Fatalf("expected 2 complete days, got %d: %v", len(w), w)
	}
	day := mustWeatherDay(t, w, "2026-07-01")
	if day.Icon != "sun" || day.Code == nil || *day.Code != 0 {
		t.Fatalf("expected sunny code 0, got %+v", day)
	}
	if day.Min == nil || *day.Min != 24.0 || day.Max == nil || *day.Max != 31.5 {
		t.Fatalf("expected temperatures 24.0/31.5 preserved, got %+v", day)
	}
	if day.Provider != "open-meteo" || day.Units != "celsius" {
		t.Fatalf("expected provider/units preserved, got %+v", day)
	}
}

func TestFetchWeatherNullCodeIsOmitted(t *testing.T) {
	weatherTestServer(t, `{"daily":{
		"time":["2026-07-01","2026-07-02"],
		"weather_code":[null,61],
		"temperature_2m_max":[31.5,22.0],
		"temperature_2m_min":[24.0,18.5]
	}}`)
	start, end := weatherTestRange()
	w := fetchOpenMeteoWeather(35.0, 135.0, start, end)
	if _, ok := w["2026-07-01"]; ok {
		t.Fatalf("null-code day must be omitted, got %v", w)
	}
	if _, ok := w["2026-07-02"]; !ok {
		t.Fatalf("complete day must survive, got %v", w)
	}
}

func TestFetchWeatherNullTemperaturesAreOmitted(t *testing.T) {
	weatherTestServer(t, `{"daily":{
		"time":["2026-07-01"],
		"weather_code":[0],
		"temperature_2m_max":[null],
		"temperature_2m_min":[24.0]
	}}`)
	start, end := weatherTestRange()
	w := fetchOpenMeteoWeather(35.0, 135.0, start, end)
	if len(w) != 0 {
		t.Fatalf("null-temperature day must be omitted, got %v", w)
	}
}

func TestFetchWeatherGenuineZeroIsPreserved(t *testing.T) {
	weatherTestServer(t, `{"daily":{
		"time":["2026-07-01"],
		"weather_code":[0],
		"temperature_2m_max":[0],
		"temperature_2m_min":[0]
	}}`)
	start, end := weatherTestRange()
	w := fetchOpenMeteoWeather(35.0, 135.0, start, end)
	day, ok := w["2026-07-01"]
	if !ok {
		t.Fatalf("genuine zeros must be kept as a complete day, got %v", w)
	}
	if day.Min == nil || *day.Min != 0 || day.Max == nil || *day.Max != 0 {
		t.Fatalf("expected min/max exactly 0, got %+v", day)
	}
	if day.Code == nil || *day.Code != 0 {
		t.Fatalf("expected code exactly 0, got %+v", day)
	}
}

func TestFetchWeatherShortArraysArePartialDays(t *testing.T) {
	weatherTestServer(t, `{"daily":{
		"time":["2026-07-01","2026-07-02"],
		"weather_code":[0],
		"temperature_2m_max":[31.5,22.0],
		"temperature_2m_min":[24.0,20.0]
	}}`)
	start, end := weatherTestRange()
	w := fetchOpenMeteoWeather(35.0, 135.0, start, end)
	if _, ok := w["2026-07-01"]; !ok {
		t.Fatalf("first day has complete entries and must survive, got %v", w)
	}
	if _, ok := w["2026-07-02"]; ok {
		t.Fatalf("day missing its array entries must be omitted, got %v", w)
	}
}

func TestFetchWeatherMissingArraysOmitAllDays(t *testing.T) {
	weatherTestServer(t, `{"daily":{"time":["2026-07-01","2026-07-02"]}}`)
	start, end := weatherTestRange()
	w := fetchOpenMeteoWeather(35.0, 135.0, start, end)
	if len(w) != 0 {
		t.Fatalf("missing arrays must yield no weather, got %v", w)
	}
}

func TestFetchWeatherAllIncompleteReturnsNil(t *testing.T) {
	weatherTestServer(t, `{"daily":{
		"time":["2026-07-01"],
		"weather_code":[null],
		"temperature_2m_max":[null],
		"temperature_2m_min":[null]
	}}`)
	start, end := weatherTestRange()
	w := fetchOpenMeteoWeather(35.0, 135.0, start, end)
	if w != nil {
		t.Fatalf("fully incomplete response must yield nil so no weather is stored, got %v", w)
	}
}
