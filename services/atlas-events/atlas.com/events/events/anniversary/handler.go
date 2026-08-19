package anniversary

import (
	"atlas-events/event/registry"
	"atlas-events/kafka/message"
	"atlas-events/kafka/message/buff"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// concurrencyKey is the single gameplay slot every ANNIVERSARY occurrence
// occupies (design §15.2, FR-UI4): at most one ANNIVERSARY occurrence can be
// active tenant-wide, regardless of definition.
const concurrencyKey = "anniversary"

// ReasonScheduledEnd is the CompletionReason for an ANNIVERSARY occurrence
// completed because its scheduled window ended (FR-A14).
const ReasonScheduledEnd = "SCHEDULED_END"

// Handler is the ANNIVERSARY registry.Handler.
type Handler struct {
	db *gorm.DB
	l  logrus.FieldLogger
}

// NewHandler constructs the ANNIVERSARY handler using the standard logger.
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db, l: logrus.StandardLogger()}
}

// NewHandlerWith constructs the ANNIVERSARY handler with an injected
// logger, so a test can capture Emit's output.
func NewHandlerWith(db *gorm.DB, l logrus.FieldLogger) *Handler {
	return &Handler{db: db, l: l}
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

// ConcurrencyKey is constant: ANNIVERSARY has no per-occurrence scope, only
// one global window can be active at a time (design §15.2, FR-UI4).
func (h *Handler) ConcurrencyKey(_ context.Context, _ json.RawMessage) (string, error) {
	return concurrencyKey, nil
}

// ConcurrencyKeyIsConstant is true: ConcurrencyKey never varies with its
// workContext argument (R33-4). The UI can render live single-occurrence
// state on the definition row rather than linking out to a filtered list.
func (h *Handler) ConcurrencyKeyIsConstant() bool { return true }

// Evaluate decides whether a TRIGGER_EVALUATION should open the ANNIVERSARY
// window. Returning (nil, nil) is the ordinary "no occurrence" outcome
// (FR-A4: the window has already fully elapsed, or has not opened yet — in
// which case the row that will do the work at scheduledStart is scheduled
// here, via Scheduler, so a definition enabled well before its window opens
// still starts on time (FR-A6)).
func (h *Handler) Evaluate(ctx context.Context, d registry.Definition, _ registry.Work) (*registry.Seed, error) {
	c, err := DecodeConfig(d.Configuration)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if !c.ScheduledEnd.After(now) {
		return nil, nil
	}
	if c.ScheduledStart.After(now) {
		if err := NewScheduler(h.l, ctx, h.db).scheduleStart(d.Id, c); err != nil {
			return nil, err
		}
		return nil, nil
	}

	oc := OccurrenceContext{
		ScheduledEnd:   c.ScheduledEnd,
		ExpMultiplier:  c.ExpMultiplier,
		DropMultiplier: c.DropMultiplier,
		BuffSourceId:   c.BuffSourceId,
	}
	raw, err := EncodeOccurrenceContext(oc)
	if err != nil {
		return nil, err
	}

	return &registry.Seed{
		Context:        raw,
		ConcurrencyKey: concurrencyKey,
	}, nil
}

// Start orchestrates the side effects of a newly created occurrence: it does
// no emitting of its own (the buff/multipliers are applied by a later task
// per R33-6 — this task confines itself to schedule/window/completion), and
// settles NextTransitionAt at the window's end (FR-A3).
func (h *Handler) Start(_ context.Context, o registry.Occurrence) (registry.Progress, error) {
	oc, err := DecodeOccurrenceContext(o.Context)
	if err != nil {
		return registry.Progress{}, err
	}
	end := oc.ScheduledEnd
	return registry.Progress{NextTransitionAt: &end}, nil
}

// Advance handles the due OCCURRENCE_TRANSITION row that fires at the
// window's scheduled end (FR-A14). It emits exactly ONE CANCEL_BY_CORRELATION
// command, carrying the occurrence id as the correlation — not one per
// online character (FR-A15) — and completes the occurrence.
func (h *Handler) Advance(ctx context.Context, o registry.Occurrence, _ registry.Work) (registry.Progress, error) {
	if err := message.Emit(h.l, ctx)(func(buf *message.Buffer) error {
		return buf.Put(buff.EnvCommandTopic, cancelByCorrelationCommandProvider(o.Id))
	}); err != nil {
		return registry.Progress{}, err
	}

	return registry.Progress{Terminal: true, CompletionReason: ReasonScheduledEnd}, nil
}

// cancelByCorrelationCommandProvider builds the single CANCEL_BY_CORRELATION
// command that sweeps every buff granted with this occurrence's id as the
// correlation, tenant-wide (FR-A15). The correlation string sent here is
// occurrenceId.String() — the exact value a later task's buff-apply path
// must also use as CorrelationId, or the sweep will not find what it
// applied.
func cancelByCorrelationCommandProvider(occurrenceId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(0)
	value := &buff.Command[buff.CancelByCorrelationCommandBody]{
		Type: buff.CommandTypeCancelByCorrelation,
		Body: buff.CancelByCorrelationCommandBody{CorrelationId: occurrenceId.String()},
	}
	return producer.SingleMessageProvider(key, value)
}
