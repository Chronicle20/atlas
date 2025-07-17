package saga

import (
	"atlas-saga-orchestrator/character"
	"atlas-saga-orchestrator/compartment"
	"atlas-saga-orchestrator/guild"
	"atlas-saga-orchestrator/invite"
	"atlas-saga-orchestrator/kafka/message/saga"
	"atlas-saga-orchestrator/kafka/producer"
	"atlas-saga-orchestrator/skill"
	"atlas-saga-orchestrator/validation"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor is the interface for saga processing
type Processor interface {
	WithCharacterProcessor(character.Processor) Processor
	WithCompartmentProcessor(compartment.Processor) Processor
	WithSkillProcessor(processor skill.Processor) Processor
	WithValidationProcessor(validation.Processor) Processor
	WithGuildProcessor(guild.Processor) Processor
	WithInviteProcessor(invite.Processor) Processor

	GetAll() ([]Saga, error)
	AllProvider() model.Provider[[]Saga]
	GetById(transactionId uuid.UUID) (Saga, error)
	ByIdProvider(transactionId uuid.UUID) model.Provider[Saga]

	Put(saga Saga) error
	MarkFurthestCompletedStepFailed(transactionId uuid.UUID) error
	MarkEarliestPendingStep(transactionId uuid.UUID, status Status) error
	MarkEarliestPendingStepCompleted(transactionId uuid.UUID) error
	StepCompleted(transactionId uuid.UUID, success bool) error
	AddStep(transactionId uuid.UUID, step Step[any]) error
	AddStepAfterCurrent(transactionId uuid.UUID, step Step[any]) error
	Step(transactionId uuid.UUID) error
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
}

// NewProcessor creates a new saga processor
func NewProcessor(logger logrus.FieldLogger, ctx context.Context) Processor {

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
	}
}

// GetAll returns all sagas for the current tenant
func (p *ProcessorImpl) GetAll() ([]Saga, error) {
	return p.AllProvider()()
}

func (p *ProcessorImpl) AllProvider() model.Provider[[]Saga] {
	return func() ([]Saga, error) {
		return GetCache().GetAll(p.t.Id()), nil
	}
}

// GetById returns a saga by its transaction ID for the current tenant
func (p *ProcessorImpl) GetById(transactionId uuid.UUID) (Saga, error) {
	return p.ByIdProvider(transactionId)()
}

func (p *ProcessorImpl) ByIdProvider(transactionId uuid.UUID) model.Provider[Saga] {
	return func() (Saga, error) {
		m, ok := GetCache().GetById(p.t.Id(), transactionId)
		if !ok {
			return Saga{}, errors.New("saga not found")
		}
		return m, nil
	}
}

// Put adds or updates a saga in the cache for the current tenant
func (p *ProcessorImpl) Put(saga Saga) error {
	p.l.WithFields(logrus.Fields{
		"transaction_id": saga.TransactionId.String(),
		"saga_type":      saga.SagaType,
		"tenant_id":      p.t.Id().String(),
	}).Debug("Inserting saga into cache")

	// Validate state consistency before inserting
	if err := saga.ValidateStateConsistency(); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": saga.TransactionId.String(),
			"saga_type":      saga.SagaType,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("State consistency validation failed before inserting saga")
		return err
	}

	GetCache().Put(p.t.Id(), saga)

	p.l.WithFields(logrus.Fields{
		"transaction_id": saga.TransactionId.String(),
		"saga_type":      saga.SagaType,
		"tenant_id":      p.t.Id().String(),
	}).Debug("Saga inserted into cache")

	return p.Step(saga.TransactionId)
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
	GetCache().Put(p.t.Id(), sagaCopy)

	return nil
}

// SafeSetStepStatus safely updates step status with validation and logging
func (p *ProcessorImpl) SafeSetStepStatus(s *Saga, stepIndex int, status Status, operation string) error {
	if err := s.SetStepStatus(stepIndex, status); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_index":     stepIndex,
			"status":         status,
			"operation":      operation,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("Failed to set step status safely")
		return err
	}

	// Validate state consistency after status change
	if err := s.ValidateStateConsistency(); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"operation":      operation,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("State consistency validation failed after safe status update")
		return err
	}

	return nil
}

