package saga

import (
	"atlas-saga-orchestrator/character"
	"atlas-saga-orchestrator/compartment"
	"atlas-saga-orchestrator/guild"
	"atlas-saga-orchestrator/invite"
	sagaMsg "atlas-saga-orchestrator/kafka/message/saga"
	"atlas-saga-orchestrator/kafka/producer"
	"atlas-saga-orchestrator/skill"
	"atlas-saga-orchestrator/validation"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Compensator interface {
	WithCharacterProcessor(character.Processor) Compensator
	WithCompartmentProcessor(compartment.Processor) Compensator
	WithSkillProcessor(skill.Processor) Compensator
	WithValidationProcessor(validation.Processor) Compensator
	WithGuildProcessor(guild.Processor) Compensator
	WithInviteProcessor(invite.Processor) Compensator

	CompensateFailedStep(s Saga) error
	compensateEquipAsset(s Saga, failedStep Step[any]) error
	compensateUnequipAsset(s Saga, failedStep Step[any]) error
	compensateCreateCharacter(s Saga, failedStep Step[any]) error
	compensateCreateAndEquipAsset(s Saga, failedStep Step[any]) error
	compensateChangeHair(s Saga, failedStep Step[any]) error
	compensateChangeFace(s Saga, failedStep Step[any]) error
	compensateChangeSkin(s Saga, failedStep Step[any]) error
	compensateStorageOperation(s Saga, failedStep Step[any]) error
	compensateSelectGachaponReward(s Saga, failedStep Step[any]) error
	compensateCharacterCreation(s Saga, failedStep Step[any]) error

	// DispatchCharacterCreationRollbacks is the dispatch half of the reverse-walk
	// compensator. It fires the inverse commands (DestroyItem / DeleteSkill /
	// DeleteCharacter-last) for each completed step of a CharacterCreation saga.
	// No lifecycle transitions, no Failed emission, no cache eviction — callers
	// handle those. Used both by the step-driven compensator and by the timer-
	// fire path in saga/timer.go (PRD §4.3 / plan Phase 4.3).
	DispatchCharacterCreationRollbacks(s Saga)
}

type CompensatorImpl struct {
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

func NewCompensator(l logrus.FieldLogger, ctx context.Context) Compensator {
	return &CompensatorImpl{
		l:       l,
		ctx:     ctx,
		t:       tenant.MustFromContext(ctx),
		charP:   character.NewProcessor(l, ctx),
		compP:   compartment.NewProcessor(l, ctx),
		skillP:  skill.NewProcessor(l, ctx),
		validP:  validation.NewProcessor(l, ctx),
		guildP:  guild.NewProcessor(l, ctx),
		inviteP: invite.NewProcessor(l, ctx),
	}
}

func (c *CompensatorImpl) WithCharacterProcessor(charP character.Processor) Compensator {
	return &CompensatorImpl{
		l:       c.l,
		ctx:     c.ctx,
		t:       c.t,
		charP:   charP,
		compP:   c.compP,
		skillP:  c.skillP,
		validP:  c.validP,
		guildP:  c.guildP,
		inviteP: c.inviteP,
	}
}

func (c *CompensatorImpl) WithCompartmentProcessor(compP compartment.Processor) Compensator {
	return &CompensatorImpl{
		l:       c.l,
		ctx:     c.ctx,
		t:       c.t,
		charP:   c.charP,
		compP:   compP,
		skillP:  c.skillP,
		validP:  c.validP,
		guildP:  c.guildP,
		inviteP: c.inviteP,
	}
}

func (c *CompensatorImpl) WithSkillProcessor(skillP skill.Processor) Compensator {
	return &CompensatorImpl{
		l:       c.l,
		ctx:     c.ctx,
		t:       c.t,
		charP:   c.charP,
		compP:   c.compP,
		skillP:  skillP,
		validP:  c.validP,
		guildP:  c.guildP,
		inviteP: c.inviteP,
	}
}

func (c *CompensatorImpl) WithValidationProcessor(validP validation.Processor) Compensator {
	return &CompensatorImpl{
		l:       c.l,
		ctx:     c.ctx,
		t:       c.t,
		charP:   c.charP,
		compP:   c.compP,
		skillP:  c.skillP,
		validP:  validP,
		guildP:  c.guildP,
		inviteP: c.inviteP,
	}
}

func (c *CompensatorImpl) WithGuildProcessor(guildP guild.Processor) Compensator {
	return &CompensatorImpl{
		l:       c.l,
		ctx:     c.ctx,
		t:       c.t,
		charP:   c.charP,
		compP:   c.compP,
		skillP:  c.skillP,
		validP:  c.validP,
		guildP:  guildP,
		inviteP: c.inviteP,
	}
}

func (c *CompensatorImpl) WithInviteProcessor(inviteP invite.Processor) Compensator {
	return &CompensatorImpl{
		l:       c.l,
		ctx:     c.ctx,
		t:       c.t,
		charP:   c.charP,
		compP:   c.compP,
		skillP:  c.skillP,
		validP:  c.validP,
		guildP:  c.guildP,
		inviteP: inviteP,
	}
}

// CompensateFailedStep handles compensation for failed steps
func (c *CompensatorImpl) CompensateFailedStep(s Saga) error {
	// Find the failed step
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex == -1 {
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"tenant_id":      c.t.Id().String(),
		}).Debug("No failed step found for compensation.")
		return nil
	}

	failedStep, _ := s.StepAt(failedStepIndex)

	// Character-creation reverse-walk (plan Phase 6). Takes precedence over the
	// per-step switch so that a CharacterCreation saga ALWAYS runs the full
	// reverse chain (DestroyItem * / DeleteSkill * / DeleteCharacter) rather
	// than only compensating the failing step.
	if s.SagaType() == CharacterCreation {
		return c.compensateCharacterCreation(s, failedStep)
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        failedStep.StepId(),
		"action":         failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Debug("Compensating failed step.")

	// Special handling for ValidateCharacterState failures
	// These are terminal failures - no compensation needed, just emit FAILED event
	if failedStep.Action() == ValidateCharacterState {
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_id":        failedStep.StepId(),
			"tenant_id":      c.t.Id().String(),
		}).Info("Validation failed - terminating saga without compensation.")

		// Cancel the Phase-4 timeout backstop and remove saga from cache.
		SagaTimers().Cancel(s.TransactionId())
		GetCache().Remove(c.ctx, s.TransactionId())

		// Extract character ID from the validation payload
		characterId := ExtractCharacterId(failedStep)

		// Emit saga failed event
		err := producer.ProviderImpl(c.l)(c.ctx)(sagaMsg.EnvStatusEventTopic)(
			FailedStatusEventProvider(s.TransactionId(), 0, characterId, string(s.SagaType()), sagaMsg.ErrorCodeUnknown, "Validation failed", failedStep.StepId()))
		if err != nil {
			c.l.WithError(err).WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"tenant_id":      c.t.Id().String(),
			}).Error("Failed to emit saga failed event.")
		}

		return nil
	}

	// Perform compensation based on the action type
	switch failedStep.Action() {
	case EquipAsset:
		return c.compensateEquipAsset(s, failedStep)
	case UnequipAsset:
		return c.compensateUnequipAsset(s, failedStep)
	case CreateCharacter:
		return c.compensateCreateCharacter(s, failedStep)
	case CreateAndEquipAsset:
		return c.compensateCreateAndEquipAsset(s, failedStep)
	case ChangeHair:
		return c.compensateChangeHair(s, failedStep)
	case ChangeFace:
		return c.compensateChangeFace(s, failedStep)
	case ChangeSkin:
		return c.compensateChangeSkin(s, failedStep)
	case AwardMesos, AcceptToStorage, AcceptToCharacter, ReleaseFromStorage, ReleaseFromCharacter:
		// Storage-related actions are terminal failures - emit error event and stop
		return c.compensateStorageOperation(s, failedStep)
	case SelectGachaponReward:
		return c.compensateSelectGachaponReward(s, failedStep)
	default:
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_id":        failedStep.StepId(),
			"action":         failedStep.Action(),
			"tenant_id":      c.t.Id().String(),
		}).Debug("No compensation logic available for action type.")
		// Mark step as compensated (remove failed status) with validation
		updatedSaga, err := s.WithStepStatus(failedStepIndex, Pending)
		if err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"step_index":     failedStepIndex,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to mark step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := updatedSaga.ValidateStateConsistency(); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after compensation")
			return err
		}

		if err := GetCache().Put(c.ctx,updatedSaga); err != nil {
			return err
		}
		return nil
	}
}

