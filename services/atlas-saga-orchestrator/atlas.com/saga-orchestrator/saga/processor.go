package saga

import (
	"atlas-saga-orchestrator/cashshop"
	"atlas-saga-orchestrator/character"
	"atlas-saga-orchestrator/compartment"
	"atlas-saga-orchestrator/guild"
	"atlas-saga-orchestrator/invite"
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	"atlas-saga-orchestrator/kafka/message/saga"
	"atlas-saga-orchestrator/mts"
	"atlas-saga-orchestrator/skill"
	"atlas-saga-orchestrator/storage"
	"atlas-saga-orchestrator/validation"
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// acceptOptions carries optional constraints applied by AcceptEvent after the
// action/kind gate. Zero value means "no additional constraint".
type acceptOptions struct {
	characterId    uint32
	hasCharacterId bool
}

// AcceptOption constrains AcceptEvent beyond the action/kind match.
type AcceptOption func(*acceptOptions)

// ForCharacter constrains acceptance to a step whose payload names this
// character. A step whose payload carries no character id (ExtractCharacterId
// returns 0) is left unconstrained, so actions this plan does not touch keep
// their current behaviour exactly.
func ForCharacter(id uint32) AcceptOption {
	return func(o *acceptOptions) {
		o.characterId = id
		o.hasCharacterId = true
	}
}

// Processor is the interface for saga processing
type Processor interface {
	WithCharacterProcessor(character.Processor) Processor
	WithCompartmentProcessor(compartment.Processor) Processor
	WithSkillProcessor(processor skill.Processor) Processor
	WithValidationProcessor(validation.Processor) Processor
	WithGuildProcessor(guild.Processor) Processor
	WithInviteProcessor(invite.Processor) Processor
	WithCashshopProcessor(cashshop.Processor) Processor

	GetAll() ([]Saga, error)
	AllProvider() model.Provider[[]Saga]
	GetById(transactionId uuid.UUID) (Saga, error)
	ByIdProvider(transactionId uuid.UUID) model.Provider[Saga]

	Put(saga Saga) error
	MarkFurthestCompletedStepFailed(transactionId uuid.UUID) error
	MarkEarliestPendingStep(transactionId uuid.UUID, status Status) error
	MarkEarliestPendingStepCompleted(transactionId uuid.UUID) error
	StepCompleted(transactionId uuid.UUID, success bool) error
	StepCompletedWithResult(transactionId uuid.UUID, success bool, result map[string]any) error
	AddStep(transactionId uuid.UUID, step Step[any]) error
	AddStepAfterCurrent(transactionId uuid.UUID, step Step[any]) error
	Step(transactionId uuid.UUID) error

	// AcceptEvent is the single gate at which a saga-tagged Kafka event is
	// matched against the saga's pending step. Returns the decision (saga +
	// step) for handler-specific post-processing on success. On any skip
	// path it debug-logs and returns ok=false.
	AcceptEvent(transactionId uuid.UUID, kind EventKind, opts ...AcceptOption) (AcceptDecision, bool)
}

// AcceptDecision is returned by Processor.AcceptEvent on the match path so
// handlers can perform payload-specific work (templateId guard, reward-notice
// pre-emit, CreateAndEquipAsset follow-up step) before calling StepCompleted.
type AcceptDecision struct {
	Saga Saga
	Step Step[any]
}

// ProcessorImpl is the implementation of the Processor interface
type ProcessorImpl struct {
	l       logrus.FieldLogger
	ctx     context.Context
	t       tenant.Model
	comp    Compensator
	handle  Handler
	charP   character.Processor
	compP   compartment.Processor
	skillP  skill.Processor
	validP  validation.Processor
	guildP  guild.Processor
	inviteP invite.Processor
	csP     cashshop.Processor
}

const maxConflictRetries = 3

// unmatchedEventWarnOnce dedupes "unmatched_event" warn logs by (transactionId, kind).
// Package-level so dedup survives the per-request Processor instances.
var unmatchedEventWarnOnce sync.Map // key = tx.String() + "|" + string(kind), value = struct{}{}

// isVersionConflict checks if an error is a VersionConflictError
func isVersionConflict(err error) bool {
	var vce *VersionConflictError
	return errors.As(err, &vce)
}

// newProcessorFn is the package-level factory for Processor instances.
// Tests may override it via SetProcessorFactoryForTest to inject mocks.
var newProcessorFn = newProcessorImpl

// NewProcessor creates a new saga processor.
func NewProcessor(logger logrus.FieldLogger, ctx context.Context) Processor {
	return newProcessorFn(logger, ctx)
}

var _ Processor = (*ProcessorImpl)(nil)

func newProcessorImpl(logger logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:       logger,
		ctx:     ctx,
		t:       tenant.MustFromContext(ctx),
		comp:    NewCompensator(logger, ctx),
		handle:  NewHandler(logger, ctx),
		charP:   character.NewProcessor(logger, ctx),
		compP:   compartment.NewProcessor(logger, ctx),
		skillP:  skill.NewProcessor(logger, ctx),
		validP:  validation.NewProcessor(logger, ctx),
		guildP:  guild.NewProcessor(logger, ctx),
		inviteP: invite.NewProcessor(logger, ctx),
		csP:     cashshop.NewProcessor(logger, ctx),
	}
}

func (p *ProcessorImpl) WithCharacterProcessor(charP character.Processor) Processor {
	return &ProcessorImpl{
		l:       p.l,
		ctx:     p.ctx,
		t:       p.t,
		comp:    p.comp.WithCharacterProcessor(charP),
		handle:  p.handle.WithCharacterProcessor(charP),
		charP:   charP,
		compP:   p.compP,
		skillP:  p.skillP,
		validP:  p.validP,
		guildP:  p.guildP,
		inviteP: p.inviteP,
		csP:     p.csP,
	}
}

func (p *ProcessorImpl) WithCompartmentProcessor(compP compartment.Processor) Processor {
	return &ProcessorImpl{
		l:       p.l,
		ctx:     p.ctx,
		t:       p.t,
		comp:    p.comp.WithCompartmentProcessor(compP),
		handle:  p.handle.WithCompartmentProcessor(compP),
		charP:   p.charP,
		compP:   compP,
		skillP:  p.skillP,
		validP:  p.validP,
		guildP:  p.guildP,
		inviteP: p.inviteP,
		csP:     p.csP,
	}
}

func (p *ProcessorImpl) WithSkillProcessor(skillP skill.Processor) Processor {
	return &ProcessorImpl{
		l:       p.l,
		ctx:     p.ctx,
		t:       p.t,
		comp:    p.comp.WithSkillProcessor(skillP),
		handle:  p.handle.WithSkillProcessor(skillP),
		charP:   p.charP,
		compP:   p.compP,
		skillP:  skillP,
		validP:  p.validP,
		guildP:  p.guildP,
		inviteP: p.inviteP,
		csP:     p.csP,
	}
}

func (p *ProcessorImpl) WithValidationProcessor(validP validation.Processor) Processor {
	return &ProcessorImpl{
		l:       p.l,
		ctx:     p.ctx,
		t:       p.t,
		comp:    p.comp.WithValidationProcessor(validP),
		handle:  p.handle.WithValidationProcessor(validP),
		charP:   p.charP,
		compP:   p.compP,
		skillP:  p.skillP,
		validP:  validP,
		guildP:  p.guildP,
		inviteP: p.inviteP,
		csP:     p.csP,
	}
}

func (p *ProcessorImpl) WithGuildProcessor(guildP guild.Processor) Processor {
	return &ProcessorImpl{
		l:       p.l,
		ctx:     p.ctx,
		t:       p.t,
		comp:    p.comp.WithGuildProcessor(guildP),
		handle:  p.handle.WithGuildProcessor(guildP),
		charP:   p.charP,
		compP:   p.compP,
		skillP:  p.skillP,
		validP:  p.validP,
		guildP:  guildP,
		inviteP: p.inviteP,
		csP:     p.csP,
	}
}

func (p *ProcessorImpl) WithInviteProcessor(inviteP invite.Processor) Processor {
	return &ProcessorImpl{
		l:       p.l,
		ctx:     p.ctx,
		t:       p.t,
		comp:    p.comp.WithInviteProcessor(inviteP),
		handle:  p.handle.WithInviteProcessor(inviteP),
		charP:   p.charP,
		compP:   p.compP,
		skillP:  p.skillP,
		validP:  p.validP,
		guildP:  p.guildP,
		inviteP: inviteP,
		csP:     p.csP,
	}
}

func (p *ProcessorImpl) WithCashshopProcessor(csP cashshop.Processor) Processor {
	return &ProcessorImpl{
		l:       p.l,
		ctx:     p.ctx,
		t:       p.t,
		comp:    p.comp.WithCashshopProcessor(csP),
		handle:  p.handle.WithCashshopProcessor(csP),
		charP:   p.charP,
		compP:   p.compP,
		skillP:  p.skillP,
		validP:  p.validP,
		guildP:  p.guildP,
		inviteP: p.inviteP,
		csP:     csP,
	}
}

// GetAll returns all sagas for the current tenant
func (p *ProcessorImpl) GetAll() ([]Saga, error) {
	return p.AllProvider()()
}

func (p *ProcessorImpl) AllProvider() model.Provider[[]Saga] {
	return func() ([]Saga, error) {
		return GetCache().GetAll(p.ctx), nil
	}
}

// GetById returns a saga by its transaction ID for the current tenant
func (p *ProcessorImpl) GetById(transactionId uuid.UUID) (Saga, error) {
	return p.ByIdProvider(transactionId)()
}

func (p *ProcessorImpl) ByIdProvider(transactionId uuid.UUID) model.Provider[Saga] {
	return func() (Saga, error) {
		m, ok := GetCache().GetById(p.ctx, transactionId)
		if !ok {
			return Saga{}, errors.New("saga not found")
		}
		return m, nil
	}
}