func (p *ProcessorImpl) StepCompleted(transactionId uuid.UUID, success bool) error {
	s, err := p.GetById(transactionId)
	if err != nil {
		return nil
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

		err = p.MarkEarliestPendingStep(transactionId, status)
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
	if err := s.SetStepStatus(furthestCompletedIndex, Failed); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_index":     furthestCompletedIndex,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("Failed to set step status to failed")
		return err
	}

	// Validate state consistency before updating cache
	if err := s.ValidateStateConsistency(); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("State consistency validation failed after marking step as failed")
		return err
	}

	// Update the saga in the cache
	GetCache().Put(p.t.Id(), s)

	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        s.Steps[furthestCompletedIndex].StepId,
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
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"tenant_id":      p.t.Id().String(),
		}).Debugf("No pending steps found to mark as [%s].", status)
		return errors.New("no pending steps found")
	}

	// Mark the step with validation
	if err := s.SetStepStatus(earliestPendingIndex, status); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_index":     earliestPendingIndex,
			"status":         status,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("Failed to set step status")
		return err
	}

	// Validate state consistency before updating cache
	if err := s.ValidateStateConsistency(); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("State consistency validation failed after marking step")
		return err
	}

	// Update the saga in the cache
	GetCache().Put(p.t.Id(), s)

	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        s.Steps[earliestPendingIndex].StepId,
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
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"tenant_id":      p.t.Id().String(),
		}).Debug("Cannot add step to a failing saga.")
		return errors.New("cannot add step to a failing saga")
	}

	// Find the index of the current step (earliest pending step)
	currentStepIndex := s.FindEarliestPendingStepIndex()
	if currentStepIndex == -1 {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"tenant_id":      p.t.Id().String(),
		}).Debug("No pending steps found to add step after.")
		return errors.New("no pending steps found")
	}

	// Validate step ID uniqueness within the saga
	for _, existingStep := range s.Steps {
		if existingStep.StepId == step.StepId {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"step_id":        step.StepId,
				"tenant_id":      p.t.Id().String(),
			}).Debug("Step ID already exists in saga.")
			return fmt.Errorf("step ID '%s' already exists in saga", step.StepId)
		}
	}

	// Insert the new step right after the current step to maintain proper ordering
	insertIndex := currentStepIndex + 1

	// Ensure the step has proper timestamps
	if step.CreatedAt.IsZero() {
		step.CreatedAt = time.Now()
	}
	if step.UpdatedAt.IsZero() {
		step.UpdatedAt = time.Now()
	}

	// Expand the slice and insert the new step
	s.Steps = append(s.Steps[:insertIndex], append([]Step[any]{step}, s.Steps[insertIndex:]...)...)

	// Validate comprehensive state consistency after insertion
	if err := s.ValidateStateConsistency(); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_id":        step.StepId,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("State consistency validation failed after step insertion")
		return err
	}

	// Update the saga in the cache atomically
	GetCache().Put(p.t.Id(), s)

	p.l.WithFields(logrus.Fields{
		"transaction_id":  s.TransactionId.String(),
		"saga_type":       s.SagaType,
		"step_id":         step.StepId,
		"action":          step.Action,
		"insert_index":    insertIndex,
		"total_steps":     len(s.Steps),
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
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"tenant_id":      p.t.Id().String(),
		}).Debug("Cannot prepend step to a failing saga.")
		return errors.New("cannot prepend step to a failing saga")
	}

	// Validate step ID uniqueness within the saga
	for _, existingStep := range s.Steps {
		if existingStep.StepId == step.StepId {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"step_id":        step.StepId,
				"tenant_id":      p.t.Id().String(),
			}).Debug("Step ID already exists in saga.")
			return fmt.Errorf("step ID '%s' already exists in saga", step.StepId)
		}
	}

	// Ensure the step has proper timestamps
	if step.CreatedAt.IsZero() {
		step.CreatedAt = time.Now()
	}
	if step.UpdatedAt.IsZero() {
		step.UpdatedAt = time.Now()
	}

	// Prepend the new step to the beginning of the steps slice
	for i, st := range s.Steps {
		if st.Status == Pending {
			s.Steps = append(s.Steps[:i+1], append([]Step[any]{step}, s.Steps[i+1:]...)...)
			break
		}
	}

	// Validate comprehensive state consistency after insertion
	if err := s.ValidateStateConsistency(); err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_id":        step.StepId,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("State consistency validation failed after step prepend")
		return err
	}

	// Update the saga in the cache atomically
	GetCache().Put(p.t.Id(), s)

	p.l.WithFields(logrus.Fields{
		"transaction_id":  s.TransactionId.String(),
		"saga_type":       s.SagaType,
		"step_id":         step.StepId,
		"action":          step.Action,
		"insert_index":    0,
		"total_steps":     len(s.Steps),
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
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"tenant_id":      p.t.Id().String(),
		}).Debug("Reverting saga step.")
		return p.comp.CompensateFailedStep(s)
	}

	st, ok := s.GetCurrentStep()
	if !ok {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"tenant_id":      p.t.Id().String(),
		}).Debug("No steps remaining to progress.")
		GetCache().Remove(p.t.Id(), s.TransactionId)

		// Emit saga completion event
		err := producer.ProviderImpl(p.l)(p.ctx)(saga.EnvStatusEventTopic)(CompletedStatusEventProvider(s.TransactionId))
		if err != nil {
			p.l.WithError(err).WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"tenant_id":      p.t.Id().String(),
			}).Error("Failed to emit saga completion event.")
		}

		return nil
	}

	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"tenant_id":      p.t.Id().String(),
	}).Debugf("Progressing saga step [%s].", st.StepId)

	// Get the handler for this action type
	handler, exists := p.handle.GetHandler(st.Action)
	if !exists {
		return fmt.Errorf("unknown action type: %s", st.Action)
	}

	// Execute the handler
	return handler(s, st)
}
