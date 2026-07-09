package pg_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/internal/domain"
	"github.com/azusachino/felicia/internal/store/pg"
)

func TestPgRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN environment variable not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Clean up old test data to ensure a reproducible run
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE journal CASCADE")

	repo := pg.NewRepository(pool)

	// 1. Create a Journal
	journal := &domain.Journal{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	err = repo.CreateJournal(ctx, journal)
	if err != nil {
		t.Fatalf("failed to create journal: %v", err)
	}

	// Retrieve Journal
	fetchedJournal, err := repo.GetJournal(ctx, journal.ID)
	if err != nil {
		t.Fatalf("failed to get journal: %v", err)
	}
	if fetchedJournal.ID != journal.ID {
		t.Errorf("expected journal ID %s, got %s", journal.ID, fetchedJournal.ID)
	}

	// 2. Upsert a Journey
	gpsTrack := orb.MultiLineString{
		orb.LineString{
			orb.Point{139.7671, 35.6812}, // Tokyo Station
			orb.Point{139.7003, 35.6895}, // Shinjuku Station
		},
	}
	journey := &domain.Journey{
		ID:             uuid.New(),
		JournalID:      journal.ID,
		Slug:           "japan-spring-2026",
		Title:          "日本春旅 2026",
		Place:          "東京",
		DateStart:      time.Now().UTC().Truncate(time.Hour),
		DateEnd:        time.Now().UTC().Add(24 * time.Hour).Truncate(time.Hour),
		GPSRoute:       gpsTrack,
		AuthoredFields: []string{},
	}
	err = repo.UpsertJourney(ctx, journey)
	if err != nil {
		t.Fatalf("failed to upsert journey: %v", err)
	}

	// Fetch and Assert
	fetchedJourney, err := repo.GetJourney(ctx, journey.ID)
	if err != nil {
		t.Fatalf("failed to get journey: %v", err)
	}
	if fetchedJourney.Title != journey.Title {
		t.Errorf("expected title %s, got %s", journey.Title, fetchedJourney.Title)
	}
	if len(fetchedJourney.GPSRoute) != 1 || len(fetchedJourney.GPSRoute[0]) != 2 {
		t.Errorf("invalid gps track length scanned")
	}

	// Test "No-Clobber" Upsert:
	// A. Title is authored
	journey.Title = "Edited Title (Human)"
	journey.AuthoredFields = []string{"title"}
	err = repo.UpsertJourney(ctx, journey)
	if err != nil {
		t.Fatalf("failed to update journey: %v", err)
	}

	// B. Try to overwrite title via automated importer update (without authoring it)
	journey2 := *journey
	journey2.Title = "Overwritten Title (Machine)"
	err = repo.UpsertJourney(ctx, &journey2)
	if err != nil {
		t.Fatalf("failed to upsert journey: %v", err)
	}

	// C. Fetch again. The Title should remain "Edited Title (Human)" since "title" is in authored_fields
	fetchedJourney2, err := repo.GetJourney(ctx, journey.ID)
	if err != nil {
		t.Fatalf("failed to get journey: %v", err)
	}
	if fetchedJourney2.Title != "Edited Title (Human)" {
		t.Errorf("expected Title to be protected and remain 'Edited Title (Human)', but got: %s", fetchedJourney2.Title)
	}

	// 3. Upsert a Memento
	memento := &domain.Memento{
		ID:             uuid.New(),
		JourneyID:      journey.ID,
		Kind:           "ticket",
		Seq:            1,
		OccurredAt:     time.Now().UTC().Truncate(time.Microsecond),
		OccurredTZ:     "Asia/Tokyo",
		Geom:           orb.Point{139.7671, 35.6812},
		Title:          "東京駅入場券",
		Place:          "東京駅",
		KindData:       []byte(`{"operator":"JR East"}`),
		AuthoredFields: []string{},
	}
	err = repo.UpsertMemento(ctx, memento)
	if err != nil {
		t.Fatalf("failed to upsert memento: %v", err)
	}

	// Fetch Memento
	fetchedMemento, err := repo.GetMemento(ctx, memento.ID)
	if err != nil {
		t.Fatalf("failed to get memento: %v", err)
	}
	if fetchedMemento.Title != memento.Title {
		t.Errorf("expected memento title %s, got %s", memento.Title, fetchedMemento.Title)
	}
	pt, ok := fetchedMemento.Geom.(orb.Point)
	if !ok || pt.X() != 139.7671 || pt.Y() != 35.6812 {
		t.Errorf("expected memento geometry Point(139.7671, 35.6812)")
	}

	// 4. Upsert a Photo
	photo := &domain.MementoPhoto{
		ID:          uuid.New(),
		MementoID:   memento.ID,
		ObjectKey:   "media/photo1.jpg",
		ContentHash: "hash123456",
		Caption:     nil,
		Seq:         1,
	}
	err = repo.UpsertPhoto(ctx, photo)
	if err != nil {
		t.Fatalf("failed to upsert photo: %v", err)
	}

	fetchedPhoto, err := repo.GetPhoto(ctx, photo.ID)
	if err != nil {
		t.Fatalf("failed to get photo: %v", err)
	}
	if fetchedPhoto.ObjectKey != photo.ObjectKey {
		t.Errorf("expected photo object key %s, got %s", photo.ObjectKey, fetchedPhoto.ObjectKey)
	}

	// 5. Upsert and List Translations
	trans := &domain.Translation{
		ID:         uuid.New(),
		OwnerType:  "memento",
		OwnerID:    memento.ID,
		Lang:       "en",
		Field:      "title",
		Value:      "Tokyo Station Ticket",
		Provenance: "machine",
	}
	err = repo.UpsertTranslation(ctx, trans)
	if err != nil {
		t.Fatalf("failed to upsert translation: %v", err)
	}

	// Test No-Clobber for translations:
	// A. Human edits translation value
	trans.Value = "Tokyo Station Admission Ticket (Human)"
	trans.Provenance = "authored"
	err = repo.UpsertTranslation(ctx, trans)
	if err != nil {
		t.Fatalf("failed to update translation: %v", err)
	}

	// B. Importer tries to overwrite it with machine translation
	trans2 := *trans
	trans2.Value = "Tokyo Station Ticket (Machine)"
	trans2.Provenance = "machine"
	err = repo.UpsertTranslation(ctx, &trans2)
	if err != nil {
		t.Fatalf("failed to upsert translation: %v", err)
	}

	// C. Fetch and assert the value is protected
	translations, err := repo.ListTranslations(ctx, "memento", memento.ID)
	if err != nil {
		t.Fatalf("failed to list translations: %v", err)
	}
	if len(translations) != 1 {
		t.Fatalf("expected 1 translation, got %d", len(translations))
	}
	if translations[0].Value != "Tokyo Station Admission Ticket (Human)" {
		t.Errorf("expected translation value to be protected, got: %s", translations[0].Value)
	}
}
