package saga

import (
	"atlas-saga-orchestrator/character"
	"atlas-saga-orchestrator/compartment"
	"atlas-saga-orchestrator/guild"
	"atlas-saga-orchestrator/invite"
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"atlas-saga-orchestrator/skill"
	"atlas-saga-orchestrator/validation"
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

// Processor is the interface for saga processing
type Processor interface {
	GetAll() ([]Saga, error)
	AllProvider() model.Provider[[]Saga]
	GetById(transactionId uuid.UUID) (Saga, error)
	ByIdProvider(transactionId uuid.UUID) model.Provider[Saga]
	Put(saga Saga) error
	MarkFurthestCompletedStepFailed(transactionId uuid.UUID) error
	MarkEarliestPendingStepCompleted(transactionId uuid.UUID) error
	AddStep(transactionId uuid.UUID, step Step[any]) error
}

// ProcessorImpl is the implementation of the Processor interface
type ProcessorImpl struct {
	l       logrus.FieldLogger
	ctx     context.Context
	t       tenant.Model
	charP   character.Processor
	compP   compartment.Processor
	skillP  skill.Processor
	validP  validation.Processor
	guildP  guild.Processor
	inviteP invite.Processor
}

// NewProcessor creates a new saga processor
func NewProcessor(logger logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	return &ProcessorImpl{
		l:       logger,
		ctx:     ctx,
		t:       tenant.MustFromContext(ctx),
		charP:   character.NewProcessor(logger, ctx),
		compP:   compartment.NewProcessor(logger, ctx),
		skillP:  skill.NewProcessor(logger, ctx),
		validP:  validation.NewProcessor(logger, ctx),
		guildP:  guild.NewProcessor(logger, ctx),
		inviteP: invite.NewProcessor(logger, ctx),
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
		"transaction_id":     s.TransactionId.String(),
		"saga_type":          s.SagaType,
		"step_id":            step.StepId,
		"action":             step.Action,
		"insert_index":       insertIndex,
		"total_steps":        len(s.Steps),
		"completed_steps":    s.GetCompletedStepCount(),
		"pending_steps":      s.GetPendingStepCount(),
		"tenant_id":          p.t.Id().String(),
	}).Debug("Added new step to saga with proper ordering.")

	return nil
}

// ActionHandler is a function type for handling different saga action types
type ActionHandler func(p *ProcessorImpl, s Saga, st Step[any]) error

// actionHandlers maps action types to their handler functions
var actionHandlers = map[Action]ActionHandler{
	AwardInventory:               handleAwardAsset, // Deprecated: Use AwardAsset instead
	AwardAsset:                   handleAwardAsset, // Preferred over AwardInventory
	WarpToRandomPortal:           handleWarpToRandomPortal,
	WarpToPortal:                 handleWarpToPortal,
	AwardExperience:              handleAwardExperience,
	AwardLevel:                   handleAwardLevel,
	AwardMesos:                   handleAwardMesos,
	DestroyAsset:                 handleDestroyAsset,
	EquipAsset:                   handleEquipAsset,
	UnequipAsset:                 handleUnequipAsset,
	ChangeJob:                    handleChangeJob,
	CreateSkill:                  handleCreateSkill,
	UpdateSkill:                  handleUpdateSkill,
	ValidateCharacterState:       handleValidateCharacterState,
	RequestGuildName:             handleRequestGuildName,
	RequestGuildEmblem:           handleRequestGuildEmblem,
	RequestGuildDisband:          handleRequestGuildDisband,
	RequestGuildCapacityIncrease: handleRequestGuildCapacityIncrease,
	CreateInvite:                 handleCreateInvite,
	CreateCharacter:              handleCreateCharacter,
	CreateAndEquipAsset:          handleCreateAndEquipAsset,
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
		return p.compensateFailedStep(s)
	}

	st, ok := s.GetCurrentStep()
	if !ok {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"tenant_id":      p.t.Id().String(),
		}).Debug("No steps remaining to progress.")
		GetCache().Remove(p.t.Id(), s.TransactionId)
		// TODO complete saga
		return nil
	}

	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"tenant_id":      p.t.Id().String(),
	}).Debugf("Progressing saga step [%s].", st.StepId)

	// Get the handler for this action type
	handler, exists := actionHandlers[st.Action]
	if !exists {
		return fmt.Errorf("unknown action type: %s", st.Action)
	}

	// Execute the handler
	return handler(p, s, st)
}

