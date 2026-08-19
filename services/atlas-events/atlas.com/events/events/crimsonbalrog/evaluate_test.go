package crimsonbalrog

import (
	"atlas-events/event/registry"
	"atlas-events/external/maps"
	"atlas-events/external/transports"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// fakes assembles a Handler wired to in-memory stand-ins for the
// atlas-transports/atlas-maps clients and the probability roll, plus the
// default definition/work-context pair every gate would ACCEPT. Each test
// mutates exactly the field(s) that should flip one gate.
type fakes struct {
	t *testing.T

	ctx        context.Context
	definition registry.Definition
	work       registry.Work
	voyageId   uuid.UUID

	route    transports.RestModel
	routeErr error

	charactersByMap map[_map.Id][]uint32
	charactersErr   error

	roll func() float64

	// logHook captures what handler()'s logger emitted, so a test can assert
	// WHICH gate rejected an evaluation. That is the whole point of the gate
	// logging: every rejection returns the same (nil, nil), so the log line
	// is the only thing distinguishing them.
	logHook *test.Hook
}

// rejectedGate returns the `gate` field of the single
// crimsonbalrog.evaluate_rejected line the handler emitted, failing the test
// unless there was exactly one.
func (f *fakes) rejectedGate() string {
	f.t.Helper()
	entries := f.logHook.AllEntries()
	if len(entries) != 1 {
		f.t.Fatalf("expected exactly 1 log entry, got %d", len(entries))
	}
	if got := entries[0].Message; got != "crimsonbalrog.evaluate_rejected" {
		f.t.Fatalf("expected a rejection entry, got message %q", got)
	}
	gate, ok := entries[0].Data["gate"]
	if !ok {
		f.t.Fatalf("rejection entry carries no gate field: %v", entries[0].Data)
	}
	return gate.(string)
}

// newFakes returns a fakes where every FR-B5 gate passes: the voyage is
// underway on the expected voyage, the definition is enabled, the roll
// undercuts the configured probability, and someone is aboard an attack map.
func newFakes(t *testing.T) *fakes {
	t.Helper()

	voyageId := uuid.New()

	// Grounded configuration values, from
	// deploy/seed/shared/all/events/definitions/event-crimson-balrog.json.
	cfg := Config{
		ApplicableRouteIds: []string{"boat-ellinia-orbis", "boat-orbis-ellinia"},
		AttackProbability:  0.42,
		MonsterId:          8150000,
		MonsterCount:       2,
		AttackMaps: []AttackMap{
			{MapId: 200090010, SpawnPositions: []Position{{X: 339, Y: 148}, {X: 339, Y: 148}}},
			{MapId: 200090000, SpawnPositions: []Position{{X: -538, Y: 143}, {X: -538, Y: 143}}},
		},
		RelatedMapIds:   []_map.Id{200090011, 200090001},
		BackgroundMusic: "Bgm04/ArabPirate",
		Visual: VisualConfig{
			Name: "CONTI_MOVE",
		},
	}
	cfgRaw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}

	wc := WorkContext{
		VoyageId:  voyageId,
		RouteId:   uuid.New(),
		WorldId:   world.Id(1),
		ChannelId: channel.Id(4),
	}
	wcRaw, err := json.Marshal(wc)
	if err != nil {
		t.Fatalf("marshal work context: %v", err)
	}

	return &fakes{
		t:   t,
		ctx: context.Background(),
		definition: registry.Definition{
			Id:            uuid.New(),
			Type:          TypeName,
			Name:          "Crimson Balrog",
			Enabled:       true,
			Configuration: cfgRaw,
		},
		work: registry.Work{
			Id:      uuid.New(),
			Type:    "TRIGGER_EVALUATION",
			Context: wcRaw,
		},
		voyageId: voyageId,
		route: transports.RestModel{
			State:    "in_transit",
			VoyageID: voyageId.String(),
		},
		charactersByMap: map[_map.Id][]uint32{200090010: {42}},
		roll:            func() float64 { return 0 },
	}
}