// compensateEquipAsset handles compensation for a failed EquipAsset operation
// by performing the reverse operation (UnequipAsset)
func (c *CompensatorImpl) compensateEquipAsset(s Saga, failedStep Step[any]) error {
	// Extract the original payload
	payload, ok := failedStep.Payload().(EquipAssetPayload)
	if !ok {
		return fmt.Errorf("invalid payload for EquipAsset compensation")
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        failedStep.StepId(),
		"character_id":   payload.CharacterId,
		"source":         payload.Source,
		"destination":    payload.Destination,
		"tenant_id":      c.t.Id().String(),
	}).Info("Compensating failed EquipAsset operation with UnequipAsset")

	// Perform the reverse operation: unequip from destination back to source
	err := c.compP.RequestUnequipAsset(s.TransactionId(), payload.CharacterId, byte(payload.InventoryType), payload.Destination, payload.Source)
	if err != nil {
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_id":        failedStep.StepId(),
			"tenant_id":      c.t.Id().String(),
		}).WithError(err).Error("Failed to compensate EquipAsset operation")
		return err
	}

	// Mark the failed step as compensated by removing it from the saga
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex != -1 {
		updatedSaga, err := s.WithStepStatus(failedStepIndex, Pending)
		if err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"step_index":     failedStepIndex,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to mark EquipAsset step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := updatedSaga.ValidateStateConsistency(); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after EquipAsset compensation")
			return err
		}

		if err := GetCache().Put(c.ctx,updatedSaga); err != nil {
			return err
		}
	}

	return nil
}