// compensateFailedStep handles compensation for failed steps
func (p *ProcessorImpl) compensateFailedStep(s Saga) error {
	// Find the failed step
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex == -1 {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"tenant_id":      p.t.Id().String(),
		}).Debug("No failed step found for compensation.")
		return nil
	}

	failedStep := s.Steps[failedStepIndex]
	
	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        failedStep.StepId,
		"action":         failedStep.Action,
		"tenant_id":      p.t.Id().String(),
	}).Debug("Compensating failed step.")

	// Perform compensation based on the action type
	switch failedStep.Action {
	case EquipAsset:
		return p.compensateEquipAsset(s, failedStep)
	case UnequipAsset:
		return p.compensateUnequipAsset(s, failedStep)
	case CreateCharacter:
		return p.compensateCreateCharacter(s, failedStep)
	case CreateAndEquipAsset:
		return p.compensateCreateAndEquipAsset(s, failedStep)
	default:
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_id":        failedStep.StepId,
			"action":         failedStep.Action,
			"tenant_id":      p.t.Id().String(),
		}).Debug("No compensation logic available for action type.")
		// Mark step as compensated (remove failed status) with validation
		if err := s.SetStepStatus(failedStepIndex, Pending); err != nil {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"step_index":     failedStepIndex,
				"tenant_id":      p.t.Id().String(),
			}).WithError(err).Error("Failed to mark step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := s.ValidateStateConsistency(); err != nil {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"tenant_id":      p.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after compensation")
			return err
		}

		GetCache().Put(p.t.Id(), s)
		return nil
	}
}

// compensateEquipAsset handles compensation for a failed EquipAsset operation
// by performing the reverse operation (UnequipAsset)
func (p *ProcessorImpl) compensateEquipAsset(s Saga, failedStep Step[any]) error {
	// Extract the original payload
	payload, ok := failedStep.Payload.(EquipAssetPayload)
	if !ok {
		return fmt.Errorf("invalid payload for EquipAsset compensation")
	}

	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        failedStep.StepId,
		"character_id":   payload.CharacterId,
		"source":         payload.Source,
		"destination":    payload.Destination,
		"tenant_id":      p.t.Id().String(),
	}).Info("Compensating failed EquipAsset operation with UnequipAsset")

	// Perform the reverse operation: unequip from destination back to source
	err := p.compP.RequestUnequipAsset(s.TransactionId, payload.CharacterId, byte(payload.InventoryType), payload.Destination, payload.Source)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_id":        failedStep.StepId,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("Failed to compensate EquipAsset operation")
		return err
	}

	// Mark the failed step as compensated by removing it from the saga
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex != -1 {
		if err := s.SetStepStatus(failedStepIndex, Pending); err != nil {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"step_index":     failedStepIndex,
				"tenant_id":      p.t.Id().String(),
			}).WithError(err).Error("Failed to mark EquipAsset step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := s.ValidateStateConsistency(); err != nil {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"tenant_id":      p.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after EquipAsset compensation")
			return err
		}

		GetCache().Put(p.t.Id(), s)
	}

	return nil
}

// compensateUnequipAsset handles compensation for a failed UnequipAsset operation
// by performing the reverse operation (EquipAsset)
func (p *ProcessorImpl) compensateUnequipAsset(s Saga, failedStep Step[any]) error {
	// Extract the original payload
	payload, ok := failedStep.Payload.(UnequipAssetPayload)
	if !ok {
		return fmt.Errorf("invalid payload for UnequipAsset compensation")
	}

	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        failedStep.StepId,
		"character_id":   payload.CharacterId,
		"source":         payload.Source,
		"destination":    payload.Destination,
		"tenant_id":      p.t.Id().String(),
	}).Info("Compensating failed UnequipAsset operation with EquipAsset")

	// Perform the reverse operation: equip from destination back to source
	err := p.compP.RequestEquipAsset(s.TransactionId, payload.CharacterId, byte(payload.InventoryType), payload.Destination, payload.Source)
	if err != nil {
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_id":        failedStep.StepId,
			"tenant_id":      p.t.Id().String(),
		}).WithError(err).Error("Failed to compensate UnequipAsset operation")
		return err
	}

	// Mark the failed step as compensated by removing it from the saga
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex != -1 {
		if err := s.SetStepStatus(failedStepIndex, Pending); err != nil {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"step_index":     failedStepIndex,
				"tenant_id":      p.t.Id().String(),
			}).WithError(err).Error("Failed to mark UnequipAsset step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := s.ValidateStateConsistency(); err != nil {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"tenant_id":      p.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after UnequipAsset compensation")
			return err
		}

		GetCache().Put(p.t.Id(), s)
	}

	return nil
}

