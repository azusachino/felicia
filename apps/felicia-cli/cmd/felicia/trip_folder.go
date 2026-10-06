package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-providers/local"
	"github.com/azusachino/felicia/apps/felicia-runtime/importer"
	"github.com/azusachino/felicia/apps/felicia-runtime/intake"
	"github.com/azusachino/felicia/apps/felicia-runtime/timezone"
)

// TripFolderConfig controls single-step ingestion of a trip folder.
type TripFolderConfig struct {
	Dir           string
	Slug          string
	Title         string
	Place         string
	WorkspaceRoot string
	Database      string
	MediaRoot     string
	From          time.Time
	To            time.Time
}

// IngestTripFolder scans a staged trip folder, plans its draft stops and mementos,
// installs its photos into mediaRoot under content-addressed digests, and applies
// the resulting package document to the canonical store.
func IngestTripFolder(ctx context.Context, cfg TripFolderConfig, repo importer.PackageStore, mediaRoot string) (*importer.ImportReport, error) {
	if cfg.Dir == "" {
		return nil, errors.New("trip folder directory is required")
	}
	if mediaRoot == "" {
		mediaRoot = cfg.MediaRoot
	}
	if mediaRoot == "" {
		return nil, errors.New("media root is required")
	}
	if repo == nil {
		return nil, errors.New("package store is required")
	}

	absDir, err := filepath.Abs(cfg.Dir)
	if err != nil || !dirExists(absDir) {
		return nil, fmt.Errorf("trip directory %q does not exist", cfg.Dir)
	}

	gpxPath, timelinePath, photosDir, sidecarPath, err := scanTripFolder(absDir)
	if err != nil {
		return nil, err
	}

	// 1. Resolve Journey identity
	slug := cfg.Slug
	if slug == "" {
		slug = slugify(filepath.Base(absDir))
	}

	var journeyID uuid.UUID
	var journalID uuid.UUID

	existingJourney, err := repo.GetJourneyBySlug(ctx, slug)
	if err == nil && existingJourney != nil {
		journeyID = existingJourney.ID
		journalID = existingJourney.JournalID
	} else {
		soleJournal, err := repo.GetSoleJournal(ctx)
		if err == nil && soleJournal != nil {
			journalID = soleJournal.ID
		} else {
			journalID = uuid.Must(uuid.NewV7())
			if err := repo.EnsureJournal(ctx, &domain.Journal{ID: journalID, CreatedAt: time.Now().UTC()}); err != nil {
				return nil, fmt.Errorf("ensure journal: %w", err)
			}
		}
		journeyID = uuid.Must(uuid.NewV7())
	}

	title := cfg.Title
	if title == "" {
		if existingJourney != nil && existingJourney.Title != "" {
			title = existingJourney.Title
		} else {
			title = titleFromSlug(slug)
		}
	}

	place := cfg.Place
	if place == "" {
		if existingJourney != nil && existingJourney.Place != "" {
			place = existingJourney.Place
		} else {
			place = title
		}
	}

	// 2. Wire local source adapters
	var sources intake.SourceSet
	if gpxPath != "" {
		sources.Routes = local.NewGPXSource(gpxPath)
	}
	if timelinePath != "" {
		sources.Visits = local.NewTimelineSource(timelinePath)
	}
	var photoAssets []domain.PhotoAsset
	if photosDir != "" {
		photoSource := local.NewPhotoSourceWithSidecar(photosDir, sidecarPath)
		sources.Media = photoSource
		if fetched, err := photoSource.FetchAssets(ctx, cfg.From, cfg.To); err == nil {
			photoAssets = fetched
		}
	}

	fingerprint, err := sourceFingerprint(gpxPath, timelinePath)
	if err != nil {
		return nil, err
	}

	// 3. Draft planning
	planService := intake.NewService(nil, nil)
	plan, err := planService.Plan(ctx, intake.PlanRequest{
		JourneyID:         journeyID,
		From:              cfg.From,
		To:                cfg.To,
		SourceFingerprint: fingerprint,
		Sources:           sources,
	})
	if err != nil {
		return nil, fmt.Errorf("plan trip folder: %w", err)
	}

	// 4. Install media files into mediaRoot
	installedPhotos, err := installMediaFiles(ctx, photosDir, mediaRoot, photoAssets)
	if err != nil {
		return nil, fmt.Errorf("install media files: %w", err)
	}

	// 5. Build PackageDocument
	doc, err := buildPackageDocument(journeyID, journalID, slug, title, place, cfg, plan, installedPhotos)
	if err != nil {
		return nil, fmt.Errorf("build package document: %w", err)
	}

	// 6. Apply package
	report, err := importer.ApplyPackage(ctx, doc, repo)
	if err != nil {
		return nil, fmt.Errorf("apply package: %w", err)
	}
	report.JourneyID = journeyID
	report.Slug = slug
	return &report, nil
}

