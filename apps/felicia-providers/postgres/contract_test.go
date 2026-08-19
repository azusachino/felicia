package postgres_test

import (
	"testing"

	"github.com/azusachino/felicia/providers/contract"
	"github.com/azusachino/felicia/providers/postgres"
)

func TestRepositoryContract(t *testing.T) {
	contract.Run(t, postgres.NewRepository(testPool(t)))
}