// compensateCreateCharacter handles compensation for a failed CreateCharacter operation
// Note: Character creation failures typically do not require compensation as the character
// creation process is atomic. If partial creation occurred, the character service should
// handle cleanup. This function exists for completeness and future extensibility.
func (p *ProcessorImpl) compensateCreateCharacter(s Saga, failedStep Step[any]) error {
	// Extract the original payload
	payload, ok := failedStep.Payload.(CharacterCreatePayload)
	if !ok {
		return fmt.Errorf("invalid payload for CreateCharacter compensation")
	}

	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        failedStep.StepId,
		"account_id":     payload.AccountId,
		"character_name": payload.Name,
		"world_id":       payload.WorldId,
		"tenant_id":      p.t.Id().String(),
	}).Info("Compensating failed CreateCharacter operation - no rollback action available")

	// Note: Currently there is no character deletion command available
	// in the character service, so we cannot perform actual rollback.
	// The character service should handle cleanup of failed character creation internally.
	// This compensation step simply acknowledges the failure and allows the saga to continue.

	// Mark the failed step as compensated by removing it from the saga
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex != -1 {
		if err := s.SetStepStatus(failedStepIndex, Pending); err != nil {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"step_index":     failedStepIndex,
				"tenant_id":      p.t.Id().String(),
			}).WithError(err).Error("Failed to mark CreateCharacter step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := s.ValidateStateConsistency(); err != nil {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"tenant_id":      p.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after CreateCharacter compensation")
			return err
		}

		GetCache().Put(p.t.Id(), s)
	}

	return nil
}

