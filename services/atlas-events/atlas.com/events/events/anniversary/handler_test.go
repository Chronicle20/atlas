package anniversary

import (
	"atlas-events/event/definition"
	"atlas-events/event/occurrence"
	"atlas-events/event/registry"
	"atlas-events/event/scheduling"
	"atlas-events/event/transition"
	"atlas-events/kafka/message/buff"
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// testTenantId is fixed (rather than generated per-test) so seedDefinition
// and testCtx agree on the same tenant without threading it through every
// helper call.
var testTenantId = uuid.MustParse("11111111-2222-3333-4444-555555555555")

// now anchors every relative time in this file's tests to a single instant,
// so "scheduledStart in the past"/"in the future" comparisons stay
// consistent within a test run.
var now = time.Now()

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return databasetest.NewInMemoryTenantDB(t, definition.MigrateTable, occurrence.MigrateTable, scheduling.MigrateTable, transition.MigrateTable)
}

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	return databasetest.TenantContext(testTenantId)
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// defOpt customizes a definition seeded by seedDefinition.
type defOpt func(*testDefSpec)

type testDefSpec struct {
	enabled bool
	cfg     Config
}

func enabled(v bool) defOpt { return func(s *testDefSpec) { s.enabled = v } }

func window(start, end time.Time) defOpt {
	return func(s *testDefSpec) {
		s.cfg.ScheduledStart = start
		s.cfg.ScheduledEnd = end
	}
}

// defaultSpec builds a valid default ANNIVERSARY config: a week-long window
// starting one hour from now.
func defaultSpec() testDefSpec {
	return testDefSpec{
		cfg: Config{
			ScheduledStart: now.Add(1 * time.Hour),
			ScheduledEnd:   now.Add(1*time.Hour + 7*24*time.Hour),
			ExpMultiplier:  2,
			DropMultiplier: 2,
			BuffSourceId:   1000000000,
		},
	}
}

// seedDefinition creates and (optionally) enables an ANNIVERSARY definition
// with a valid base configuration, customized by opts. It calls
// definition.NewProcessor directly, NOT orchestration.SetEnabled, since
// these tests exercise Scheduler.OnDefinitionEnabled explicitly and must not
// double-schedule.
var registerOnce sync.Once

func ensureHandlerRegistered(db *gorm.DB) {
	registerOnce.Do(func() { registry.Register(NewHandler(db)) })
}

func seedDefinition(t *testing.T, db *gorm.DB, theType string, opts ...defOpt) definition.Model {
	t.Helper()
	ensureHandlerRegistered(db)

	spec := defaultSpec()
	for _, o := range opts {
		o(&spec)
	}

	raw, err := json.Marshal(spec.cfg)
	must(t, err)

	m, err := definition.NewBuilder(theType, "test-definition").SetConfiguration(raw).Build()
	must(t, err)

	p := definition.NewProcessor(testLogger(t), testCtx(t), db)
	created, err := p.Create(m)
	must(t, err)

	if spec.enabled {
		created, err = p.SetEnabled(created.Id(), true)
		must(t, err)
	}

	return created
}

// definitionWith builds a bare registry.Definition (not DB-backed), reusing
// the same testDefSpec/defOpt machinery, for tests that call Handler.Evaluate
// directly without a definition row.
func definitionWith(opts ...defOpt) registry.Definition {
	spec := defaultSpec()
	for _, o := range opts {
		o(&spec)
	}
	raw, _ := json.Marshal(spec.cfg)
	return registry.Definition{
		Id:            uuid.New(),
		Type:          TypeName,
		Name:          "test-definition",
		Enabled:       spec.enabled,
		Configuration: raw,
	}
}

// seedActiveOccurrence seeds a definition and one ACTIVE occurrence for it,
// with an OccurrenceContext built from the definition's own config.
func seedActiveOccurrence(t *testing.T, db *gorm.DB, theType string, opts ...defOpt) occurrence.Model {
	t.Helper()

	d := seedDefinition(t, db, theType, opts...)
	c, err := DecodeConfig(d.Configuration())
	must(t, err)

	oc := OccurrenceContext{
		ScheduledEnd:   c.ScheduledEnd,
		ExpMultiplier:  c.ExpMultiplier,
		DropMultiplier: c.DropMultiplier,
		BuffSourceId:   c.BuffSourceId,
	}
	raw, err := EncodeOccurrenceContext(oc)
	must(t, err)

	m, err := occurrence.NewBuilder(d.Id(), theType).
		SetState(occurrence.StateActive).
		SetContext(raw).
		SetConcurrencyKey(concurrencyKey).
		SetStartedAt(time.Now()).
		Build()
	must(t, err)

	tn := tenant.MustFromContext(testCtx(t))
	entity, err := occurrence.ToEntity(m, tn.Id())
	must(t, err)
	must(t, db.Create(&entity).Error)

	made, err := occurrence.Make(entity)
	must(t, err)
	return made
}