// Put adds or updates a saga in the cache for the current tenant
func (p *ProcessorImpl) Put(saga Saga) error {
	p.l.WithFields(logrus.Fields{
		"transaction_id": saga.TransactionId().String(),
		"saga_type":      saga.SagaType(),
		"tenant_id":      p.t.Id().String(),
	}).Debug("Inserting saga into cache")

	// Validate state consistency before inserting
	if err := saga.ValidateStateConsistency(); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": saga.TransactionId().String(),
			"saga_type":      saga.SagaType(),
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("State consistency validation failed before inserting saga")
		return err
	}

	if err := GetCache().Put(p.ctx, saga); err != nil {
		return err
	}

	// Arm the per-saga timeout backstop. A saga that never produces a
	// terminal-state transition (all downstream services wedged / timed out)
	// will trip this timer and emit Failed with ErrorCodeSagaTimeout.
	// See PRD §4.1 / plan Phase 4. The timer is cancelled on every normal
	// terminal transition below.
	SagaTimers().Schedule(p.l, p.t, saga.TransactionId(), saga.Timeout())

	p.l.WithFields(logrus.Fields{
		"transaction_id": saga.TransactionId().String(),
		"saga_type":      saga.SagaType(),
		"timeout":        saga.Timeout().String(),
		"tenant_id":      p.t.Id().String(),
	}).Debug("Saga inserted into cache")

	return p.Step(saga.TransactionId())
}

// AtomicUpdateSaga performs an atomic update of saga state with consistency validation
func (p *ProcessorImpl) AtomicUpdateSaga(transactionId uuid.UUID, updateFunc func(*Saga) error) error {
	s, err := p.GetById(transactionId)
	if err != nil {
		return err
	}

	// Create a copy for safe modification
	sagaCopy := s

	// Apply the update function
	if err := updateFunc(&sagaCopy); err != nil {
		return err
	}

	// Validate state consistency after update
	if err := sagaCopy.ValidateStateConsistency(); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": transactionId.String(),
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("State consistency validation failed in atomic update")
		return err
	}

	// Update the cache atomically
	return GetCache().Put(p.ctx, sagaCopy)
}

// SafeSetStepStatus safely updates step status with validation and logging
func (p *ProcessorImpl) SafeSetStepStatus(s *Saga, stepIndex int, status Status, operation string) error {
	updated, err := s.WithStepStatus(stepIndex, status)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_index":     stepIndex,
			"status":         status,
			"operation":      operation,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("Failed to set step status safely")
		return err
	}
	*s = updated

	// Validate state consistency after status change
	if err := s.ValidateStateConsistency(); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"operation":      operation,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("State consistency validation failed after safe status update")
		return err
	}

	return nil
}

func (p *ProcessorImpl) StepCompleted(transactionId uuid.UUID, success bool) error {
	return p.StepCompletedWithResult(transactionId, success, nil)
}

func (p *ProcessorImpl) StepCompletedWithResult(transactionId uuid.UUID, success bool, result map[string]any) error {
	for attempt := 1; attempt <= maxConflictRetries; attempt++ {
		err := p.stepCompletedWithResultOnce(transactionId, success, result)
		if err == nil {
			return nil
		}
		if !isVersionConflict(err) {
			return err
		}
		p.l.WithFields(logrus.Fields{
			"transaction_id": transactionId.String(),
			"attempt":        attempt,
			"tenant_id":      p.t.Id().String(),
		}).Warn("Version conflict in StepCompletedWithResult, retrying.")
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("max retries exceeded for step completion on saga %s", transactionId.String())
}

// AcceptEvent is the single gate at which a saga-tagged Kafka event is
// matched against the saga's pending step. See PRD §4 and plan Task 3.
func (p *ProcessorImpl) AcceptEvent(transactionId uuid.UUID, kind EventKind, opts ...AcceptOption) (AcceptDecision, bool) {
	var o acceptOptions
	for _, opt := range opts {
		opt(&o)
	}
	if transactionId == uuid.Nil {
		LogSkip(p.l, logrus.Fields{
			"event_kind": kind,
		}, SkipReasonNilTransactionId)
		return AcceptDecision{}, false
	}
	s, err := p.GetById(transactionId)
	if err != nil {
		LogSkip(p.l, logrus.Fields{
			"transaction_id": transactionId.String(),
			"event_kind":     kind,
		}, SkipReasonSagaNotFound)
		return AcceptDecision{}, false
	}
	// Terminal lifecycle states are absorbing (PRD §4.1): a saga the timer
	// or a failure path has already moved to compensating/failed/completed
	// can never be advanced by a late step event. A cache miss on
	// GetLifecycle (hard-deleted in-memory entry racing GetById) falls
	// through to the existing checks.
	if lc, ok := GetCache().GetLifecycle(p.ctx, transactionId); ok && lc != SagaLifecyclePending {
		p.absorbLateTerminalEvent(s, lc, kind)
		return AcceptDecision{}, false
	}
	step, ok := s.GetCurrentStep()
	if !ok {
		LogSkip(p.l, logrus.Fields{
			"transaction_id": transactionId.String(),
			"event_kind":     kind,
		}, SkipReasonNoPendingStep)
		p.maybeWarnUnmatchedEvent(s, kind)
		return AcceptDecision{}, false
	}
	if !StepAcceptsEvent(step.Action(), kind) {
		LogSkip(p.l, logrus.Fields{
			"transaction_id": transactionId.String(),
			"step_id":        step.StepId(),
			"step_action":    step.Action(),
			"event_kind":     kind,
		}, SkipReasonActionMismatch)
		p.maybeWarnUnmatchedEvent(s, kind)
		return AcceptDecision{}, false
	}
	// Character-id guard (FR-1.3). Runs last, after the action/kind gate, so
	// a mismatch is reported as its own reason rather than being masked by
	// action_mismatch. maybeWarnUnmatchedEvent is deliberately NOT called
	// here: a cross-character map_changed is expected traffic under the
	// party-quest fan-out, not an anomaly.
	if o.hasCharacterId {
		if want := ExtractCharacterId(step); want != 0 && want != o.characterId {
			LogSkip(p.l, logrus.Fields{
				"transaction_id":     transactionId.String(),
				"step_id":            step.StepId(),
				"step_action":        step.Action(),
				"event_kind":         kind,
				"event_character_id": o.characterId,
				"step_character_id":  want,
			}, SkipReasonCharacterIdMismatch)
			return AcceptDecision{}, false
		}
	}
	return AcceptDecision{Saga: s, Step: step}, true
}

// maybeWarnUnmatchedEvent emits a warn log (once per (transactionId, kind)) when
// a saga-tagged event arrives but no pending step in the saga accepts the
// event kind. If any pending step does accept the kind, the event is
// considered merely out-of-order and no warn is emitted. See PRD §4 and
// plan Task 17.
func (p *ProcessorImpl) maybeWarnUnmatchedEvent(s Saga, kind EventKind) {
	for _, st := range s.Steps() {
		if st.Status() != Pending {
			continue
		}
		if StepAcceptsEvent(st.Action(), kind) {
			return
		}
	}
	key := s.TransactionId().String() + "|" + string(kind)
	if _, loaded := unmatchedEventWarnOnce.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"event_kind":     kind,
		"reason":         SkipReasonUnmatchedEvent,
	}).Warn("Saga event arrived but no pending step accepts it.")
}

// absorbLateTerminalEvent handles a step event that arrived after the saga's
// lifecycle went terminal. The event never advances the saga; a success-
// outcome event for a compensable in-flight step is routed into single-step
// compensation so its real side effect is rolled back (PRD §4.2/§4.3).
func (p *ProcessorImpl) absorbLateTerminalEvent(s Saga, lc SagaLifecycleState, kind EventKind) {
	// Unknown kinds default to failure: never dispatch a rollback for an
	// effect we cannot classify.
	outcome := OutcomeFailure
	if o, ok := EventOutcomeOf(kind); ok {
		outcome = o
	}
	// Steps dispatch serially, so the earliest pending step IS the in-flight
	// one; match it exactly as the happy path would (design §3.2).
	step, stepOk := s.GetCurrentStep()
	matched := stepOk && StepAcceptsEvent(step.Action(), kind)
	p.absorbLateTerminal(s, lc, string(kind), outcome, step, matched)
}

// absorbLateTerminal is the shared core for the AcceptEvent fast-path gate
// and the stepCompletedWithResultOnce commit-time gate.
func (p *ProcessorImpl) absorbLateTerminal(s Saga, lc SagaLifecycleState, eventKind string, outcome EventOutcome, step Step[any], matched bool) {
	fields := logrus.Fields{
		"transaction_id":  s.TransactionId().String(),
		"event_kind":      eventKind,
		"lifecycle_state": string(lc),
		"saga_type":       s.SagaType(),
		"tenant_id":       p.t.Id().String(),
	}
	if matched {
		fields["step_id"] = step.StepId()
	}
	LogSkip(p.l, fields, SkipReasonSagaTerminal)

	compensated := false
	if matched && outcome == OutcomeSuccess {
		var err error
		compensated, err = p.comp.CompensateLateStep(s, step)
		if err != nil {
			p.l.WithFields(fields).WithError(err).Error("Late-success compensation failed.")
		}
	}

	// task-040 span-metrics pipeline: the counter is
	// traces_spanmetrics_calls_total{span_name="saga.late_event_absorbed"}.
	// transaction.id is on the forbidden-attribute list; it lives in the log
	// line above instead.
	_, span := otel.GetTracerProvider().Tracer("atlas-saga-orchestrator").Start(p.ctx, "saga.late_event_absorbed")
	span.SetAttributes(
		attribute.String("tenant.id", p.t.Id().String()),
		attribute.String("saga.type", string(s.SagaType())),
		attribute.String("saga.lifecycle_state", string(lc)),
		attribute.String("late.outcome", string(outcome)),
		attribute.Bool("late.compensated", compensated),
	)
	span.End()
}