// compensateUnequipAsset handles compensation for a failed UnequipAsset operation
// by performing the reverse operation (EquipAsset)
func (c *CompensatorImpl) compensateUnequipAsset(s Saga, failedStep Step[any]) error {
	// Extract the original payload
	payload, ok := failedStep.Payload().(UnequipAssetPayload)
	if !ok {
		return fmt.Errorf("invalid payload for UnequipAsset compensation")
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        failedStep.StepId(),
		"character_id":   payload.CharacterId,
		"source":         payload.Source,
		"destination":    payload.Destination,
		"tenant_id":      c.t.Id().String(),
	}).Info("Compensating failed UnequipAsset operation with EquipAsset")

	// Perform the reverse operation: equip from destination back to source
	err := c.compP.RequestEquipAsset(s.TransactionId(), payload.CharacterId, byte(payload.InventoryType), payload.Destination, payload.Source)
	if err != nil {
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_id":        failedStep.StepId(),
			"tenant_id":      c.t.Id().String(),
		}).WithError(err).Error("Failed to compensate UnequipAsset operation")
		return err
	}

	// Mark the failed step as compensated by removing it from the saga
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex != -1 {
		updatedSaga, err := s.WithStepStatus(failedStepIndex, Pending)
		if err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"step_index":     failedStepIndex,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to mark UnequipAsset step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := updatedSaga.ValidateStateConsistency(); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after UnequipAsset compensation")
			return err
		}

		if err := GetCache().Put(c.ctx,updatedSaga); err != nil {
			return err
		}
	}

	return nil
}