func (f *fakes) handler() *Handler {
	return &Handler{
		roll: f.roll,
		transports: func(context.Context) transports.Processor {
			return fakeTransportsProcessor{route: f.route, err: f.routeErr}
		},
		maps: func(context.Context) maps.Processor {
			return fakeMapsProcessor{byMap: f.charactersByMap, err: f.charactersErr}
		},
		// Evaluate names its rejecting gate on this logger, and Start (Task
		// 25) threads it into message.Emit. A null logger keeps test output
		// pristine while its hook keeps every entry assertable.
		l: f.logger(),
	}
}

// logger builds the null logger handler() installs, retaining its hook on
// the fakes so rejectedGate can read back what was emitted.
func (f *fakes) logger() logrus.FieldLogger {
	l, hook := test.NewNullLogger()
	f.logHook = hook
	return l
}

type fakeTransportsProcessor struct {
	route transports.RestModel
	err   error
}

func (f fakeTransportsProcessor) GetRoute(uuid.UUID) (transports.RestModel, error) {
	return f.route, f.err
}

var _ transports.Processor = fakeTransportsProcessor{}

type fakeMapsProcessor struct {
	byMap map[_map.Id][]uint32
	err   error
}

func (f fakeMapsProcessor) CharacterIdsInMap(fm field.Model) ([]uint32, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.byMap[fm.MapId()], nil
}

var _ maps.Processor = fakeMapsProcessor{}

// Each rejection path asserts NO occurrence is seeded — that is what preserves
// the occurrence table's meaning as a history of real events (§4).
func TestEvaluateRejectionPaths(t *testing.T) {
	// Every case asserts the logged gate as well as the nil seed: the seed
	// alone cannot tell these five apart, which is exactly the ambiguity the
	// gate field exists to remove.
	for _, tc := range []struct {
		name  string
		setup func(*fakes)
		gate  string
	}{
		{"voyage already arrived", func(f *fakes) { f.route.State = "awaiting_return" }, gateNotUnderway},
		{"voyage replaced by the next trip", func(f *fakes) { f.route.VoyageID = uuid.New().String() }, gateNotUnderway},
		{"definition disabled since departure", func(f *fakes) { f.definition.Enabled = false }, gateNotEnabled},
		{"probability roll failed", func(f *fakes) { f.roll = func() float64 { return 0.99 } }, gateRoll},
		{"nobody aboard", func(f *fakes) { f.charactersByMap = map[_map.Id][]uint32{} }, gateNobodyAboard},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakes(t)
			tc.setup(f)
			seed, err := f.handler().Evaluate(f.ctx, f.definition, f.work)
			if err != nil {
				t.Fatalf("rejection must not be an error: %v", err)
			}
			if seed != nil {
				t.Fatalf("expected no occurrence, got %+v", seed)
			}
			if got := f.rejectedGate(); got != tc.gate {
				t.Fatalf("logged gate = %q, want %q", got, tc.gate)
			}
		})
	}
}

// FR-B6: "aboard" is the UNION of attack maps and related maps — a character in
// the cabin counts.
func TestCharacterInTheCabinCountsAsAboard(t *testing.T) {
	f := newFakes(t)
	f.charactersByMap = map[_map.Id][]uint32{200090011: {42}} // cabin only

	seed, err := f.handler().Evaluate(f.ctx, f.definition, f.work)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if seed == nil {
		t.Fatalf("cabin occupancy must count as aboard (FR-B6)")
	}
}