func (p *ProcessorImpl) stepCompletedWithResultOnce(transactionId uuid.UUID, success bool, result map[string]any) error {
	s, err := p.GetById(transactionId)
	if err != nil {
		return nil
	}

	// Commit-time terminal gate (design §3.3a): AcceptEvent's fast-path
	// check can race the timeout transition. This guards the only function
	// that performs the forward write; the TryTransition version bump
	// (store.go) forces any concurrent optimistic writer back through here
	// via VersionConflictError retry. Outcome comes from the caller's
	// success flag — no kind table needed on this path.
	if lc, ok := GetCache().GetLifecycle(p.ctx, transactionId); ok && lc != SagaLifecyclePending {
		outcome := OutcomeFailure
		if success {
			outcome = OutcomeSuccess
		}
		step, stepOk := s.GetCurrentStep()
		p.absorbLateTerminal(s, lc, "step_completed", outcome, step, stepOk)
		return nil
	}

	// Idempotency guard: if there are no pending steps and the saga is not failing,
	// this is a duplicate event — the saga already advanced past this point.
	if !s.Failing() && s.FindEarliestPendingStepIndex() == -1 {
		p.l.WithFields(logrus.Fields{
			"transaction_id": transactionId.String(),
			"tenant_id":      p.t.Id().String(),
		}).Debug("Duplicate step completion received — saga has no pending steps, ignoring.")
		return nil
	}

	// Terminal-state guard: the first failure-signal takes Pending → Compensating.
	// Later callers (late StepCompleted(false), timer) see the transition fail and
	// short-circuit, which prevents duplicate Failed emissions in Phase 3+. See PRD §4.7.
	if !success && !s.Failing() {
		if !GetCache().TryTransition(p.ctx, transactionId, SagaLifecyclePending, SagaLifecycleCompensating) {
			p.l.WithFields(logrus.Fields{
				"transaction_id": transactionId.String(),
				"saga_type":      s.SagaType(),
				"tenant_id":      p.t.Id().String(),
			}).Info("saga already terminal, late completion ignored")
			return nil
		}
		// Cancel the Phase-4 timeout backstop — we have taken over failure handling.
		SagaTimers().Cancel(transactionId)
	}

	if s.Failing() {
		err = p.MarkFurthestCompletedStepFailed(transactionId)
		if err != nil {
			return err
		}
	} else {
		status := Failed
		if success {
			status = Completed
		}

		err = p.MarkEarliestPendingStepWithResult(transactionId, status, result)
		if err != nil {
			return err
		}
	}
	return p.Step(transactionId)
}

// MarkFurthestCompletedStepFailed marks the furthest completed step as failed
func (p *ProcessorImpl) MarkFurthestCompletedStepFailed(transactionId uuid.UUID) error {
	s, err := p.GetById(transactionId)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": transactionId.String(),
			"tenant_id":      p.t.Id().String(),
		}).Debug("Unable to locate saga for marking furthest completed step as failed.")
		return err
	}

	// Find the furthest completed step (last one with status "completed")
	furthestCompletedIndex := s.FindFurthestCompletedStepIndex()

	// If no completed step was found, return an error
	if furthestCompletedIndex == -1 {
		return nil
	}

	// Mark the step as failed with validation
	s, err = s.WithStepStatus(furthestCompletedIndex, Failed)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_index":     furthestCompletedIndex,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("Failed to set step status to failed")
		return err
	}

	// Validate state consistency before updating cache
	if err := s.ValidateStateConsistency(); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("State consistency validation failed after marking step as failed")
		return err
	}

	// Update the saga in the cache
	if err := GetCache().Put(p.ctx, s); err != nil {
		return err
	}

	step, _ := s.StepAt(furthestCompletedIndex)
	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        step.StepId(),
		"tenant_id":      p.t.Id().String(),
	}).Debug("Marked furthest completed step as failed.")

	return nil
}

// MarkEarliestPendingStepCompleted marks the earliest pending step as completed
func (p *ProcessorImpl) MarkEarliestPendingStepCompleted(transactionId uuid.UUID) error {
	return p.MarkEarliestPendingStep(transactionId, Completed)
}

// MarkEarliestPendingStep marks the earliest pending step
func (p *ProcessorImpl) MarkEarliestPendingStep(transactionId uuid.UUID, status Status) error {
	s, err := p.GetById(transactionId)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": transactionId.String(),
			"tenant_id":      p.t.Id().String(),
		}).Debugf("Unable to locate saga for marking earliest pending step as [%s].", status)
		return err
	}

	// Find the earliest pending step (first one with status "pending")
	earliestPendingIndex := s.FindEarliestPendingStepIndex()

	// If no pending step was found, return an error
	if earliestPendingIndex == -1 {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"tenant_id":      p.t.Id().String(),
		}).Debugf("No pending steps found to mark as [%s].", status)
		return errors.New("no pending steps found")
	}

	// Mark the step with validation
	s, err = s.WithStepStatus(earliestPendingIndex, status)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_index":     earliestPendingIndex,
			"status":         status,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("Failed to set step status")
		return err
	}

	// Validate state consistency before updating cache
	if err := s.ValidateStateConsistency(); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("State consistency validation failed after marking step")
		return err
	}

	// Update the saga in the cache
	if err := GetCache().Put(p.ctx, s); err != nil {
		return err
	}

	step, _ := s.StepAt(earliestPendingIndex)
	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        step.StepId(),
		"tenant_id":      p.t.Id().String(),
	}).Debugf("Marked earliest pending step as [%s].", status)

	return nil
}

// MarkEarliestPendingStepWithResult marks the earliest pending step and attaches a result
func (p *ProcessorImpl) MarkEarliestPendingStepWithResult(transactionId uuid.UUID, status Status, result map[string]any) error {
	s, err := p.GetById(transactionId)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": transactionId.String(),
			"tenant_id":      p.t.Id().String(),
		}).Debugf("Unable to locate saga for marking earliest pending step as [%s].", status)
		return err
	}

	// Find the earliest pending step (first one with status "pending")
	earliestPendingIndex := s.FindEarliestPendingStepIndex()

	// If no pending step was found, return an error
	if earliestPendingIndex == -1 {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"tenant_id":      p.t.Id().String(),
		}).Debugf("No pending steps found to mark as [%s].", status)
		return errors.New("no pending steps found")
	}

	// Mark the step with validation
	s, err = s.WithStepStatus(earliestPendingIndex, status)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_index":     earliestPendingIndex,
			"status":         status,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("Failed to set step status")
		return err
	}

	// Attach result if provided
	if result != nil {
		s, err = s.WithStepResult(earliestPendingIndex, result)
		if err != nil {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"step_index":     earliestPendingIndex,
				"tenant_id":      p.t.Id().String(),
			}).WithError(err).Error("Failed to set step result")
			return err
		}
	}

	// Forward result to subsequent step payloads for CharacterCreation sagas
	if s.SagaType() == CharacterCreation && status == Completed {
		s = forwardCharacterCreationResult(p.l, s)
	}

	// Validate state consistency before updating cache
	if err := s.ValidateStateConsistency(); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("State consistency validation failed after marking step")
		return err
	}

	// Update the saga in the cache
	if err := GetCache().Put(p.ctx, s); err != nil {
		return err
	}

	step, _ := s.StepAt(earliestPendingIndex)
	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        step.StepId(),
		"tenant_id":      p.t.Id().String(),
	}).Debugf("Marked earliest pending step as [%s].", status)

	return nil
}

// AddStep adds a new step to the saga with proper ordering and transaction management
func (p *ProcessorImpl) AddStep(transactionId uuid.UUID, step Step[any]) error {
	s, err := p.GetById(transactionId)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": transactionId.String(),
			"tenant_id":      p.t.Id().String(),
		}).Debug("Unable to locate saga for adding step.")
		return err
	}

	// Validate that the saga is in a valid state for adding steps
	if s.Failing() {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"tenant_id":      p.t.Id().String(),
		}).Debug("Cannot add step to a failing saga.")
		return errors.New("cannot add step to a failing saga")
	}

	// Find the index of the current step (earliest pending step)
	currentStepIndex := s.FindEarliestPendingStepIndex()
	if currentStepIndex == -1 {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"tenant_id":      p.t.Id().String(),
		}).Debug("No pending steps found to add step after.")
		return errors.New("no pending steps found")
	}

	// Validate step ID uniqueness within the saga
	for _, existingStep := range s.Steps() {
		if existingStep.StepId() == step.StepId() {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"step_id":        step.StepId(),
				"tenant_id":      p.t.Id().String(),
			}).Debug("Step ID already exists in saga.")
			return fmt.Errorf("step ID '%s' already exists in saga", step.StepId())
		}
	}

	// Insert the new step right after the current step to maintain proper ordering
	insertIndex := currentStepIndex + 1

	// Use WithStepAfterIndex to add the step at the right position
	s, err = s.WithStepAfterIndex(currentStepIndex, step)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_id":        step.StepId(),
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("Failed to add step to saga")
		return err
	}

	// Validate comprehensive state consistency after insertion
	if err := s.ValidateStateConsistency(); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_id":        step.StepId(),
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("State consistency validation failed after step insertion")
		return err
	}

	// Update the saga in the cache atomically
	if err := GetCache().Put(p.ctx, s); err != nil {
		return err
	}

	p.l.WithFields(logrus.Fields{
		"transaction_id":  s.TransactionId().String(),
		"saga_type":       s.SagaType(),
		"step_id":         step.StepId(),
		"action":          step.Action(),
		"insert_index":    insertIndex,
		"total_steps":     s.StepCount(),
		"completed_steps": s.GetCompletedStepCount(),
		"pending_steps":   s.GetPendingStepCount(),
		"tenant_id":       p.t.Id().String(),
	}).Debug("Added new step to saga with proper ordering.")

	return nil
}

// AddStepAfterCurrent adds a new step to the saga's step list after the current step.
func (p *ProcessorImpl) AddStepAfterCurrent(transactionId uuid.UUID, step Step[any]) error {
	s, err := p.GetById(transactionId)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": transactionId.String(),
			"tenant_id":      p.t.Id().String(),
		}).Debug("Unable to locate saga for prepending step.")
		return err
	}

	// Validate that the saga is in a valid state for adding steps
	if s.Failing() {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"tenant_id":      p.t.Id().String(),
		}).Debug("Cannot prepend step to a failing saga.")
		return errors.New("cannot prepend step to a failing saga")
	}

	// Validate step ID uniqueness within the saga
	for _, existingStep := range s.Steps() {
		if existingStep.StepId() == step.StepId() {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"step_id":        step.StepId(),
				"tenant_id":      p.t.Id().String(),
			}).Debug("Step ID already exists in saga.")
			return fmt.Errorf("step ID '%s' already exists in saga", step.StepId())
		}
	}

	// Find the first pending step and add after it
	for i, st := range s.Steps() {
		if st.Status() == Pending {
			s, err = s.WithStepAfterIndex(i, step)
			if err != nil {
				p.l.WithFields(logrus.Fields{
					"transaction_id": s.TransactionId().String(),
					"saga_type":      s.SagaType(),
					"step_id":        step.StepId(),
					"tenant_id":      p.t.Id().String(),
				}).WithError(err).Error("Failed to add step after current")
				return err
			}
			break
		}
	}

	// Validate comprehensive state consistency after insertion
	if err := s.ValidateStateConsistency(); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_id":        step.StepId(),
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("State consistency validation failed after step prepend")
		return err
	}

	// Update the saga in the cache atomically
	if err := GetCache().Put(p.ctx, s); err != nil {
		return err
	}

	p.l.WithFields(logrus.Fields{
		"transaction_id":  s.TransactionId().String(),
		"saga_type":       s.SagaType(),
		"step_id":         step.StepId(),
		"action":          step.Action(),
		"insert_index":    0,
		"total_steps":     s.StepCount(),
		"completed_steps": s.GetCompletedStepCount(),
		"pending_steps":   s.GetPendingStepCount(),
		"tenant_id":       p.t.Id().String(),
	}).Debug("Prepended new step to saga at the beginning.")

	return nil
}