// compensateCreateCharacter handles compensation for a failed CreateCharacter operation
// Note: Character creation failures typically do not require compensation as the character
// creation process is atomic. If partial creation occurred, the character service should
// handle cleanup. This function exists for completeness and future extensibility.
func (c *CompensatorImpl) compensateCreateCharacter(s Saga, failedStep Step[any]) error {
	// Extract the original payload
	payload, ok := failedStep.Payload().(CharacterCreatePayload)
	if !ok {
		return fmt.Errorf("invalid payload for CreateCharacter compensation")
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        failedStep.StepId(),
		"account_id":     payload.AccountId,
		"character_name": payload.Name,
		"world_id":       payload.WorldId,
		"tenant_id":      c.t.Id().String(),
	}).Info("Compensating failed CreateCharacter operation - no rollback action available")

	// Note: Currently there is no character deletion command available
	// in the character service, so we cannot perform actual rollback.
	// The character service should handle cleanup of failed character creation internally.
	// This compensation step simply acknowledges the failure and allows the saga to continue.

	// Mark the failed step as compensated by removing it from the saga
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex != -1 {
		updatedSaga, err := s.WithStepStatus(failedStepIndex, Pending)
		if err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"step_index":     failedStepIndex,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to mark CreateCharacter step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := updatedSaga.ValidateStateConsistency(); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after CreateCharacter compensation")
			return err
		}

		if err := GetCache().Put(c.ctx,updatedSaga); err != nil {
			return err
		}
	}

	return nil
}

// CompensateCreateAndEquipAsset handles compensation for a failed CreateAndEquipAsset operation
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
func (c *CompensatorImpl) compensateCreateAndEquipAsset(s Saga, failedStep Step[any]) error {
	// Extract the original payload
	payload, ok := failedStep.Payload().(CreateAndEquipAssetPayload)
	if !ok {
		return fmt.Errorf("invalid payload for CreateAndEquipAsset compensation")
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        failedStep.StepId(),
		"character_id":   payload.CharacterId,
		"template_id":    payload.Item.TemplateId,
		"quantity":       payload.Item.Quantity,
		"tenant_id":      c.t.Id().String(),
	}).Info("Compensating failed CreateAndEquipAsset operation")

	// For CreateAndEquipAsset, we need to determine if the asset was actually created
	// If the failure happened during the asset creation phase, no compensation is needed
	// If the failure happened during the equipment phase, we need to destroy the created asset

	// Check if there are any auto-generated equip steps in this saga
	// If an auto-equip step exists, it means the asset was successfully created
	// and the failure occurred during the equipment phase
	autoEquipStepExists := false
	for _, step := range s.Steps() {
		if step.Action() == EquipAsset && strings.HasPrefix(step.StepId(), "auto_equip_step_") {
			autoEquipStepExists = true
			break
		}
	}

	if autoEquipStepExists {
		// Asset was created but equipment failed - need to destroy the created asset
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_id":        failedStep.StepId(),
			"character_id":   payload.CharacterId,
			"template_id":    payload.Item.TemplateId,
			"quantity":       payload.Item.Quantity,
			"tenant_id":      c.t.Id().String(),
		}).Info("Auto-equip step found - destroying created asset for compensation")

		// Destroy the created asset (removeAll = false, destroy exact quantity created)
		err := c.compP.RequestDestroyItem(s.TransactionId(), payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity, false)
		if err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"step_id":        failedStep.StepId(),
				"character_id":   payload.CharacterId,
				"template_id":    payload.Item.TemplateId,
				"quantity":       payload.Item.Quantity,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to destroy created asset during CreateAndEquipAsset compensation")
			return err
		}

		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_id":        failedStep.StepId(),
			"character_id":   payload.CharacterId,
			"template_id":    payload.Item.TemplateId,
			"quantity":       payload.Item.Quantity,
			"tenant_id":      c.t.Id().String(),
		}).Info("Successfully destroyed created asset during CreateAndEquipAsset compensation")
	} else {
		// No auto-equip step found - asset creation failed, no compensation needed
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"step_id":        failedStep.StepId(),
			"character_id":   payload.CharacterId,
			"template_id":    payload.Item.TemplateId,
			"quantity":       payload.Item.Quantity,
			"tenant_id":      c.t.Id().String(),
		}).Info("No auto-equip step found - asset creation failed, no compensation needed")
	}

	// Mark the failed step as compensated
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex != -1 {
		updatedSaga, err := s.WithStepStatus(failedStepIndex, Pending)
		if err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"step_index":     failedStepIndex,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to mark CreateAndEquipAsset step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := updatedSaga.ValidateStateConsistency(); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after CreateAndEquipAsset compensation")
			return err
		}

		if err := GetCache().Put(c.ctx,updatedSaga); err != nil {
			return err
		}
	}

	return nil
}