// compensateCreateAndEquipAsset handles compensation for a failed CreateAndEquipAsset operation
// This compound action has two phases:
// 1. Asset creation (handled by handleCreateAndEquipAsset)
// 2. Dynamic equipment step creation (handled by compartment consumer)
//
// Compensation scenarios:
// - Phase 1 failure: No compensation needed since nothing was created
// - Phase 2 failure: Need to destroy the created asset since it was successfully created but failed to equip
//
// Note: This function is called when the CreateAndEquipAsset step itself fails,
// not when the dynamically created EquipAsset step fails (that uses compensateEquipAsset)
func (p *ProcessorImpl) compensateCreateAndEquipAsset(s Saga, failedStep Step[any]) error {
	// Extract the original payload
	payload, ok := failedStep.Payload.(CreateAndEquipAssetPayload)
	if !ok {
		return fmt.Errorf("invalid payload for CreateAndEquipAsset compensation")
	}

	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        failedStep.StepId,
		"character_id":   payload.CharacterId,
		"template_id":    payload.Item.TemplateId,
		"quantity":       payload.Item.Quantity,
		"tenant_id":      p.t.Id().String(),
	}).Info("Compensating failed CreateAndEquipAsset operation")

	// For CreateAndEquipAsset, we need to determine if the asset was actually created
	// If the failure happened during the asset creation phase, no compensation is needed
	// If the failure happened during the equipment phase, we need to destroy the created asset
	
	// Check if there are any auto-generated equip steps in this saga
	// If an auto-equip step exists, it means the asset was successfully created
	// and the failure occurred during the equipment phase
	autoEquipStepExists := false
	for _, step := range s.Steps {
		if step.Action == EquipAsset && strings.HasPrefix(step.StepId, "auto_equip_step_") {
			autoEquipStepExists = true
			break
		}
	}

	if autoEquipStepExists {
		// Asset was created but equipment failed - need to destroy the created asset
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_id":        failedStep.StepId,
			"character_id":   payload.CharacterId,
			"template_id":    payload.Item.TemplateId,
			"quantity":       payload.Item.Quantity,
			"tenant_id":      p.t.Id().String(),
		}).Info("Auto-equip step found - destroying created asset for compensation")

		// Destroy the created asset
		err := p.compP.RequestDestroyItem(s.TransactionId, payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity)
		if err != nil {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"step_id":        failedStep.StepId,
				"character_id":   payload.CharacterId,
				"template_id":    payload.Item.TemplateId,
				"quantity":       payload.Item.Quantity,
				"tenant_id":      p.t.Id().String(),
			}).WithError(err).Error("Failed to destroy created asset during CreateAndEquipAsset compensation")
			return err
		}

		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_id":        failedStep.StepId,
			"character_id":   payload.CharacterId,
			"template_id":    payload.Item.TemplateId,
			"quantity":       payload.Item.Quantity,
			"tenant_id":      p.t.Id().String(),
		}).Info("Successfully destroyed created asset during CreateAndEquipAsset compensation")
	} else {
		// No auto-equip step found - asset creation failed, no compensation needed
		p.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_id":        failedStep.StepId,
			"character_id":   payload.CharacterId,
			"template_id":    payload.Item.TemplateId,
			"quantity":       payload.Item.Quantity,
			"tenant_id":      p.t.Id().String(),
		}).Info("No auto-equip step found - asset creation failed, no compensation needed")
	}

	// Mark the failed step as compensated
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex != -1 {
		if err := s.SetStepStatus(failedStepIndex, Pending); err != nil {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"step_index":     failedStepIndex,
				"tenant_id":      p.t.Id().String(),
			}).WithError(err).Error("Failed to mark CreateAndEquipAsset step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := s.ValidateStateConsistency(); err != nil {
			p.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"tenant_id":      p.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after CreateAndEquipAsset compensation")
			return err
		}

		GetCache().Put(p.t.Id(), s)
	}

	return nil
}

// logActionError logs an error that occurred during action processing
func (p *ProcessorImpl) logActionError(s Saga, st Step[any], err error, errorMsg string) {
	p.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        st.StepId,
		"tenant_id":      p.t.Id().String(),
	}).WithError(err).Error(errorMsg)
}

