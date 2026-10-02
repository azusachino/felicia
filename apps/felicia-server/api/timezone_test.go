package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-server/api"
)

func TestAdminTimezoneDefaultsAndExplicitValues(t *testing.T) {
	for _, test := range []struct {
		name, explicit, existing, want string
		geom                           any
		missingNoRows                  bool
	}{
		{"GPS", "", "", "America/New_York", map[string]any{"type": "Point", "coordinates": []float64{-74.006, 40.7128}}, false},
		{"journey fallback", "", "", "Asia/Tokyo", nil, true},
		{"explicit UTC", "UTC", "", "UTC", map[string]any{"type": "Point", "coordinates": []float64{139.6917, 35.6895}}, false},
		{"omitted edit preserves zone", "", "Europe/London", "Europe/London", map[string]any{"type": "Point", "coordinates": []float64{139.6917, 35.6895}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := newMockRepository()
			if test.missingNoRows {
				repo.missingMementoErr = sql.ErrNoRows
			}
			jid, mid := uuid.New(), uuid.New()
			repo.journeys[jid] = &domain.Journey{ID: jid, GPSRoute: orb.MultiLineString{{{139.6917, 35.6895}, {139.7, 35.7}}}}
			if test.existing != "" {
				repo.mementos[mid] = &domain.Memento{ID: mid, JourneyID: jid, State: domain.MementoDraft, OccurredTZ: test.existing}
			}
			body, err := json.Marshal(map[string]any{
				"id": mid, "journey_id": jid, "kind": "goods", "state": "draft", "title": "Test",
				"occurred_at": "2026-07-01T23:30:00Z", "occurred_tz": test.explicit, "geom": test.geom, "kind_data": map[string]any{},
			})
			if err != nil {
				t.Fatal(err)
			}
			handler := api.NewServer(repo, loadKinds(t), api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler()
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/admin/mementos", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}
			if got := repo.mementos[mid].OccurredTZ; got != test.want {
				t.Fatalf("zone=%q want=%q", got, test.want)
			}
		})
	}
}
