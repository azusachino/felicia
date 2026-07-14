package postgres_test

import (
	"testing"

	"github.com/azusachino/felicia/apps/providers/contract"
	"github.com/azusachino/felicia/apps/providers/postgres"
)

func TestRepositoryContract(t *testing.T) {
	contract.Run(t, postgres.NewRepository(testPool(t)))
}
