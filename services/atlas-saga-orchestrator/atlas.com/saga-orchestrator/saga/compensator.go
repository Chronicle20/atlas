package saga

import (
	"atlas-saga-orchestrator/cashshop"
	"atlas-saga-orchestrator/character"
	"atlas-saga-orchestrator/compartment"
	"atlas-saga-orchestrator/guild"
	"atlas-saga-orchestrator/invite"
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	character2 "atlas-saga-orchestrator/kafka/message/character"
	sagaMsg "atlas-saga-orchestrator/kafka/message/saga"
	"atlas-saga-orchestrator/mts"
	"atlas-saga-orchestrator/skill"
	"atlas-saga-orchestrator/validation"
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"strings"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Compensator interface {
	WithCharacterProcessor(character.Processor) Compensator
	WithCompartmentProcessor(compartment.Processor) Compensator
	WithSkillProcessor(skill.Processor) Compensator
	WithValidationProcessor(validation.Processor) Compensator
	WithGuildProcessor(guild.Processor) Compensator
	WithInviteProcessor(invite.Processor) Compensator
	WithCashshopProcessor(cashshop.Processor) Compensator
	WithMtsProcessor(mts.Processor) Compensator

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
	compensatePetEvolution(s Saga, failedStep Step[any]) error
	compensateCashItemUse(s Saga, failedStep Step[any]) error
	compensatePointReset(s Saga, failedStep Step[any]) error
	compensateMtsOperation(s Saga, failedStep Step[any]) error

	// DispatchMtsOperationRollbacks reverse-walks the completed steps of an MTS
	// saga (TransferToMts / WithdrawFromMts / MtsSettlePurchase) and dispatches the
	// inverse for each: AwardCurrency → negated re-credit/debit, ReleaseFromCharacter
	// → AcceptToCharacter (re-grant), ReleaseFromMtsHolding → RestoreMtsHolding,
	// AcceptToMtsListing → DestroyItem-not-needed (handled by atomic tx; see code).
	// No lifecycle transitions, no Failed emission, no cache eviction — callers
	// handle those. This is the dupe-safety core (design §4.1).
	DispatchMtsOperationRollbacks(s Saga)

	// DispatchCharacterCreationRollbacks is the dispatch half of the reverse-walk
	// compensator. It fires the inverse commands (DestroyItem / DeleteSkill /
	// DeleteCharacter-last) for each completed step of a CharacterCreation saga.
	// No lifecycle transitions, no Failed emission, no cache eviction — callers
	// handle those. Used both by the step-driven compensator and by the timer-
	// fire path in saga/timer.go (PRD §4.3 / plan Phase 4.3).
	DispatchCharacterCreationRollbacks(s Saga)

	// DispatchPetEvolutionRollbacks reverse-walks the completed steps of a
	// PetEvolution saga, refunding the destroyed Rock (DestroyAsset → CreateItem)
	// and the deducted mesos (AwardMesos → inverse credit). No lifecycle
	// transitions, no Failed emission, no cache eviction — callers handle those.
	DispatchPetEvolutionRollbacks(s Saga)

	// DispatchCashItemUseRollbacks reverse-walks the completed steps of a
	// cash-item-use saga (ItemTagUse/SealingLockUse/IncubatorUse), re-creating
	// every consumed item (DestroyAsset/DestroyAssetFromSlot → CreateItem) and
	// destroying every awarded result (AwardAsset → DestroyItem). No lifecycle
	// transitions, no Failed emission, no cache eviction — callers handle those.
	DispatchCashItemUseRollbacks(s Saga)

	// DispatchPointResetRollbacks reverse-walks the completed steps of a
	// point_reset saga, re-awarding the destroyed AP/SP Reset item
	// (DestroyAsset → CreateItem). No lifecycle transitions, no Failed emission,
	// no cache eviction — callers handle those.
	DispatchPointResetRollbacks(s Saga)

	// CompensateLateStep dispatches the single-step inverse for a step whose
	// success event arrived after the saga went terminal (PRD §4.3, design
	// §3.4/§3.5). Pure dispatch — no lifecycle transitions, no Failed
	// emission, no cache eviction. Claim-then-dispatch: the lateCompensated
	// marker is persisted BEFORE the inverse goes out, giving at-most-once
	// rollback, because the negation inverses (mesos/currency/exp/fame) are
	// not idempotent downstream — at-least-once would double-refund. A crash
	// between claim and dispatch loses the rollback but is auditable via the
	// saga_terminal log + span emitted by the caller. Returns true only when
	// an inverse command was dispatched by this call.
	CompensateLateStep(s Saga, step Step[any]) (bool, error)
}

type CompensatorImpl struct {
	l         logrus.FieldLogger
	ctx       context.Context
	t         tenant.Model
	charP     character.Processor
	compP     compartment.Processor
	skillP    skill.Processor
	validP    validation.Processor
	guildP    guild.Processor
	inviteP   invite.Processor
	cashshopP cashshop.Processor
	mtsP      mts.Processor
}

func NewCompensator(l logrus.FieldLogger, ctx context.Context) Compensator {
	return &CompensatorImpl{
		l:         l,
		ctx:       ctx,
		t:         tenant.MustFromContext(ctx),
		charP:     character.NewProcessor(l, ctx),
		compP:     compartment.NewProcessor(l, ctx),
		skillP:    skill.NewProcessor(l, ctx),
		validP:    validation.NewProcessor(l, ctx),
		guildP:    guild.NewProcessor(l, ctx),
		inviteP:   invite.NewProcessor(l, ctx),
		cashshopP: cashshop.NewProcessor(l, ctx),
		mtsP:      mts.NewProcessor(l, ctx),
	}
}

// copy returns a shallow clone of the compensator so the With* setters can
// override a single processor without re-listing every field at each call site.
func (c *CompensatorImpl) copy() *CompensatorImpl {
	cp := *c
	return &cp
}

func (c *CompensatorImpl) WithCharacterProcessor(charP character.Processor) Compensator {
	n := c.copy()
	n.charP = charP
	return n
}

func (c *CompensatorImpl) WithCompartmentProcessor(compP compartment.Processor) Compensator {
	n := c.copy()
	n.compP = compP
	return n
}

func (c *CompensatorImpl) WithSkillProcessor(skillP skill.Processor) Compensator {
	n := c.copy()
	n.skillP = skillP
	return n
}

func (c *CompensatorImpl) WithValidationProcessor(validP validation.Processor) Compensator {
	n := c.copy()
	n.validP = validP
	return n
}

func (c *CompensatorImpl) WithGuildProcessor(guildP guild.Processor) Compensator {
	n := c.copy()
	n.guildP = guildP
	return n
}

func (c *CompensatorImpl) WithInviteProcessor(inviteP invite.Processor) Compensator {
	n := c.copy()
	n.inviteP = inviteP
	return n
}

func (c *CompensatorImpl) WithCashshopProcessor(cashshopP cashshop.Processor) Compensator {
	n := c.copy()
	n.cashshopP = cashshopP
	return n
}

func (c *CompensatorImpl) WithMtsProcessor(mtsP mts.Processor) Compensator {
	n := c.copy()
	n.mtsP = mtsP
	return n
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

	// Pet-evolution reverse-walk (plan Task 18). A failed evolve_pet must refund
	// the already-completed destroy_item (the Rock) and award_mesos (the cost)
	// rather than only compensating the failed step.
	if s.SagaType() == PetEvolution {
		return c.compensatePetEvolution(s, failedStep)
	}

	// Cash-item-use reverse-walk (Task 10). A failed item_tag_use /
	// sealing_lock_use / incubator_use must refund the already-completed
	// consume steps (the tagged/sealed/incubated item) and undo any awarded
	// result rather than only compensating the failed step.
	if s.SagaType() == ItemTagUse || s.SagaType() == SealingLockUse || s.SagaType() == IncubatorUse {
		return c.compensateCashItemUse(s, failedStep)
	}

	// Point-reset reverse-walk (task-126, shape B). A destroy-first saga:
	// invert the already-completed destroy_asset via re-award, then emit the
	// saga-failed event carrying the service's machine-readable error code
	// (threaded via the failed step's result map) so atlas-channel can render
	// specific pink text (Task 14).
	if s.SagaType() == PointReset {
		return c.compensatePointReset(s, failedStep)
	}

	// MTS reverse-walk (task-102 §4.1 — the dupe-safety core). A failed
	// TransferToMts / WithdrawFromMts / MtsSettlePurchase must undo every
	// already-completed step so exactly one custody copy of the item exists at
	// every instant and currency nets to zero, rather than only compensating the
	// failed step.
	if s.SagaType() == MtsOperation {
		return c.compensateMtsOperation(s, failedStep)
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

		if err := GetCache().Put(c.ctx, updatedSaga); err != nil {
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

		if err := GetCache().Put(c.ctx, updatedSaga); err != nil {
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

		if err := GetCache().Put(c.ctx, updatedSaga); err != nil {
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

		if err := GetCache().Put(c.ctx, updatedSaga); err != nil {
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

		if err := GetCache().Put(c.ctx, updatedSaga); err != nil {
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

		if err := GetCache().Put(c.ctx, updatedSaga); err != nil {
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

		if err := GetCache().Put(c.ctx, updatedSaga); err != nil {
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

		if err := GetCache().Put(c.ctx, updatedSaga); err != nil {
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

// compensatePetEvolution is the pet-evolution reverse-walk compensator (plan
// Task 18). On a failed evolve_pet it walks the saga's completed steps in
// reverse and refunds the destroyed Rock (DestroyAsset → CreateItem) and the
// deducted mesos (AwardMesos → inverse credit), emits exactly one
// StatusEventTypeFailed, cancels the Phase-4 timer, and evicts the saga.
//
// Double-emission is prevented by TryTransition(Compensating → Failed): if the
// timer already emitted Failed, the transition is refused and this function
// returns without re-emitting. Mirrors compensateCharacterCreation.
func (c *CompensatorImpl) compensatePetEvolution(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"failed_step":    failedStep.StepId(),
		"failed_action":  failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Info("PetEvolution saga failing — dispatching reverse-walk compensation.")

	c.DispatchPetEvolutionRollbacks(s)

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

	reason := fmt.Sprintf("Pet evolution failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if err := EmitSagaFailed(c.l, c.ctx, s, sagaMsg.ErrorCodeUnknown, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event after pet-evolution compensation.")
		return err
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"tenant_id":      c.t.Id().String(),
	}).Info("Pet-evolution reverse-walk compensation complete; saga terminated.")
	return nil
}

// DispatchPetEvolutionRollbacks reverse-walks the saga's completed steps and
// dispatches the inverse compensation command for each. This is the pure
// "dispatch" half — no lifecycle transitions, no event emission, no cache
// eviction. Callers are responsible for those.
//
// Inverses:
//   - DestroyAsset (Rock destroyed)  → CreateItem (refund the Rock).
//   - AwardMesos   (negative cost)   → AwardMesos with -Amount (re-credit the
//     player so they net back to even). The consume step deducts with a
//     negative amount; negating it restores the mesos.
//
// evolve_pet produced no committed mutation on failure, so it has no inverse.
// An error refunding one step does not abort the chain.
func (c *CompensatorImpl) DispatchPetEvolutionRollbacks(s Saga) {
	steps := s.Steps()
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Status() != Completed {
			continue
		}
		switch step.Action() {
		case DestroyAsset:
			if payload, ok := step.Payload().(DestroyAssetPayload); ok {
				qty := payload.Quantity
				if qty == 0 {
					qty = 1
				}
				if err := c.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, time.Time{}); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"template_id":    payload.TemplateId,
					}).Error("Reverse-walk: DestroyAsset → CreateItem dispatch failed; continuing chain.")
				}
			}
		case AwardMesos:
			if payload, ok := step.Payload().(AwardMesosPayload); ok {
				ch := channel.NewModel(payload.WorldId, payload.ChannelId)
				if err := c.charP.AwardMesosAndEmit(s.TransactionId(), ch, payload.CharacterId, payload.CharacterId, "SYSTEM", -payload.Amount, false); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"amount":         payload.Amount,
					}).Error("Reverse-walk: AwardMesos refund dispatch failed; continuing chain.")
				}
			}
		}
	}
}

// compensateCashItemUse is the reverse-walk compensator for cash-item-use
// sagas (ItemTagUse/SealingLockUse/IncubatorUse — Task 10). On a failed step
// (e.g. the terminal incubator_result emit) it walks the saga's completed
// steps in reverse, re-creating consumed items and destroying awarded
// results, emits exactly one StatusEventTypeFailed, cancels the Phase-4
// timer, and evicts the saga. The FAILED event is what triggers the channel's
// INCUBATOR_RESULT(0) announcement.
//
// Double-emission is prevented by TryTransition(Compensating → Failed): if the
// timer already emitted Failed, the transition is refused and this function
// returns without re-emitting. Mirrors compensatePetEvolution.
func (c *CompensatorImpl) compensateCashItemUse(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"failed_step":    failedStep.StepId(),
		"failed_action":  failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Info("Cash-item-use saga failing — dispatching reverse-walk compensation.")

	c.DispatchCashItemUseRollbacks(s)

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

	reason := fmt.Sprintf("Cash item use (%s) failed at step [%s] action [%s]", s.SagaType(), failedStep.StepId(), failedStep.Action())
	if err := EmitSagaFailed(c.l, c.ctx, s, sagaMsg.ErrorCodeUnknown, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event after cash-item-use compensation.")
		return err
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"tenant_id":      c.t.Id().String(),
	}).Info("Cash-item-use reverse-walk compensation complete; saga terminated.")
	return nil
}