// compensateChangeHair handles compensation for a failed ChangeHair operation
// Note: Currently cosmetic changes cannot be fully rolled back because the saga payload
// does not capture the original cosmetic value before the change. The character already
// has the new hair style applied. Future enhancement could store the old value for rollback.
func (c *CompensatorImpl) compensateChangeHair(s Saga, failedStep Step[any]) error {
	// Extract the original payload
	payload, ok := failedStep.Payload().(ChangeHairPayload)
	if !ok {
		return fmt.Errorf("invalid payload for ChangeHair compensation")
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        failedStep.StepId(),
		"character_id":   payload.CharacterId,
		"new_style_id":   payload.StyleId,
		"tenant_id":      c.t.Id().String(),
	}).Info("Compensating failed ChangeHair operation - no rollback action available")

	// Note: To support full rollback, we would need to:
	// 1. Capture the old hair style before applying the change
	// 2. Store it in the saga payload or metadata
	// 3. Revert to the old style here
	// For now, the character retains the new hair style even if the saga fails.

	// Mark the failed step as compensated
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex != -1 {
		updatedSaga, err := s.WithStepStatus(failedStepIndex, Pending)
		if err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"step_index":     failedStepIndex,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to mark ChangeHair step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := updatedSaga.ValidateStateConsistency(); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after ChangeHair compensation")
			return err
		}

		if err := GetCache().Put(c.ctx,updatedSaga); err != nil {
			return err
		}
	}

	return nil
}

// compensateChangeFace handles compensation for a failed ChangeFace operation
// Note: Currently cosmetic changes cannot be fully rolled back because the saga payload
// does not capture the original cosmetic value before the change. The character already
// has the new face style applied. Future enhancement could store the old value for rollback.
func (c *CompensatorImpl) compensateChangeFace(s Saga, failedStep Step[any]) error {
	// Extract the original payload
	payload, ok := failedStep.Payload().(ChangeFacePayload)
	if !ok {
		return fmt.Errorf("invalid payload for ChangeFace compensation")
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        failedStep.StepId(),
		"character_id":   payload.CharacterId,
		"new_style_id":   payload.StyleId,
		"tenant_id":      c.t.Id().String(),
	}).Info("Compensating failed ChangeFace operation - no rollback action available")

	// Note: To support full rollback, we would need to:
	// 1. Capture the old face style before applying the change
	// 2. Store it in the saga payload or metadata
	// 3. Revert to the old style here
	// For now, the character retains the new face style even if the saga fails.

	// Mark the failed step as compensated
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex != -1 {
		updatedSaga, err := s.WithStepStatus(failedStepIndex, Pending)
		if err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"step_index":     failedStepIndex,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to mark ChangeFace step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := updatedSaga.ValidateStateConsistency(); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after ChangeFace compensation")
			return err
		}

		if err := GetCache().Put(c.ctx,updatedSaga); err != nil {
			return err
		}
	}

	return nil
}

