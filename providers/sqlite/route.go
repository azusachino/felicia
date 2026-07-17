package sqlite

import (
	"context"
	"database/sql"
	"slices"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core/domain"
)

// CreateTransitLeg stores an authored route segment.
func (r *Repository) CreateTransitLeg(ctx context.Context, leg *domain.TransitLegInput) error {
	geom := orb.LineString{leg.Origin, leg.Dest}
	encoded, err := encodeGeometry(geom)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO tb_transit_legs(id, journey_id, seq, origin_label, dest_label, geom, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, idString(leg.ID), idString(leg.JourneyID), leg.Seq, nullableString(leg.OriginLabel), nullableString(leg.DestLabel), encoded, now())
	return err
}

// ListTransitLegsByJourney retrieves route segments in sequence order.
func (r *Repository) ListTransitLegsByJourney(ctx context.Context, journeyID uuid.UUID) ([]*domain.TransitLeg, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, seq, origin_label, dest_label, geom FROM tb_transit_legs WHERE journey_id = ? ORDER BY seq", idString(journeyID))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var result []*domain.TransitLeg
	for rows.Next() {
		var rawID string
		var seq int
		var origin, dest, geom sql.NullString
		if err := rows.Scan(&rawID, &seq, &origin, &dest, &geom); err != nil {
			return nil, err
		}
		id, err := parseID(rawID)
		if err != nil {
			return nil, err
		}
		decoded, err := decodeGeometry(geom)
		if err != nil {
			return nil, err
		}
		line, ok := decoded.(orb.LineString)
		if !ok || len(line) < 2 {
			continue
		}
		result = append(result, &domain.TransitLeg{ID: id, JourneyID: journeyID, Seq: seq, OriginLabel: readString(origin), DestLabel: readString(dest), Geom: line})
	}
	return result, rows.Err()
}

// DeleteTransitLeg removes an authored route segment.
func (r *Repository) DeleteTransitLeg(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM tb_transit_legs WHERE id = ?", idString(id))
	return err
}

// GetDisplayRoute composes the stored GPS route and authored transit legs.
func (r *Repository) GetDisplayRoute(ctx context.Context, journeyID uuid.UUID) (orb.MultiLineString, error) {
	journey, err := r.GetJourney(ctx, journeyID)
	if err != nil {
		return nil, err
	}
	route := slices.Clone(journey.GPSRoute)
	legs, err := r.ListTransitLegsByJourney(ctx, journeyID)
	if err != nil {
		return nil, err
	}
	for _, leg := range legs {
		route = append(route, leg.Geom)
	}
	return route, nil
}

// SnapToRoute returns the route snap for a journey.
func (r *Repository) SnapToRoute(ctx context.Context, journeyID uuid.UUID, point orb.Point) (*orb.Point, error) {
	route, err := r.GetDisplayRoute(ctx, journeyID)
	if err != nil {
		return nil, err
	}
	if len(route) == 0 {
		return nil, nil
	}
	return &point, nil
}