// DispatchCashItemUseRollbacks reverse-walks the saga's completed steps and
// dispatches the inverse compensation command for each. This is the pure
// "dispatch" half — no lifecycle transitions, no event emission, no cache
// eviction. Callers are responsible for those.
//
// Inverses:
//   - DestroyAsset (item consumed by templateId) → CreateItem (refund it).
//   - DestroyAssetFromSlot (item consumed from a specific slot, e.g. the tag/
//     seal item or the incubator's sacrificed target) → CreateItem, using the
//     TemplateId carried on the payload. A payload with no TemplateId is
//     skipped (nothing to re-create) rather than issuing a zero-templateId
//     create.
//   - AwardAsset (a granted result, e.g. the incubator's produced item)  →
//     DestroyItem (mirrors DispatchCharacterCreationRollbacks's AwardAsset
//     inverse).
//
// An error refunding one step does not abort the chain.
func (c *CompensatorImpl) DispatchCashItemUseRollbacks(s Saga) {
	steps := s.Steps()
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Status() != Completed {
			continue
		}
		switch step.Action() {
		case DestroyAsset:
			if payload, ok := step.Payload().(DestroyAssetPayload); ok {
				qty := payload.Quantity
				if qty == 0 {
					qty = 1
				}
				if err := c.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, time.Time{}); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"template_id":    payload.TemplateId,
					}).Error("Reverse-walk: DestroyAsset -> CreateItem dispatch failed; continuing chain.")
				}
			}
		case DestroyAssetFromSlot:
			if payload, ok := step.Payload().(DestroyAssetFromSlotPayload); ok {
				if payload.TemplateId == 0 {
					c.l.WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
					}).Error("Reverse-walk: DestroyAssetFromSlot payload has no templateId; cannot re-create.")
					continue
				}
				qty := payload.Quantity
				if qty == 0 {
					qty = 1
				}
				if err := c.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, time.Time{}); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"template_id":    payload.TemplateId,
					}).Error("Reverse-walk: DestroyAssetFromSlot -> CreateItem dispatch failed; continuing chain.")
				}
			}
		case AwardAsset:
			if payload, ok := step.Payload().(AwardItemActionPayload); ok {
				if err := c.compP.RequestDestroyItem(s.TransactionId(), payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity, false); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"template_id":    payload.Item.TemplateId,
					}).Error("Reverse-walk: AwardAsset -> DestroyItem dispatch failed; continuing chain.")
				}
			}
		}
	}
}

