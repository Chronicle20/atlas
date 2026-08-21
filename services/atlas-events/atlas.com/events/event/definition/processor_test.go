package definition

import (
	"atlas-events/event/registry"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, MigrateTable)
}

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	return databasetest.TenantContext(uuid.New())
}

func registryReset(t *testing.T) {
	t.Helper()
	registry.ResetForTest()
	t.Cleanup(func() { registry.ResetForTest() })
}

type rejectingHandler struct {
	t   string
	err error
}

func (h rejectingHandler) Type() string { return h.t }
func (h rejectingHandler) ValidateConfiguration(json.RawMessage) error {
	return h.err
}

func (h rejectingHandler) ConcurrencyKey(context.Context, json.RawMessage) (string, error) {
	return "", nil
}

func (h rejectingHandler) ConcurrencyKeyIsConstant() bool { return false }
func (h rejectingHandler) Evaluate(context.Context, registry.Definition, registry.Work) (*registry.Seed, error) {
	return nil, nil
}

func (h rejectingHandler) Start(context.Context, registry.Occurrence) (registry.Progress, error) {
	return registry.Progress{}, nil
}

func (h rejectingHandler) Advance(context.Context, registry.Occurrence, registry.Work) (registry.Progress, error) {
	return registry.Progress{}, nil
}

type acceptingHandler struct {
	t string
}

func (h acceptingHandler) Type() string                                { return h.t }
func (h acceptingHandler) ValidateConfiguration(json.RawMessage) error { return nil }
func (h acceptingHandler) ConcurrencyKey(context.Context, json.RawMessage) (string, error) {
	return "", nil
}

func (h acceptingHandler) ConcurrencyKeyIsConstant() bool { return false }
func (h acceptingHandler) Evaluate(context.Context, registry.Definition, registry.Work) (*registry.Seed, error) {
	return nil, nil
}

func (h acceptingHandler) Start(context.Context, registry.Occurrence) (registry.Progress, error) {
	return registry.Progress{}, nil
}

func (h acceptingHandler) Advance(context.Context, registry.Occurrence, registry.Work) (registry.Progress, error) {
	return registry.Progress{}, nil
}

// FR-D6: an invalid configuration is rejected on WRITE, by the handler that
// owns the type, rather than persisted and failing later at trigger time.
func TestCreateRejectsConfigurationTheHandlerRefuses(t *testing.T) {
	registryReset(t)
	registry.Register(rejectingHandler{t: "PICKY", err: errors.New("monsterCount must be > 0")})

	db := newTestDB(t)
	m, _ := NewBuilder("PICKY", "n").SetConfiguration(json.RawMessage(`{"monsterCount":0}`)).Build()

	if _, err := NewProcessor(testLogger(t), testCtx(t), db).Create(m); err == nil {
		t.Fatalf("expected the handler's validation error to be surfaced")
	}
	var count int64
	db.Model(&Entity{}).Count(&count)
	if count != 0 {
		t.Fatalf("invalid definition was persisted anyway")
	}
}

// A type with no registered handler is rejected at write time, not silently
// stored to fail at trigger time.
func TestCreateRejectsUnknownType(t *testing.T) {
	registryReset(t)
	db := newTestDB(t)
	m, _ := NewBuilder("NO_HANDLER", "n").SetConfiguration(json.RawMessage(`{}`)).Build()

	if _, err := NewProcessor(testLogger(t), testCtx(t), db).Create(m); err == nil {
		t.Fatalf("expected an error for a type with no handler")
	}
}

// FR-D5: disabling a definition must not touch occurrences. This processor has
// no path that writes an occurrence at all — the assertion is that SetEnabled
// returns the updated model and nothing else changes.
func TestSetEnabledTogglesOnly(t *testing.T) {
	registryReset(t)
	registry.Register(acceptingHandler{t: "OK"})
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)

	m, _ := NewBuilder("OK", "n").SetConfiguration(json.RawMessage(`{}`)).Build()
	created, err := p.Create(m)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Enabled() {
		t.Fatalf("definitions must be created disabled unless asked otherwise")
	}

	updated, err := p.SetEnabled(created.Id(), true)
	if err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	if !updated.Enabled() || updated.Id() != created.Id() || updated.Type() != created.Type() {
		t.Fatalf("SetEnabled changed more than enabled: %+v", updated)
	}
}

// A nonexistent id must surface gorm.ErrRecordNotFound from the setEnabled
// administrator itself, not silently no-op with a nil error. This calls the
// unexported administrator directly (bypassing Processor.SetEnabled's
// follow-up GetById read) so the assertion pins the administrator's own
// contract rather than an accidental side effect of the processor's next
// call.
func TestAdministratorSetEnabledReturnsRecordNotFoundForMissingId(t *testing.T) {
	db := newTestDB(t)

	err := setEnabled(db)(uuid.New())(true)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("setEnabled on a missing id = %v, want gorm.ErrRecordNotFound", err)
	}
}

// Processor.SetEnabled must propagate that same not-found error to its
// caller — the REST layer's errors.Is(err, gorm.ErrRecordNotFound) check
// depends on it.
func TestSetEnabledReturnsRecordNotFoundForMissingId(t *testing.T) {
	registryReset(t)
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)

	if _, err := p.SetEnabled(uuid.New(), true); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("SetEnabled on a missing id = %v, want gorm.ErrRecordNotFound", err)
	}
}

