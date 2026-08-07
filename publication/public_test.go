package publication

import (
	"bytes"
	"context"
	"encoding/json"
	"image/jpeg"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core/domain"
)

func stringPtr(value string) *string { return &value }

func int64Ptr(value int64) *int64 { return &value }

func TestPublishedMementosFiltersAndSorts(t *testing.T) {
	journeyID := uuid.New()
	base := time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC)
	published2 := &domain.Memento{ID: uuid.New(), JourneyID: journeyID, Seq: 2, OccurredAt: base.Add(time.Hour), State: domain.MementoPublished}
	published1 := &domain.Memento{ID: uuid.New(), JourneyID: journeyID, Seq: 1, OccurredAt: base, State: domain.MementoPublished}
	draft := &domain.Memento{ID: uuid.New(), JourneyID: journeyID, Seq: 3, OccurredAt: base, State: domain.MementoDraft}
	candidate := &domain.Memento{ID: uuid.New(), JourneyID: journeyID, Seq: 4, OccurredAt: base, State: domain.MementoCandidateState}

	got := PublishedMementos([]*domain.Memento{published2, draft, candidate, published1})
	if len(got) != 2 {
		t.Fatalf("published count = %d, want 2", len(got))
	}
	if got[0].ID != published1.ID || got[1].ID != published2.ID {
		t.Errorf("published order = [%s %s], want seq order [%s %s]", got[0].ID, got[1].ID, published1.ID, published2.ID)
	}
}

func TestNewStaticMementoCarriesAuthoredFields(t *testing.T) {
	takenAt := time.Date(2026, 7, 2, 9, 30, 0, 0, time.UTC)
	photo := &domain.MementoPhoto{ID: uuid.New(), MementoID: uuid.New(), ObjectKey: "media/a.jpg", ContentHash: "abc", Seq: 1, TakenAt: &takenAt}
	memento := &domain.Memento{
		ID: uuid.New(), JourneyID: uuid.New(), Kind: "goods", Seq: 1,
		OccurredAt: time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC), OccurredTZ: "Asia/Tokyo",
		Geom: orb.Point{139.7, 35.6}, Title: "fuwamiku", Place: "Akihabara",
		Vendor: stringPtr("Animate"), Essay: stringPtr("the essay body"),
		PriceAmount: int64Ptr(2400), PriceCurrency: stringPtr("JPY"),
		State: domain.MementoPublished,
	}

	projected := NewStaticMemento(memento, []StaticPhoto{NewStaticPhoto(photo)})

	raw, err := json.Marshal(projected)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["essay"] != "the essay body" {
		t.Errorf("essay = %v, want authored essay", decoded["essay"])
	}
	if decoded["vendor"] != "Animate" {
		t.Errorf("vendor = %v, want Animate", decoded["vendor"])
	}
	if decoded["price_amount"] != float64(2400) {
		t.Errorf("price_amount = %v, want 2400", decoded["price_amount"])
	}
	if decoded["price_currency"] != "JPY" {
		t.Errorf("price_currency = %v, want JPY", decoded["price_currency"])
	}
	photos := decoded["photos"].([]any)
	if got := photos[0].(map[string]any)["taken_at"]; got != "2026-07-02T09:30:00Z" {
		t.Errorf("taken_at = %v, want RFC3339 timestamp", got)
	}
}

type fakeReadModel struct {
	journeys     []*domain.Journey
	mementos     map[uuid.UUID][]*domain.Memento
	photos       map[uuid.UUID][]*domain.MementoPhoto
	journal      *domain.Journal
	siteSettings map[uuid.UUID]*domain.SiteSettings
}

func (f *fakeReadModel) ListJourneys(context.Context) ([]*domain.Journey, error) {
	return f.journeys, nil
}

func (f *fakeReadModel) ListMementosByJourney(_ context.Context, id uuid.UUID) ([]*domain.Memento, error) {
	return f.mementos[id], nil
}

func (f *fakeReadModel) ListPhotosByMemento(_ context.Context, id uuid.UUID) ([]*domain.MementoPhoto, error) {
	return f.photos[id], nil
}

func (f *fakeReadModel) GetSoleJournal(context.Context) (*domain.Journal, error) {
	if f.journal == nil {
		return nil, domain.ErrNotFound
	}
	return f.journal, nil
}

func (f *fakeReadModel) GetSiteSettings(_ context.Context, journalID uuid.UUID) (*domain.SiteSettings, error) {
	settings, ok := f.siteSettings[journalID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return settings, nil
}

// fakeMediaSource serves a real (if tiny) JPEG for every key: the compiler
// sanitizes media it reads, so a stand-in must be a decodable image.
type fakeMediaSource struct{}

func (fakeMediaSource) Open(context.Context, string) (io.ReadCloser, error) {
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, gradient(8, 8), nil); err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(encoded.Bytes())), nil
}

