package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

func TestMissingPhotoUsesDomainNotFound(t *testing.T) {
	repo, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repo.Close() }()
	if _, err := repo.GetPhoto(context.Background(), uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing photo error = %v, want domain.ErrNotFound", err)
	}
}
