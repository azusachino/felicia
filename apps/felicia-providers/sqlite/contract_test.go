package sqlite_test

import (
	"testing"

	"github.com/azusachino/felicia/providers/contract"
	"github.com/azusachino/felicia/providers/sqlite"
)

func TestRepositoryContract(t *testing.T) {
	repo, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer repo.Close()

	contract.Run(t, repo)
}