func (p *ProcessorImpl) Step(transactionId uuid.UUID) error {
	s, err := p.GetById(transactionId)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": transactionId.String(),
			"tenant_id":      p.t.Id().String(),
		}).Debug("Unable to locate saga being stepped.")
		return err
	}

	if s.Failing() {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"tenant_id":      p.t.Id().String(),
		}).Debug("Reverting saga step.")
		return p.comp.CompensateFailedStep(s)
	}

	st, ok := s.GetCurrentStep()
	if !ok {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"tenant_id":      p.t.Id().String(),
		}).Debug("No steps remaining to progress.")

		// Terminal-state guard: only the first observer of the success-terminal
		// state gets to emit Completed. A late observer (e.g., re-entry after
		// another goroutine already handled completion) sees the transition fail
		// and returns silently.
		if !GetCache().TryTransition(p.ctx, s.TransactionId(), SagaLifecyclePending, SagaLifecycleCompleted) {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"tenant_id":      p.t.Id().String(),
			}).Info("saga already terminal, completion emission skipped")
			return nil
		}

		// Cancel the Phase-4 timeout backstop — normal terminal, no timer needed.
		SagaTimers().Cancel(s.TransactionId())
		GetCache().Remove(p.ctx, s.TransactionId())

		// Emit saga completion event
		err := producer.ProviderImpl(p.l)(p.ctx)(saga.EnvStatusEventTopic)(CompletedStatusEventProvider(s))
		if err != nil {
			p.l.WithError(err).WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"tenant_id":      p.t.Id().String(),
			}).Error("Failed to emit saga completion event.")
		}

		return nil
	}

	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"tenant_id":      p.t.Id().String(),
	}).Debugf("Progressing saga step [%s].", st.StepId())

	// Check if this is a high-level action that needs expansion
	if isExpandableAction(st.Action()) {
		p.l.Debugf("Expanding high-level action [%s] into concrete steps", st.Action())
		err := p.expandAndProcessStep(s, st)
		if err != nil {
			p.l.WithError(err).WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"step_id":        st.StepId(),
				"action":         st.Action(),
				"tenant_id":      p.t.Id().String(),
			}).Error("Failed to expand saga step, marking as failed")

			// Mark the step as failed and trigger compensation
			markErr := p.MarkEarliestPendingStep(s.TransactionId(), Failed)
			if markErr != nil {
				p.l.WithError(markErr).Error("Failed to mark expansion error step as failed")
				return markErr
			}

			// Trigger the next step (which will start compensation)
			return p.Step(s.TransactionId())
		}
		return nil
	}

	// Get the handler for this action type
	handler, exists := p.handle.GetHandler(st.Action())
	if !exists {
		unknownErr := fmt.Errorf("unknown action type: %s", st.Action())
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_id":        st.StepId(),
			"action":         st.Action(),
			"tenant_id":      p.t.Id().String(),
		}).Error("Unknown action type encountered. Saga cannot be processed.")
		p.emitFailedFromStepSyncError(s, st, unknownErr)
		return unknownErr
	}

	// Execute the handler. A synchronous error here would otherwise travel back
	// to the Kafka consumer and be dropped (PRD §4.2 / plan Phase 3.2). Take the
	// terminal-state guard and emit StatusEventTypeFailed before propagating the
	// error up (propagation preserved so the Kafka consumer can log with context).
	handlerErr := handler(s, st)
	if handlerErr != nil {
		p.l.WithError(handlerErr).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_id":        st.StepId(),
			"action":         st.Action(),
			"tenant_id":      p.t.Id().String(),
		}).Error("Saga step handler returned a synchronous error.")
		p.emitFailedFromStepSyncError(s, st, handlerErr)
	}
	return handlerErr
}

// emitFailedFromStepSyncError takes the terminal-state guard and emits Failed
// for a synchronous step-handler error (PRD §4.2 / plan Phase 3.2). A loser
// (state already non-Pending) logs and returns without emitting.
//
// The saga is intentionally NOT evicted from the cache here: existing per-step
// compensation flows (compensateEquipAsset, compensateCreateAndEquipAsset, etc.)
// still run for non-character-creation sagas, and the Phase-6 reverse-walk
// evicts for character-creation sagas.
func (p *ProcessorImpl) emitFailedFromStepSyncError(s Saga, st Step[any], cause error) {
	if !GetCache().TryTransition(p.ctx, s.TransactionId(), SagaLifecyclePending, SagaLifecycleCompensating) {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"step_id":        st.StepId(),
			"tenant_id":      p.t.Id().String(),
		}).Info("saga already terminal, step sync-error emission skipped")
		return
	}
	// Cancel the Phase-4 timeout backstop — we have taken over failure handling.
	SagaTimers().Cancel(s.TransactionId())

	emitErr := EmitSagaFailed(p.l, p.ctx, s, saga.ErrorCodeUnknown, cause.Error(), st.StepId())
	if emitErr != nil {
		p.l.WithError(emitErr).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"step_id":        st.StepId(),
			"tenant_id":      p.t.Id().String(),
		}).Error("Failed to emit saga failed event on step sync-error.")
	}
}

// isExpandableAction reports whether an action is a high-level composite that
// must be routed to expandAndProcessStep rather than handled atomically via
// GetHandler. This MUST stay in sync with the switch in expandAndProcessStep:
// any action with an expand* case there must be listed here, or the step falls
// through to GetHandler and fails with "unknown action type" at runtime (the
// MTS composites hit exactly that gap — the expansion existed but this gate
// didn't list them, and unit tests that called expand* directly never exercised
// the gate). TestIsExpandableActionCoversExpansionSwitch pins the two in sync.
func isExpandableAction(a Action) bool {
	switch a {
	case TransferToStorage, WithdrawFromStorage,
		TransferToCashShop, WithdrawFromCashShop,
		TransferToMts, WithdrawFromMts, MtsSettlePurchase,
		TradeSettlement, TransferToTrade, TradeUnwind:
		return true
	default:
		return false
	}
}

// expandAndProcessStep expands high-level actions into concrete steps and processes them
func (p *ProcessorImpl) expandAndProcessStep(s Saga, st Step[any]) error {
	// Find the index of the current step
	currentStepIndex := -1
	for i, step := range s.Steps() {
		if step.StepId() == st.StepId() {
			currentStepIndex = i
			break
		}
	}

	if currentStepIndex == -1 {
		return fmt.Errorf("could not find current step [%s] in saga", st.StepId())
	}

	var newSteps []Step[any]
	var err error

	switch st.Action() {
	case TransferToStorage:
		newSteps, err = p.expandTransferToStorage(st)
	case WithdrawFromStorage:
		newSteps, err = p.expandWithdrawFromStorage(st)
	case TransferToCashShop:
		newSteps, err = p.expandTransferToCashShop(st)
	case WithdrawFromCashShop:
		newSteps, err = p.expandWithdrawFromCashShop(st)
	case TransferToMts:
		newSteps, err = p.expandTransferToMts(st)
	case WithdrawFromMts:
		newSteps, err = p.expandWithdrawFromMts(st)
	case MtsSettlePurchase:
		newSteps, err = p.expandMtsSettlePurchase(st)
	case TradeSettlement:
		newSteps, err = p.expandTradeSettlement(st)
	case TransferToTrade:
		newSteps, err = p.expandTransferToTrade(st)
	case TradeUnwind:
		newSteps, err = p.expandTradeUnwind(st)
	default:
		return fmt.Errorf("unknown high-level action for expansion: %s", st.Action())
	}

	if err != nil {
		return err
	}

	// Replace the high-level step with expanded concrete steps
	err = p.AtomicUpdateSaga(s.TransactionId(), func(saga *Saga) error {
		// Remove the high-level step and insert the expanded steps
		// Build a new steps slice: steps before current + new steps + steps after current
		existingSteps := saga.Steps()
		updatedSteps := make([]Step[any], 0, len(existingSteps)-1+len(newSteps))
		updatedSteps = append(updatedSteps, existingSteps[:currentStepIndex]...)
		updatedSteps = append(updatedSteps, newSteps...)
		updatedSteps = append(updatedSteps, existingSteps[currentStepIndex+1:]...)

		// Rebuild the saga with all the new steps
		var updated Saga
		var stepErr error
		updated = *saga
		// Clear steps and add each one
		builder := NewBuilder().
			SetTransactionId(saga.TransactionId()).
			SetSagaType(saga.SagaType()).
			SetInitiatedBy(saga.InitiatedBy())
		for _, step := range updatedSteps {
			builder = builder.AddStep(step.StepId(), step.Status(), step.Action(), step.Payload())
		}
		updated, stepErr = builder.Build()
		if stepErr != nil {
			return stepErr
		}
		*saga = updated
		return nil
	})
	if err != nil {
		return err
	}

	p.l.Debugf("Expanded step [%s] into %d concrete steps", st.StepId(), len(newSteps))

	// Recursively process the first expanded step
	return p.Step(s.TransactionId())
}

