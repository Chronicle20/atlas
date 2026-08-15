package occurrence

import (
	"atlas-events/event/definition"
	"atlas-events/event/transition"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, MigrateTable, transition.MigrateTable)
}

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

// testTenantId is a fixed tenant shared by every occurrence package test that
// calls testCtx. Each test gets its own fresh in-memory DB (newTestDB), so a
// shared id across tests is safe; a fixed id (rather than a fresh uuid.New()
// per call) is what lets the REST harness's doGET issue an HTTP request
// against the same tenant a test wrote its fixtures under, without threading
// a tenant id through every helper call.
var testTenantId = uuid.New()

func testCtx(t *testing.T) context.Context {
	t.Helper()
	return databasetest.TenantContext(testTenantId)
}

// testDefinition builds a definition.Model with a deterministic id for
// theType, so two calls with the same type within a test represent the SAME
// definition — required for the concurrency-key uniqueness tests, which
// create occurrences from "separate" calls that must still collide.
func testDefinition(t *testing.T, theType string) definition.Model {
	t.Helper()
	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte(theType))
	m, err := definition.NewBuilder(theType, theType).SetId(id).SetEnabled(true).Build()
	if err != nil {
		t.Fatalf("testDefinition: %v", err)
	}
	return m
}
