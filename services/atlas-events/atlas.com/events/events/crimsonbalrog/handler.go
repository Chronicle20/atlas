package crimsonbalrog

import (
	"atlas-events/event/registry"
	"atlas-events/external/maps"
	"atlas-events/external/transports"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/sirupsen/logrus"
)

// Handler is the CRIMSON_BALROG registry.Handler.
//
// roll and the two client constructors are fields, not direct calls, so a
// test can pin the roll and fake the atlas-transports/atlas-maps clients
// (Task 24's evaluate_test.go) without a network dependency; NewHandler
// wires the real ones. l is the logger Start (Task 25) threads into
// message.Emit for the visual event and spawn commands.
type Handler struct {
	roll       func() float64
	transports func(ctx context.Context) transports.Processor
	maps       func(ctx context.Context) maps.Processor
	l          logrus.FieldLogger
}

// NewHandler constructs the CRIMSON_BALROG handler.
func NewHandler() registry.Handler {
	return &Handler{
		roll: rand.Float64,
		transports: func(ctx context.Context) transports.Processor {
			return transports.NewProcessor(logrus.StandardLogger(), ctx)
		},
		maps: func(ctx context.Context) maps.Processor {
			return maps.NewProcessor(logrus.StandardLogger(), ctx)
		},
		l: logrus.StandardLogger(),
	}
}

// compile-time assertion
var _ registry.Handler = (*Handler)(nil)

// Type is the definition type this handler serves. Used as the registry key.
func (h *Handler) Type() string { return TypeName }

// ValidateConfiguration rejects a definition whose configuration this handler
// cannot interpret (FR-D6); returns a field-scoped error.
func (h *Handler) ValidateConfiguration(raw json.RawMessage) error {
	c, err := DecodeConfig(raw)
	if err != nil {
		return err
	}
	return c.Validate()
}

// ConcurrencyKey scopes an occurrence to one voyage in one channel of one
// world (FR-V2). The generic layer turns this into a unique constraint among
// ACTIVE occurrences, which is design §5.3's third idempotency guard: even if
// Kafka dedup and the work-row state machine BOTH fail, the second occurrence
// insert fails rather than producing two live Balrog attacks on one voyage.
func (h *Handler) ConcurrencyKey(_ context.Context, workContext json.RawMessage) (string, error) {
	var wc WorkContext
	if err := json.Unmarshal(workContext, &wc); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s|%d|%d", wc.VoyageId, wc.WorldId, wc.ChannelId), nil
}

// Evaluate decides whether a TRIGGER_EVALUATION should produce an occurrence.
// Implemented in evaluate.go (Task 24).

// Start orchestrates the side effects of a newly created occurrence.
// Implemented in start.go (Task 25).

// Advance handles a due OCCURRENCE_TRANSITION row. CRIMSON_BALROG never
// schedules one (FR-B17: completion is externally driven by monster
// elimination or voyage arrival, not by a timed transition — start.go's
// Start always returns a nil NextTransitionAt). A row of this work type for
// this occurrence type is therefore a bug — a stray scheduled transition, or
// a future code path that started scheduling one without updating this
// handler — and must surface loudly as a FAILED work row with a named
// reason (event/scheduling/processor.go's applyOutcome), not be swallowed as
// a silent no-op.
func (h *Handler) Advance(_ context.Context, o registry.Occurrence, w registry.Work) (registry.Progress, error) {
	return registry.Progress{}, fmt.Errorf("crimsonbalrog: unexpected %s work for occurrence %s", w.Type, o.Id)
}

// Complete is cleanup for a terminal transition (FR-B18, FR-B19, FR-A15),
// driven by the generic scheduling layer rather than by this package's own
// consumers (arrival.go's ArrivalProcessor, monsters.go's MonsterProcessor).
// It delegates to the same emitCleanup (arrival.go) those two paths use, so
// a completion driven by the generic layer produces identical wire traffic
// to one driven by the consumer — no second, divergent HIDE/DESTROY_BY_SOURCE
// shape (ruling 4). reason is unused: emitCleanup's wire traffic does not
// depend on WHY the occurrence completed, only on what it owns.
func (h *Handler) Complete(ctx context.Context, o registry.Occurrence, _ string) error {
	return emitCleanup(h.l, ctx, o.Id, o.Context)
}