func scanTripFolder(dir string) (gpxPath, timelinePath, photosDir, sidecarPath string, err error) {
	routeGPX := filepath.Join(dir, "route.gpx")
	if fileExists(routeGPX) {
		gpxPath = routeGPX
	} else {
		entries, err := os.ReadDir(dir)
		if err == nil {
			var gpxFiles []string
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".gpx") {
					gpxFiles = append(gpxFiles, filepath.Join(dir, e.Name()))
				}
			}
			sort.Strings(gpxFiles)
			if len(gpxFiles) > 0 {
				gpxPath = gpxFiles[0]
			}
		}
	}

	tl1 := filepath.Join(dir, "timeline.json")
	tl2 := filepath.Join(dir, "Timeline.json")
	if fileExists(tl1) {
		timelinePath = tl1
	} else if fileExists(tl2) {
		timelinePath = tl2
	}

	if gpxPath == "" && timelinePath == "" {
		return "", "", "", "", errors.New("trip folder requires route.gpx or timeline.json")
	}

	photosCandidate := filepath.Join(dir, "photos")
	if dirExists(photosCandidate) {
		photosDir = photosCandidate
	} else {
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, e := range entries {
				if !e.IsDir() && isImageFile(e.Name()) {
					photosDir = dir
					break
				}
			}
		}
	}

	sidecarCandidate := filepath.Join(dir, "photos.jsonl")
	if fileExists(sidecarCandidate) {
		sidecarPath = sidecarCandidate
	}

	return gpxPath, timelinePath, photosDir, sidecarPath, nil
}

type installedPhoto struct {
	Path      string
	Digest    string
	ObjectKey string
	Asset     domain.MediaAsset
}

func installMediaFiles(ctx context.Context, photosDir, mediaRoot string, assets []domain.PhotoAsset) ([]installedPhoto, error) {
	if photosDir == "" {
		return nil, nil
	}
	var filePaths []string
	err := filepath.WalkDir(photosDir, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if isImageFile(d.Name()) {
			filePaths = append(filePaths, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk photos directory %s: %w", photosDir, err)
	}
	sort.Strings(filePaths)

	assetByRel := make(map[string]domain.PhotoAsset, len(assets))
	for _, a := range assets {
		assetByRel[filepath.ToSlash(a.ID)] = a
		assetByRel[filepath.Base(a.ID)] = a
	}

	installed := make([]installedPhoto, 0, len(filePaths))
	seenDigests := make(map[string]bool)

	for _, fp := range filePaths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		data, err := os.ReadFile(fp)
		if err != nil {
			return nil, fmt.Errorf("read photo %s: %w", fp, err)
		}
		digest := importer.MediaDigest(data)
		if seenDigests[digest] {
			continue
		}
		seenDigests[digest] = true

		rel, err := filepath.Rel(photosDir, fp)
		if err != nil {
			rel = filepath.Base(fp)
		}
		objectKey := importer.MediaObjectKey(filepath.ToSlash(rel), digest)
		targetPath := filepath.Join(mediaRoot, filepath.FromSlash(objectKey))

		if !fileExists(targetPath) {
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
				return nil, fmt.Errorf("create media directory %s: %w", filepath.Dir(targetPath), err)
			}
			if err := writeAtomic(targetPath, data); err != nil {
				return nil, fmt.Errorf("install media %s: %w", fp, err)
			}
		}

		asset, ok := assetByRel[filepath.ToSlash(rel)]
		if !ok {
			asset, ok = assetByRel[filepath.Base(fp)]
			if !ok {
				asset = domain.MediaAsset{
					ID:       filepath.ToSlash(rel),
					Kind:     domain.MediaImage,
					Checksum: "sha256:" + digest,
					Title:    strings.TrimSuffix(filepath.Base(fp), filepath.Ext(fp)),
					Provider: "local",
				}
			}
		}

		installed = append(installed, installedPhoto{
			Path:      fp,
			Digest:    digest,
			ObjectKey: objectKey,
			Asset:     asset,
		})
	}
	return installed, nil
}