// handleAwardAsset handles the AwardAsset and AwardInventory actions
func handleAwardAsset(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(AwardItemActionPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := p.compP.RequestCreateItem(s.TransactionId, payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity)

	if err != nil {
		p.logActionError(s, st, err, "Unable to award asset.")
		return err
	}

	return nil
}

// handleAwardInventory is a wrapper for handleAwardAsset for backward compatibility
// Deprecated: Use handleAwardAsset instead
func handleAwardInventory(p *ProcessorImpl, s Saga, st Step[any]) error {
	return handleAwardAsset(p, s, st)
}

// handleWarpToRandomPortal handles the WarpToRandomPortal action
func handleWarpToRandomPortal(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(WarpToRandomPortalPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	f, ok := field.FromId(payload.FieldId)
	if !ok {
		return errors.New("invalid field id")
	}

	err := p.charP.WarpRandomAndEmit(s.TransactionId, payload.CharacterId, f)

	if err != nil {
		p.logActionError(s, st, err, "Unable to warp to random portal.")
		return err
	}

	return nil
}

// handleWarpToPortal handles the WarpToPortal action
func handleWarpToPortal(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(WarpToPortalPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	f, ok := field.FromId(payload.FieldId)
	if !ok {
		return errors.New("invalid field id")
	}

	err := p.charP.WarpToPortalAndEmit(s.TransactionId, payload.CharacterId, f, model.FixedProvider(payload.PortalId))

	if err != nil {
		p.logActionError(s, st, err, "Unable to warp to specific portal.")
		return err
	}

	return nil
}

// handleAwardExperience handles the AwardExperience action
func handleAwardExperience(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(AwardExperiencePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	eds := TransformExperienceDistributions(payload.Distributions)
	err := p.charP.AwardExperienceAndEmit(s.TransactionId, payload.WorldId, payload.CharacterId, payload.ChannelId, eds)

	if err != nil {
		p.logActionError(s, st, err, "Unable to award experience.")
		return err
	}

	return nil
}

// handleAwardLevel handles the AwardLevel action
func handleAwardLevel(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(AwardLevelPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := p.charP.AwardLevelAndEmit(s.TransactionId, payload.WorldId, payload.CharacterId, payload.ChannelId, payload.Amount)

	if err != nil {
		p.logActionError(s, st, err, "Unable to award level.")
		return err
	}

	return nil
}

// handleAwardMesos handles the AwardMesos action
func handleAwardMesos(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(AwardMesosPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := p.charP.AwardMesosAndEmit(s.TransactionId, payload.WorldId, payload.CharacterId, payload.ChannelId, payload.ActorId, payload.ActorType, payload.Amount)

	if err != nil {
		p.logActionError(s, st, err, "Unable to award mesos.")
		return err
	}

	return nil
}

// handleDestroyAsset handles the DestroyAsset action
func handleDestroyAsset(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(DestroyAssetPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := p.compP.RequestDestroyItem(s.TransactionId, payload.CharacterId, payload.TemplateId, payload.Quantity)

	if err != nil {
		p.logActionError(s, st, err, "Unable to destroy asset.")
		return err
	}

	return nil
}

// handleEquipAsset handles the EquipAsset action
func handleEquipAsset(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(EquipAssetPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := p.compP.RequestEquipAsset(s.TransactionId, payload.CharacterId, byte(payload.InventoryType), payload.Source, payload.Destination)

	if err != nil {
		p.logActionError(s, st, err, "Unable to equip asset.")
		return err
	}

	return nil
}

// handleUnequipAsset handles the UnequipAsset action
func handleUnequipAsset(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(UnequipAssetPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := p.compP.RequestUnequipAsset(s.TransactionId, payload.CharacterId, byte(payload.InventoryType), payload.Source, payload.Destination)

	if err != nil {
		p.logActionError(s, st, err, "Unable to unequip asset.")
		return err
	}

	return nil
}

// handleChangeJob handles the ChangeJob action
func handleChangeJob(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(ChangeJobPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := p.charP.ChangeJobAndEmit(s.TransactionId, payload.WorldId, payload.CharacterId, payload.ChannelId, payload.JobId)

	if err != nil {
		p.logActionError(s, st, err, "Unable to change job.")
		return err
	}

	return nil
}

// handleCreateSkill handles the CreateSkill action
func handleCreateSkill(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(CreateSkillPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := p.skillP.RequestCreateAndEmit(s.TransactionId, payload.CharacterId, payload.SkillId, payload.Level, payload.MasterLevel, payload.Expiration)

	if err != nil {
		p.logActionError(s, st, err, "Unable to create skill.")
		return err
	}

	return nil
}

// handleUpdateSkill handles the UpdateSkill action
func handleUpdateSkill(p *ProcessorImpl, s Saga, st Step[any]) error {
	payload, ok := st.Payload.(UpdateSkillPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := p.skillP.RequestUpdateAndEmit(s.TransactionId, payload.CharacterId, payload.SkillId, payload.Level, payload.MasterLevel, payload.Expiration)

	if err != nil {
		p.logActionError(s, st, err, "Unable to update skill.")
		return err
	}

	return nil
}

func TransformExperienceDistributions(source []ExperienceDistributions) []character2.ExperienceDistributions {
	target := make([]character2.ExperienceDistributions, len(source))

	for i, s := range source {
		target[i] = character2.ExperienceDistributions{
			ExperienceType: s.ExperienceType,
			Amount:         s.Amount,
			Attr1:          s.Attr1,
		}
	}

	return target
}

// handleValidateCharacterState handles the ValidateCharacterState action
func handleValidateCharacterState(p *ProcessorImpl, s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(ValidateCharacterStatePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the validation processor
	result, err := p.validP.ValidateCharacterState(payload.CharacterId, payload.Conditions)
	if err != nil {
		p.logActionError(s, st, err, "Unable to validate character state.")
		return err
	}

	// Check if validation passed
	if !result.Passed() {
		// If validation failed, mark the step as failed
		err := fmt.Errorf("character state validation failed: %v", result.Details())
		p.logActionError(s, st, err, "Character state validation failed.")
		return err
	}

	return nil
}

// handleRequestGuildName handles the RequestGuildName action
func handleRequestGuildName(p *ProcessorImpl, s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(RequestGuildNamePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the guild processor
	err := p.guildP.RequestName(s.TransactionId, payload.WorldId, payload.ChannelId, payload.CharacterId)
	if err != nil {
		p.logActionError(s, st, err, "Unable to request guild name.")
		return err
	}

	return nil
}

// handleRequestGuildEmblem handles the RequestGuildEmblem action
func handleRequestGuildEmblem(p *ProcessorImpl, s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(RequestGuildEmblemPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the guild processor
	err := p.guildP.RequestEmblem(s.TransactionId, payload.WorldId, payload.ChannelId, payload.CharacterId)
	if err != nil {
		p.logActionError(s, st, err, "Unable to request guild emblem.")
		return err
	}

	return nil
}

// handleRequestGuildDisband handles the RequestGuildDisband action
func handleRequestGuildDisband(p *ProcessorImpl, s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(RequestGuildDisbandPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the guild processor
	err := p.guildP.RequestDisband(s.TransactionId, payload.WorldId, payload.ChannelId, payload.CharacterId)
	if err != nil {
		p.logActionError(s, st, err, "Unable to request guild disband.")
		return err
	}

	return nil
}

// handleRequestGuildCapacityIncrease handles the RequestGuildCapacityIncrease action
func handleRequestGuildCapacityIncrease(p *ProcessorImpl, s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(RequestGuildCapacityIncreasePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the guild processor
	err := p.guildP.RequestCapacityIncrease(s.TransactionId, payload.WorldId, payload.ChannelId, payload.CharacterId)
	if err != nil {
		p.logActionError(s, st, err, "Unable to request guild capacity increase.")
		return err
	}

	return nil
}

// handleCreateInvite handles the CreateInvite action
func handleCreateInvite(p *ProcessorImpl, s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(CreateInvitePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the invite processor
	err := p.inviteP.Create(s.TransactionId, payload.InviteType, payload.OriginatorId, payload.WorldId, payload.ReferenceId, payload.TargetId)
	if err != nil {
		p.logActionError(s, st, err, "Unable to create invitation.")
		return err
	}

	return nil
}

// handleCreateCharacter handles the CreateCharacter action
func handleCreateCharacter(p *ProcessorImpl, s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(CharacterCreatePayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Call the character processor
	err := p.charP.RequestCreateCharacter(s.TransactionId, payload.AccountId, payload.Name, payload.WorldId, payload.ChannelId, payload.JobId, payload.Gender, payload.Face, payload.Hair, payload.HairColor, payload.Skin, payload.Top, payload.Bottom, payload.Shoes, payload.Weapon, payload.MapId)
	if err != nil {
		p.logActionError(s, st, err, "Unable to create character.")
		return err
	}

	return nil
}

// handleCreateAndEquipAsset handles the CreateAndEquipAsset action
// This is a compound action that first creates an asset (internally using award_asset semantics)
// and then dynamically creates an equip_asset step when the creation succeeds
func handleCreateAndEquipAsset(p *ProcessorImpl, s Saga, st Step[any]) error {
	// Extract the payload
	payload, ok := st.Payload.(CreateAndEquipAssetPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	// Step 1: Internal award_asset - Create the item using the same logic as handleAwardAsset
	// Convert saga payload to compartment payload to avoid import cycle
	compartmentPayload := compartment.CreateAndEquipAssetPayload{
		CharacterId: payload.CharacterId,
		Item: compartment.ItemPayload{
			TemplateId: payload.Item.TemplateId,
			Quantity:   payload.Item.Quantity,
		},
	}
	
	err := p.compP.RequestCreateAndEquipAsset(s.TransactionId, compartmentPayload)
	if err != nil {
		p.logActionError(s, st, err, "Unable to create asset for create_and_equip_asset.")
		return err
	}

	// Note: Step 2 (dynamic equip_asset step creation) will be handled by the compartment consumer
	// when it receives the StatusEventTypeCreated event from the compartment service.
	// The consumer will detect this is a CreateAndEquipAsset step and add the equip_asset step.

	return nil
}
