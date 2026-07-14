package pg_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/core/domain"
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

}

func TestPgTransitLegsAndRoute(t *testing.T) {
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

	_, _ = pool.Exec(ctx, "TRUNCATE TABLE journal CASCADE")
	repo := pg.NewRepository(pool)

	journal := &domain.Journal{ID: uuid.New(), CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	if err := repo.CreateJournal(ctx, journal); err != nil {
		t.Fatalf("failed to create journal: %v", err)
	}

	// A journey with a 2-point Dawarich track.
	journey := &domain.Journey{
		ID:        uuid.New(),
		JournalID: journal.ID,
		Slug:      "leg-test-2026",
		Title:     "航路テスト",
		Place:     "東京",
		DateStart: time.Now().UTC().Truncate(time.Hour),
		DateEnd:   time.Now().UTC().Add(24 * time.Hour).Truncate(time.Hour),
		GPSRoute: orb.MultiLineString{
			orb.LineString{orb.Point{139.7671, 35.6812}, orb.Point{139.7003, 35.6895}},
		},
		AuthoredFields: []string{},
	}
	if err := repo.UpsertJourney(ctx, journey); err != nil {
		t.Fatalf("failed to upsert journey: %v", err)
	}

	// Add a HND -> KIX geodesic leg (100km segments).
	hnd := orb.Point{139.7798, 35.5494}
	kix := orb.Point{135.2381, 34.4342}
	legID := uuid.New()
	origin, dest := "HND", "KIX"
	if err := repo.CreateTransitLeg(ctx, &domain.TransitLegInput{
		ID:          legID,
		JourneyID:   journey.ID,
		Seq:         0,
		OriginLabel: &origin,
		DestLabel:   &dest,
		Origin:      hnd,
		Dest:        kix,
		SegmentLenM: 100000,
	}); err != nil {
		t.Fatalf("failed to create transit leg: %v", err)
	}

	// The leg geom must be a densified great-circle arc, not a straight chord.
	legs, err := repo.ListTransitLegsByJourney(ctx, journey.ID)
	if err != nil {
		t.Fatalf("failed to list transit legs: %v", err)
	}
	if len(legs) != 1 {
		t.Fatalf("expected 1 leg, got %d", len(legs))
	}
	if len(legs[0].Geom) <= 2 {
		t.Errorf("expected densified arc (>2 points), got %d", len(legs[0].Geom))
	}
	if legs[0].OriginLabel == nil || *legs[0].OriginLabel != "HND" {
		t.Errorf("expected origin label HND, got %v", legs[0].OriginLabel)
	}

	// The display route is the union of the GPS track (1 segment) and the leg
	// (1 segment) -> at least 2 line segments.
	route, err := repo.GetDisplayRoute(ctx, journey.ID)
	if err != nil {
		t.Fatalf("failed to get display route: %v", err)
	}
	if len(route) < 2 {
		t.Errorf("expected >=2 segments in composed route, got %d", len(route))
	}

	// Snapping an off-route point returns a point that lies within the route's
	// extent (proximity snap over track ∪ legs). Exact on-route distance is
	// verified by the ST_Distance smoke test; here we assert a sane result.
	snapped, err := repo.SnapToRoute(ctx, journey.ID, orb.Point{137.0, 35.0})
	if err != nil {
		t.Fatalf("failed to snap to route: %v", err)
	}
	if snapped == nil {
		t.Fatal("expected a snapped point, got nil")
	}
	if snapped.X() < 135.0 || snapped.X() > 140.0 || snapped.Y() < 34.0 || snapped.Y() > 36.0 {
		t.Errorf("snapped point %v is outside the route extent", *snapped)
	}

	// Deleting the leg leaves no legs.
	if err := repo.DeleteTransitLeg(ctx, legID); err != nil {
		t.Fatalf("failed to delete transit leg: %v", err)
	}
	legs, err = repo.ListTransitLegsByJourney(ctx, journey.ID)
	if err != nil {
		t.Fatalf("failed to list transit legs after delete: %v", err)
	}
	if len(legs) != 0 {
		t.Errorf("expected 0 legs after delete, got %d", len(legs))
	}

	// A journey with neither track nor legs snaps to nil.
	bare := &domain.Journey{
		ID:             uuid.New(),
		JournalID:      journal.ID,
		Slug:           "bare-2026",
		Title:          "空路",
		Place:          "無",
		DateStart:      time.Now().UTC().Truncate(time.Hour),
		DateEnd:        time.Now().UTC().Add(24 * time.Hour).Truncate(time.Hour),
		AuthoredFields: []string{},
	}
	if err := repo.UpsertJourney(ctx, bare); err != nil {
		t.Fatalf("failed to upsert bare journey: %v", err)
	}
	snapped, err = repo.SnapToRoute(ctx, bare.ID, orb.Point{137.0, 35.0})
	if err != nil {
		t.Fatalf("failed to snap on empty route: %v", err)
	}
	if snapped != nil {
		t.Errorf("expected nil snap on empty route, got %v", *snapped)
	}
}