// expandTransferToStorage expands TransferToStorage into ReleaseFromCharacter + AcceptToStorage
func (p *ProcessorImpl) expandTransferToStorage(st Step[any]) ([]Step[any], error) {
	payload, ok := st.Payload().(TransferToStoragePayload)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for TransferToStorage")
	}

	// Lookup the source asset from character inventory
	p.l.Debugf("Looking up source asset for character [%d] inventory [%d] slot [%d]",
		payload.CharacterId, payload.SourceInventoryType, payload.SourceSlot)

	comp, err := compartment.RequestCompartment(p.l, p.ctx)(payload.CharacterId, payload.SourceInventoryType)
	if err != nil {
		return nil, fmt.Errorf("unable to lookup character [%d] inventory compartment: %w", payload.CharacterId, err)
	}

	// Find the asset at the source slot
	var foundAsset *compartment.AssetRestModel
	for i := range comp.Assets {
		if comp.Assets[i].Slot == payload.SourceSlot {
			foundAsset = &comp.Assets[i]
			break
		}
	}

	if foundAsset == nil {
		return nil, fmt.Errorf("no asset found at slot [%d] in character [%d] inventory [%d]",
			payload.SourceSlot, payload.CharacterId, payload.SourceInventoryType)
	}

	p.l.Debugf("Found source asset template [%d] at slot [%d]", foundAsset.TemplateId, foundAsset.Slot)

	// Convert string ID to uint32 for assetId
	var assetId uint32
	fmt.Sscanf(foundAsset.Id, "%d", &assetId)

	// Resolve quantity: if payload.Quantity is 0 (meaning "take all"), use the asset's quantity
	actualQuantity := payload.Quantity
	if actualQuantity == 0 && foundAsset.Quantity > 0 {
		actualQuantity = foundAsset.Quantity
		p.l.Debugf("Resolved quantity from asset: %d", actualQuantity)
	}

	// Build AssetData from flat REST model
	assetData := assetDataFromCompartmentAsset(foundAsset)
	// Override quantity with the resolved actual quantity. The snapshot is of the
	// whole SOURCE STACK, but accept_to_storage RECREATES the asset from it, so
	// AssetData.Quantity is what lands in storage — depositing 1 of a 200 stack
	// would otherwise release 1 and store 200. This mirrors
	// expandWithdrawFromStorage, which resolves the same way on the way back.
	assetData.Quantity = actualQuantity

	// Create expanded steps: RELEASE first (soft-delete), then ACCEPT (create in destination)
	steps := []Step[any]{
		NewStep[any](
			"release_from_character",
			Pending,
			ReleaseFromCharacter,
			ReleaseFromCharacterPayload{
				TransactionId: payload.TransactionId,
				CharacterId:   payload.CharacterId,
				InventoryType: payload.SourceInventoryType,
				AssetId:       assetId,
				Quantity:      payload.Quantity,
			},
		),
		NewStep[any](
			"accept_to_storage",
			Pending,
			AcceptToStorage,
			AcceptToStoragePayload{
				TransactionId: payload.TransactionId,
				WorldId:       payload.WorldId,
				AccountId:     payload.AccountId,
				CharacterId:   payload.CharacterId,
				TemplateId:    foundAsset.TemplateId,
				AssetData:     assetData,
			},
		),
	}

	return steps, nil
}

// expandWithdrawFromStorage expands WithdrawFromStorage into ReleaseFromStorage + AcceptToCharacter
func (p *ProcessorImpl) expandWithdrawFromStorage(st Step[any]) ([]Step[any], error) {
	payload, ok := st.Payload().(WithdrawFromStoragePayload)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for WithdrawFromStorage")
	}

	// Lookup the source asset from storage projection
	p.l.Debugf("Looking up source asset from projection for character [%d] compartment type [%d] slot [%d]",
		payload.CharacterId, payload.InventoryType, payload.SourceSlot)

	foundAsset, err := storage.RequestProjectionAsset(p.l, p.ctx)(payload.CharacterId, payload.InventoryType, payload.SourceSlot)
	if err != nil {
		return nil, fmt.Errorf("unable to lookup projection asset for character [%d] compartment [%d] slot [%d]: %w",
			payload.CharacterId, payload.InventoryType, payload.SourceSlot, err)
	}

	p.l.Debugf("Found source storage asset template [%d] quantity [%d]", foundAsset.TemplateId, foundAsset.Quantity)

	// Resolve quantity: if payload.Quantity is 0 (meaning "take all"), use the asset's quantity
	actualQuantity := payload.Quantity
	if actualQuantity == 0 && foundAsset.Quantity > 0 {
		actualQuantity = foundAsset.Quantity
		p.l.Debugf("Resolved quantity from asset: %d", actualQuantity)
	}

	// Build AssetData from flat storage projection model
	assetData := assetDataFromStorageProjectionAsset(&foundAsset)
	// Override quantity with the resolved actual quantity
	assetData.Quantity = actualQuantity

	// Create expanded steps: RELEASE first (soft-delete), then ACCEPT (create in destination)
	steps := []Step[any]{
		NewStep[any](
			"release_from_storage",
			Pending,
			ReleaseFromStorage,
			ReleaseFromStoragePayload{
				TransactionId: payload.TransactionId,
				WorldId:       payload.WorldId,
				AccountId:     payload.AccountId,
				CharacterId:   payload.CharacterId,
				AssetId:       foundAsset.Id,
				Quantity:      payload.Quantity,
			},
		),
		NewStep[any](
			"accept_to_character",
			Pending,
			AcceptToCharacter,
			AcceptToCharacterPayload{
				TransactionId: payload.TransactionId,
				CharacterId:   payload.CharacterId,
				InventoryType: payload.InventoryType,
				TemplateId:    foundAsset.TemplateId,
				AssetData:     assetData,
			},
		),
	}

	return steps, nil
}

// expandTransferToTrade expands the task-205 transfer_to_trade composite into
// release_from_character + accept_to_trade — a staged item genuinely leaving
// its owner's compartment for atlas-trades' escrow custody (design §5A.4).
//
// Order is not cosmetic. The release is what makes atlas-inventory publish the
// asset deletion, which atlas-channel turns into an INVENTORY_OPERATION whose
// leading exclRequestSent bool clears the client's m_bExclRequestSent. Nothing
// else in the trade flow clears it, so an expansion that accepted first (or
// skipped the release) would leave the trade dialog permanently unable to stage
// anything more — the defect this whole amendment exists to fix (design §5A.1).
//
// The snapshot is read here rather than carried on the composite, matching
// expandTransferToMts: a snapshot minted at submission time could disagree with
// the asset by the time the release actually runs.
func (p *ProcessorImpl) expandTransferToTrade(st Step[any]) ([]Step[any], error) {
	payload, ok := st.Payload().(TransferToTradePayload)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for TransferToTrade")
	}

	p.l.Debugf("Looking up staged asset for character [%d] inventory [%d] assetId [%d]",
		payload.CharacterId, payload.SourceInventoryType, payload.AssetId)

	comp, err := compartment.RequestCompartment(p.l, p.ctx)(payload.CharacterId, payload.SourceInventoryType)
	if err != nil {
		return nil, fmt.Errorf("unable to lookup character [%d] inventory compartment: %w", payload.CharacterId, err)
	}

	// The JSON:API id is a string. One that does not parse is SKIPPED rather than
	// coerced: an unchecked Sscanf leaves the target at zero, and zero is a
	// legitimate-looking asset id — so an unparseable id would silently match a
	// payload asking for asset 0 and stage the wrong item.
	var foundAsset *compartment.AssetRestModel
	for i := range comp.Assets {
		assetId, perr := strconv.ParseUint(comp.Assets[i].Id, 10, 32)
		if perr != nil {
			p.l.WithError(perr).Warnf("Asset id [%s] in character [%d]'s compartment is not numeric. Skipping it.", comp.Assets[i].Id, payload.CharacterId)
			continue
		}
		if uint32(assetId) == payload.AssetId {
			foundAsset = &comp.Assets[i]
			break
		}
	}

	// A missing asset means it was dropped, used or moved between the client's
	// PUT_ITEM and this expansion. Failing here refuses the stage; atlas-trades
	// turns that into ITEM_REFUSED, which unlocks the client (design §5A.6).
	if foundAsset == nil {
		return nil, fmt.Errorf("no asset found with id [%d] in character [%d] inventory [%d]",
			payload.AssetId, payload.CharacterId, payload.SourceInventoryType)
	}

	p.l.Debugf("Found staged asset template [%d] id [%s] for trade escrow", foundAsset.TemplateId, foundAsset.Id)

	steps := []Step[any]{
		NewStep[any](
			"release_from_character",
			Pending,
			ReleaseFromCharacter,
			ReleaseFromCharacterPayload{
				TransactionId: payload.TransactionId,
				CharacterId:   payload.CharacterId,
				InventoryType: payload.SourceInventoryType,
				AssetId:       payload.AssetId,
				Quantity:      payload.Quantity,
			},
		),
		NewStep[any](
			"accept_to_trade",
			Pending,
			AcceptToTrade,
			AcceptToTradePayload{
				TransactionId:       payload.TransactionId,
				EscrowId:            payload.EscrowId,
				RoomId:              payload.RoomId,
				OwnerId:             payload.CharacterId,
				TradeSlot:           payload.TradeSlot,
				SourceInventoryType: payload.SourceInventoryType,
				AssetId:             payload.AssetId,

				// Snapshot taken HERE and carried from here on. This is the last
				// point at which the asset can be read at all: the
				// release_from_character above deletes it, and it is then in
				// nobody's compartment until the trade settles or unwinds.
				//
				// Quantity is overridden with the STAGED amount from the
				// composite, never the compartment stack's — a partial stage of
				// 1-of-40 must escrow 1.
				Snapshot: assetSnapshotFromCompartmentAsset(foundAsset, payload.Quantity),
			},
		),
	}
	return steps, nil
}