// compensateChangeSkin handles compensation for a failed ChangeSkin operation
// Note: Currently cosmetic changes cannot be fully rolled back because the saga payload
// does not capture the original cosmetic value before the change. The character already
// has the new skin color applied. Future enhancement could store the old value for rollback.
func (c *CompensatorImpl) compensateChangeSkin(s Saga, failedStep Step[any]) error {
	// Extract the original payload
	payload, ok := failedStep.Payload().(ChangeSkinPayload)
	if !ok {
		return fmt.Errorf("invalid payload for ChangeSkin compensation")
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        failedStep.StepId(),
		"character_id":   payload.CharacterId,
		"new_style_id":   payload.StyleId,
		"tenant_id":      c.t.Id().String(),
	}).Info("Compensating failed ChangeSkin operation - no rollback action available")

	// Note: To support full rollback, we would need to:
	// 1. Capture the old skin color before applying the change
	// 2. Store it in the saga payload or metadata
	// 3. Revert to the old color here
	// For now, the character retains the new skin color even if the saga fails.

	// Mark the failed step as compensated
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex != -1 {
		updatedSaga, err := s.WithStepStatus(failedStepIndex, Pending)
		if err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"step_index":     failedStepIndex,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to mark ChangeSkin step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := updatedSaga.ValidateStateConsistency(); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"saga_type":      s.SagaType(),
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after ChangeSkin compensation")
			return err
		}

		if err := GetCache().Put(c.ctx,updatedSaga); err != nil {
			return err
		}
	}

	return nil
}

// compensateStorageOperation handles compensation for storage-related operation failures.
// These are terminal failures that emit an error event to notify the client.
// No rollback is performed - the saga simply terminates with an appropriate error code.
func (c *CompensatorImpl) compensateStorageOperation(s Saga, failedStep Step[any]) error {
	// Extract character ID from the failed step's payload
	characterId := ExtractCharacterId(failedStep)

	// Determine the appropriate error code based on the saga and failed step
	errorCode := DetermineErrorCode(s, failedStep)

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        failedStep.StepId(),
		"action":         failedStep.Action(),
		"character_id":   characterId,
		"error_code":     errorCode,
		"tenant_id":      c.t.Id().String(),
	}).Info("Storage operation failed - terminating saga with error event.")

	// Cancel the Phase-4 timeout backstop and remove saga from cache.
	SagaTimers().Cancel(s.TransactionId())
	GetCache().Remove(c.ctx, s.TransactionId())

	// Emit saga failed event with context-appropriate error information
	err := producer.ProviderImpl(c.l)(c.ctx)(sagaMsg.EnvStatusEventTopic)(
		FailedStatusEventProvider(
			s.TransactionId(),
			0,
			characterId,
			string(s.SagaType()),
			errorCode,
			fmt.Sprintf("Storage operation failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action()),
			failedStep.StepId(),
		))
	if err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"character_id":   characterId,
			"error_code":     errorCode,
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event for storage operation.")
		return err
	}

	return nil
}

