package environments

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// testEntity is a SQLite-compatible version of Entity for testing - the
// production Entity declares `default:uuid_generate_v4()`, a postgres-only
// function SQLite's CREATE TABLE syntax cannot express (see
// tenants/processor_test.go, services/processor_test.go for the same
// pattern).
type testEntity struct {
	Id        uuid.UUID       `gorm:"type:text;primaryKey"`
	Name      string          `gorm:"not null;uniqueIndex"`
	Baseline  string          `gorm:"not null"`
	Namespace string          `gorm:"not null"`
	Tenant    string          `gorm:"not null;default:''"`
	Overrides json.RawMessage `gorm:"type:text;not null"`
	Phase     string          `gorm:"not null"`
}

func (testEntity) TableName() string { return "environments" }

// testDatabase returns a SQLite in-memory database migrated for both the
// environments table and the outbox, with EnvEnvironmentStatusTopic set so
// enqueueEnvironmentStatus is not a no-op - matching the real deployment
// where the topic env var is always set.
func testDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	if err := db.AutoMigrate(&testEntity{}); err != nil {
		t.Fatalf("failed to migrate environments: %v", err)
	}
	if err := outbox.Migration(db); err != nil {
		t.Fatalf("failed to migrate outbox: %v", err)
	}

	t.Setenv(EnvEnvironmentStatusTopic, "EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS")

	return db
}

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

// envContext gives the processor a context carrying an environment id, the
// same way a real request/consumer would - even though ProcessorImpl only
// threads it through to db.WithContext today, every real caller runs under
// one.
func envContext(t *testing.T, name string) context.Context {
	t.Helper()
	return env.WithContext(context.Background(), env.Id(name))
}

// outboxRow is the subset of outbox.Entity the tests assert on, named to
// match the fixture in the task-19 brief (rows[0].Key / rows[0].Payload).
type outboxRow struct {
	Key     []byte
	Payload []byte
}

func readOutbox(t *testing.T, db *gorm.DB) []outboxRow {
	t.Helper()
	var rows []outbox.Entity
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	out := make([]outboxRow, len(rows))
	for i, r := range rows {
		out[i] = outboxRow{Key: r.MessageKey, Payload: r.MessageValue}
	}
	return out
}