// compensatePointReset is the point-reset reverse-walk compensator (task-126,
// design §3 shape B). On a failed transfer_ap / transfer_sp it re-awards the
// already-consumed AP/SP Reset item (destroy-first saga) and emits exactly one
// StatusEventTypeFailed carrying the service's machine-readable error code and
// detail, threaded off the failed step's result map (Task 14 contract:
// reason = errorDetail). Mirrors compensatePetEvolution for the lifecycle
// idioms — TryTransition(Compensating → Failed) guards against a double-emit
// where the Phase-4 timer already emitted Failed.
func (c *CompensatorImpl) compensatePointReset(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"failed_step":    failedStep.StepId(),
		"failed_action":  failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Info("PointReset saga failing — dispatching reverse-walk compensation.")

	c.DispatchPointResetRollbacks(s)

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

	errorCode, reason := pointResetFailureFields(failedStep)
	if err := EmitSagaFailed(c.l, c.ctx, s, errorCode, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event after point-reset compensation.")
		return err
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"tenant_id":      c.t.Id().String(),
	}).Info("Point-reset reverse-walk compensation complete; saga terminated.")
	return nil
}

// pointResetFailureFields extracts the machine-readable error code and the
// human/detail reason to place on the saga-failed event from a failed
// point_reset step. Per the Task 14 error-threading contract, the failed
// step's result map carries `errorCode` + `errorDetail`; reason is the
// errorDetail (the channel branch reads Body.Reason as the detail carrier,
// e.g. the offending stat name). Falls back to ErrorCodeUnknown + a generic
// reason when the result map lacks the keys.
func pointResetFailureFields(failedStep Step[any]) (string, string) {
	errorCode := sagaMsg.ErrorCodeUnknown
	reason := fmt.Sprintf("Point reset failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if res := failedStep.Result(); res != nil {
		if v, ok := res["errorCode"].(string); ok && v != "" {
			errorCode = v
		}
		if v, ok := res["errorDetail"].(string); ok && v != "" {
			reason = v
		}
	}
	return errorCode, reason
}