// compensateSelectGachaponReward handles compensation for a failed SelectGachaponReward operation.
// When reward selection fails, the gachapon ticket has already been destroyed (prior DestroyAsset step).
// Compensation re-awards the ticket by walking backwards through completed steps to find the
// DestroyAsset and re-creating the item. The saga is then terminated with a failure event.
func (c *CompensatorImpl) compensateSelectGachaponReward(s Saga, failedStep Step[any]) error {
	payload, ok := failedStep.Payload().(SelectGachaponRewardPayload)
	if !ok {
		return fmt.Errorf("invalid payload for SelectGachaponReward compensation")
	}

	characterId := payload.CharacterId

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        failedStep.StepId(),
		"character_id":   characterId,
		"gachapon_id":    payload.GachaponId,
		"tenant_id":      c.t.Id().String(),
	}).Info("Compensating failed SelectGachaponReward - re-awarding destroyed ticket.")

	// Walk backwards through completed steps to find DestroyAsset steps that need reversal
	for _, step := range s.Steps() {
		if step.Status() != Completed {
			continue
		}
		if step.Action() != DestroyAsset {
			continue
		}
		destroyPayload, ok := step.Payload().(DestroyAssetPayload)
		if !ok {
			continue
		}

		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"step_id":        step.StepId(),
			"character_id":   destroyPayload.CharacterId,
			"template_id":    destroyPayload.TemplateId,
			"quantity":       destroyPayload.Quantity,
			"tenant_id":      c.t.Id().String(),
		}).Info("Re-awarding destroyed asset as compensation.")

		err := c.compP.RequestCreateItem(s.TransactionId(), destroyPayload.CharacterId, destroyPayload.TemplateId, destroyPayload.Quantity, time.Time{})
		if err != nil {
			c.l.WithError(err).WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"character_id":   destroyPayload.CharacterId,
				"template_id":    destroyPayload.TemplateId,
				"tenant_id":      c.t.Id().String(),
			}).Error("Failed to re-award destroyed asset during SelectGachaponReward compensation.")
			return err
		}
	}

	// Cancel the Phase-4 timeout backstop and remove saga from cache.
	SagaTimers().Cancel(s.TransactionId())
	GetCache().Remove(c.ctx, s.TransactionId())

	// Emit saga failed event
	err := producer.ProviderImpl(c.l)(c.ctx)(sagaMsg.EnvStatusEventTopic)(
		FailedStatusEventProvider(
			s.TransactionId(),
			0,
			characterId,
			string(s.SagaType()),
			sagaMsg.ErrorCodeUnknown,
			fmt.Sprintf("Gachapon reward selection failed at step [%s]", failedStep.StepId()),
			failedStep.StepId(),
		))
	if err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"saga_type":      s.SagaType(),
			"character_id":   characterId,
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event for gachapon compensation.")
		return err
	}

	return nil
}

// compensateCharacterCreation is the character-creation reverse-walk
// compensator (PRD §4.3 / plan Phase 6). On a character-creation failure it
// walks the saga's completed steps in reverse, dispatches the inverse command
// for each (fire-and-forget; Phase-5 compensators are idempotent on missing
// rows), emits exactly one StatusEventTypeFailed, cancels the Phase-4 timer,
// and evicts the saga from cache.
//
// CreateCharacter is dispatched LAST so item/skill rows referencing the
// character are cleaned up first.
//
// Double-emission is prevented by TryTransition(Compensating → Failed): if the
// timer already emitted Failed (via ErrorCodeSagaTimeout), the transition is
// refused and this function returns without re-emitting. See PRD §4.7.
func (c *CompensatorImpl) compensateCharacterCreation(s Saga, failedStep Step[any]) error {
	accountId, characterId := ExtractCharacterCreationIds(s)

	c.l.WithFields(logrus.Fields{
		"transaction_id":  s.TransactionId().String(),
		"failed_step":     failedStep.StepId(),
		"failed_action":   failedStep.Action(),
		"character_id":    characterId,
		"account_id":      accountId,
		"tenant_id":       c.t.Id().String(),
		"total_steps":     s.StepCount(),
		"completed_steps": s.GetCompletedStepCount(),
	}).Info("CharacterCreation saga failing — dispatching reverse-walk compensation.")

	c.DispatchCharacterCreationRollbacks(s)

	// Phase-4 timer already fired? Its handleSagaTimeout takes Pending →
	// Compensating, dispatches rollbacks, and emits Failed(ErrorCodeSagaTimeout).
	// In that race, the timer beats us to Compensating → Failed, and this
	// step-triggered reverse walk returns without a second emit.
	if !GetCache().TryTransition(c.ctx, s.TransactionId(), SagaLifecycleCompensating, SagaLifecycleFailed) {
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Info("saga already in terminal Failed state; reverse-walk emission skipped.")
		SagaTimers().Cancel(s.TransactionId())
		GetCache().Remove(c.ctx, s.TransactionId())
		return nil
	}

	SagaTimers().Cancel(s.TransactionId())
	GetCache().Remove(c.ctx, s.TransactionId())

	reason := fmt.Sprintf("Character creation failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if err := EmitSagaFailed(c.l, c.ctx, s, sagaMsg.ErrorCodeUnknown, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event after character-creation compensation.")
		return err
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"character_id":   characterId,
		"account_id":     accountId,
		"tenant_id":      c.t.Id().String(),
	}).Info("Character-creation reverse-walk compensation complete; saga terminated.")
	return nil
}

