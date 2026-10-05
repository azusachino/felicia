// Package photos owns private original-image validation and photo creation.
// Upload creates a new row on each successful call; callers must not retry an
// unknown HTTP outcome automatically. Publication still sanitizes originals.
package photos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // Register JPEG decoding for image.DecodeConfig.
	_ "image/png"  // Register PNG decoding for image.DecodeConfig.
	"path"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "golang.org/x/image/webp" // Register WebP decoding for image.DecodeConfig.

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-core/ports"
	"github.com/azusachino/felicia/apps/felicia-runtime/importer"
)

// MaxUploadBytes bounds one private original image.
const MaxUploadBytes = 20 << 20

// ErrTooLarge identifies an image exceeding the shared upload limit.
var ErrTooLarge = errors.New("photo upload is too large (maximum 20 MiB)")

// InvalidImageError describes an unsupported or unsafe image payload.
type InvalidImageError struct{ reason string }

func (err *InvalidImageError) Error() string { return err.reason }

// ImageFormat validates supported formats and dimensions without decoding pixels.
func ImageFormat(data []byte) (string, error) {
	if len(data) > MaxUploadBytes {
		return "", ErrTooLarge
	}
	if len(data) >= 12 && string(data[4:8]) == "ftyp" {
		brand := string(data[8:12])
		if brand == "heic" || brand == "heix" || brand == "hevc" || brand == "hevx" || brand == "mif1" || brand == "msf1" {
			return "", &InvalidImageError{"HEIC/HEIF photos are not supported yet; convert to JPEG before uploading"}
		}
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", &InvalidImageError{"uploaded file is not a supported JPEG, PNG, or WebP image"}
	}
	if format != "jpeg" && format != "png" && format != "webp" {
		return "", &InvalidImageError{fmt.Sprintf("unsupported image format %q; use JPEG, PNG, or WebP", format)}
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > 50_000 || config.Height > 50_000 || int64(config.Width)*int64(config.Height) > 50_000_000 {
		return "", &InvalidImageError{"image dimensions are too large"}
	}
	return format, nil
}

type photoStore interface {
	GetMemento(context.Context, uuid.UUID) (*domain.Memento, error)
	GetPhoto(context.Context, uuid.UUID) (*domain.MementoPhoto, error)
	ListPhotosByMemento(context.Context, uuid.UUID) ([]*domain.MementoPhoto, error)
	UpsertPhoto(context.Context, *domain.MementoPhoto) error
}

// Service creates photo identities and stores their private original bytes.
type Service struct {
	repo photoStore
	blob ports.BlobStore
	mu   sync.Mutex
}

// New binds photo creation to the workspace's metadata and private blob stores.
func New(repo photoStore, blob ports.BlobStore) *Service { return &Service{repo: repo, blob: blob} }

// Upload stores bytes under a digest-derived key. It accepts no filenames or
// source paths. Sequence assignment is serialized within this service instance.
func (s *Service) Upload(ctx context.Context, mementoID uuid.UUID, data []byte) (*domain.MementoPhoto, error) {
	format, err := ImageFormat(data)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.repo.GetMemento(ctx, mementoID); err != nil {
		return nil, fmt.Errorf("get memento: %w", err)
	}
	existing, err := s.repo.ListPhotosByMemento(ctx, mementoID)
	if err != nil {
		return nil, fmt.Errorf("list photos: %w", err)
	}
	seq := 0
	for _, photo := range existing {
		if photo.Seq >= seq {
			seq = photo.Seq + 1
		}
	}
	digest := importer.MediaDigest(data)
	extension := map[string]string{"jpeg": ".jpg", "png": ".png", "webp": ".webp"}[format]
	photo := &domain.MementoPhoto{ID: uuid.Must(uuid.NewV7()), MementoID: mementoID,
		ContentHash: "sha256:" + digest, Seq: seq, CreatedAt: time.Now().UTC()}
	// A per-photo key makes compensation safe: never delete an original shared
	// with an existing photo or another process uploading identical bytes.
	photo.ObjectKey = path.Join("media", digest, photo.ID.String(), "original"+extension)
	if err := s.blob.Put(ctx, photo.ObjectKey, data); err != nil {
		return nil, fmt.Errorf("store original: %w", err)
	}
	if err := s.repo.UpsertPhoto(ctx, photo); err != nil {
		// Cancellation can race a committed write. Only compensate when the
		// repository positively confirms that this new identity is absent.
		cleanupCtx := context.WithoutCancel(ctx)
		if _, lookupErr := s.repo.GetPhoto(cleanupCtx, photo.ID); errors.Is(lookupErr, domain.ErrNotFound) {
			if cleanupErr := s.blob.Delete(cleanupCtx, photo.ObjectKey); cleanupErr != nil {
				return nil, fmt.Errorf("save photo and remove original: %w", errors.Join(err, cleanupErr))
			}
		}
		return nil, fmt.Errorf("save photo: %w", err)
	}
	return photo, nil
}
