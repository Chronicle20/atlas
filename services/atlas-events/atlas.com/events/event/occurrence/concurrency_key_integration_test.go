//go:build integration

// concurrency_key_integration_test.go proves the ONE thing the SQLite harness
// cannot distinguish (task-19 review F1): whether a COMPLETED occurrence's
// concurrency key still blocks a fresh occurrence from reusing it. The
// untargeted `clause.OnConflict{DoNothing: true}` insert suppresses on ANY
// unique-index violation, so a stale, state-blind index
// (ux_event_occurrence_concurrency_key) would silently swallow the re-create
// even though `ux_occ_concurrency` — scoped to state = 'ACTIVE' — should have
// let it through. SQLite's ON CONFLICT DO NOTHING handling differs from
// Postgres's, so only a real Postgres reproduces this. Modelled on
// event/scheduling/poller_integration_test.go's testcontainers setup.
// Excluded from the default `go test ./...` gate by the `integration` build
// tag, same as every other testcontainers test in this repo.
package occurrence

import (
	"atlas-events/event/definition"
	"atlas-events/event/registry"
	"atlas-events/event/transition"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newPGIntegrationTestLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

// TestConcurrencyKeyIsReusableAfterCompletionOnPostgres is the create ->
// complete -> re-create case the existing
// TestConcurrencyKeyRejectsASecondActiveOccurrence cannot exercise: both rows
// there are ACTIVE, so both the stale index and ux_occ_concurrency fire
// identically. Here the first occurrence is completed before the second
// insert, so only the stale, state-blind index would reject it.
func TestConcurrencyKeyIsReusableAfterCompletionOnPostgres(t *testing.T) {
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, definition.MigrateTable(db))
	require.NoError(t, MigrateTable(db))
	require.NoError(t, transition.MigrateTable(db))

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	pgCtx := tenant.WithContext(context.Background(), tm)

	p := NewProcessor(newPGIntegrationTestLogger(t), pgCtx, db)

	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte("CRIMSON_BALROG"))
	d, err := definition.NewBuilder("CRIMSON_BALROG", "CRIMSON_BALROG").SetId(id).SetEnabled(true).Build()
	require.NoError(t, err)
	require.NoError(t, db.WithContext(pgCtx).Create(&definition.Entity{
		ID: id, TenantID: tm.Id(), Type: "CRIMSON_BALROG", Name: "CRIMSON_BALROG",
		Enabled: true, Configuration: "{}",
	}).Error)

	seed := registry.Seed{Stage: "ATTACKING", ConcurrencyKey: "v1|1|4", WorldId: 1, ChannelId: 4}

	first, err := p.CreateFromSeed(d, seed, "work-1")
	require.NoError(t, err)

	won, err := p.Complete(first.Id(), "MONSTERS_ELIMINATED", transition.TriggerTypeMonsterKilled, "work-1")
	require.NoError(t, err)
	require.True(t, won)

	// The defect: with the stale ux_event_occurrence_concurrency_key index in
	// place, this second insert is silently suppressed by the untargeted
	// ON CONFLICT DO NOTHING even though the first occurrence is COMPLETED —
	// ux_occ_concurrency (state = 'ACTIVE') would happily allow it.
	second, err := p.CreateFromSeed(d, seed, "work-2")
	require.NoError(t, err, "a repeating event must be able to run again after completion")
	require.NotEqual(t, first.Id(), second.Id())
	require.Equal(t, StateActive, second.State())
}