func TestGetByIdReturnsRecordNotFoundForMissingRow(t *testing.T) {
	registryReset(t)
	db := newTestDB(t)
	if _, err := NewProcessor(testLogger(t), testCtx(t), db).GetById(uuid.New()); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("GetById on a missing id = %v, want gorm.ErrRecordNotFound", err)
	}
}

func TestGetAllPagedIsTenantScoped(t *testing.T) {
	registryReset(t)
	registry.Register(acceptingHandler{t: "OK"})
	db := newTestDB(t)

	ctxA := databasetest.TenantContext(uuid.New())
	ctxB := databasetest.TenantContext(uuid.New())
	pa := NewProcessor(testLogger(t), ctxA, db)
	pb := NewProcessor(testLogger(t), ctxB, db)

	m, _ := NewBuilder("OK", "n").SetConfiguration(json.RawMessage(`{}`)).Build()
	if _, err := pa.Create(m); err != nil {
		t.Fatalf("Create: %v", err)
	}

	paged, err := pa.GetAllPaged(model.Page{Number: 1, Size: 50})
	if err != nil {
		t.Fatalf("GetAllPaged: %v", err)
	}
	if paged.Total != 1 {
		t.Fatalf("GetAllPaged total = %d, want 1", paged.Total)
	}

	otherPaged, err := pb.GetAllPaged(model.Page{Number: 1, Size: 50})
	if err != nil {
		t.Fatalf("GetAllPaged (other tenant): %v", err)
	}
	if otherPaged.Total != 0 {
		t.Fatalf("another tenant saw %d definitions, want 0", otherPaged.Total)
	}
}

func TestGetByTypeAndGetEnabledByType(t *testing.T) {
	registryReset(t)
	registry.Register(acceptingHandler{t: "OK"})
	db := newTestDB(t)
	p := NewProcessor(testLogger(t), testCtx(t), db)

	m1, _ := NewBuilder("OK", "one").SetConfiguration(json.RawMessage(`{}`)).Build()
	m2, _ := NewBuilder("OK", "two").SetConfiguration(json.RawMessage(`{}`)).Build()
	created1, err := p.Create(m1)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := p.Create(m2); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := p.SetEnabled(created1.Id(), true); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}

	all, err := p.GetByType("OK")
	if err != nil {
		t.Fatalf("GetByType: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("GetByType returned %d rows, want 2", len(all))
	}

	enabled, err := p.GetEnabledByType("OK")
	if err != nil {
		t.Fatalf("GetEnabledByType: %v", err)
	}
	if len(enabled) != 1 || enabled[0].Id() != created1.Id() {
		t.Fatalf("GetEnabledByType returned %+v, want only [%s]", enabled, created1.Id())
	}
}

// FR-UI4/R33-4: singleOccurrence now asks the handler directly via
// ConcurrencyKeyIsConstant rather than probing ConcurrencyKey with two
// payloads and comparing — the constantKeyHandler/varyingKeyHandler stubs
// below express constancy through that method, not through
// probe-branching ConcurrencyKey bodies (a real handler never branches on
// probe content, which was exactly what made the old double-probe unsound —
// see registry.Handler.ConcurrencyKeyIsConstant's doc comment).
func TestSingleOccurrenceDerivation(t *testing.T) {
	registryReset(t)
	registry.Register(constantKeyHandler{t: "CONST"})
	registry.Register(varyingKeyHandler{t: "VARY"})

	if !singleOccurrence(context.Background(), "CONST") {
		t.Fatalf("CONST: expected a constant concurrency key to report singleOccurrence=true")
	}
	if singleOccurrence(context.Background(), "VARY") {
		t.Fatalf("VARY: expected a varying concurrency key to report singleOccurrence=false")
	}
	if singleOccurrence(context.Background(), "NO_HANDLER") {
		t.Fatalf("an unregistered type must report singleOccurrence=false")
	}
}

type constantKeyHandler struct{ t string }

func (h constantKeyHandler) Type() string                                { return h.t }
func (h constantKeyHandler) ValidateConfiguration(json.RawMessage) error { return nil }
func (h constantKeyHandler) ConcurrencyKey(context.Context, json.RawMessage) (string, error) {
	return "constant", nil
}
func (h constantKeyHandler) ConcurrencyKeyIsConstant() bool { return true }

func (h constantKeyHandler) Evaluate(context.Context, registry.Definition, registry.Work) (*registry.Seed, error) {
	return nil, nil
}

func (h constantKeyHandler) Start(context.Context, registry.Occurrence) (registry.Progress, error) {
	return registry.Progress{}, nil
}

func (h constantKeyHandler) Advance(context.Context, registry.Occurrence, registry.Work) (registry.Progress, error) {
	return registry.Progress{}, nil
}

type varyingKeyHandler struct{ t string }

func (h varyingKeyHandler) Type() string                                { return h.t }
func (h varyingKeyHandler) ValidateConfiguration(json.RawMessage) error { return nil }
func (h varyingKeyHandler) ConcurrencyKey(_ context.Context, workContext json.RawMessage) (string, error) {
	return string(workContext), nil
}
func (h varyingKeyHandler) ConcurrencyKeyIsConstant() bool { return false }

func (h varyingKeyHandler) Evaluate(context.Context, registry.Definition, registry.Work) (*registry.Seed, error) {
	return nil, nil
}

func (h varyingKeyHandler) Start(context.Context, registry.Occurrence) (registry.Progress, error) {
	return registry.Progress{}, nil
}

func (h varyingKeyHandler) Advance(context.Context, registry.Occurrence, registry.Work) (registry.Progress, error) {
	return registry.Progress{}, nil
}