// DispatchCharacterCreationRollbacks walks the saga's completed steps in
// reverse and dispatches the inverse compensation command for each. This is
// the pure "dispatch" half of the reverse-walk — no lifecycle transitions,
// no event emission, no cache eviction. Callers are responsible for those.
//
// CreateCharacter is deferred to the end so item/skill inverses are in flight
// before the character row is removed. Phase-5 compensators are idempotent on
// missing rows, so out-of-order downstream arrival is safe.
func (c *CompensatorImpl) DispatchCharacterCreationRollbacks(s Saga) {
	_, characterId := ExtractCharacterCreationIds(s)
	worldId := extractCharacterCreationWorldId(s)

	var deleteCharacterStep *Step[any]

	steps := s.Steps()
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Status() != Completed {
			continue
		}
		switch step.Action() {
		case AwardAsset:
			if payload, ok := step.Payload().(AwardItemActionPayload); ok {
				if err := c.compP.RequestDestroyItem(s.TransactionId(), payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity, false); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"template_id":    payload.Item.TemplateId,
					}).Error("Reverse-walk: AwardAsset → DestroyItem dispatch failed; continuing chain.")
				}
			}
		case CreateAndEquipAsset:
			if payload, ok := step.Payload().(CreateAndEquipAssetPayload); ok {
				if err := c.compP.RequestDestroyItem(s.TransactionId(), payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity, false); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"template_id":    payload.Item.TemplateId,
					}).Error("Reverse-walk: CreateAndEquipAsset → DestroyItem dispatch failed; continuing chain.")
				}
			}
		case CreateSkill:
			if payload, ok := step.Payload().(CreateSkillPayload); ok {
				if err := c.skillP.RequestDeleteSkill(s.TransactionId(), payload.WorldId, payload.CharacterId, payload.SkillId); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"skill_id":       payload.SkillId,
					}).Error("Reverse-walk: CreateSkill → DeleteSkill dispatch failed; continuing chain.")
				}
			}
		case CreateCharacter:
			// Defer to the end — deleting the character first would orphan
			// item/skill inverses still in flight.
			sCopy := step
			deleteCharacterStep = &sCopy
		}
	}

	if deleteCharacterStep != nil && characterId != 0 {
		if err := c.charP.RequestDeleteCharacter(s.TransactionId(), characterId, worldId); err != nil {
			c.l.WithError(err).WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"step_id":        deleteCharacterStep.StepId(),
				"character_id":   characterId,
			}).Error("Reverse-walk: CreateCharacter → DeleteCharacter dispatch failed; continuing chain.")
		}
	}
}

// extractCharacterCreationWorldId reads the WorldId out of the CharacterCreate
// step's payload. Returns 0 if the step is not present.
func extractCharacterCreationWorldId(s Saga) world.Id {
	for _, step := range s.Steps() {
		if step.Action() != CreateCharacter {
			continue
		}
		if p, ok := step.Payload().(CharacterCreatePayload); ok {
			return p.WorldId
		}
	}
	return 0
}