// DispatchPointResetRollbacks reverse-walks the saga's completed steps and
// re-awards each destroyed AP/SP Reset item (DestroyAsset → CreateItem). This
// is the pure "dispatch" half — no lifecycle transitions, no event emission,
// no cache eviction. Only Completed destroy steps are inverted; the failed
// transfer step produced no committed mutation and has no inverse. An error
// re-awarding one step does not abort the chain.
func (c *CompensatorImpl) DispatchPointResetRollbacks(s Saga) {
	steps := s.Steps()
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Status() != Completed {
			continue
		}
		if step.Action() != DestroyAsset {
			continue
		}
		if payload, ok := step.Payload().(DestroyAssetPayload); ok {
			qty := payload.Quantity
			if qty == 0 {
				qty = 1
			}
			if err := c.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, time.Time{}); err != nil {
				c.l.WithError(err).WithFields(logrus.Fields{
					"transaction_id": s.TransactionId().String(),
					"step_id":        step.StepId(),
					"template_id":    payload.TemplateId,
				}).Error("Reverse-walk: DestroyAsset → CreateItem dispatch failed; continuing chain.")
			}
		}
	}
}

// compensateMtsOperation is the MTS reverse-walk compensator (task-102 §4.1 —
// the dupe-safety core). On a failed TransferToMts / WithdrawFromMts /
// MtsSettlePurchase it walks the saga's completed steps in reverse, dispatches
// the inverse for each, emits exactly one StatusEventTypeFailed, cancels the
// Phase-4 timer, and evicts the saga.
//
// Double-emission is prevented by TryTransition(Compensating → Failed): if the
// timer already emitted Failed, the transition is refused and this function
// returns without re-emitting. Mirrors compensatePetEvolution.
func (c *CompensatorImpl) compensateMtsOperation(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"failed_step":    failedStep.StepId(),
		"failed_action":  failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Info("MTS saga failing — dispatching reverse-walk compensation.")

	c.DispatchMtsOperationRollbacks(s)

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

	reason := fmt.Sprintf("MTS operation failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if err := EmitSagaFailed(c.l, c.ctx, s, sagaMsg.ErrorCodeUnknown, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event after MTS compensation.")
		return err
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"tenant_id":      c.t.Id().String(),
	}).Info("MTS reverse-walk compensation complete; saga terminated.")
	return nil
}