// expandTradeSettlement expands the task-205 trade_settlement composite into
// the concrete two-party swap (design §5A.7).
//
// Under escrow-at-staging both sides' items are already in atlas-trades'
// custody, so this expander performs NO compartment lookups: the item snapshot
// travels on the payload, because there is no inventory row left to read. That
// also retires the whole class of mid-trade substitution checks the old
// reserve-based expander needed — an escrowed asset cannot be moved, merged,
// dropped or swapped for another instance.
//
// Step order still matters: ALL releases precede ALL accepts, so a slot freed by
// an outgoing item is available to an incoming one, and a failure in either
// side's release compensates before anything has been created.
//
// Meso is CREDIT-ONLY. The staged amount was debited when it was staged
// (design §5A.5); the tax is destroyed by crediting the receiver less than was
// escrowed. An expander that also emitted the old negative leg would charge the
// giver twice — see TestExpandTradeSettlementEmitsNoNegativeAward.
//
// Sides is a [2]TradeSettlementSide array, so "the other side" is `1-si` by
// construction. Side ORDER carries no role meaning: each side is the giver of
// its own contribution and the receiver of the other's.
func (p *ProcessorImpl) expandTradeSettlement(st Step[any]) ([]Step[any], error) {
	payload, ok := st.Payload().(TradeSettlementPayload)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for TradeSettlement")
	}
	if payload.Sides[0].CharacterId == payload.Sides[1].CharacterId {
		return nil, fmt.Errorf("trade settlement names character [%d] on both sides", payload.Sides[0].CharacterId)
	}
	// MesoDelivered is uint32 but AwardMesosPayload.Amount is int32. A value
	// above MaxInt32 would wrap on conversion and turn a credit into a debit.
	for _, side := range payload.Sides {
		if side.MesoDelivered > math.MaxInt32 {
			return nil, fmt.Errorf("trade settlement delivered meso for character [%d] exceeds int32 range (%d)", side.CharacterId, side.MesoDelivered)
		}
	}

	steps := make([]Step[any], 0)

	// 1. Every release from escrow, both sides.
	for _, side := range payload.Sides {
		for _, it := range side.Items {
			steps = append(steps, NewStep[any](
				fmt.Sprintf("release_from_trade_%d_%d", side.CharacterId, it.AssetId),
				Pending,
				ReleaseFromTrade,
				ReleaseFromTradePayload{
					TransactionId: payload.TransactionId,
					EscrowId:      it.EscrowId,
				},
			))
		}
	}

	// 2. Every accept, crossed: side 0's items go to side 1 and vice versa.
	for si := range payload.Sides {
		recipient := payload.Sides[1-si].CharacterId
		for _, it := range payload.Sides[si].Items {
			steps = append(steps, NewStep[any](
				fmt.Sprintf("accept_to_character_%d_%d", recipient, it.AssetId),
				Pending,
				AcceptToCharacter,
				AcceptToCharacterPayload{
					TransactionId: payload.TransactionId,
					CharacterId:   uint32(recipient),
					InventoryType: byte(it.InventoryType),
					TemplateId:    it.Snapshot.TemplateId,
					AssetData:     assetDataFromSnapshot(it.Snapshot),
				},
			))
		}
	}

	// 3. Meso, per side that staged any: credit the post-tax amount to the OTHER
	//    side. No debit — that already happened at stage time.
	for si, side := range payload.Sides {
		if side.MesoDelivered == 0 {
			continue
		}
		receiver := payload.Sides[1-si].CharacterId
		steps = append(steps, NewStep[any](
			fmt.Sprintf("award_mesos_credit_%d", receiver),
			Pending,
			AwardMesos,
			AwardMesosPayload{
				CharacterId: uint32(receiver),
				WorldId:     payload.WorldId,
				ChannelId:   payload.ChannelId,
				ActorId:     uint32(side.CharacterId),
				ActorType:   "CHARACTER",
				Amount:      int32(side.MesoDelivered),
			},
		))
	}

	return steps, nil
}

// expandTradeUnwind expands the task-205 trade_unwind composite: every escrowed
// item goes back to the character it came from, and every escrowed meso is
// refunded in full (design §5A.8).
//
// It is the teardown twin of expandTradeSettlement and deliberately a separate
// composite. The only difference is arithmetic — a refund is untaxed and a
// delivery is not — but folding them together would have put an "is this a
// refund?" branch inside the expander, which is exactly the kind of conditional
// that silently taxes a refund one release later.
func (p *ProcessorImpl) expandTradeUnwind(st Step[any]) ([]Step[any], error) {
	payload, ok := st.Payload().(TradeUnwindPayload)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for TradeUnwind")
	}
	for _, m := range payload.Mesos {
		if m.Amount > math.MaxInt32 {
			return nil, fmt.Errorf("trade unwind refund for character [%d] exceeds int32 range (%d)", m.CharacterId, m.Amount)
		}
	}

	steps := make([]Step[any], 0)

	// Releases first, for the same reason settlement orders them first: a
	// failure before anything has been created leaves nothing to unpick.
	for _, ui := range payload.Items {
		steps = append(steps, NewStep[any](
			fmt.Sprintf("release_from_trade_%d_%d", ui.OwnerId, ui.Item.AssetId),
			Pending,
			ReleaseFromTrade,
			ReleaseFromTradePayload{
				TransactionId: payload.TransactionId,
				EscrowId:      ui.Item.EscrowId,
			},
		))
	}

	for _, ui := range payload.Items {
		steps = append(steps, NewStep[any](
			fmt.Sprintf("accept_to_character_%d_%d", ui.OwnerId, ui.Item.AssetId),
			Pending,
			AcceptToCharacter,
			AcceptToCharacterPayload{
				TransactionId: payload.TransactionId,
				CharacterId:   uint32(ui.OwnerId),
				InventoryType: byte(ui.Item.InventoryType),
				TemplateId:    ui.Item.Snapshot.TemplateId,
				AssetData:     assetDataFromSnapshot(ui.Item.Snapshot),
			},
		))
	}

	for _, m := range payload.Mesos {
		if m.Amount == 0 {
			continue
		}
		steps = append(steps, NewStep[any](
			fmt.Sprintf("award_mesos_refund_%d", m.CharacterId),
			Pending,
			AwardMesos,
			AwardMesosPayload{
				CharacterId: uint32(m.CharacterId),
				WorldId:     m.WorldId,
				ChannelId:   m.ChannelId,
				ActorId:     uint32(m.CharacterId),
				ActorType:   "SYSTEM",
				Amount:      int32(m.Amount),
			},
		))
	}

	return steps, nil
}

// assetDataFromSnapshot rebuilds an inventory AssetData from an escrow snapshot,
// so a delivery, a refund or a staging rollback restores the asset rather than a
// bare template (FR-10.3).
//
// Expiration, CashId, Rechargeable and PetId are as load-bearing as the equip
// stats: a cash item without its serial is a different item to the client, a pet
// without its id is an empty shell, and a timed item without its expiry becomes
// permanent. The bespoke stat list this replaced carried none of the four, and
// cash items and pets are stageable (atlas-trades trade/restriction.go), so the
// loss was reachable by any player.
//
// Quantity comes from the snapshot, which holds the STAGED quantity — a partial
// stage of 1 out of 200 escrowed 1, and must deliver 1.
func assetDataFromSnapshot(s AssetSnapshot) asset2.AssetData {
	return asset2.AssetData{
		Expiration:     s.Expiration,
		Quantity:       s.Quantity,
		Owner:          s.Owner,
		Flag:           s.Flag,
		Rechargeable:   s.Rechargeable,
		Strength:       s.Strength,
		Dexterity:      s.Dexterity,
		Intelligence:   s.Intelligence,
		Luck:           s.Luck,
		Hp:             s.Hp,
		Mp:             s.Mp,
		WeaponAttack:   s.WeaponAttack,
		MagicAttack:    s.MagicAttack,
		WeaponDefense:  s.WeaponDefense,
		MagicDefense:   s.MagicDefense,
		Accuracy:       s.Accuracy,
		Avoidability:   s.Avoidability,
		Hands:          s.Hands,
		Speed:          s.Speed,
		Jump:           s.Jump,
		Slots:          s.Slots,
		LevelType:      s.LevelType,
		Level:          s.Level,
		Experience:     s.Experience,
		HammersApplied: s.HammersApplied,
		CashId:         s.CashId,
		PetId:          s.PetId,
	}
}

// expandTransferToCashShop expands TransferToCashShop into ReleaseFromCharacter + AcceptToCashShop
func (p *ProcessorImpl) expandTransferToCashShop(st Step[any]) ([]Step[any], error) {
	payload, ok := st.Payload().(TransferToCashShopPayload)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for TransferToCashShop")
	}

	// Lookup the source asset from character inventory
	p.l.Debugf("Looking up source asset for character [%d] inventory [%d] cashId [%d]",
		payload.CharacterId, payload.SourceInventoryType, payload.CashId)

	comp, err := compartment.RequestCompartment(p.l, p.ctx)(payload.CharacterId, payload.SourceInventoryType)
	if err != nil {
		return nil, fmt.Errorf("unable to lookup character [%d] inventory compartment: %w", payload.CharacterId, err)
	}

	// Find the asset with matching CashId (now a flat field on the REST model)
	var foundAsset *compartment.AssetRestModel
	for i := range comp.Assets {
		if comp.Assets[i].CashId == payload.CashId {
			foundAsset = &comp.Assets[i]
			break
		}
	}

	if foundAsset == nil {
		return nil, fmt.Errorf("no asset found with cashId [%d] in character [%d] inventory [%d]",
			payload.CashId, payload.CharacterId, payload.SourceInventoryType)
	}

	p.l.Debugf("Found source asset template [%d] with cashId [%d]", foundAsset.TemplateId, foundAsset.CashId)

	// Convert string ID to uint32 for assetId
	var assetId uint32
	fmt.Sscanf(foundAsset.Id, "%d", &assetId)

	// Get cash shop compartment to get the compartment ID
	cashComp, err := cashshop.RequestCompartment(p.l, p.ctx)(payload.AccountId, payload.CompartmentType)
	if err != nil {
		return nil, fmt.Errorf("unable to lookup cash shop compartment for account [%d] type [%d]: %w",
			payload.AccountId, payload.CompartmentType, err)
	}

	// Create expanded steps: RELEASE first (soft-delete), then ACCEPT (create in destination)
	steps := []Step[any]{
		NewStep[any](
			"release_from_character",
			Pending,
			ReleaseFromCharacter,
			ReleaseFromCharacterPayload{
				TransactionId: payload.TransactionId,
				CharacterId:   payload.CharacterId,
				InventoryType: payload.SourceInventoryType,
				AssetId:       assetId,
				Quantity:      0, // Release all (cash items don't have partial quantity)
			},
		),
		NewStep[any](
			"accept_to_cash_shop",
			Pending,
			AcceptToCashShop,
			AcceptToCashShopPayload{
				TransactionId:   payload.TransactionId,
				CharacterId:     payload.CharacterId,
				AccountId:       payload.AccountId,
				CompartmentId:   cashComp.Id,
				CompartmentType: payload.CompartmentType,
				CashId:          payload.CashId,
				TemplateId:      foundAsset.TemplateId,
				Quantity:        foundAsset.Quantity,
				CommodityId:     foundAsset.CommodityId,
				PurchasedBy:     foundAsset.PurchaseBy,
				Flag:            foundAsset.Flag,
			},
		),
	}

	return steps, nil
}