func TestCreatingAnEnvironmentEnqueuesAnOutboxEnvelope(t *testing.T) {
	db := testDatabase(t)
	tenantId := uuid.New()
	p := NewProcessor(testLogger(t), envContext(t, "main"), db)

	_, err := p.Create(NewBuilder().
		SetName("pr-123").
		SetBaseline("main").
		SetNamespace("atlas-pr-123").
		SetTenant(tenantId.String()).
		SetOverride("atlas-login", "atlas-pr-123").
		SetOverride("atlas-channel", "atlas-pr-123").
		SetPhase(env.PhaseProvisioning).
		Build())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	rows := readOutbox(t, db)
	if len(rows) != 1 {
		t.Fatalf("outbox has %d rows, want 1", len(rows))
	}
	if got := string(rows[0].Key); got != "environment:pr-123" {
		t.Fatalf("key = %q, want \"environment:pr-123\"", got)
	}

	var envelope struct {
		Config env.Record `json:"config"`
	}
	if err := json.Unmarshal(rows[0].Payload, &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Config.Namespace != "atlas-pr-123" ||
		envelope.Config.Overrides["atlas-login"] != "atlas-pr-123" ||
		envelope.Config.Phase != env.PhaseProvisioning {
		t.Fatalf("envelope config = %+v", envelope.Config)
	}
	// Full round-trip: every RestModel field the wire shape carries must
	// survive unchanged, not just the three spot-checked above. A producer
	// that dropped Baseline or Tenant into the envelope would pass the
	// spot-check but fail here.
	if envelope.Config.Name != "pr-123" || envelope.Config.Baseline != "main" ||
		envelope.Config.Tenant != tenantId.String() ||
		envelope.Config.Overrides["atlas-channel"] != "atlas-pr-123" {
		t.Fatalf("envelope config incomplete = %+v", envelope.Config)
	}

	// Read back the persisted row directly, rather than trusting the
	// caller's validated input: a create() that silently persisted an
	// empty Name would still pass every assertion above.
	persisted, err := p.GetByName("pr-123")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if persisted.Name != "pr-123" {
		t.Fatalf("persisted name = %q, want \"pr-123\"", persisted.Name)
	}
}

func TestCreatingAnEnvironmentRejectsAMalformedName(t *testing.T) {
	db := testDatabase(t)
	p := NewProcessor(testLogger(t), envContext(t, "main"), db)

	if _, err := p.Create(NewBuilder().SetName("PR_123").
		SetBaseline("main").SetNamespace("x").Build()); err == nil {
		t.Fatal("malformed environment name accepted; ingest must validate (P2)")
	}

	rows := readOutbox(t, db)
	if len(rows) != 0 {
		t.Fatalf("rejected create still enqueued %d outbox row(s)", len(rows))
	}
}

func TestCreatingAnEnvironmentRejectsAnEmptyName(t *testing.T) {
	db := testDatabase(t)
	p := NewProcessor(testLogger(t), envContext(t, "main"), db)

	if _, err := p.Create(NewBuilder().SetBaseline("main").SetNamespace("x").Build()); err == nil {
		t.Fatal("empty environment name accepted; the empty id is env's legacy value and must never be persisted")
	}
}

// TestRepublishReemitsTheUnchangedRecord pins heartbeat.go's contract: the
// republished envelope's config must be byte-for-byte the same record that
// was created, field for field. A heartbeat that rebuilt the record and
// dropped a field (e.g. forgot Overrides, the exact failure mode
// controller note 6 warns about) would enqueue a second row but with an
// incomplete config - this test discriminates that from a correct republish.
func TestRepublishReemitsTheUnchangedRecord(t *testing.T) {
	db := testDatabase(t)
	p := NewProcessor(testLogger(t), envContext(t, "main"), db)

	created, err := p.Create(NewBuilder().
		SetName("main").
		SetBaseline("main").
		SetNamespace("atlas").
		SetTenant("tenant-1").
		SetOverride("atlas-login", "atlas").
		SetPhase(env.PhaseActive).
		Build())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := p.Republish(env.Id(created.Name)); err != nil {
		t.Fatalf("Republish: %v", err)
	}

	rows := readOutbox(t, db)
	if len(rows) != 2 {
		t.Fatalf("outbox has %d rows, want 2 (create + republish)", len(rows))
	}

	var first, second struct {
		Config env.Record `json:"config"`
	}
	if err := json.Unmarshal(rows[0].Payload, &first); err != nil {
		t.Fatalf("decode first envelope: %v", err)
	}
	if err := json.Unmarshal(rows[1].Payload, &second); err != nil {
		t.Fatalf("decode second envelope: %v", err)
	}
	if !reflect.DeepEqual(first.Config, second.Config) {
		t.Fatalf("republished record diverged: create=%+v republish=%+v", first.Config, second.Config)
	}
	if second.Config.Namespace != "atlas" || second.Config.Overrides["atlas-login"] != "atlas" {
		t.Fatalf("republished record incomplete = %+v", second.Config)
	}
}

// TestCreatingAnEnvironmentRejectsAnInvalidPhase pins the outbox side of
// ErrInvalidPhase: a junk phase must not merely fail Create, it must never
// reach the outbox. The phase is published on a compacted topic and decoded
// by every Task 20 subscriber, so an enqueued junk value poisons every pod's
// registry projection until the key is overwritten.
func TestCreatingAnEnvironmentRejectsAnInvalidPhase(t *testing.T) {
	db := testDatabase(t)
	p := NewProcessor(testLogger(t), envContext(t, "main"), db)

	_, err := p.Create(NewBuilder().
		SetName("pr-9").
		SetBaseline("main").
		SetNamespace("x").
		SetPhase("banana").
		Build())
	if !errors.Is(err, ErrInvalidPhase) {
		t.Fatalf("err = %v, want ErrInvalidPhase", err)
	}

	rows := readOutbox(t, db)
	if len(rows) != 0 {
		t.Fatalf("rejected create still enqueued %d outbox row(s)", len(rows))
	}
}

// TestUpdatingAnEnvironmentRejectsAnInvalidPhase mirrors the create-path
// test for UpdateByName: an update to a junk phase must fail with
// ErrInvalidPhase and must not enqueue an additional outbox row beyond the
// one from the initial valid create.
func TestUpdatingAnEnvironmentRejectsAnInvalidPhase(t *testing.T) {
	db := testDatabase(t)
	p := NewProcessor(testLogger(t), envContext(t, "main"), db)

	created, err := p.Create(NewBuilder().
		SetName("pr-10").
		SetBaseline("main").
		SetNamespace("x").
		SetPhase(env.PhaseProvisioning).
		Build())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = p.UpdateByName("pr-10", NewBuilder().
		SetName(created.Name).
		SetBaseline(created.Baseline).
		SetNamespace(created.Namespace).
		SetPhase("banana").
		Build())
	if !errors.Is(err, ErrInvalidPhase) {
		t.Fatalf("err = %v, want ErrInvalidPhase", err)
	}

	rows := readOutbox(t, db)
	if len(rows) != 1 {
		t.Fatalf("outbox has %d rows, want 1 (create only)", len(rows))
	}
}

// TestUpdatingAnEnvironmentRejectsAnIllegalPhaseTransition pins PRD FR-5.1's
// literal chain (PROVISIONING -> ACTIVE -> DEACTIVATING -> DELETED): no
// skip-ahead, no reverting. It also asserts the no-op transition (X -> X)
// is still ACCEPTED, since the heartbeat path re-PATCHes the same phase - a
// validator that rejected every transition, including the no-op, would pass
// the skip/revert cases here but silently break the heartbeat.
func TestUpdatingAnEnvironmentRejectsAnIllegalPhaseTransition(t *testing.T) {
	create := func(t *testing.T, db *gorm.DB, p Processor, name, phase string) RestModel {
		t.Helper()
		created, err := p.Create(NewBuilder().
			SetName(name).
			SetBaseline("main").
			SetNamespace("x").
			SetPhase(phase).
			Build())
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		return created
	}

	t.Run("skips two steps", func(t *testing.T) {
		db := testDatabase(t)
		p := NewProcessor(testLogger(t), envContext(t, "main"), db)
		created := create(t, db, p, "pr-11", env.PhaseProvisioning)

		_, err := p.UpdateByName("pr-11", NewBuilder().
			SetName(created.Name).
			SetBaseline(created.Baseline).
			SetNamespace(created.Namespace).
			SetPhase(env.PhaseDeleted).
			Build())
		if !errors.Is(err, ErrIllegalPhaseTransition) {
			t.Fatalf("err = %v, want ErrIllegalPhaseTransition", err)
		}
	})

	t.Run("reverts", func(t *testing.T) {
		db := testDatabase(t)
		p := NewProcessor(testLogger(t), envContext(t, "main"), db)
		created := create(t, db, p, "pr-12", env.PhaseActive)

		_, err := p.UpdateByName("pr-12", NewBuilder().
			SetName(created.Name).
			SetBaseline(created.Baseline).
			SetNamespace(created.Namespace).
			SetPhase(env.PhaseProvisioning).
			Build())
		if !errors.Is(err, ErrIllegalPhaseTransition) {
			t.Fatalf("err = %v, want ErrIllegalPhaseTransition", err)
		}
	})

	t.Run("noop is accepted", func(t *testing.T) {
		db := testDatabase(t)
		p := NewProcessor(testLogger(t), envContext(t, "main"), db)
		created := create(t, db, p, "pr-13", env.PhaseActive)

		updated, err := p.UpdateByName("pr-13", NewBuilder().
			SetName(created.Name).
			SetBaseline(created.Baseline).
			SetNamespace(created.Namespace).
			SetPhase(env.PhaseActive).
			Build())
		if err != nil {
			t.Fatalf("noop transition (ACTIVE -> ACTIVE) rejected: %v", err)
		}
		if updated.Phase != env.PhaseActive {
			t.Fatalf("phase = %q, want %q", updated.Phase, env.PhaseActive)
		}
	})

	t.Run("advances one step", func(t *testing.T) {
		db := testDatabase(t)
		p := NewProcessor(testLogger(t), envContext(t, "main"), db)
		created := create(t, db, p, "pr-14", env.PhaseProvisioning)

		updated, err := p.UpdateByName("pr-14", NewBuilder().
			SetName(created.Name).
			SetBaseline(created.Baseline).
			SetNamespace(created.Namespace).
			SetPhase(env.PhaseActive).
			Build())
		if err != nil {
			t.Fatalf("legal one-step transition (PROVISIONING -> ACTIVE) rejected: %v", err)
		}
		if updated.Phase != env.PhaseActive {
			t.Fatalf("phase = %q, want %q", updated.Phase, env.PhaseActive)
		}
	})
}