// DispatchMtsOperationRollbacks reverse-walks the saga's completed steps and
// dispatches the inverse compensation command for each. This is the pure
// "dispatch" half — no lifecycle transitions, no event emission, no cache
// eviction. Callers are responsible for those. An error dispatching one inverse
// does not abort the chain.
//
// Inverses (design §4.1):
//   - AwardCurrency (settlement debit/credit) → AwardCurrency with -Amount: the
//     buyer debit (negative amount) re-credits, the seller credit (positive
//     amount) debits. Net currency change is zero. REUSES the cash-shop wallet
//     dispatch — no duplicate command.
//   - ReleaseFromCharacter (TransferToMts: item left inventory) → re-grant the
//     item to the character via RequestAcceptAsset, reconstructing the equip
//     snapshot from the saga's AcceptToMtsListing step so stats survive.
//   - ReleaseFromMtsHolding (WithdrawFromMts: holding soft-deleted) →
//     RestoreMtsHolding (un-soft-delete the same holding row).
//
// Steps that committed no compensable mutation have no inverse:
//   - AcceptToMtsListing failing leaves no listing row (its own atomic tx rolled
//     back), so there is nothing to un-accept; the ReleaseFromCharacter inverse
//     above re-grants the item.
//   - MtsMoveListingToHolding failing leaves the listing `active` with no buyer
//     holding (its own atomic tx rolled back), so there is nothing to un-move;
//     only the two AwardCurrency steps need reversal. It is the LAST settlement
//     step, so it is never a Completed-then-compensated step.
func (c *CompensatorImpl) DispatchMtsOperationRollbacks(s Saga) {
	// Locate the AcceptToMtsListing snapshot (if any) so a ReleaseFromCharacter
	// inverse can re-grant with the original equip stats.
	var listingSnapshot *AcceptToMtsListingPayload
	for _, step := range s.Steps() {
		if step.Action() != AcceptToMtsListing {
			continue
		}
		if p, ok := step.Payload().(AcceptToMtsListingPayload); ok {
			pc := p
			listingSnapshot = &pc
			break
		}
	}

	steps := s.Steps()
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Status() != Completed {
			continue
		}
		switch step.Action() {
		case AwardCurrency:
			if payload, ok := step.Payload().(AwardCurrencyPayload); ok {
				if err := c.cashshopP.AwardCurrencyAndEmit(s.TransactionId(), payload.AccountId, payload.CurrencyType, -payload.Amount); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"account_id":     payload.AccountId,
						"amount":         payload.Amount,
					}).Error("Reverse-walk: AwardCurrency reversal dispatch failed; continuing chain.")
				}
			}
		case ReleaseFromCharacter:
			if payload, ok := step.Payload().(ReleaseFromCharacterPayload); ok {
				if listingSnapshot == nil {
					c.l.WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"character_id":   payload.CharacterId,
					}).Error("Reverse-walk: ReleaseFromCharacter has no AcceptToMtsListing snapshot to re-grant; skipping.")
					continue
				}
				assetData := assetDataFromMtsListingSnapshot(*listingSnapshot)
				if err := c.compP.RequestAcceptAsset(s.TransactionId(), payload.CharacterId, payload.InventoryType, listingSnapshot.TemplateId, assetData); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"character_id":   payload.CharacterId,
						"template_id":    listingSnapshot.TemplateId,
					}).Error("Reverse-walk: ReleaseFromCharacter → AcceptToCharacter re-grant dispatch failed; continuing chain.")
				}
			}
		case ReleaseFromMtsHolding:
			if payload, ok := step.Payload().(ReleaseFromMtsHoldingPayload); ok {
				if err := c.mtsP.RestoreMtsHoldingAndEmit(s.TransactionId(), payload.HoldingId); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"holding_id":     payload.HoldingId.String(),
					}).Error("Reverse-walk: ReleaseFromMtsHolding → RestoreMtsHolding dispatch failed; continuing chain.")
				}
			}
		}
	}
}