// FR-B9/FR-B10/FR-API8: attack maps are visual, related maps are not.
func TestSuccessfulEvaluationSeedsTheCorrectScope(t *testing.T) {
	f := newFakes(t)
	f.charactersByMap = map[_map.Id][]uint32{200090010: {42}}

	seed, err := f.handler().Evaluate(f.ctx, f.definition, f.work)
	if err != nil || seed == nil {
		t.Fatalf("Evaluate: seed=%v err=%v", seed, err)
	}
	if seed.Stage != StageAttacking {
		t.Fatalf("stage = %q, want %q", seed.Stage, StageAttacking)
	}
	if seed.WorldId != 1 || seed.ChannelId != 4 || seed.VoyageId != f.voyageId {
		t.Fatalf("scope = %d/%d/%s", seed.WorldId, seed.ChannelId, seed.VoyageId)
	}
	got := map[_map.Id]bool{}
	for _, ms := range seed.Maps {
		got[ms.MapId] = ms.Visual
	}
	if !got[200090010] {
		t.Fatalf("attack map must be visual")
	}
	if got[200090011] {
		t.Fatalf("cabin must NOT be visual (FR-B13)")
	}

	// The generic dispatch path (event/scheduling and event/occurrence
	// processors) copies Seed.ConcurrencyKey verbatim and never calls
	// h.ConcurrencyKey itself — Evaluate is the only place that fills it in
	// (evaluate.go). An empty key would silently defeat the single-occurrence
	// guarantee, so pin it against the handler's own ConcurrencyKey contract
	// rather than merely asserting non-empty.
	wantKey, err := f.handler().ConcurrencyKey(f.ctx, f.work.Context)
	if err != nil {
		t.Fatalf("ConcurrencyKey: %v", err)
	}
	if wantKey == "" {
		t.Fatalf("test setup produced an empty concurrency key")
	}
	if seed.ConcurrencyKey != wantKey {
		t.Fatalf("seed.ConcurrencyKey = %q, want %q (from h.ConcurrencyKey)", seed.ConcurrencyKey, wantKey)
	}
}

// An unreachable dependency must RETRY, not be read as a negative answer. This
// is the difference between "nobody was aboard" and "we could not tell".
func TestUnreachableDependenciesReturnErrors(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*fakes)
	}{
		{"transports down", func(f *fakes) { f.routeErr = errors.New("connection refused") }},
		{"maps down", func(f *fakes) { f.charactersErr = errors.New("connection refused") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakes(t)
			tc.setup(f)
			if _, err := f.handler().Evaluate(f.ctx, f.definition, f.work); err == nil {
				t.Fatalf("expected an error so the work row retries")
			}
		})
	}
}

// Enabling a definition schedules one generic TRIGGER_EVALUATION with an
// empty work context. For an externally-triggered event that means "nothing
// to do" — not an error, and certainly not an occurrence with a zero voyage
// id.
func TestCrimsonBalrogEvaluateIgnoresAnEmptyWorkContext(t *testing.T) {
	f := newFakes(t)
	f.work.Context = json.RawMessage(`{}`)

	seed, err := f.handler().Evaluate(f.ctx, f.definition, f.work)
	if err != nil {
		t.Fatalf("an enable-triggered evaluation must not error: %v", err)
	}
	if seed != nil {
		t.Fatalf("created an occurrence with no voyage: %+v", seed)
	}
	if got := f.rejectedGate(); got != gateNoVoyage {
		t.Fatalf("logged gate = %q, want %q", got, gateNoVoyage)
	}
}

// A successful evaluation says so, and names the roll that carried it: a
// silent success would leave the operator unable to distinguish "the event
// fired" from "the log line for this voyage is missing".
func TestSuccessfulEvaluationLogsTheSeed(t *testing.T) {
	f := newFakes(t)

	seed, err := f.handler().Evaluate(f.ctx, f.definition, f.work)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if seed == nil {
		t.Fatal("expected an occurrence seed")
	}

	entries := f.logHook.AllEntries()
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 log entry, got %d", len(entries))
	}
	if got := entries[0].Message; got != "crimsonbalrog.evaluate_seeded" {
		t.Fatalf("log message = %q, want crimsonbalrog.evaluate_seeded", got)
	}
	if _, ok := entries[0].Data["rolled"]; !ok {
		t.Fatalf("seeded entry carries no rolled field: %v", entries[0].Data)
	}
}