func buildPackageDocument(
	journeyID, journalID uuid.UUID,
	slug, title, place string,
	cfg TripFolderConfig,
	plan intake.DraftPlan,
	installedPhotos []installedPhoto,
) (*importer.PackageDocument, error) {
	dateStart := plan.DateStart
	dateEnd := plan.DateEnd
	if dateStart.IsZero() {
		if !cfg.From.IsZero() {
			dateStart = cfg.From
		} else {
			dateStart = time.Now().UTC()
		}
	}
	if dateEnd.IsZero() {
		if !cfg.To.IsZero() {
			dateEnd = cfg.To
		} else {
			dateEnd = dateStart
		}
	}
	if dateEnd.Before(dateStart) {
		dateEnd = dateStart
	}
	dateStart = time.Date(dateStart.Year(), dateStart.Month(), dateStart.Day(), 0, 0, 0, 0, time.UTC)
	dateEnd = time.Date(dateEnd.Year(), dateEnd.Month(), dateEnd.Day(), 0, 0, 0, 0, time.UTC)

	var gpsRoute orb.MultiLineString
	for _, route := range plan.Routes {
		if len(route.Line) >= 2 {
			gpsRoute = append(gpsRoute, route.Line)
		}
	}

	sourceRef := "trip-folder"
	journey := &domain.Journey{
		ID:        journeyID,
		JournalID: journalID,
		Slug:      slug,
		SourceRef: &sourceRef,
		Title:     title,
		Place:     place,
		DateStart: dateStart,
		DateEnd:   dateEnd,
		GPSRoute:  gpsRoute,
	}

	doc := &importer.PackageDocument{
		Journey:  journey,
		Stops:    make([]*domain.StopCandidate, 0, len(plan.Stops)),
		Mementos: make([]*domain.Memento, 0, len(plan.Mementos)),
		Photos:   make([]*domain.MementoPhoto, 0),
	}

	for i := range plan.Stops {
		sc := plan.Stops[i]
		if sc.ID == uuid.Nil {
			sc.ID = uuid.NewSHA1(uuid.Nil, []byte("journey:"+journeyID.String()+":stop:"+sc.Identity.Key))
		}
		if sc.Label == "" {
			sc.Label = fmt.Sprintf("Stop %d", i+1)
		}
		if sc.Confidence <= 0 {
			sc.Confidence = 0.5
		}
		doc.Stops = append(doc.Stops, &sc)
	}

	photoFilesByDigest := make(map[string]installedPhoto, len(installedPhotos))
	for _, ip := range installedPhotos {
		photoFilesByDigest[ip.Digest] = ip
	}
	attachedPhotos := make(map[string]bool, len(installedPhotos))

	mementoSeq := 1
	for _, cand := range plan.Mementos {
		mementoID := uuid.NewSHA1(uuid.Nil, []byte("journey:"+journeyID.String()+":memento:"+cand.StopKey))
		kind := cand.Kind
		if kind == "" {
			kind = "souvenir"
		}
		memTitle := cand.Title
		if memTitle == "" {
			memTitle = cand.Place
		}
		if memTitle == "" {
			memTitle = fmt.Sprintf("Stop %d", mementoSeq)
		}
		memPlace := cand.Place
		if memPlace == "" {
			memPlace = memTitle
		}
		kindData := cand.KindData
		if kindData == nil {
			kindData = map[string]any{}
		}
		if kindData["name"] == nil || kindData["name"] == "" {
			kindData["name"] = memTitle
		}
		kindDataBytes, err := json.Marshal(kindData)
		if err != nil {
			return nil, fmt.Errorf("marshal memento %s kind data: %w", mementoID, err)
		}

		tz := cand.OccurredTZ
		if tz == "" {
			tz = timezone.Default("", cand.Geom, timezone.Journey(gpsRoute))
		}

		memento := &domain.Memento{
			ID:         mementoID,
			JourneyID:  journeyID,
			Kind:       kind,
			Seq:        mementoSeq,
			OccurredAt: cand.OccurredAt,
			OccurredTZ: tz,
			Geom:       cand.Geom,
			Title:      memTitle,
			Place:      memPlace,
			KindData:   kindDataBytes,
			State:      domain.MementoCandidateState,
		}
		doc.Mementos = append(doc.Mementos, memento)

		photoSeq := 1
		for _, mediaAsset := range cand.Media {
			digest := strings.TrimPrefix(mediaAsset.Checksum, "sha256:")
			if digest == "" || attachedPhotos[digest] {
				continue
			}
			photoFile, ok := photoFilesByDigest[digest]
			if !ok {
				continue
			}
			attachedPhotos[digest] = true
			photoID := uuid.NewSHA1(uuid.Nil, []byte("photo:"+journeyID.String()+":"+digest))
			var takenAtPtr *time.Time
			if !mediaAsset.At.IsZero() {
				t := mediaAsset.At
				takenAtPtr = &t
			}
			var captionPtr *string
			if mediaAsset.Title != "" {
				c := mediaAsset.Title
				captionPtr = &c
			}
			srcRef := mediaAsset.SourceRef
			var srcRefPtr *string
			if srcRef != "" {
				srcRefPtr = &srcRef
			}
			doc.Photos = append(doc.Photos, &domain.MementoPhoto{
				ID:          photoID,
				MementoID:   mementoID,
				ObjectKey:   photoFile.ObjectKey,
				ContentHash: "sha256:" + digest,
				Caption:     captionPtr,
				Seq:         photoSeq,
				TakenAt:     takenAtPtr,
				SourceRef:   srcRefPtr,
				CreatedAt:   time.Now().UTC(),
			})
			photoSeq++
		}
		mementoSeq++
	}

	var unattachedPhotoFiles []installedPhoto
	for _, pf := range installedPhotos {
		if !attachedPhotos[pf.Digest] {
			unattachedPhotoFiles = append(unattachedPhotoFiles, pf)
		}
	}

	if len(unattachedPhotoFiles) > 0 {
		var targetMemento *domain.Memento
		if len(doc.Mementos) == 0 {
			defaultID := uuid.NewSHA1(uuid.Nil, []byte("journey:"+journeyID.String()+":memento:default"))
			kindDataBytes, _ := json.Marshal(map[string]any{"name": title})
			var geom orb.Geometry
			if len(gpsRoute) > 0 && len(gpsRoute[0]) > 0 {
				geom = gpsRoute[0][0]
			}
			targetMemento = &domain.Memento{
				ID:         defaultID,
				JourneyID:  journeyID,
				Kind:       "souvenir",
				Seq:        1,
				OccurredAt: dateStart,
				OccurredTZ: timezone.Default("", geom, timezone.Journey(gpsRoute)),
				Geom:       geom,
				Title:      title,
				Place:      place,
				KindData:   kindDataBytes,
				State:      domain.MementoCandidateState,
			}
			doc.Mementos = append(doc.Mementos, targetMemento)
		} else {
			targetMemento = doc.Mementos[0]
		}

		currentSeq := 0
		for _, p := range doc.Photos {
			if p.MementoID == targetMemento.ID && p.Seq > currentSeq {
				currentSeq = p.Seq
			}
		}

		for _, pf := range unattachedPhotoFiles {
			attachedPhotos[pf.Digest] = true
			currentSeq++
			photoID := uuid.NewSHA1(uuid.Nil, []byte("photo:"+journeyID.String()+":"+pf.Digest))
			var takenAtPtr *time.Time
			if !pf.Asset.At.IsZero() {
				t := pf.Asset.At
				takenAtPtr = &t
			}
			var captionPtr *string
			if pf.Asset.Title != "" {
				c := pf.Asset.Title
				captionPtr = &c
			}
			var srcRefPtr *string
			if pf.Asset.SourceRef != "" {
				s := pf.Asset.SourceRef
				srcRefPtr = &s
			}
			doc.Photos = append(doc.Photos, &domain.MementoPhoto{
				ID:          photoID,
				MementoID:   targetMemento.ID,
				ObjectKey:   pf.ObjectKey,
				ContentHash: "sha256:" + pf.Digest,
				Caption:     captionPtr,
				Seq:         currentSeq,
				TakenAt:     takenAtPtr,
				SourceRef:   srcRefPtr,
				CreatedAt:   time.Now().UTC(),
			})
		}
	}

	return doc, nil
}

func isImageFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	default:
		return false
	}
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func writeAtomic(target string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(target), ".felicia-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpName, target)
}

func slugify(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var buf strings.Builder
	lastDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			buf.WriteRune(r)
			lastDash = false
		} else if !lastDash && buf.Len() > 0 {
			buf.WriteByte('-')
			lastDash = true
		}
	}
	res := strings.Trim(buf.String(), "-")
	if res == "" {
		return "journey"
	}
	return res
}

func titleFromSlug(slug string) string {
	parts := strings.FieldsFunc(slug, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, part := range parts {
		if len(part) > 0 {
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			parts[i] = string(runes)
		}
	}
	res := strings.Join(parts, " ")
	if res == "" {
		return "Journey"
	}
	return res
}
