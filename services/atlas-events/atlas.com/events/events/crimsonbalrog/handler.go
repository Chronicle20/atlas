package crimsonbalrog

import (
	"atlas-events/event/registry"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrNotImplemented marks a Handler method filled in later in Phase E (Tasks
// 24, 25, 27). No caller may treat it as a normal outcome.
var ErrNotImplemented = errors.New("crimsonbalrog: not implemented")

// Handler is the CRIMSON_BALROG registry.Handler. This task writes only the
// static half (Type, ValidateConfiguration, ConcurrencyKey); Evaluate, Start,
// Advance and Complete are filled in by Tasks 24, 25 and 27 within the same
// phase and branch.
type Handler struct{}

// NewHandler constructs the CRIMSON_BALROG handler.
func NewHandler() registry.Handler {
	return &Handler{}
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
// Filled in by Task 24.
func (h *Handler) Evaluate(_ context.Context, _ registry.Definition, _ registry.Work) (*registry.Seed, error) {
	return nil, fmt.Errorf("Evaluate: %w", ErrNotImplemented)
}

// Start orchestrates the side effects of a newly created occurrence.
// Filled in by Task 25.
func (h *Handler) Start(_ context.Context, _ registry.Occurrence) (registry.Progress, error) {
	return registry.Progress{}, fmt.Errorf("Start: %w", ErrNotImplemented)
}

// Advance handles a due OCCURRENCE_TRANSITION row. Filled in by Task 25.
func (h *Handler) Advance(_ context.Context, _ registry.Occurrence, _ registry.Work) (registry.Progress, error) {
	return registry.Progress{}, fmt.Errorf("Advance: %w", ErrNotImplemented)
}

// Complete is cleanup for a terminal transition. Filled in by Task 27.
func (h *Handler) Complete(_ context.Context, _ registry.Occurrence, _ string) error {
	return fmt.Errorf("Complete: %w", ErrNotImplemented)
}
