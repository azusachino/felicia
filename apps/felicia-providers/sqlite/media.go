package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// GetPhoto retrieves a memento photo by ID.
func (r *Repository) GetPhoto(ctx context.Context, id uuid.UUID) (*domain.MementoPhoto, error) {
	row := r.db.QueryRowContext(ctx, "SELECT memento_id, object_key, content_hash, caption, seq, taken_at, source_ref, created_at FROM tb_memento_photos WHERE id = ?", idString(id))
	photo, err := scanPhoto(row, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return photo, err
}

// ListPhotosByMemento retrieves a memento's photos in sequence order.
func (r *Repository) ListPhotosByMemento(ctx context.Context, mementoID uuid.UUID) ([]*domain.MementoPhoto, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, object_key, content_hash, caption, seq, taken_at, source_ref, created_at FROM tb_memento_photos WHERE memento_id = ? ORDER BY seq", idString(mementoID))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var result []*domain.MementoPhoto
	for rows.Next() {
		var rawID string
		var objectKey, hash string
		var caption, takenAt, sourceRef, created sql.NullString
		var seq int
		if err := rows.Scan(&rawID, &objectKey, &hash, &caption, &seq, &takenAt, &sourceRef, &created); err != nil {
			return nil, err
		}
		id, err := parseID(rawID)
		if err != nil {
			return nil, err
		}
		result = append(result, photoFromValues(id, mementoID, objectKey, hash, caption, seq, takenAt, sourceRef, created))
	}
	return result, rows.Err()
}

// CreatePhotoWithNextSequence inserts a new photo row and assigns the next
// sequence position for its memento in one INSERT statement. The correlated
// MAX(seq) subquery runs inside the insert's own write transaction, and
// SQLite serializes writers, so concurrent creations each observe the prior
// row instead of a shared stale maximum. This is a plain INSERT with no
// conflict clause: duplicate identities and duplicate (memento, content
// hash) pairs surface as constraint errors, never as silent skips, and
// explicit authored ordering via UpsertPhoto is untouched.
func (r *Repository) CreatePhotoWithNextSequence(ctx context.Context, photo *domain.MementoPhoto) (int, error) {
	var assigned int
	err := r.db.QueryRowContext(ctx, `INSERT INTO tb_memento_photos(id, memento_id, object_key, content_hash, caption, seq, taken_at, source_ref, created_at)
SELECT ?, ?, ?, ?, ?, COALESCE((SELECT MAX(seq) FROM tb_memento_photos WHERE memento_id = ?), -1) + 1, ?, ?, ?
RETURNING seq`,
		idString(photo.ID), idString(photo.MementoID), photo.ObjectKey, photo.ContentHash, nullableString(photo.Caption), idString(photo.MementoID), nullableTimePtr(photo.TakenAt), nullableString(photo.SourceRef), timeOrNow(photo.CreatedAt)).Scan(&assigned)
	if err != nil {
		return 0, fmt.Errorf("create photo with next sequence: %w", err)
	}
	return assigned, nil
}

// UpsertPhoto inserts or updates a memento photo.
func (r *Repository) UpsertPhoto(ctx context.Context, photo *domain.MementoPhoto) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO tb_memento_photos(id, memento_id, object_key, content_hash, caption, seq, taken_at, source_ref, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET object_key=excluded.object_key, content_hash=excluded.content_hash, caption=excluded.caption, seq=excluded.seq, taken_at=excluded.taken_at, source_ref=excluded.source_ref`, idString(photo.ID), idString(photo.MementoID), photo.ObjectKey, photo.ContentHash, nullableString(photo.Caption), photo.Seq, nullableTimePtr(photo.TakenAt), nullableString(photo.SourceRef), timeOrNow(photo.CreatedAt))
	return err
}

func scanPhoto(row scanner, id uuid.UUID) (*domain.MementoPhoto, error) {
	var rawMemento, objectKey, hash string
	var caption, takenAt, sourceRef, created sql.NullString
	var seq int
	if err := row.Scan(&rawMemento, &objectKey, &hash, &caption, &seq, &takenAt, &sourceRef, &created); err != nil {
		return nil, err
	}
	mementoID, err := parseID(rawMemento)
	if err != nil {
		return nil, err
	}
	return photoFromValues(id, mementoID, objectKey, hash, caption, seq, takenAt, sourceRef, created), nil
}

func photoFromValues(id, mementoID uuid.UUID, objectKey, hash string, caption sql.NullString, seq int, takenAt, sourceRef, created sql.NullString) *domain.MementoPhoto {
	return &domain.MementoPhoto{ID: id, MementoID: mementoID, ObjectKey: objectKey, ContentHash: hash, Caption: readString(caption), Seq: seq, TakenAt: timePtr(takenAt), SourceRef: readString(sourceRef), CreatedAt: readTime(created)}
}