// expandWithdrawFromCashShop expands WithdrawFromCashShop into ReleaseFromCashShop + AcceptToCharacter
func (p *ProcessorImpl) expandWithdrawFromCashShop(st Step[any]) ([]Step[any], error) {
	payload, ok := st.Payload().(WithdrawFromCashShopPayload)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for WithdrawFromCashShop")
	}

	// Lookup the source item from cash shop compartment
	p.l.Debugf("Looking up source item from cash shop for account [%d] compartment type [%d] cashId [%d]",
		payload.AccountId, payload.CompartmentType, payload.CashId)

	cashComp, err := cashshop.RequestCompartment(p.l, p.ctx)(payload.AccountId, payload.CompartmentType)
	if err != nil {
		return nil, fmt.Errorf("unable to lookup cash shop compartment for account [%d] type [%d]: %w",
			payload.AccountId, payload.CompartmentType, err)
	}

	// Find the asset with the matching CashId (now a flat field)
	var foundAsset *cashshop.AssetRestModel
	for i := range cashComp.Assets {
		if uint64(cashComp.Assets[i].CashId) == payload.CashId {
			foundAsset = &cashComp.Assets[i]
			break
		}
	}

	if foundAsset == nil {
		return nil, fmt.Errorf("no item found with cashId [%d] in cash shop compartment for account [%d]",
			payload.CashId, payload.AccountId)
	}

	p.l.Debugf("Found source cash shop item template [%d] with quantity [%d] purchased by [%d]",
		foundAsset.TemplateId, foundAsset.Quantity, foundAsset.PurchasedBy)

	// Build AssetData from cashshop flat REST model for the character inventory accept
	assetData := asset2.AssetData{
		Quantity:    foundAsset.Quantity,
		Flag:        foundAsset.Flag,
		CashId:      foundAsset.CashId,
		CommodityId: foundAsset.CommodityId,
		PetId:       foundAsset.PetId,
		PurchaseBy:  foundAsset.PurchasedBy,
		Expiration:  foundAsset.Expiration,
	}

	// Create expanded steps: RELEASE first (soft-delete), then ACCEPT (create in destination)
	steps := []Step[any]{
		NewStep[any](
			"release_from_cash_shop",
			Pending,
			ReleaseFromCashShop,
			ReleaseFromCashShopPayload{
				TransactionId:   payload.TransactionId,
				CharacterId:     payload.CharacterId,
				AccountId:       payload.AccountId,
				CompartmentId:   cashComp.Id,
				CompartmentType: payload.CompartmentType,
				AssetId:         foundAsset.Id,
				CashId:          foundAsset.CashId,
				TemplateId:      foundAsset.TemplateId,
			},
		),
		NewStep[any](
			"accept_to_character",
			Pending,
			AcceptToCharacter,
			AcceptToCharacterPayload{
				TransactionId: payload.TransactionId,
				CharacterId:   payload.CharacterId,
				InventoryType: payload.InventoryType,
				TemplateId:    foundAsset.TemplateId,
				AssetData:     assetData,
			},
		),
	}

	return steps, nil
}

// expandTransferToMts expands TransferToMts into ReleaseFromCharacter + AcceptToMtsListing.
// Mirrors expandTransferToCashShop: it looks up the source asset from the character's
// inventory by AssetId, captures the full item snapshot, then builds a release step
// (item leaves inventory FIRST) followed by an accept step that carries the snapshot
// plus the seller's sale params (copied from the TransferToMtsPayload) so atlas-mts can
// CREATE the listing row. N=2 steps (record for timeout scaling, design §4.3).
func (p *ProcessorImpl) expandTransferToMts(st Step[any]) ([]Step[any], error) {
	payload, ok := st.Payload().(TransferToMtsPayload)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for TransferToMts")
	}

	p.l.Debugf("Looking up source asset for character [%d] inventory [%d] assetId [%d]",
		payload.CharacterId, payload.SourceInventoryType, payload.AssetId)

	comp, err := compartment.RequestCompartment(p.l, p.ctx)(payload.CharacterId, payload.SourceInventoryType)
	if err != nil {
		return nil, fmt.Errorf("unable to lookup character [%d] inventory compartment: %w", payload.CharacterId, err)
	}

	// Find the asset by id.
	var foundAsset *compartment.AssetRestModel
	for i := range comp.Assets {
		var assetId uint32
		fmt.Sscanf(comp.Assets[i].Id, "%d", &assetId)
		if assetId == payload.AssetId {
			foundAsset = &comp.Assets[i]
			break
		}
	}

	if foundAsset == nil {
		return nil, fmt.Errorf("no asset found with id [%d] in character [%d] inventory [%d]",
			payload.AssetId, payload.CharacterId, payload.SourceInventoryType)
	}

	p.l.Debugf("Found source asset template [%d] id [%s] for MTS listing", foundAsset.TemplateId, foundAsset.Id)

	steps := []Step[any]{
		NewStep[any](
			"release_from_character",
			Pending,
			ReleaseFromCharacter,
			ReleaseFromCharacterPayload{
				TransactionId: payload.TransactionId,
				CharacterId:   payload.CharacterId,
				InventoryType: payload.SourceInventoryType,
				AssetId:       payload.AssetId,
				Quantity:      payload.Quantity,
			},
		),
		NewStep[any](
			"accept_to_mts_listing",
			Pending,
			AcceptToMtsListing,
			AcceptToMtsListingPayload{
				TransactionId:   payload.TransactionId,
				ListingId:       payload.ListingId,
				WorldId:         payload.WorldId,
				SellerId:        payload.CharacterId,
				SellerAccountId: payload.SellerAccountId,
				SellerName:      payload.SellerName,
				SaleType:        payload.SaleType,

				// Item snapshot captured from inventory.
				TemplateId:    foundAsset.TemplateId,
				Quantity:      payload.Quantity,
				Strength:      foundAsset.Strength,
				Dexterity:     foundAsset.Dexterity,
				Intelligence:  foundAsset.Intelligence,
				Luck:          foundAsset.Luck,
				HP:            foundAsset.Hp,
				MP:            foundAsset.Mp,
				WeaponAttack:  foundAsset.WeaponAttack,
				MagicAttack:   foundAsset.MagicAttack,
				WeaponDefense: foundAsset.WeaponDefense,
				MagicDefense:  foundAsset.MagicDefense,
				Accuracy:      foundAsset.Accuracy,
				Avoidability:  foundAsset.Avoidability,
				Hands:         foundAsset.Hands,
				Speed:         foundAsset.Speed,
				Jump:          foundAsset.Jump,
				Slots:         foundAsset.Slots,
				Level:         foundAsset.Level,
				ItemExp:       foundAsset.Experience,
				Flags:         foundAsset.Flag,
				Owner:         foundAsset.Owner,

				// Sale params copied from the seller's TransferToMts payload.
				ListValue:      payload.ListValue,
				BuyNowPrice:    payload.BuyNowPrice,
				CommissionRate: payload.CommissionRate,
				Category:       payload.Category,
				SubCategory:    payload.SubCategory,
				EndsAt:         payload.EndsAt,
				MinIncrement:   payload.MinIncrement,

				// Offer link copied through so the created listing records its want-ad.
				OfferWishSerial:  payload.OfferWishSerial,
				OfferWishOwnerId: payload.OfferWishOwnerId,
			},
		),
	}

	return steps, nil
}

// expandWithdrawFromMts expands WithdrawFromMts into ReleaseFromMtsHolding + AcceptToCharacter.
// The item snapshot for the accept_to_character step is looked up from atlas-mts (the
// holding row) by HoldingId, mirroring how expandWithdrawFromCashShop reads its source
// snapshot from the cash-shop compartment. N=2 steps.
func (p *ProcessorImpl) expandWithdrawFromMts(st Step[any]) ([]Step[any], error) {
	payload, ok := st.Payload().(WithdrawFromMtsPayload)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for WithdrawFromMts")
	}

	p.l.Debugf("Looking up MTS holding [%s] for character [%d] world [%d]",
		payload.HoldingId, payload.CharacterId, payload.WorldId)

	holdings, err := mts.RequestHoldings(p.l, p.ctx)(payload.CharacterId, byte(payload.WorldId))
	if err != nil {
		return nil, fmt.Errorf("unable to lookup MTS holdings for character [%d]: %w", payload.CharacterId, err)
	}

	var found *mts.HoldingRestModel
	for i := range holdings {
		if holdings[i].Id == payload.HoldingId.String() {
			found = &holdings[i]
			break
		}
	}

	if found == nil {
		return nil, fmt.Errorf("no holding found with id [%s] for character [%d]", payload.HoldingId, payload.CharacterId)
	}

	p.l.Debugf("Found MTS holding template [%d] quantity [%d] for take-home", found.TemplateId, found.Quantity)

	// Derive the destination inventory type from the holding's item template. The
	// channel passes InventoryType=0 as an advisory placeholder (the wire carries
	// only nITCSN), and 0 matches no compartment (valid types are 1-5), so the
	// accept_to_character step would error and roll the saga back, stranding the
	// item in holding. Derive it here from the template the same way every other
	// grant path does (cash-shop withdraw, RequestCreateItem, the asset consumer).
	inventoryType, ok := inventory.TypeFromItemId(item.Id(found.TemplateId))
	if !ok {
		return nil, fmt.Errorf("unable to derive inventory type for holding template [%d] (character [%d])", found.TemplateId, payload.CharacterId)
	}

	assetData := asset2.AssetData{
		Quantity:      found.Quantity,
		Flag:          found.Flags,
		Strength:      found.Strength,
		Dexterity:     found.Dexterity,
		Intelligence:  found.Intelligence,
		Luck:          found.Luck,
		Hp:            found.HP,
		Mp:            found.MP,
		WeaponAttack:  found.WeaponAttack,
		MagicAttack:   found.MagicAttack,
		WeaponDefense: found.WeaponDefense,
		MagicDefense:  found.MagicDefense,
		Accuracy:      found.Accuracy,
		Avoidability:  found.Avoidability,
		Hands:         found.Hands,
		Speed:         found.Speed,
		Jump:          found.Jump,
		Slots:         found.Slots,
		Level:         found.Level,
		Experience:    found.ItemExp,
	}

	steps := []Step[any]{
		NewStep[any](
			"release_from_mts_holding",
			Pending,
			ReleaseFromMtsHolding,
			ReleaseFromMtsHoldingPayload{
				TransactionId: payload.TransactionId,
				HoldingId:     payload.HoldingId,
			},
		),
		NewStep[any](
			"accept_to_character",
			Pending,
			AcceptToCharacter,
			AcceptToCharacterPayload{
				TransactionId: payload.TransactionId,
				CharacterId:   payload.CharacterId,
				InventoryType: byte(inventoryType),
				TemplateId:    found.TemplateId,
				AssetData:     assetData,
			},
		),
	}

	return steps, nil
}

