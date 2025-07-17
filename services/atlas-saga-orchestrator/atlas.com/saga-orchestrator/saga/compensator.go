package saga

import (
	"atlas-saga-orchestrator/character"
	"atlas-saga-orchestrator/compartment"
	"atlas-saga-orchestrator/guild"
	"atlas-saga-orchestrator/invite"
	"atlas-saga-orchestrator/skill"
	"atlas-saga-orchestrator/validation"
	"context"
	"fmt"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"strings"
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
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"tenant_id":      c.t.Id().String(),
		}).Debug("No failed step found for compensation.")
		return nil
	}

	failedStep := s.Steps[failedStepIndex]

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        failedStep.StepId,
		"action":         failedStep.Action,
		"tenant_id":      c.t.Id().String(),
	}).Debug("Compensating failed step.")

	// Perform compensation based on the action type
	switch failedStep.Action {
	case EquipAsset:
		return c.compensateEquipAsset(s, failedStep)
	case UnequipAsset:
		return c.compensateUnequipAsset(s, failedStep)
	case CreateCharacter:
		return c.compensateCreateCharacter(s, failedStep)
	case CreateAndEquipAsset:
		return c.compensateCreateAndEquipAsset(s, failedStep)
	default:
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_id":        failedStep.StepId,
			"action":         failedStep.Action,
			"tenant_id":      c.t.Id().String(),
		}).Debug("No compensation logic available for action type.")
		// Mark step as compensated (remove failed status) with validation
		if err := s.SetStepStatus(failedStepIndex, Pending); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"step_index":     failedStepIndex,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to mark step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := s.ValidateStateConsistency(); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after compensation")
			return err
		}

		GetCache().Put(c.t.Id(), s)
		return nil
	}
}

// compensateEquipAsset handles compensation for a failed EquipAsset operation
// by performing the reverse operation (UnequipAsset)
func (c *CompensatorImpl) compensateEquipAsset(s Saga, failedStep Step[any]) error {
	// Extract the original payload
	payload, ok := failedStep.Payload.(EquipAssetPayload)
	if !ok {
		return fmt.Errorf("invalid payload for EquipAsset compensation")
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        failedStep.StepId,
		"character_id":   payload.CharacterId,
		"source":         payload.Source,
		"destination":    payload.Destination,
		"tenant_id":      c.t.Id().String(),
	}).Info("Compensating failed EquipAsset operation with UnequipAsset")

	// Perform the reverse operation: unequip from destination back to source
	err := c.compP.RequestUnequipAsset(s.TransactionId, payload.CharacterId, byte(payload.InventoryType), payload.Destination, payload.Source)
	if err != nil {
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_id":        failedStep.StepId,
			"tenant_id":      c.t.Id().String(),
		}).WithError(err).Error("Failed to compensate EquipAsset operation")
		return err
	}

	// Mark the failed step as compensated by removing it from the saga
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex != -1 {
		if err := s.SetStepStatus(failedStepIndex, Pending); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"step_index":     failedStepIndex,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to mark EquipAsset step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := s.ValidateStateConsistency(); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after EquipAsset compensation")
			return err
		}

		GetCache().Put(c.t.Id(), s)
	}

	return nil
}

// compensateUnequipAsset handles compensation for a failed UnequipAsset operation
// by performing the reverse operation (EquipAsset)
func (c *CompensatorImpl) compensateUnequipAsset(s Saga, failedStep Step[any]) error {
	// Extract the original payload
	payload, ok := failedStep.Payload.(UnequipAssetPayload)
	if !ok {
		return fmt.Errorf("invalid payload for UnequipAsset compensation")
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        failedStep.StepId,
		"character_id":   payload.CharacterId,
		"source":         payload.Source,
		"destination":    payload.Destination,
		"tenant_id":      c.t.Id().String(),
	}).Info("Compensating failed UnequipAsset operation with EquipAsset")

	// Perform the reverse operation: equip from destination back to source
	err := c.compP.RequestEquipAsset(s.TransactionId, payload.CharacterId, byte(payload.InventoryType), payload.Destination, payload.Source)
	if err != nil {
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_id":        failedStep.StepId,
			"tenant_id":      c.t.Id().String(),
		}).WithError(err).Error("Failed to compensate UnequipAsset operation")
		return err
	}

	// Mark the failed step as compensated by removing it from the saga
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex != -1 {
		if err := s.SetStepStatus(failedStepIndex, Pending); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"step_index":     failedStepIndex,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to mark UnequipAsset step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := s.ValidateStateConsistency(); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after UnequipAsset compensation")
			return err
		}

		GetCache().Put(c.t.Id(), s)
	}

	return nil
}