func readAllWork(t *testing.T, db *gorm.DB) []scheduling.Entity {
	t.Helper()
	var out []scheduling.Entity
	if err := db.Order("execute_at asc").Find(&out).Error; err != nil {
		t.Fatalf("readAllWork: %v", err)
	}
	return out
}

// registryOccurrence narrows an occurrence.Model to the read-only view a
// registry.Handler receives. Mirrors the unexported toRegistryOccurrence in
// event/scheduling/processor.go — cannot reuse it directly, so redefined
// identically here.
func registryOccurrence(o occurrence.Model) registry.Occurrence {
	return registry.Occurrence{
		Id:           o.Id(),
		DefinitionId: o.DefinitionId(),
		Type:         o.Type(),
		Stage:        o.Stage(),
		Context:      o.Context(),
		WorldId:      o.WorldId(),
		ChannelId:    o.ChannelId(),
		VoyageId:     o.VoyageId(),
		StartedAt:    o.StartedAt(),
	}
}

// emitted records every message Advance's tests produce. Installed once per
// package (producer.Manager caches one writer per topic for the lifetime of
// the singleton — producertest.Capture's own doc comment); each test that
// reads it goes through newEmitCapture, which resets it first.
var emitted *producertest.Capture

func TestMain(m *testing.M) {
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
}

// emitCapture embeds logrus.FieldLogger so *emitCapture itself satisfies
// logrus.FieldLogger via promotion — it can be passed directly as
// NewHandlerWith(db, f)'s second argument.
type emitCapture struct {
	logrus.FieldLogger
	t *testing.T
}

func newEmitCapture(t *testing.T) *emitCapture {
	t.Helper()
	emitted.Reset()
	l, _ := test.NewNullLogger()
	return &emitCapture{FieldLogger: l, t: t}
}

// emitted decodes every message captured on topic whose Type equals wantType
// as a buff.Command[buff.CancelByCorrelationCommandBody].
func (f *emitCapture) emitted(topic topic.Token, wantType string) []buff.Command[buff.CancelByCorrelationCommandBody] {
	f.t.Helper()
	var out []buff.Command[buff.CancelByCorrelationCommandBody]
	for _, m := range emitted.Messages(string(topic)) {
		var c buff.Command[buff.CancelByCorrelationCommandBody]
		if err := json.Unmarshal(m.Value, &c); err != nil {
			f.t.Fatalf("decode buff command: %v", err)
		}
		if c.Type != wantType {
			continue
		}
		out = append(out, c)
	}
	return out
}

// FR-A2/FR-A6: enabling a definition schedules the row that will do the
// work — at scheduledStart if the window is still future, at ~now if the
// window has already opened, and nothing at all if the whole window has
// already elapsed (FR-A4 applied at enable time too).
func TestEnablingSchedulesTheStart(t *testing.T) {
	cases := []struct {
		name       string
		start, end time.Time
		wantRows   int
		wantAt     *time.Time
	}{
		{
			name:     "future window",
			start:    now.Add(1 * time.Hour),
			end:      now.Add(2 * time.Hour),
			wantRows: 1,
			wantAt:   timePtr(now.Add(1 * time.Hour)),
		},
		{
			name:     "already started",
			start:    now.Add(-1 * time.Hour),
			end:      now.Add(1 * time.Hour),
			wantRows: 1,
			wantAt:   nil, // ~now, checked separately
		},
		{
			name:     "window fully elapsed",
			start:    now.Add(-2 * time.Hour),
			end:      now.Add(-1 * time.Hour),
			wantRows: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			// enabled(true): OnDefinitionEnabled itself never reads d.Enabled()
			// (schedule.go's scheduleStart only decodes Configuration), so the
			// flag does not gate this assertion. It is set anyway because the
			// real caller (orchestration.SetEnabled) always flips the
			// definition to enabled BEFORE invoking the scheduling side effect
			// this test drives directly — seeding it enabled keeps the fixture
			// representative of that precondition instead of an unreachable
			// still-disabled state.
			d := seedDefinition(t, db, TypeName, window(tc.start, tc.end), enabled(true))

			must(t, NewScheduler(testLogger(t), testCtx(t), db).OnDefinitionEnabled(d))

			work := readAllWork(t, db)
			if len(work) != tc.wantRows {
				t.Fatalf("scheduled %d rows, want %d", len(work), tc.wantRows)
			}
			if tc.wantRows == 0 {
				return
			}
			if work[0].Type != scheduling.WorkTypeTriggerEvaluation {
				t.Fatalf("type = %s", work[0].Type)
			}
			if tc.wantAt != nil {
				if !work[0].ExecuteAt.Equal(*tc.wantAt) {
					t.Fatalf("executeAt = %s, want %s", work[0].ExecuteAt, *tc.wantAt)
				}
			} else {
				// "already started": scheduled for ~now, not scheduledStart.
				if work[0].ExecuteAt.Before(now.Add(-1*time.Minute)) || work[0].ExecuteAt.After(now.Add(1*time.Minute)) {
					t.Fatalf("executeAt = %s, want ~now (%s)", work[0].ExecuteAt, now)
				}
				if work[0].ExecuteAt.Equal(tc.start) {
					t.Fatalf("executeAt = scheduledStart (%s), want ~now — the window is already open", tc.start)
				}
			}
		})
	}
}