// expandMtsSettlePurchase expands MtsSettlePurchase into the three ordered money-mover
// steps: debit the buyer's prepaid wallet FIRST (so a mid-saga failure grants nothing),
// credit the seller's points wallet, then move the listing custody to the buyer's
// holding. Commission = markedUpPrice − listValue is never credited (the sink).
// currencyType: 3=prepaid (buyer debit), 2=points (seller credit). N=3 steps.
func (p *ProcessorImpl) expandMtsSettlePurchase(st Step[any]) ([]Step[any], error) {
	payload, ok := st.Payload().(MtsSettlePurchasePayload)
	if !ok {
		return nil, fmt.Errorf("invalid payload type for MtsSettlePurchase")
	}

	const (
		currencyTypePoints  = uint32(2)
		currencyTypePrepaid = uint32(3)
	)

	steps := []Step[any]{
		// 1. Debit buyer prepaid by the marked-up price (negative amount).
		NewStep[any](
			"award_currency_buyer",
			Pending,
			AwardCurrency,
			AwardCurrencyPayload{
				CharacterId:  payload.BuyerId,
				AccountId:    payload.BuyerAccountId,
				CurrencyType: currencyTypePrepaid,
				Amount:       -payload.MarkedUpPrice,
			},
		),
		// 2. Credit seller points by the list value (positive amount).
		NewStep[any](
			"award_currency_seller",
			Pending,
			AwardCurrency,
			AwardCurrencyPayload{
				CharacterId:  payload.SellerId,
				AccountId:    payload.SellerAccountId,
				CurrencyType: currencyTypePoints,
				Amount:       payload.ListValue,
			},
		),
		// 3. Move listing custody to the buyer's holding.
		NewStep[any](
			"mts_move_listing_to_holding",
			Pending,
			MtsMoveListingToHolding,
			MtsMoveListingToHoldingPayload{
				TransactionId: payload.TransactionId,
				ListingId:     payload.ListingId,
				BuyerId:       payload.BuyerId,
				WorldId:       payload.WorldId,
				ResultKind:    payload.ResultKind,
				Price:         payload.Price,
			},
		),
	}

	return steps, nil
}

// assetSnapshotFromCompartmentAsset flattens a compartment asset into the shared
// AssetSnapshot, overriding the stack quantity with the amount actually being
// moved (a partial stage escrows fewer than the compartment row holds).
//
// It exists because escrow-at-staging deletes the compartment row: after this
// point there is nothing left to read, so every downstream consumer — the escrow
// table, the settlement re-grant, the unwind refund, the staging compensator and
// atlas-channel's trade frame — is served from this one snapshot.
//
// The pet block is deliberately limited to PetId. compartment.AssetRestModel
// mirrors atlas-inventory's asset resource, which carries the pet's id but not
// its name, level, closeness or fullness — those live in atlas-pets and are
// joined in by whoever renders the pet. Fabricating them here would be inventing
// state, so they stay zero and the pet is identified by its id.
func assetSnapshotFromCompartmentAsset(a *compartment.AssetRestModel, quantity uint32) AssetSnapshot {
	return AssetSnapshot{
		Slot:           a.Slot,
		TemplateId:     a.TemplateId,
		Expiration:     a.Expiration,
		CashId:         a.CashId,
		Quantity:       quantity,
		Flag:           a.Flag,
		Owner:          a.Owner,
		Rechargeable:   a.Rechargeable,
		Strength:       a.Strength,
		Dexterity:      a.Dexterity,
		Intelligence:   a.Intelligence,
		Luck:           a.Luck,
		Hp:             a.Hp,
		Mp:             a.Mp,
		WeaponAttack:   a.WeaponAttack,
		MagicAttack:    a.MagicAttack,
		WeaponDefense:  a.WeaponDefense,
		MagicDefense:   a.MagicDefense,
		Accuracy:       a.Accuracy,
		Avoidability:   a.Avoidability,
		Hands:          a.Hands,
		Speed:          a.Speed,
		Jump:           a.Jump,
		Slots:          a.Slots,
		LevelType:      a.LevelType,
		Level:          a.Level,
		Experience:     a.Experience,
		HammersApplied: a.HammersApplied,
		PetId:          a.PetId,
	}
}

// assetDataFromCompartmentAsset converts a compartment AssetRestModel to an AssetData struct
func assetDataFromCompartmentAsset(a *compartment.AssetRestModel) asset2.AssetData {
	return asset2.AssetData{
		Expiration:     a.Expiration,
		CreatedAt:      a.CreatedAt,
		Quantity:       a.Quantity,
		OwnerId:        a.OwnerId,
		Owner:          a.Owner,
		Flag:           a.Flag,
		Rechargeable:   a.Rechargeable,
		Strength:       a.Strength,
		Dexterity:      a.Dexterity,
		Intelligence:   a.Intelligence,
		Luck:           a.Luck,
		Hp:             a.Hp,
		Mp:             a.Mp,
		WeaponAttack:   a.WeaponAttack,
		MagicAttack:    a.MagicAttack,
		WeaponDefense:  a.WeaponDefense,
		MagicDefense:   a.MagicDefense,
		Accuracy:       a.Accuracy,
		Avoidability:   a.Avoidability,
		Hands:          a.Hands,
		Speed:          a.Speed,
		Jump:           a.Jump,
		Slots:          a.Slots,
		Locked:         a.Locked,
		Spikes:         a.Spikes,
		KarmaUsed:      a.KarmaUsed,
		Cold:           a.Cold,
		CanBeTraded:    a.CanBeTraded,
		LevelType:      a.LevelType,
		Level:          a.Level,
		Experience:     a.Experience,
		HammersApplied: a.HammersApplied,
		EquippedSince:  a.EquippedSince,
		CashId:         a.CashId,
		CommodityId:    a.CommodityId,
		PurchaseBy:     a.PurchaseBy,
		PetId:          a.PetId,
	}
}

// forwardCharacterCreationResult extracts characterId from the completed CreateCharacter step
// and injects it into all remaining pending step payloads that have a CharacterId field.
// This enables a single unified CharacterCreation saga where subsequent steps (AwardAsset,
// CreateAndEquipAsset, CreateSkill) are submitted with characterId=0 as a sentinel value,
// then rewritten once the actual characterId is known.
func forwardCharacterCreationResult(l logrus.FieldLogger, s Saga) Saga {
	// Find the CreateCharacter step result
	var characterId uint32
	for _, step := range s.Steps() {
		if step.Action() == CreateCharacter && step.Status() == Completed && step.Result() != nil {
			characterId = extractUint32(step.Result(), "characterId")
			break
		}
	}
	if characterId == 0 {
		return s
	}

	l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"character_id":   characterId,
	}).Debug("Forwarding characterId from CreateCharacter step to pending steps.")

	// Rewrite pending step payloads with the characterId
	for i, step := range s.Steps() {
		if step.Status() != Pending {
			continue
		}
		var updated Saga
		var err error
		switch p := step.Payload().(type) {
		case AwardItemActionPayload:
			p.CharacterId = characterId
			updated, err = s.WithStepPayload(i, p)
		case CreateAndEquipAssetPayload:
			p.CharacterId = characterId
			updated, err = s.WithStepPayload(i, p)
		case CreateSkillPayload:
			p.CharacterId = characterId
			updated, err = s.WithStepPayload(i, p)
		case AwaitInventoryCreatedPayload:
			p.CharacterId = characterId
			updated, err = s.WithStepPayload(i, p)
		default:
			continue
		}
		if err != nil {
			l.WithError(err).WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"step_index":     i,
			}).Error("Failed to forward characterId to step payload")
			continue
		}
		s = updated
	}
	return s
}

// extractUint32 extracts a uint32 value from a map[string]any, handling both uint32 and float64 types
// (float64 occurs after JSON round-trip through PostgreSQL storage)
func extractUint32(m map[string]any, key string) uint32 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case uint32:
		return val
	case float64:
		return uint32(val)
	case int:
		return uint32(val)
	case int64:
		return uint32(val)
	default:
		return 0
	}
}

// assetDataFromStorageProjectionAsset converts a storage ProjectionAssetRestModel to an AssetData struct
func assetDataFromStorageProjectionAsset(a *storage.ProjectionAssetRestModel) asset2.AssetData {
	return asset2.AssetData{
		Expiration:     a.Expiration,
		Quantity:       a.Quantity,
		OwnerId:        a.OwnerId,
		Owner:          a.Owner,
		Flag:           a.Flag,
		Rechargeable:   a.Rechargeable,
		Strength:       a.Strength,
		Dexterity:      a.Dexterity,
		Intelligence:   a.Intelligence,
		Luck:           a.Luck,
		Hp:             a.Hp,
		Mp:             a.Mp,
		WeaponAttack:   a.WeaponAttack,
		MagicAttack:    a.MagicAttack,
		WeaponDefense:  a.WeaponDefense,
		MagicDefense:   a.MagicDefense,
		Accuracy:       a.Accuracy,
		Avoidability:   a.Avoidability,
		Hands:          a.Hands,
		Speed:          a.Speed,
		Jump:           a.Jump,
		Slots:          a.Slots,
		Locked:         a.Locked,
		Spikes:         a.Spikes,
		KarmaUsed:      a.KarmaUsed,
		Cold:           a.Cold,
		CanBeTraded:    a.CanBeTraded,
		LevelType:      a.LevelType,
		Level:          a.Level,
		Experience:     a.Experience,
		HammersApplied: a.HammersApplied,
		EquippedSince:  a.EquippedSince,
		CashId:         a.CashId,
		CommodityId:    a.CommodityId,
		PurchaseBy:     a.PurchaseBy,
		PetId:          a.PetId,
	}
}
