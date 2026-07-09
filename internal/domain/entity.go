package domain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
)

// ErrNotFound is returned when a requested entity does not exist.
var ErrNotFound = errors.New("entity not found")

// Journal is the root container of journeys.
type Journal struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

// Journey represents a travel trip.
type Journey struct {
	ID             uuid.UUID           `json:"id"`
	JournalID      uuid.UUID           `json:"journal_id"`
	Slug           string              `json:"slug"`
	SourceRef      *string             `json:"source_ref,omitempty"`
	Title          string              `json:"title"` // Canonical Japanese (ja)
	Place          string              `json:"place"`
	Country        *string             `json:"country,omitempty"`
	Region         *string             `json:"region,omitempty"`
	DateStart      time.Time           `json:"date_start"`
	DateEnd        time.Time           `json:"date_end"`
	GPSRoute       orb.MultiLineString `json:"gps_route,omitempty"`
	AuthoredFields []string            `json:"authored_fields"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// Memento represents an object that anchors a memory.
type Memento struct {
	ID             uuid.UUID       `json:"id"`
	JourneyID      uuid.UUID       `json:"journey_id"`
	Kind           string          `json:"kind"`
	Seq            int             `json:"seq"`
	OccurredAt     time.Time       `json:"occurred_at"`
	OccurredTZ     string          `json:"occurred_tz"`
	Geom           orb.Geometry    `json:"geom"`
	Title          string          `json:"title"` // Canonical Japanese (ja)
	Place          string          `json:"place"`
	Vendor         *string         `json:"vendor,omitempty"`
	Essay          *string         `json:"essay,omitempty"`
	PriceAmount    *int64          `json:"price_amount,omitempty"`
	PriceCurrency  *string         `json:"price_currency,omitempty"`
	KindData       json.RawMessage `json:"kind_data"` // JSONB payload
	SourceRef      *string         `json:"source_ref,omitempty"`
	AuthoredFields []string        `json:"authored_fields"`
	OrphanedAt     *time.Time      `json:"orphaned_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Translation represents a translated string in the translations sidecar.
type Translation struct {
	ID         uuid.UUID `json:"id"`
	OwnerType  string    `json:"owner_type"` // 'journey', 'memento', 'photo'
	OwnerID    uuid.UUID `json:"owner_id"`
	Lang       string    `json:"lang"` // 'en', 'zh'
	Field      string    `json:"field"`
	Value      string    `json:"value"`
	Provenance string    `json:"provenance"` // 'machine', 'authored'
	UpdatedAt  time.Time `json:"updated_at"`
}

// TransitLeg is an authored route segment (a flight, ferry, or GPS-gap fill).
// Its Geom is a great-circle arc generated server-side. Legs are kept separate
// from journeys.gps_route and composed into the display route at read time.
type TransitLeg struct {
	ID          uuid.UUID      `json:"id"`
	JourneyID   uuid.UUID      `json:"journey_id"`
	Seq         int            `json:"seq"`
	OriginLabel *string        `json:"origin_label,omitempty"`
	DestLabel   *string        `json:"dest_label,omitempty"`
	Geom        orb.LineString `json:"geom"`
	CreatedAt   time.Time      `json:"created_at"`
}

// TransitLegInput is the input to create a leg. The geodesic arc is generated in
// PostGIS from the endpoints (ST_Segmentize on geography), so the caller supplies
// endpoints and a max segment length in metres rather than a geometry.
type TransitLegInput struct {
	ID          uuid.UUID
	JourneyID   uuid.UUID
	Seq         int
	OriginLabel *string
	DestLabel   *string
	Origin      orb.Point // [lng, lat]
	Dest        orb.Point // [lng, lat]
	SegmentLenM float64
}

// MementoPhoto is a collectible photo associated with a memento.
type MementoPhoto struct {
	ID          uuid.UUID  `json:"id"`
	MementoID   uuid.UUID  `json:"memento_id"`
	ObjectKey   string     `json:"object_key"`
	ContentHash string     `json:"content_hash"`
	Caption     *string    `json:"caption,omitempty"` // Canonical Japanese (ja)
	Seq         int        `json:"seq"`
	TakenAt     *time.Time `json:"taken_at,omitempty"`
	SourceRef   *string    `json:"source_ref,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Repository defines the contract for persisting and retrieving felicia data.
type Repository interface {
	// Journal operations
	GetJournal(ctx context.Context, id uuid.UUID) (*Journal, error)
	CreateJournal(ctx context.Context, journal *Journal) error

	// Journey operations
	GetJourney(ctx context.Context, id uuid.UUID) (*Journey, error)
	GetJourneyBySlug(ctx context.Context, slug string) (*Journey, error)
	ListJourneys(ctx context.Context) ([]*Journey, error)
	UpsertJourney(ctx context.Context, journey *Journey) error

	// Memento operations
	GetMemento(ctx context.Context, id uuid.UUID) (*Memento, error)
	ListMementosByJourney(ctx context.Context, journeyID uuid.UUID) ([]*Memento, error)
	UpsertMemento(ctx context.Context, memento *Memento) error

	// TransitLeg operations (authored legs, union-at-read)
	CreateTransitLeg(ctx context.Context, leg *TransitLegInput) error
	ListTransitLegsByJourney(ctx context.Context, journeyID uuid.UUID) ([]*TransitLeg, error)
	DeleteTransitLeg(ctx context.Context, id uuid.UUID) error

	// Route composition: display route = gps_route ∪ transit legs, and proximity
	// snap of a point onto that composed route. Both return nil when empty.
	GetDisplayRoute(ctx context.Context, journeyID uuid.UUID) (orb.MultiLineString, error)
	SnapToRoute(ctx context.Context, journeyID uuid.UUID, pt orb.Point) (*orb.Point, error)

	// Photo operations
	GetPhoto(ctx context.Context, id uuid.UUID) (*MementoPhoto, error)
	ListPhotosByMemento(ctx context.Context, mementoID uuid.UUID) ([]*MementoPhoto, error)
	UpsertPhoto(ctx context.Context, photo *MementoPhoto) error

	// Translation operations
	ListTranslations(ctx context.Context, ownerType string, ownerID uuid.UUID) ([]*Translation, error)
	UpsertTranslation(ctx context.Context, translation *Translation) error
}
