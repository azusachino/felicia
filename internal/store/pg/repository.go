// Package pg implements the PostgreSQL 18 database store repository.
package pg

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkb"

	"github.com/azusachino/felicia/internal/domain"
	"github.com/azusachino/felicia/internal/store/pg/db"
)

type pgRepository struct {
	q *db.Queries
}

// NewRepository creates a new domain.Repository implementation backed by Postgres.
func NewRepository(d db.DBTX) domain.Repository {
	return &pgRepository{
		q: db.New(d),
	}
}

// Journal operations

func (r *pgRepository) GetJournal(ctx context.Context, id uuid.UUID) (*domain.Journal, error) {
	row, err := r.q.GetJournal(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domain.Journal{
		ID:        row.ID,
		CreatedAt: fromTimestamptz(row.CreatedAt),
	}, nil
}

func (r *pgRepository) CreateJournal(ctx context.Context, journal *domain.Journal) error {
	return r.q.CreateJournal(ctx, db.CreateJournalParams{
		ID:        journal.ID,
		CreatedAt: toTimestamptz(journal.CreatedAt),
	})
}

// Journey operations

func (r *pgRepository) GetJourney(ctx context.Context, id uuid.UUID) (*domain.Journey, error) {
	row, err := r.q.GetJourney(ctx, id)
	if err != nil {
		return nil, err
	}
	return toDomainJourney(row.ID, row.JournalID, row.Slug, row.SourceRef, row.Title, row.Place, row.Country, row.Region, row.DateStart, row.DateEnd, row.GpsRouteWkb, row.AuthoredFields, row.CreatedAt, row.UpdatedAt)
}

func (r *pgRepository) GetJourneyBySlug(ctx context.Context, slug string) (*domain.Journey, error) {
	row, err := r.q.GetJourneyBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return toDomainJourney(row.ID, row.JournalID, row.Slug, row.SourceRef, row.Title, row.Place, row.Country, row.Region, row.DateStart, row.DateEnd, row.GpsRouteWkb, row.AuthoredFields, row.CreatedAt, row.UpdatedAt)
}

func (r *pgRepository) ListJourneys(ctx context.Context) ([]*domain.Journey, error) {
	rows, err := r.q.ListJourneys(ctx)
	if err != nil {
		return nil, err
	}
	var res []*domain.Journey
	for _, row := range rows {
		j, err := toDomainJourney(row.ID, row.JournalID, row.Slug, row.SourceRef, row.Title, row.Place, row.Country, row.Region, row.DateStart, row.DateEnd, row.GpsRouteWkb, row.AuthoredFields, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		res = append(res, j)
	}
	return res, nil
}

func (r *pgRepository) UpsertJourney(ctx context.Context, journey *domain.Journey) error {
	var gpsRouteBytes []byte
	var err error
	if len(journey.GPSRoute) > 0 {
		gpsRouteBytes, err = wkb.Marshal(journey.GPSRoute)
		if err != nil {
			return err
		}
	}

	return r.q.UpsertJourney(ctx, db.UpsertJourneyParams{
		ID:             journey.ID,
		JournalID:      journey.JournalID,
		Slug:           journey.Slug,
		SourceRef:      toText(journey.SourceRef),
		Title:          journey.Title,
		Place:          journey.Place,
		Country:        toText(journey.Country),
		Region:         toText(journey.Region),
		DateStart:      toDate(journey.DateStart),
		DateEnd:        toDate(journey.DateEnd),
		StGeomfromwkb:  gpsRouteBytes,
		AuthoredFields: journey.AuthoredFields,
	})
}

// Memento operations

func (r *pgRepository) GetMemento(ctx context.Context, id uuid.UUID) (*domain.Memento, error) {
	row, err := r.q.GetMemento(ctx, id)
	if err != nil {
		return nil, err
	}
	return toDomainMemento(row.ID, row.JourneyID, row.Kind, row.Seq, row.OccurredAt, row.OccurredTz, row.GeomWkb, row.Title, row.Place, row.Vendor, row.Essay, row.PriceAmount, row.PriceCurrency, row.KindData, row.SourceRef, row.AuthoredFields, row.OrphanedAt, row.CreatedAt, row.UpdatedAt)
}

func (r *pgRepository) ListMementosByJourney(ctx context.Context, journeyID uuid.UUID) ([]*domain.Memento, error) {
	rows, err := r.q.ListMementosByJourney(ctx, journeyID)
	if err != nil {
		return nil, err
	}
	var res []*domain.Memento
	for _, row := range rows {
		m, err := toDomainMemento(row.ID, row.JourneyID, row.Kind, row.Seq, row.OccurredAt, row.OccurredTz, row.GeomWkb, row.Title, row.Place, row.Vendor, row.Essay, row.PriceAmount, row.PriceCurrency, row.KindData, row.SourceRef, row.AuthoredFields, row.OrphanedAt, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		res = append(res, m)
	}
	return res, nil
}

func (r *pgRepository) UpsertMemento(ctx context.Context, memento *domain.Memento) error {
	var geomBytes []byte
	var err error
	if memento.Geom != nil {
		geomBytes, err = wkb.Marshal(memento.Geom)
		if err != nil {
			return err
		}
	}

	return r.q.UpsertMemento(ctx, db.UpsertMementoParams{
		ID:             memento.ID,
		JourneyID:      memento.JourneyID,
		Kind:           memento.Kind,
		Seq:            int32(memento.Seq),
		OccurredAt:     toTimestamptz(memento.OccurredAt),
		OccurredTz:     memento.OccurredTZ,
		StGeomfromwkb:  geomBytes,
		Title:          memento.Title,
		Place:          memento.Place,
		Vendor:         toText(memento.Vendor),
		Essay:          toText(memento.Essay),
		PriceAmount:    toInt8(memento.PriceAmount),
		PriceCurrency:  toText(memento.PriceCurrency),
		KindData:       memento.KindData,
		SourceRef:      toText(memento.SourceRef),
		AuthoredFields: memento.AuthoredFields,
		OrphanedAt:     toTimestamptzPtr(memento.OrphanedAt),
	})
}

// Photo operations

func (r *pgRepository) GetPhoto(ctx context.Context, id uuid.UUID) (*domain.MementoPhoto, error) {
	row, err := r.q.GetPhoto(ctx, id)
	if err != nil {
		return nil, err
	}
	return &domain.MementoPhoto{
		ID:          row.ID,
		MementoID:   row.MementoID,
		ObjectKey:   row.ObjectKey,
		ContentHash: row.ContentHash,
		Caption:     fromText(row.Caption),
		Seq:         int(row.Seq),
		TakenAt:     fromTimestamptzPtr(row.TakenAt),
		SourceRef:   fromText(row.SourceRef),
		CreatedAt:   fromTimestamptz(row.CreatedAt),
	}, nil
}

func (r *pgRepository) ListPhotosByMemento(ctx context.Context, mementoID uuid.UUID) ([]*domain.MementoPhoto, error) {
	rows, err := r.q.ListPhotosByMemento(ctx, mementoID)
	if err != nil {
		return nil, err
	}
	var res []*domain.MementoPhoto
	for _, row := range rows {
		res = append(res, &domain.MementoPhoto{
			ID:          row.ID,
			MementoID:   row.MementoID,
			ObjectKey:   row.ObjectKey,
			ContentHash: row.ContentHash,
			Caption:     fromText(row.Caption),
			Seq:         int(row.Seq),
			TakenAt:     fromTimestamptzPtr(row.TakenAt),
			SourceRef:   fromText(row.SourceRef),
			CreatedAt:   fromTimestamptz(row.CreatedAt),
		})
	}
	return res, nil
}

func (r *pgRepository) UpsertPhoto(ctx context.Context, photo *domain.MementoPhoto) error {
	return r.q.UpsertPhoto(ctx, db.UpsertPhotoParams{
		ID:          photo.ID,
		MementoID:   photo.MementoID,
		ObjectKey:   photo.ObjectKey,
		ContentHash: photo.ContentHash,
		Caption:     toText(photo.Caption),
		Seq:         int32(photo.Seq),
		TakenAt:     toTimestamptzPtr(photo.TakenAt),
		SourceRef:   toText(photo.SourceRef),
	})
}

// Translation operations

func (r *pgRepository) ListTranslations(ctx context.Context, ownerType string, ownerID uuid.UUID) ([]*domain.Translation, error) {
	rows, err := r.q.ListTranslations(ctx, db.ListTranslationsParams{
		OwnerType: ownerType,
		OwnerID:   ownerID,
	})
	if err != nil {
		return nil, err
	}
	var res []*domain.Translation
	for _, row := range rows {
		res = append(res, &domain.Translation{
			ID:         row.ID,
			OwnerType:  row.OwnerType,
			OwnerID:    row.OwnerID,
			Lang:       row.Lang,
			Field:      row.Field,
			Value:      row.Value,
			Provenance: row.Provenance,
			UpdatedAt:  fromTimestamptz(row.UpdatedAt),
		})
	}
	return res, nil
}

func (r *pgRepository) UpsertTranslation(ctx context.Context, translation *domain.Translation) error {
	return r.q.UpsertTranslation(ctx, db.UpsertTranslationParams{
		ID:         translation.ID,
		OwnerType:  translation.OwnerType,
		OwnerID:    translation.OwnerID,
		Lang:       translation.Lang,
		Field:      translation.Field,
		Value:      translation.Value,
		Provenance: translation.Provenance,
	})
}

// Helper converters

func toDomainJourney(
	id uuid.UUID, journalID uuid.UUID, slug string, sourceRef pgtype.Text,
	title string, place string, country pgtype.Text, region pgtype.Text,
	dateStart pgtype.Date, dateEnd pgtype.Date, gpsRouteWkb interface{},
	authoredFields []string, createdAt pgtype.Timestamptz, updatedAt pgtype.Timestamptz,
) (*domain.Journey, error) {
	var gpsRoute orb.MultiLineString
	if gpsRouteWkb != nil {
		if bytes, ok := gpsRouteWkb.([]byte); ok && len(bytes) > 0 {
			geom, err := wkb.Unmarshal(bytes)
			if err != nil {
				return nil, err
			}
			if mls, ok := geom.(orb.MultiLineString); ok {
				gpsRoute = mls
			}
		}
	}

	return &domain.Journey{
		ID:             id,
		JournalID:      journalID,
		Slug:           slug,
		SourceRef:      fromText(sourceRef),
		Title:          title,
		Place:          place,
		Country:        fromText(country),
		Region:         fromText(region),
		DateStart:      fromDate(dateStart),
		DateEnd:        fromDate(dateEnd),
		GPSRoute:       gpsRoute,
		AuthoredFields: authoredFields,
		CreatedAt:      fromTimestamptz(createdAt),
		UpdatedAt:      fromTimestamptz(updatedAt),
	}, nil
}

func toDomainMemento(
	id uuid.UUID, journeyID uuid.UUID, kind string, seq int32,
	occurredAt pgtype.Timestamptz, occurredTz string, geomWkb interface{},
	title string, place string, vendor pgtype.Text, essay pgtype.Text,
	priceAmount pgtype.Int8, priceCurrency pgtype.Text, kindData []byte,
	sourceRef pgtype.Text, authoredFields []string, orphanedAt pgtype.Timestamptz,
	createdAt pgtype.Timestamptz, updatedAt pgtype.Timestamptz,
) (*domain.Memento, error) {
	var geom orb.Geometry
	if geomWkb != nil {
		if bytes, ok := geomWkb.([]byte); ok && len(bytes) > 0 {
			g, err := wkb.Unmarshal(bytes)
			if err != nil {
				return nil, err
			}
			geom = g
		}
	}

	return &domain.Memento{
		ID:             id,
		JourneyID:      journeyID,
		Kind:           kind,
		Seq:            int(seq),
		OccurredAt:     fromTimestamptz(occurredAt),
		OccurredTZ:     occurredTz,
		Geom:           geom,
		Title:          title,
		Place:          place,
		Vendor:         fromText(vendor),
		Essay:          fromText(essay),
		PriceAmount:    fromInt8(priceAmount),
		PriceCurrency:  fromText(priceCurrency),
		KindData:       kindData,
		SourceRef:      fromText(sourceRef),
		AuthoredFields: authoredFields,
		OrphanedAt:     fromTimestamptzPtr(orphanedAt),
		CreatedAt:      fromTimestamptz(createdAt),
		UpdatedAt:      fromTimestamptz(updatedAt),
	}, nil
}

func toText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func fromText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func toInt8(i *int64) pgtype.Int8 {
	if i == nil {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: *i, Valid: true}
}

func fromInt8(i pgtype.Int8) *int64 {
	if !i.Valid {
		return nil
	}
	return &i.Int64
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func toTimestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil || t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func fromTimestamptz(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func fromTimestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func toDate(t time.Time) pgtype.Date {
	if t.IsZero() {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: t, Valid: true}
}

func fromDate(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}