type memoryArtifactWriter struct {
	json  map[string][]byte
	media []string
	// mediaBytes retains what was written so a test can prove the artifact
	// holds the sanitized derivative rather than the original.
	mediaBytes map[string][]byte
}

func (w *memoryArtifactWriter) WriteJSON(path string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if w.json == nil {
		w.json = map[string][]byte{}
	}
	w.json[path] = raw
	return nil
}

func (w *memoryArtifactWriter) WriteMedia(path string, source io.Reader) error {
	data, err := io.ReadAll(source)
	if err != nil {
		return err
	}
	if w.mediaBytes == nil {
		w.mediaBytes = map[string][]byte{}
	}
	w.mediaBytes[path] = data
	w.media = append(w.media, path)
	return nil
}

func TestCompileEmitsPublishedOnlyAndSkipsUnpublishedJourneys(t *testing.T) {
	journalID := uuid.New()
	publishedJourney := &domain.Journey{ID: uuid.New(), JournalID: journalID, Slug: "tokyo", Title: "Tokyo", Place: "Tokyo",
		DateStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), DateEnd: time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)}
	draftJourney := &domain.Journey{ID: uuid.New(), JournalID: journalID, Slug: "osaka-draft", Title: "Osaka draft", Place: "Osaka",
		DateStart: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), DateEnd: time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)}

	publishedMemento := &domain.Memento{ID: uuid.New(), JourneyID: publishedJourney.ID, Kind: "goods", Seq: 1,
		OccurredAt: time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC), OccurredTZ: "Asia/Tokyo",
		Geom: orb.Point{139.7, 35.6}, Title: "published stub", Place: "Akihabara",
		Essay: stringPtr("published essay"), State: domain.MementoPublished}
	draftMemento := &domain.Memento{ID: uuid.New(), JourneyID: publishedJourney.ID, Kind: "goods", Seq: 2,
		OccurredAt: time.Date(2026, 7, 2, 13, 0, 0, 0, time.UTC), OccurredTZ: "Asia/Tokyo",
		Geom: orb.Point{139.8, 35.7}, Title: "secret draft", Place: "Ueno",
		Essay: stringPtr("draft essay must not leak"), State: domain.MementoDraft}
	draftOnlyMemento := &domain.Memento{ID: uuid.New(), JourneyID: draftJourney.ID, Kind: "goods", Seq: 1,
		OccurredAt: time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC), OccurredTZ: "Asia/Tokyo",
		Geom: orb.Point{135.5, 34.7}, Title: "unpublished", Place: "Namba", State: domain.MementoDraft}

	read := &fakeReadModel{
		journeys: []*domain.Journey{publishedJourney, draftJourney},
		mementos: map[uuid.UUID][]*domain.Memento{
			publishedJourney.ID: {publishedMemento, draftMemento},
			draftJourney.ID:     {draftOnlyMemento},
		},
		photos:  map[uuid.UUID][]*domain.MementoPhoto{},
		journal: &domain.Journal{ID: journalID},
	}
	writer := &memoryArtifactWriter{}

	report, err := StaticCompiler{}.Compile(context.Background(), Input{}, read, fakeMediaSource{}, writer)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if report.Journeys != 1 || report.Mementos != 1 {
		t.Errorf("report = %+v, want 1 journey / 1 memento", report)
	}

	index := writer.json["api/v1/journeys.json"]
	var listed []JourneyListItem
	if err := json.Unmarshal(index, &listed); err != nil {
		t.Fatalf("unmarshal index: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != publishedJourney.ID {
		t.Fatalf("index = %s, want only the journey with published content", index)
	}
	if listed[0].MementoCount != 1 {
		t.Errorf("memento_count = %d, want published-only count 1", listed[0].MementoCount)
	}

	if _, ok := writer.json["api/v1/journeys/"+draftJourney.ID.String()+".json"]; ok {
		t.Errorf("draft-only journey detail must not be emitted")
	}
	if _, ok := writer.json["api/v1/journeys/"+draftJourney.ID.String()+"/mementos.json"]; ok {
		t.Errorf("draft-only journey mementos must not be emitted")
	}

	mementosJSON := writer.json["api/v1/journeys/"+publishedJourney.ID.String()+"/mementos.json"]
	if strings.Contains(string(mementosJSON), "draft essay must not leak") {
		t.Errorf("draft memento leaked into the public artifact: %s", mementosJSON)
	}
	if !strings.Contains(string(mementosJSON), "published essay") {
		t.Errorf("authored essay missing from the public artifact: %s", mementosJSON)
	}
}