// compensateCreateCharacter handles compensation for a failed CreateCharacter operation
// Note: Character creation failures typically do not require compensation as the character
// creation process is atomic. If partial creation occurred, the character service should
// handle cleanup. This function exists for completeness and future extensibility.
func (c *CompensatorImpl) compensateCreateCharacter(s Saga, failedStep Step[any]) error {
	// Extract the original payload
	payload, ok := failedStep.Payload.(CharacterCreatePayload)
	if !ok {
		return fmt.Errorf("invalid payload for CreateCharacter compensation")
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        failedStep.StepId,
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
		if err := s.SetStepStatus(failedStepIndex, Pending); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"step_index":     failedStepIndex,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to mark CreateCharacter step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := s.ValidateStateConsistency(); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after CreateCharacter compensation")
			return err
		}

		GetCache().Put(c.t.Id(), s)
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
	payload, ok := failedStep.Payload.(CreateAndEquipAssetPayload)
	if !ok {
		return fmt.Errorf("invalid payload for CreateAndEquipAsset compensation")
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId.String(),
		"saga_type":      s.SagaType,
		"step_id":        failedStep.StepId,
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
	for _, step := range s.Steps {
		if step.Action == EquipAsset && strings.HasPrefix(step.StepId, "auto_equip_step_") {
			autoEquipStepExists = true
			break
		}
	}

	if autoEquipStepExists {
		// Asset was created but equipment failed - need to destroy the created asset
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_id":        failedStep.StepId,
			"character_id":   payload.CharacterId,
			"template_id":    payload.Item.TemplateId,
			"quantity":       payload.Item.Quantity,
			"tenant_id":      c.t.Id().String(),
		}).Info("Auto-equip step found - destroying created asset for compensation")

		// Destroy the created asset
		err := c.compP.RequestDestroyItem(s.TransactionId, payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity)
		if err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"step_id":        failedStep.StepId,
				"character_id":   payload.CharacterId,
				"template_id":    payload.Item.TemplateId,
				"quantity":       payload.Item.Quantity,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to destroy created asset during CreateAndEquipAsset compensation")
			return err
		}

		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_id":        failedStep.StepId,
			"character_id":   payload.CharacterId,
			"template_id":    payload.Item.TemplateId,
			"quantity":       payload.Item.Quantity,
			"tenant_id":      c.t.Id().String(),
		}).Info("Successfully destroyed created asset during CreateAndEquipAsset compensation")
	} else {
		// No auto-equip step found - asset creation failed, no compensation needed
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId.String(),
			"saga_type":      s.SagaType,
			"step_id":        failedStep.StepId,
			"character_id":   payload.CharacterId,
			"template_id":    payload.Item.TemplateId,
			"quantity":       payload.Item.Quantity,
			"tenant_id":      c.t.Id().String(),
		}).Info("No auto-equip step found - asset creation failed, no compensation needed")
	}

	// Mark the failed step as compensated
	failedStepIndex := s.FindFailedStepIndex()
	if failedStepIndex != -1 {
		if err := s.SetStepStatus(failedStepIndex, Pending); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"step_index":     failedStepIndex,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("Failed to mark CreateAndEquipAsset step as compensated")
			return err
		}

		// Validate state consistency before updating cache
		if err := s.ValidateStateConsistency(); err != nil {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId.String(),
				"saga_type":      s.SagaType,
				"tenant_id":      c.t.Id().String(),
			}).WithError(err).Error("State consistency validation failed after CreateAndEquipAsset compensation")
			return err
		}

		GetCache().Put(c.t.Id(), s)
	}

	return nil
}
