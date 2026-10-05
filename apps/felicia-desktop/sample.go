package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
	publication "github.com/azusachino/felicia/apps/felicia-publication"
)

// Every opening starts from the same synthetic fixture in a fresh temporary root.
// This runs before resolving the author's workspace; no personal path is read.
func createSampleWorkspace() (root string, err error) {
	root, err = os.MkdirTemp("", "felicia-sample-")
	if err != nil {
		return "", err
	}
	createdRoot := root
	defer func() {
		if err != nil {
			_ = os.RemoveAll(createdRoot)
		}
	}()
	media := filepath.Join(root, "media")
	if err = os.MkdirAll(media, 0o700); err != nil {
		return "", err
	}
	repo, err := sqlite.Open(filepath.Join(root, "felicia.sqlite"))
	if err != nil {
		return "", err
	}
	defer func() { _ = repo.Close() }()
	ctx := context.Background()
	journalID := uuid.MustParse("0190cbde-f300-7000-8000-000000000001")
	journeyID := uuid.MustParse("0190cbde-f300-7000-8000-000000000002")
	day := time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)
	if err = repo.CreateJournal(ctx, &domain.Journal{ID: journalID, CreatedAt: day}); err != nil {
		return "", err
	}
	points := []orb.Point{{135.7671, 34.9858}, {135.7727, 35.0030}, {135.7780, 35.0120}}
	journey := &domain.Journey{ID: journeyID, JournalID: journalID, Slug: "sample-kyoto-afternoon", Title: "A Kyoto afternoon · Sample trip", Place: "Kyoto, Japan", DateStart: day, DateEnd: day.Add(24 * time.Hour), GPSRoute: orb.MultiLineString{orb.LineString(points)}, AuthoredFields: domain.ClaimJourneyAuthorship(nil), CreatedAt: day}
	if err = repo.UpsertJourney(ctx, journey); err != nil {
		return "", err
	}
	entries := []struct {
		title, essay string
		state        domain.MementoState
	}{
		{"A little paper keepsake", "This is a fictional afternoon, not a record of a real person's travels.\n\nA memento is anything that anchors a memory: a ticket, a receipt, a postcard. Open this one to try editing the title and essay. The two pictures are generated illustrations, not personal photos.\n\nSave keeps your changes in this sample workspace. Build updates the site preview. Closing the sample leaves your real trips untouched.", domain.MementoPublished},
		{"Coffee by the river", "A draft is a place to write before publishing. Try changing this sentence, then Save. This sample starts fresh the next time you open it.", domain.MementoDraft},
		{"A postcard waiting for a story", "Candidates are memories waiting to be reviewed. Give this one a title and a story when you are ready.", domain.MementoCandidateState},
	}
	var firstID uuid.UUID
	for i, entry := range entries {
		id := uuid.MustParse(fmt.Sprintf("0190cbde-f300-7000-8000-%012d", i+10))
		if i == 0 {
			firstID = id
		}
		data, _ := json.Marshal(map[string]string{"name": entry.title, "origin": "Fictional sample"})
		item := &domain.Memento{ID: id, JourneyID: journeyID, Kind: "souvenir", Seq: i, OccurredAt: day.Add(time.Duration(i+9) * time.Hour), OccurredTZ: "Asia/Tokyo", Geom: points[i], Title: entry.title, Place: "Kyoto · fictional stop", Essay: &entry.essay, KindData: data, State: entry.state, AuthoredFields: []string{"title", "essay", "kind", "kind_data", "place", "geom", "occurred_at", "occurred_tz"}, CreatedAt: day}
		if err = repo.UpsertMemento(ctx, item); err != nil {
			return "", err
		}
	}
	for i := range 2 {
		pic := image.NewRGBA(image.Rect(0, 0, 960, 640))
		draw.Draw(pic, pic.Bounds(), &image.Uniform{color.RGBA{225, 231, 227, 255}}, image.Point{}, draw.Src)
		draw.Draw(pic, image.Rect(0, 360, 960, 640), &image.Uniform{color.RGBA{91, 128, 114, 255}}, image.Point{}, draw.Src)
		draw.Draw(pic, image.Rect(100+i*260, 150, 360+i*260, 430), &image.Uniform{color.RGBA{190, 123, 72, 255}}, image.Point{}, draw.Src)
		var encoded bytes.Buffer
		if err = png.Encode(&encoded, pic); err != nil {
			return "", err
		}
		hash := sha256.Sum256(encoded.Bytes())
		key := hex.EncodeToString(hash[:]) + ".png"
		if err = os.WriteFile(filepath.Join(media, key), encoded.Bytes(), 0o600); err != nil {
			return "", err
		}
		caption := fmt.Sprintf("Generated sample illustration %d · no personal photography", i+1)
		photo := &domain.MementoPhoto{ID: uuid.MustParse(fmt.Sprintf("0190cbde-f300-7000-8000-%012d", i+20)), MementoID: firstID, ObjectKey: key, ContentHash: hex.EncodeToString(hash[:]), Caption: &caption, Seq: i, CreatedAt: day}
		if err = repo.UpsertPhoto(ctx, photo); err != nil {
			return "", err
		}
	}
	writer := &publication.FileArtifactWriter{Root: filepath.Join(root, "site")}
	_, err = (publication.StaticCompiler{}).Compile(ctx, publication.Input{}, repo, publication.FileMediaSource{Root: media}, writer)
	if err != nil {
		return "", err
	}
	if _, err = writer.Finalize(); err != nil {
		return "", err
	}
	return root, nil
}