func timePtr(t time.Time) *time.Time { return &t }

// FR-A3: Start returns NextTransitionAt = scheduledEnd and is not terminal.
func TestStartSchedulesTheEndTransition(t *testing.T) {
	db := newTestDB(t)
	o := seedActiveOccurrence(t, db, TypeName)

	p, err := NewHandler(db).Start(testCtx(t), registryOccurrence(o))
	must(t, err)

	if p.Terminal {
		t.Fatalf("Start returned Terminal = true, want false (FR-A3)")
	}
	oc, err := DecodeOccurrenceContext(o.Context())
	must(t, err)
	if p.NextTransitionAt == nil || !p.NextTransitionAt.Equal(oc.ScheduledEnd) {
		t.Fatalf("NextTransitionAt = %v, want %s", p.NextTransitionAt, oc.ScheduledEnd)
	}
}

// FR-A4: Evaluate refuses a fully-elapsed window — no retroactive occurrence.
func TestEvaluateRefusesAFullyElapsedWindow(t *testing.T) {
	db := newTestDB(t)
	d := definitionWith(window(now.Add(-2*time.Hour), now.Add(-1*time.Hour)))

	seed, err := NewHandler(db).Evaluate(testCtx(t), d, registry.Work{Type: scheduling.WorkTypeTriggerEvaluation, Context: json.RawMessage("{}")})
	must(t, err)
	if seed != nil {
		t.Fatalf("Evaluate returned a seed for an elapsed window, want nil (FR-A4)")
	}
}

// FR-A6: Evaluate on a not-yet-open window schedules the row that will do
// the work at scheduledStart, via Scheduler — end-to-end through Evaluate
// itself, not just Scheduler in isolation.
func TestEvaluateSchedulesTheStartWhenWindowNotYetOpen(t *testing.T) {
	db := newTestDB(t)
	start := now.Add(1 * time.Hour)
	end := now.Add(2 * time.Hour)
	dm := seedDefinition(t, db, TypeName, window(start, end))
	d := registry.Definition{Id: dm.Id(), Type: dm.Type(), Name: dm.Name(), Enabled: dm.Enabled(), Configuration: dm.Configuration()}

	seed, err := NewHandler(db).Evaluate(testCtx(t), d, registry.Work{Type: scheduling.WorkTypeTriggerEvaluation, Context: json.RawMessage("{}")})
	must(t, err)
	if seed != nil {
		t.Fatalf("Evaluate returned a seed before the window opened, want nil")
	}

	work := readAllWork(t, db)
	if len(work) != 1 {
		t.Fatalf("scheduled %d rows, want 1", len(work))
	}
	if !work[0].ExecuteAt.Equal(start) {
		t.Fatalf("executeAt = %s, want scheduledStart %s", work[0].ExecuteAt, start)
	}
}

// FR-A14/FR-A15: Advance completes with ReasonScheduledEnd and emits exactly
// ONE CANCEL_BY_CORRELATION, not one per character — carrying the occurrence
// id as the correlation.
func TestAdvanceCompletesAndCancelsOnce(t *testing.T) {
	db := newTestDB(t)
	o := seedActiveOccurrence(t, db, TypeName)
	f := newEmitCapture(t)

	p, err := NewHandlerWith(db, f).Advance(testCtx(t), registryOccurrence(o), registry.Work{Type: scheduling.WorkTypeOccurrenceTransition})
	must(t, err)

	if !p.Terminal {
		t.Fatalf("Advance returned Terminal = false, want true")
	}
	if p.CompletionReason != ReasonScheduledEnd {
		t.Fatalf("CompletionReason = %s, want %s", p.CompletionReason, ReasonScheduledEnd)
	}

	cancels := f.emitted(buff.EnvCommandTopic, buff.CommandTypeCancelByCorrelation)
	if len(cancels) != 1 {
		t.Fatalf("emitted %d CANCEL_BY_CORRELATION commands, want exactly 1 (FR-A15)", len(cancels))
	}
	if cancels[0].Body.CorrelationId != o.Id().String() {
		t.Fatalf("CorrelationId = %s, want %s", cancels[0].Body.CorrelationId, o.Id())
	}
}

// design §15.2, FR-UI4: at most one ANNIVERSARY occurrence globally.
func TestConcurrencyKeyIsConstant(t *testing.T) {
	if !NewHandler(newTestDB(t)).ConcurrencyKeyIsConstant() {
		t.Fatalf("ANNIVERSARY must report a constant concurrency key")
	}
}
