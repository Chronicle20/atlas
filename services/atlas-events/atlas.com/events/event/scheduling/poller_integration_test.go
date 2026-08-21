//go:build integration

// poller_integration_test.go proves the ONE thing poller_test.go's SQLite
// harness cannot: that SKIP LOCKED actually isolates concurrent claimers on a
// real Postgres, so two replicas polling the same table never execute the
// same row (FR-S6, FR-N6). Modelled on
// libs/atlas-outbox/lock_test.go's testcontainers setup. Excluded from the
// default `go test ./...` gate by the `integration` build tag, same as every
// other testcontainers test in this repo (controller ruling, task-18 brief).
package scheduling

import (
	"atlas-events/event/definition"
	"atlas-events/event/occurrence"
	"atlas-events/event/transition"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newIntegrationTestLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

// TestTwoInstancesNeverExecuteTheSameRow is PRD §20.3's explicit demand for a
// CONCURRENT test rather than inspection: two processor instances, sharing
// one real Postgres database, claiming and executing 50 due rows. SKIP
// LOCKED must mean each row is claimed — and therefore executed — by exactly
// one instance.
func TestTwoInstancesNeverExecuteTheSameRow(t *testing.T) {
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db1, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	db2, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, MigrateTable(db1))
	require.NoError(t, definition.MigrateTable(db1))
	require.NoError(t, occurrence.MigrateTable(db1))
	require.NoError(t, transition.MigrateTable(db1))

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	seedCtx := tenant.WithContext(context.Background(), tm)

	const rows = 50
	defId := uuid.New()
	require.NoError(t, db1.WithContext(seedCtx).Create(&definition.Entity{
		ID: defId, TenantID: tm.Id(), Type: "SHARED_TYPE", Name: "SHARED_TYPE",
		Enabled: true, Configuration: "{}",
	}).Error)
	for i := 0; i < rows; i++ {
		m, err := NewBuilder(defId, WorkTypeTriggerEvaluation).
			SetExecuteAt(time.Now().Add(-time.Minute)).
			Build()
		require.NoError(t, err)
		entity, err := ToEntity(m, tm)
		require.NoError(t, err)
		require.NoError(t, db1.WithContext(seedCtx).Create(&entity).Error)
	}

	var mu sync.Mutex
	executed := map[uuid.UUID]int{}
	record := func(m Model) error {
		mu.Lock()
		defer mu.Unlock()
		executed[m.Id()]++
		return nil
	}

	var wg sync.WaitGroup
	for _, instance := range []string{"a", "b"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			db := db1
			if id == "b" {
				db = db2
			}
			p := NewProcessorWithExecutor(newIntegrationTestLogger(t), context.Background(), db, record)
			for i := 0; i < 20; i++ {
				claimed, err := p.ClaimBatch(id, 10)
				if err != nil {
					t.Errorf("claim: %v", err)
					return
				}
				for _, m := range claimed {
					if err := p.ExecuteOne(m); err != nil {
						t.Errorf("execute: %v", err)
					}
				}
			}
		}(instance)
	}
	wg.Wait()

	if len(executed) != rows {
		t.Fatalf("executed %d distinct rows, want %d", len(executed), rows)
	}
	for id, n := range executed {
		if n != 1 {
			t.Fatalf("row %s executed %d times", id, n)
		}
	}
}