// TestCompileWritesSiteSettingsWithDefaultsOnFreshDB asserts api/v1/site.json
// is always written, even on a fresh DB with a bootstrapped journal but no
// saved site settings — reflecting domain.DefaultSiteSettings (ADMIN-02 M2 §4).
func TestCompileWritesSiteSettingsWithDefaultsOnFreshDB(t *testing.T) {
	journalID := uuid.New()
	read := &fakeReadModel{journal: &domain.Journal{ID: journalID}}
	writer := &memoryArtifactWriter{}

	report, err := StaticCompiler{}.Compile(context.Background(), Input{}, read, fakeMediaSource{}, writer)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if report.Journeys != 0 {
		t.Errorf("report.Journeys = %d, want 0 on a fresh DB", report.Journeys)
	}

	raw, ok := writer.json["api/v1/site.json"]
	if !ok {
		t.Fatal("api/v1/site.json was not written on a fresh DB")
	}
	var got StaticSiteSettings
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal site.json: %v", err)
	}
	want := NewStaticSiteSettings(domain.DefaultSiteSettings(journalID))
	if got != want {
		t.Errorf("site.json = %+v, want defaults %+v", got, want)
	}
}

// staticMediaSource serves the same fixed bytes for every requested key.
type staticMediaSource struct{ data []byte }

func (s staticMediaSource) Open(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.data)), nil
}

// readModelWithPublishedPhoto builds the smallest read model that drives the
// compiler's media loop: one published memento carrying one photo.
func readModelWithPublishedPhoto(objectKey string) *fakeReadModel {
	journalID := uuid.New()
	journey := &domain.Journey{ID: uuid.New(), JournalID: journalID, Slug: "kyoto", Title: "Kyoto", Place: "Kyoto",
		DateStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), DateEnd: time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)}
	memento := &domain.Memento{ID: uuid.New(), JourneyID: journey.ID, Kind: "goods", Seq: 1,
		OccurredAt: time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC), OccurredTZ: "Asia/Tokyo",
		Geom: orb.Point{135.7, 35.0}, Title: "stub", Place: "Gion", State: domain.MementoPublished}
	photo := &domain.MementoPhoto{ID: uuid.New(), MementoID: memento.ID, ObjectKey: objectKey, ContentHash: "sha256:fixture", Seq: 1}
	return &fakeReadModel{
		journeys: []*domain.Journey{journey},
		mementos: map[uuid.UUID][]*domain.Memento{journey.ID: {memento}},
		photos:   map[uuid.UUID][]*domain.MementoPhoto{memento.ID: {photo}},
		journal:  &domain.Journal{ID: journalID},
	}
}

// TestCompileWritesSanitizedDerivativeNotTheOriginal is the compiler-level half
// of the privacy invariant: the artifact must hold the stripped derivative, and
// the object key must be unchanged so the JSON projection still resolves.
func TestCompileWritesSanitizedDerivativeNotTheOriginal(t *testing.T) {
	const objectKey = "media/kyoto.jpg"
	original := jpegWithGPSExif(t, gradient(64, 48), 1)
	read := readModelWithPublishedPhoto(objectKey)
	writer := &memoryArtifactWriter{}

	report, err := StaticCompiler{}.Compile(context.Background(), Input{}, read, staticMediaSource{data: original}, writer)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if report.Media != 1 {
		t.Fatalf("report.Media = %d, want 1", report.Media)
	}

	published, ok := writer.mediaBytes[objectKey]
	if !ok {
		t.Fatalf("compile did not write %s (wrote %v)", objectKey, writer.media)
	}
	if bytes.Equal(published, original) {
		t.Fatal("compile copied the original bytes verbatim")
	}
	if _, hasExif := jpegExifTIFF(published); hasExif {
		t.Error("published media still carries an APP1/Exif segment")
	}
	if bytes.Contains(published, gpsLatitudeSentinel) || bytes.Contains(published, gpsLongitudeSentinel) {
		t.Error("published media still carries the GPS coordinates")
	}
}

// TestCompileFailsOnUnsanitizableMedia: packaging must fail rather than publish
// a format whose metadata we cannot strip.
func TestCompileFailsOnUnsanitizableMedia(t *testing.T) {
	read := readModelWithPublishedPhoto("media/scan.tiff")
	writer := &memoryArtifactWriter{}

	_, err := StaticCompiler{}.Compile(context.Background(), Input{}, read, staticMediaSource{data: []byte("II*\x00")}, writer)
	if err == nil {
		t.Fatal("compile succeeded, want a failure instead of publishing unprocessed media")
	}
	if len(writer.media) != 0 {
		t.Errorf("compile wrote media %v, want none", writer.media)
	}
}