// assetDataFromMtsListingSnapshot reconstructs an inventory AssetData from the
// item snapshot carried on an AcceptToMtsListing step, so a TransferToMts
// compensation re-grants the released item with its original equip stats intact.
func assetDataFromMtsListingSnapshot(p AcceptToMtsListingPayload) asset2.AssetData {
	return asset2.AssetData{
		Quantity:      p.Quantity,
		Strength:      p.Strength,
		Dexterity:     p.Dexterity,
		Intelligence:  p.Intelligence,
		Luck:          p.Luck,
		Hp:            p.HP,
		Mp:            p.MP,
		WeaponAttack:  p.WeaponAttack,
		MagicAttack:   p.MagicAttack,
		WeaponDefense: p.WeaponDefense,
		MagicDefense:  p.MagicDefense,
		Accuracy:      p.Accuracy,
		Avoidability:  p.Avoidability,
		Hands:         p.Hands,
		Speed:         p.Speed,
		Jump:          p.Jump,
		Slots:         p.Slots,
		LevelType:     p.ItemLevel,
		Level:         p.Level,
		Experience:    p.ItemExp,
		Flag:          p.Flags,
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

// lateCompensableActions is the v1 compensable set (design §3.4): the full
// value-transfer class that broke the task-102 invariant. Everything else is
// absorb-only and logged as late_effect_unrecoverable when hit.
// DestroyAssetFromSlot is deliberately absent: its payload carries no
// TemplateId, so the destroyed item cannot be recreated from the step alone.
//
// MTS custody actions (task-102) all have late inverses:
//   - ReleaseFromMtsHolding (take-home): RestoreMtsHolding un-soft-deletes the
//     holding so a late release doesn't orphan the item.
//   - AcceptToMtsListing (list): RemoveMtsListing hard-deletes the spurious
//     still-active listing a late accept created after the list saga's
//     compensation already re-granted the item to the seller (the guard is
//     state=active, so a listing acted on in the interim is left alone).
//   - MtsMoveListingToHolding (buy): RestoreListingFromHolding soft-deletes the
//     deterministic buyer holding and returns the listing sold->active, so a buy
//     that lands late after the buyer's prepaid was refunded delivers no free
//     item. (The currency legs are AwardCurrency, already covered.)
//
// task-136 removed the timeout trigger, so a late MTS custody success is now rare.
var lateCompensableActions = map[Action]struct{}{
	AwardAsset:              {},
	CreateAndEquipAsset:     {},
	CreateSkill:             {},
	CreateCharacter:         {},
	AwaitCharacterCreated:   {},
	DestroyAsset:            {},
	AwardMesos:              {},
	AwardCurrency:           {},
	AwardExperience:         {},
	DeductExperience:        {},
	AwardFame:               {},
	EquipAsset:              {},
	UnequipAsset:            {},
	ReleaseFromMtsHolding:   {},
	AcceptToMtsListing:      {},
	MtsMoveListingToHolding: {},
}

func (c *CompensatorImpl) CompensateLateStep(s Saga, step Step[any]) (bool, error) {
	fields := logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        step.StepId(),
		"step_action":    step.Action(),
		"tenant_id":      c.t.Id().String(),
	}

	if _, ok := lateCompensableActions[step.Action()]; !ok {
		fields["reason"] = "late_effect_unrecoverable"
		c.l.WithFields(fields).Warn("Late-successful step has no registered inverse; its effect is orphaned.")
		return false, nil
	}

	// DestroyAsset with RemoveAll=true is not recoverable from the step
	// payload: Quantity is 0/unset because the destroy consumed "everything"
	// rather than an explicit count. Recreating a fabricated quantity would
	// silently under- (or over-) refund the player, so route this into the
	// same absorb-only path as a non-compensable action. Explicit-quantity
	// DestroyAsset steps (RemoveAll=false) still compensate normally.
	if step.Action() == DestroyAsset {
		if payload, ok := step.Payload().(DestroyAssetPayload); ok && payload.RemoveAll {
			fields["reason"] = "late_effect_unrecoverable"
			c.l.WithFields(fields).Warn("Late-successful DestroyAsset used RemoveAll; destroyed quantity is not recoverable from the step payload, its effect is orphaned.")
			return false, nil
		}
	}

	claimed, err := c.claimLateCompensation(s.TransactionId(), step.StepId())
	if err != nil {
		return false, err
	}
	if !claimed {
		c.l.WithFields(fields).Debug("Late-success compensation already claimed; duplicate delivery ignored.")
		return false, nil
	}

	if err := c.dispatchLateInverse(s, step); err != nil {
		// The claim is already persisted: at-most-once means we do NOT retry
		// dispatch on a later redelivery. Log loudly for the audit trail.
		fields["reason"] = "late_effect_dispatch_failed"
		c.l.WithFields(fields).WithError(err).Error("Late-success inverse dispatch failed after claim.")
		return false, err
	}

	fields["reason"] = "late_effect_compensated"
	c.l.WithFields(fields).Info("Late-successful step routed into compensation; effect rolled back.")
	return true, nil
}

// claimLateCompensation atomically sets the step's lateCompensated marker.
// Returns false when the marker was already set (duplicate delivery). Only
// the goroutine whose Put wins the optimistic-version race proceeds to
// dispatch; losers re-read and observe the marker.
func (c *CompensatorImpl) claimLateCompensation(transactionId uuid.UUID, stepId string) (bool, error) {
	for attempt := 1; attempt <= maxConflictRetries; attempt++ {
		s, ok := GetCache().GetById(c.ctx, transactionId)
		if !ok {
			return false, errors.New("saga not found while claiming late compensation")
		}
		index := -1
		for i, st := range s.Steps() {
			if st.StepId() == stepId {
				index = i
				break
			}
		}
		if index == -1 {
			return false, fmt.Errorf("step [%s] not found while claiming late compensation", stepId)
		}
		st, _ := s.StepAt(index)
		if st.LateCompensated() {
			return false, nil
		}
		updated, err := s.WithStepLateCompensated(index)
		if err != nil {
			return false, err
		}
		err = GetCache().Put(c.ctx, updated)
		if err == nil {
			return true, nil
		}
		if !isVersionConflict(err) {
			return false, err
		}
	}
	return false, fmt.Errorf("max retries exceeded claiming late compensation for saga %s", transactionId.String())
}

// dispatchLateInverse fires the single-step inverse computed from the STEP
// payload (never the event payload), reusing the reverse-walk idioms.
func (c *CompensatorImpl) dispatchLateInverse(s Saga, step Step[any]) error {
	switch step.Action() {
	case AwardAsset:
		payload, ok := step.Payload().(AwardItemActionPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late AwardAsset compensation")
		}
		return c.compP.RequestDestroyItem(s.TransactionId(), payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity, false)
	case CreateAndEquipAsset:
		payload, ok := step.Payload().(CreateAndEquipAssetPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late CreateAndEquipAsset compensation")
		}
		return c.compP.RequestDestroyItem(s.TransactionId(), payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity, false)
	case CreateSkill:
		payload, ok := step.Payload().(CreateSkillPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late CreateSkill compensation")
		}
		return c.skillP.RequestDeleteSkill(s.TransactionId(), payload.WorldId, payload.CharacterId, payload.SkillId)
	case CreateCharacter, AwaitCharacterCreated:
		_, characterId := ExtractCharacterCreationIds(s)
		worldId := extractCharacterCreationWorldId(s)
		if characterId == 0 {
			return fmt.Errorf("late character-creation compensation: character id unresolved")
		}
		return c.charP.RequestDeleteCharacter(s.TransactionId(), characterId, worldId)
	case DestroyAsset:
		// RemoveAll=true is excluded upstream in CompensateLateStep (payload
		// carries no recoverable quantity), so only explicit-quantity destroys
		// reach here — recreate exactly what was destroyed.
		payload, ok := step.Payload().(DestroyAssetPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late DestroyAsset compensation")
		}
		return c.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, payload.Quantity, time.Time{})
	case AwardMesos:
		payload, ok := step.Payload().(AwardMesosPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late AwardMesos compensation")
		}
		ch := channel.NewModel(payload.WorldId, payload.ChannelId)
		return c.charP.AwardMesosAndEmit(s.TransactionId(), ch, payload.CharacterId, payload.CharacterId, "SYSTEM", -payload.Amount, false)
	case AwardCurrency:
		payload, ok := step.Payload().(AwardCurrencyPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late AwardCurrency compensation")
		}
		return c.cashshopP.AwardCurrencyAndEmit(s.TransactionId(), payload.AccountId, payload.CurrencyType, -payload.Amount)
	case AwardExperience:
		payload, ok := step.Payload().(AwardExperiencePayload)
		if !ok {
			return fmt.Errorf("invalid payload for late AwardExperience compensation")
		}
		var total uint32
		for _, d := range payload.Distributions {
			total += d.Amount
		}
		ch := channel.NewModel(payload.WorldId, payload.ChannelId)
		return c.charP.DeductExperienceAndEmit(s.TransactionId(), ch, payload.CharacterId, total)
	case DeductExperience:
		payload, ok := step.Payload().(DeductExperiencePayload)
		if !ok {
			return fmt.Errorf("invalid payload for late DeductExperience compensation")
		}
		ch := channel.NewModel(payload.WorldId, payload.ChannelId)
		return c.charP.AwardExperienceAndEmit(s.TransactionId(), ch, payload.CharacterId,
			[]character2.ExperienceDistributions{{ExperienceType: "WHITE", Amount: payload.Amount}}, false)
	case AwardFame:
		payload, ok := step.Payload().(AwardFamePayload)
		if !ok {
			return fmt.Errorf("invalid payload for late AwardFame compensation")
		}
		ch := channel.NewModel(payload.WorldId, payload.ChannelId)
		return c.charP.AwardFameAndEmit(s.TransactionId(), ch, payload.CharacterId, -payload.Amount)
	case EquipAsset:
		payload, ok := step.Payload().(EquipAssetPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late EquipAsset compensation")
		}
		return c.compP.RequestUnequipAsset(s.TransactionId(), payload.CharacterId, byte(payload.InventoryType), payload.Destination, payload.Source)
	case UnequipAsset:
		payload, ok := step.Payload().(UnequipAssetPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late UnequipAsset compensation")
		}
		return c.compP.RequestEquipAsset(s.TransactionId(), payload.CharacterId, byte(payload.InventoryType), payload.Destination, payload.Source)
	case ReleaseFromMtsHolding:
		// A take-home that soft-deleted the holding but landed late after the
		// saga terminated: un-soft-delete the holding so the item stays in MTS
		// (recoverable) rather than orphaned. Same inverse the reverse-walk uses.
		payload, ok := step.Payload().(ReleaseFromMtsHoldingPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late ReleaseFromMtsHolding compensation")
		}
		return c.mtsP.RestoreMtsHoldingAndEmit(s.TransactionId(), payload.HoldingId)
	case AcceptToMtsListing:
		// A late list-accept created the listing after the saga's compensation
		// already re-granted the item to the seller (release_from_character always
		// precedes accept, so its inverse ran): remove the now-duplicate listing.
		// The atlas-mts guard deletes only a still-active listing.
		payload, ok := step.Payload().(AcceptToMtsListingPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late AcceptToMtsListing compensation")
		}
		return c.mtsP.RemoveMtsListingAndEmit(s.TransactionId(), payload.ListingId)
	case MtsMoveListingToHolding:
		// A late settlement-move delivered the item to the buyer's holding and
		// marked the listing sold after the buyer's prepaid was already refunded:
		// soft-delete the buyer holding and return the listing to active so the
		// buyer keeps nothing and the item is re-listed.
		payload, ok := step.Payload().(MtsMoveListingToHoldingPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late MtsMoveListingToHolding compensation")
		}
		return c.mtsP.RestoreListingFromHoldingAndEmit(s.TransactionId(), payload.ListingId, payload.BuyerId)
	}
	return fmt.Errorf("no late inverse registered for action %s", step.Action())
}
