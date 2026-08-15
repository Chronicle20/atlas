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
	"atlas-saga-orchestrator/pet"
	"atlas-saga-orchestrator/skill"
	"atlas-saga-orchestrator/trade"
	"atlas-saga-orchestrator/validation"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	WithTradeProcessor(trade.Processor) Compensator
	WithPetProcessor(pet.Processor) Compensator

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
	compensateMesoSackUse(s Saga, failedStep Step[any]) error
	compensatePetNameTagUse(s Saga, failedStep Step[any]) error
	compensateMtsOperation(s Saga, failedStep Step[any]) error
	compensateNoteSend(s Saga, failedStep Step[any]) error
	compensateSkillBookUse(s Saga, failedStep Step[any]) error
	compensateTradeTransaction(s Saga, failedStep Step[any]) error

	// DispatchTradeTransactionRollbacks reverse-walks the completed steps of a
	// trade_settlement saga (task-205) and dispatches the inverse for each:
	// AwardMesos → negated re-credit/debit, AcceptToCharacter → DestroyItem,
	// ReleaseFromCharacter → AcceptToCharacter (re-grant from the paired
	// accept's snapshot). Without it a settlement that fails partway leaves a
	// HALF-SWAP — one side's goods moved, the other's destroyed. Each inverse is
	// claimed once via the per-step lateCompensated marker, so a repeat walk
	// cannot double-credit. No lifecycle transitions, no Failed emission, no
	// cache eviction — callers handle those.
	DispatchTradeTransactionRollbacks(s Saga)

	// DispatchMtsOperationRollbacks reverse-walks the completed steps of an MTS
	// saga (TransferToMts / WithdrawFromMts / MtsSettlePurchase) and dispatches the
	// inverse for each: AwardCurrency → negated re-credit/debit, ReleaseFromCharacter
	// → AcceptToCharacter (re-grant), ReleaseFromMtsHolding → RestoreMtsHolding,
	// AcceptToMtsListing → DestroyItem-not-needed (handled by atomic tx; see code).
	// No lifecycle transitions, no Failed emission, no cache eviction — callers
	// handle those. This is the dupe-safety core (design §4.1).
	DispatchMtsOperationRollbacks(s Saga)

	// DispatchTradeStagingRollbacks reverse-walks the completed steps of a
	// trade-STAGING saga (transfer_to_trade). Exported for the same reason the
	// two above are: the tests drive it directly, avoiding the EmitSagaFailed
	// Kafka path.
	DispatchTradeStagingRollbacks(s Saga)

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

	// DispatchNoteSendRollbacks reverse-walks the completed steps of a
	// note_send saga, refunding the destroyed Note item (DestroyAsset →
	// CreateItem). A failed consume_note_item (destroy) step has nothing
	// completed to refund. No lifecycle transitions, no Failed emission, no
	// cache eviction — callers handle those.
	DispatchNoteSendRollbacks(s Saga)

	// DispatchCashItemUseRollbacks reverse-walks the completed steps of a
	// cash-item-use saga (ItemTagUse/SealingLockUse/IncubatorUse/
	// KarmaScissorsUse/RemoteMerchant/ScriptedItemUse/RemoteNpcUse), re-creating
	// every consumed item (DestroyAsset/DestroyAssetFromSlot → CreateItem),
	// destroying every awarded result (AwardAsset → DestroyItem), clearing every
	// applied karma mark (ApplyAssetKarma → clear), closing every opened shop
	// (OpenNpcShop → EXIT), and ending every opened dialogue
	// (StartItemConversation/StartNpcConversation → END_CONVERSATION). No
	// lifecycle transitions, no Failed emission, no cache eviction — callers
	// handle those.
	DispatchCashItemUseRollbacks(s Saga)

	// DispatchPointResetRollbacks reverse-walks the completed steps of a
	// point_reset saga, re-awarding the destroyed AP/SP Reset item
	// (DestroyAsset → CreateItem). No lifecycle transitions, no Failed emission,
	// no cache eviction — callers handle those.
	DispatchPointResetRollbacks(s Saga)

	// DispatchMesoSackRollbacks reverse-walks the completed steps of a
	// meso_sack_use saga and refunds the consumed sack (DestroyAsset →
	// CreateItem). The failed award_mesos step committed nothing
	// (RequestChangeMeso rejects inside its own transaction) and has no
	// inverse. No lifecycle transitions, no Failed emission, no cache
	// eviction — callers handle those.
	DispatchMesoSackRollbacks(s Saga)

	// DispatchPetNameTagRollbacks reverse-walks the completed steps of a
	// pet_name_tag_use saga and reverts the pet's name by re-issuing RENAME
	// with the PreviousName captured at build time. Pure dispatch half — no
	// lifecycle transitions, no Failed emission, no cache eviction.
	//
	// Known, accepted limitation: if some other actor renames the pet between
	// step 1 and this revert, the revert restores a stale name. A rename is
	// player-initiated, serialized by the client's exclusive-request gate, and
	// the window is one Kafka round trip — a compare-and-swap revert keyed on
	// the applied name buys almost nothing for real complexity (design §3.7).
	DispatchPetNameTagRollbacks(s Saga)

	// DispatchSkillBookUseRollbacks reverse-walks the completed steps of a
	// skill_book_use saga and re-awards the destroyed book (task-125). Pure
	// dispatch half — no lifecycle transitions, no event emission.
	DispatchSkillBookUseRollbacks(s Saga)

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
	tradeP    trade.Processor
	petP      pet.Processor
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
		tradeP:    trade.NewProcessor(l, ctx),
		petP:      pet.NewProcessor(l, ctx),
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

func (c *CompensatorImpl) WithTradeProcessor(tradeP trade.Processor) Compensator {
	n := c.copy()
	n.tradeP = tradeP
	return n
}

func (c *CompensatorImpl) WithPetProcessor(petP pet.Processor) Compensator {
	n := c.copy()
	n.petP = petP
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

	// Cash-item-use reverse-walk (Task 10; expiration_extender_use added
	// task-222, remote_merchant added task-221, karma_scissors_use added
	// task-223, scripted_item_use/remote_npc_use added task-230). A failed
	// item_tag_use / sealing_lock_use / incubator_use /
	// expiration_extender_use / remote_merchant / karma_scissors_use /
	// scripted_item_use / remote_npc_use must refund the already-completed
	// consume steps (the tagged/sealed/incubated/extender item, or the
	// destroyed scissors) and undo any awarded result or applied karma mark —
	// or close any opened shop, or end any opened dialogue — rather than only
	// compensating the failed step.
	if s.SagaType() == ItemTagUse || s.SagaType() == SealingLockUse || s.SagaType() == IncubatorUse || s.SagaType() == ExpirationExtenderUse || s.SagaType() == RemoteMerchant || s.SagaType() == KarmaScissorsUse || s.SagaType() == ScriptedItemUse || s.SagaType() == RemoteNpcUse {
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

	// Meso-sack reverse-walk. Destroy-first, like point_reset: invert the
	// completed consume_meso_sack via re-award, then emit the saga-failed event
	// carrying the character id (EmitSagaFailed's meso-sack arm) and the
	// threaded error code, so atlas-channel can render the ceiling message and
	// release the client's exclusive-request gate.
	if s.SagaType() == MesoSackUse {
		return c.compensateMesoSackUse(s, failedStep)
	}

	// Pet-name-tag reverse-walk. Rename-first, consume-second: invert the
	// completed rename_pet step by reverting to the PreviousName captured at
	// build time, then emit the saga-failed event carrying the character id
	// so atlas-channel can release the client's exclusive-request gate.
	if s.SagaType() == PetNameTagUse {
		return c.compensatePetNameTagUse(s, failedStep)
	}

	// MTS reverse-walk (task-102 §4.1 — the dupe-safety core). A failed
	// TransferToMts / WithdrawFromMts / MtsSettlePurchase must undo every
	// already-completed step so exactly one custody copy of the item exists at
	// every instant and currency nets to zero, rather than only compensating the
	// failed step.
	if s.SagaType() == MtsOperation {
		return c.compensateMtsOperation(s, failedStep)
	}

	// Trade reverse-walk (task-205). A settlement is a two-party swap: without a
	// reverse-walk a failure partway through leaves a HALF-SWAP — release A,
	// release B, accept→B, then accept→A fails means A's item is soft-deleted
	// and B holds both. The per-action switch below routes those steps to
	// compensateStorageOperation, which by its own contract performs NO
	// rollback, so the trade arm must be taken ahead of it.
	if s.SagaType() == TradeTransaction {
		return c.compensateTradeTransaction(s, failedStep)
	}

	// Trade-staging reverse-walk (task-205 amendment, design §5A.4). Staging is
	// the single-item release+accept shape, so it mirrors the MTS arm above
	// rather than the two-party swap arm: a failed accept_to_trade must re-grant
	// the already-released asset, or the player's item is destroyed by staging
	// it. Must be taken ahead of the per-action switch, which routes
	// ReleaseFromCharacter to compensateStorageOperation — a no-rollback path.
	if s.SagaType() == TradeStaging {
		return c.compensateTradeStaging(s, failedStep)
	}

	// Note-send reverse-walk: a failed create_note must refund the
	// already-destroyed Note item; a failed consume_note_item has nothing to
	// refund. Either way the saga terminates with one Failed emission so the
	// channel can announce SEND_ERROR (unlocking the client).
	if s.SagaType() == NoteSend {
		return c.compensateNoteSend(s, failedStep)
	}

	// Skill-book reverse-walk (task-125). A failed create_skill/update_skill
	// must re-award the already-destroyed book rather than only compensating
	// the failed step; a failed destroy step has nothing to reverse.
	if s.SagaType() == SkillBookUse {
		return c.compensateSkillBookUse(s, failedStep)
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

// compensateSkillBookUse is the skill_book_use reverse-walk compensator
// (task-125). On a failed create_skill/update_skill it re-awards the book
// destroyed by the completed destroy_asset_from_slot step (using the
// payload's TemplateId), emits exactly one StatusEventTypeFailed, cancels
// the Phase-4 timer, and evicts the saga. A failed destroy step (first
// step) has no completed steps to reverse — the walk is a no-op and the
// saga just terminates with the failed event.
//
// Double-emission is prevented by TryTransition(Compensating → Failed): if
// the timer already emitted Failed, the transition is refused and this
// function returns without re-emitting. Mirrors compensatePetEvolution.
func (c *CompensatorImpl) compensateSkillBookUse(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"failed_step":    failedStep.StepId(),
		"failed_action":  failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Info("SkillBookUse saga failing — dispatching reverse-walk compensation.")

	c.DispatchSkillBookUseRollbacks(s)

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

	reason := fmt.Sprintf("Skill book use failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if err := EmitSagaFailed(c.l, c.ctx, s, sagaMsg.ErrorCodeUnknown, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event after skill-book compensation.")
		return err
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"tenant_id":      c.t.Id().String(),
	}).Info("Skill-book reverse-walk compensation complete; saga terminated.")
	return nil
}

// DispatchSkillBookUseRollbacks reverse-walks the saga's completed steps and
// re-awards each completed destroy_asset_from_slot via CreateItem using the
// payload's TemplateId. Pure dispatch half — callers own lifecycle/emission.
// Slot position is not preserved (the freed slot guarantees space). A destroy
// payload without TemplateId (legacy producer) cannot be re-awarded and is
// skipped with an error log. An error re-awarding one step does not abort the
// chain.
func (c *CompensatorImpl) DispatchSkillBookUseRollbacks(s Saga) {
	steps := s.Steps()
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Status() != Completed {
			continue
		}
		if step.Action() != DestroyAssetFromSlot {
			continue
		}
		payload, ok := step.Payload().(DestroyAssetFromSlotPayload)
		if !ok {
			continue
		}
		if payload.TemplateId == 0 {
			c.l.WithFields(logrus.Fields{
				"transaction_id": s.TransactionId().String(),
				"step_id":        step.StepId(),
				"tenant_id":      c.t.Id().String(),
			}).Error("Reverse-walk: destroy step carries no TemplateId; cannot re-award the book.")
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
}

// compensateCashItemUse is the reverse-walk compensator for cash-item-use
// sagas (ItemTagUse/SealingLockUse/IncubatorUse — Task 10; RemoteMerchant —
// task-221; KarmaScissorsUse — task-223). On a failed step (e.g. the terminal
// incubator_result emit, or consume_remote_merchant_item) it walks the saga's
// completed steps in reverse, re-creating consumed items, destroying awarded
// results, clearing any applied karma mark, and closing any opened shop,
// emits exactly one StatusEventTypeFailed, cancels the Phase-4 timer, and
// evicts the saga. The FAILED event is what triggers the channel's
// INCUBATOR_RESULT(0) announcement (or, for remote_merchant, the shop's
// already-dispatched EXIT).
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
//   - ApplyAssetKarma (target marked one-trade-enabled) → RequestApplyKarma(clear=true).
//   - OpenNpcShop (a remote-merchant shop opened before the consume step —
//     task-221 FR-4.5) → EXIT, via EmitNpcShopExit, so the player is not
//     left standing in a shop they did not pay for.
//   - StartItemConversation / StartNpcConversation (a scripted-item or
//     remote-npc dialogue opened before the destroy step — task-230) →
//     END_CONVERSATION, via EmitNpcConversationEnd. The destroy step is the
//     last step for these sagas, so the only path reaching this arm is
//     "dialogue opened, destroy failed" — a UI teardown, not an item restore.
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
		case ApplyAssetKarma:
			// Inverse of a completed mark: clear it. A saga that failed after the
			// mark was applied must not leave a free trade behind (FR-6.6).
			if payload, ok := step.Payload().(ApplyAssetKarmaPayload); ok {
				if err := c.compP.RequestApplyKarma(s.TransactionId(), payload.CharacterId, payload.InventoryType, payload.Slot, payload.ScissorsKarma, true); err != nil {
					c.l.WithError(err).Errorf("Unable to clear the karma mark for character [%d] in inventory [%d] slot [%d] during compensation.", payload.CharacterId, payload.InventoryType, payload.Slot)
				}
			}
		case OpenNpcShop:
			// The inverse of "opened a shop" is "close it". Without this a
			// failed destroy step leaves the player standing in a shop they
			// did not pay for (task-221 FR-4.5).
			if payload, ok := step.Payload().(OpenNpcShopPayload); ok {
				if err := EmitNpcShopExit(c.l, c.ctx, s.TransactionId(), payload.CharacterId); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"character_id":   payload.CharacterId,
					}).Error("Reverse-walk: OpenNpcShop -> EXIT dispatch failed; continuing chain.")
				}
			}
		case StartItemConversation:
			// The inverse of "opened a dialogue" is "close it". Because the
			// destroy is the LAST step, the only path that reaches here is
			// "conversation opened, destroy failed" — rare, and its
			// compensation is a UI teardown rather than an item restore. That
			// asymmetry is the point of the conversation-first ordering.
			if payload, ok := step.Payload().(StartItemConversationPayload); ok {
				if err := EmitNpcConversationEnd(c.l, c.ctx, s.TransactionId(), payload.CharacterId, payload.NpcTemplateId); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"character_id":   payload.CharacterId,
						"item_id":        payload.ItemId,
					}).Error("Reverse-walk: StartItemConversation -> END_CONVERSATION dispatch failed; continuing chain.")
				}
			}
		case StartNpcConversation:
			if payload, ok := step.Payload().(StartNpcConversationPayload); ok {
				if err := EmitNpcConversationEnd(c.l, c.ctx, s.TransactionId(), payload.CharacterId, payload.NpcTemplateId); err != nil {
					c.l.WithError(err).WithFields(logrus.Fields{
						"transaction_id": s.TransactionId().String(),
						"step_id":        step.StepId(),
						"character_id":   payload.CharacterId,
					}).Error("Reverse-walk: StartNpcConversation -> END_CONVERSATION dispatch failed; continuing chain.")
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

// compensateMesoSackUse is the meso_sack_use reverse-walk compensator: on a
// failed award_mesos it refunds the already-consumed sack and emits exactly one
// StatusEventTypeFailed carrying atlas-character's machine-readable error code
// (threaded off the failed step's result map by handleCharacterMesoErrorEvent).
// TryTransition(Compensating → Failed) guards against a double-emit where the
// timeout backstop already emitted Failed.
func (c *CompensatorImpl) compensateMesoSackUse(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"failed_step":    failedStep.StepId(),
		"failed_action":  failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Info("MesoSackUse saga failing — dispatching reverse-walk compensation.")

	c.DispatchMesoSackRollbacks(s)

	if !GetCache().TryTransition(c.ctx, s.TransactionId(), SagaLifecycleCompensating, SagaLifecycleFailed) {
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Info("saga already in terminal Failed state; meso-sack emission skipped.")
		SagaTimers().Cancel(s.TransactionId())
		GetCache().Remove(c.ctx, s.TransactionId())
		return nil
	}

	SagaTimers().Cancel(s.TransactionId())
	GetCache().Remove(c.ctx, s.TransactionId())

	errorCode := mesoSackErrorCode(failedStep)
	reason := fmt.Sprintf("Meso sack use failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if err := EmitSagaFailed(c.l, c.ctx, s, errorCode, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event after meso-sack compensation.")
		return err
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"tenant_id":      c.t.Id().String(),
	}).Info("Meso-sack reverse-walk compensation complete; saga terminated.")
	return nil
}

// mesoSackErrorCode reads the machine-readable code atlas-character supplied
// (MESO_OVERFLOW / NOT_ENOUGH_MESO) off the failed step's result map. A
// destroy-step failure or a timeout has no such map, so the channel renders the
// generic message rather than falsely claiming a meso ceiling.
func mesoSackErrorCode(failedStep Step[any]) string {
	if res := failedStep.Result(); res != nil {
		if v, ok := res["errorCode"].(string); ok && v != "" {
			return v
		}
	}
	return sagaMsg.ErrorCodeUnknown
}

// mesoSackCharacterId resolves the character to notify. The AwardMesos payload
// is present on every meso_sack_use saga by construction; the DestroyAsset
// payload is the belt-and-braces fallback (same shape as compensateNoteSend).
func mesoSackCharacterId(s Saga) uint32 {
	for _, step := range s.Steps() {
		if step.Action() == AwardMesos {
			if id := ExtractCharacterId(step); id != 0 {
				return id
			}
		}
	}
	for _, step := range s.Steps() {
		if id := ExtractCharacterId(step); id != 0 {
			return id
		}
	}
	return 0
}

// DispatchMesoSackRollbacks reverse-walks the saga's completed steps and
// refunds the consumed sack (DestroyAsset → RequestCreateItem). Pure dispatch
// half — no lifecycle transitions, no event emission, no cache eviction. Only
// Completed destroy steps are inverted; the failed award step committed nothing
// and has no inverse. The refund lands in the first free CASH slot, matching
// every other refund path — DestroyAsset is template-keyed, not slot-keyed.
// An error refunding one step does not abort the walk.
func (c *CompensatorImpl) DispatchMesoSackRollbacks(s Saga) {
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
				}).Error("Reverse-walk: meso sack DestroyAsset → CreateItem dispatch failed; continuing chain.")
			}
		}
	}
}

// compensatePetNameTagUse is the pet_name_tag_use reverse-walk compensator:
// on a failed consume_pet_name_tag it reverts the already-completed rename
// (the rename-first, consume-second ordering means the tag was never spent,
// but the pet's name was already applied) and emits exactly one
// StatusEventTypeFailed. TryTransition(Compensating → Failed) guards against
// a double-emit where the timeout backstop already emitted Failed.
func (c *CompensatorImpl) compensatePetNameTagUse(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"failed_step":    failedStep.StepId(),
		"failed_action":  failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Info("PetNameTagUse saga failing — dispatching reverse-walk compensation.")

	c.DispatchPetNameTagRollbacks(s)

	if !GetCache().TryTransition(c.ctx, s.TransactionId(), SagaLifecycleCompensating, SagaLifecycleFailed) {
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Info("saga already in terminal Failed state; pet-name-tag emission skipped.")
		SagaTimers().Cancel(s.TransactionId())
		GetCache().Remove(c.ctx, s.TransactionId())
		return nil
	}

	SagaTimers().Cancel(s.TransactionId())
	GetCache().Remove(c.ctx, s.TransactionId())

	reason := fmt.Sprintf("Pet name tag use failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if err := EmitSagaFailed(c.l, c.ctx, s, sagaMsg.ErrorCodeUnknown, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event after pet-name-tag compensation.")
		return err
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"tenant_id":      c.t.Id().String(),
	}).Info("Pet-name-tag reverse-walk compensation complete; saga terminated.")
	return nil
}

// petNameTagCharacterId resolves the character to notify. The RenamePet
// payload is present on every pet_name_tag_use saga by construction; the
// generic ExtractCharacterId walk is the belt-and-braces fallback (same
// shape as mesoSackCharacterId).
func petNameTagCharacterId(s Saga) uint32 {
	for _, step := range s.Steps() {
		if step.Action() == RenamePet {
			if payload, ok := step.Payload().(RenamePetPayload); ok && payload.CharacterId != 0 {
				return payload.CharacterId
			}
		}
	}
	for _, step := range s.Steps() {
		if id := ExtractCharacterId(step); id != 0 {
			return id
		}
	}
	return 0
}

// DispatchPetNameTagRollbacks reverse-walks the saga's completed steps and
// reverts the pet's name by re-issuing RENAME with the PreviousName captured
// at build time (RenamePet → RenameAndEmit). Pure dispatch half — no
// lifecycle transitions, no event emission, no cache eviction. Only a
// Completed rename step is inverted; a rename step that never completed
// applied nothing and has no inverse. An error reverting the rename does not
// abort the walk.
func (c *CompensatorImpl) DispatchPetNameTagRollbacks(s Saga) {
	steps := s.Steps()
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Status() != Completed {
			continue
		}
		if step.Action() != RenamePet {
			continue
		}
		if payload, ok := step.Payload().(RenamePetPayload); ok {
			if err := c.petP.RenameAndEmit(s.TransactionId(), payload.PetId, payload.CharacterId, payload.PreviousName); err != nil {
				c.l.WithError(err).WithFields(logrus.Fields{
					"transaction_id": s.TransactionId().String(),
					"step_id":        step.StepId(),
					"pet_id":         payload.PetId,
				}).Error("Reverse-walk: pet name tag RenamePet revert dispatch failed; continuing chain.")
			}
		}
	}
}

// compensateNoteSend terminates a failing note_send saga: dispatches the
// item refund, then emits exactly one StatusEventTypeFailed carrying the
// SENDER's characterId (the channel's saga consumer announces MEMO_RESULT
// SEND_ERROR to that session, which also releases the client's
// exclusive-request lock). Double-emission is prevented by
// TryTransition(Compensating → Failed), mirroring compensatePetEvolution.
//
// EmitSagaFailed is not used here: it extracts characterId via
// ExtractCharacterCreationIds, which only recognizes a CreateCharacter step
// and would yield 0 for a note_send saga. The sender id is instead read
// directly off the CreateNote step's payload (falling back to the failed
// DestroyAsset step's CharacterId when create_note itself never ran).
func (c *CompensatorImpl) compensateNoteSend(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"failed_step":    failedStep.StepId(),
		"failed_action":  failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Info("NoteSend saga failing — dispatching compensation.")

	c.DispatchNoteSendRollbacks(s)

	// The sender's characterId rides in the Failed event so the channel can
	// notify the right session.
	var senderId uint32
	for _, step := range s.Steps() {
		if p, ok := step.Payload().(CreateNotePayload); ok {
			senderId = p.SenderId
			break
		}
	}
	if senderId == 0 {
		if payload, ok := failedStep.Payload().(DestroyAssetPayload); ok {
			senderId = payload.CharacterId
		}
	}

	if !GetCache().TryTransition(c.ctx, s.TransactionId(), SagaLifecycleCompensating, SagaLifecycleFailed) {
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Info("saga already in terminal Failed state; note-send emission skipped.")
		SagaTimers().Cancel(s.TransactionId())
		GetCache().Remove(c.ctx, s.TransactionId())
		return nil
	}

	SagaTimers().Cancel(s.TransactionId())
	GetCache().Remove(c.ctx, s.TransactionId())

	reason := fmt.Sprintf("Note send failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if err := EmitSagaFailedByIds(c.l, c.ctx, s.TransactionId(), string(s.SagaType()), 0, senderId, sagaMsg.ErrorCodeUnknown, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event after note-send compensation.")
		return err
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"tenant_id":      c.t.Id().String(),
	}).Info("Note-send compensation complete; saga terminated.")
	return nil
}

// DispatchNoteSendRollbacks reverse-walks a note_send saga's completed steps
// and refunds the destroyed Note item (DestroyAsset → RequestCreateItem).
// This is the pure "dispatch" half — no lifecycle transitions, no event
// emission, no cache eviction. Callers are responsible for those. Only
// Completed destroy steps are inverted; a failed consume_note_item produced
// no committed mutation and has no inverse. An error refunding does not
// abort the walk.
func (c *CompensatorImpl) DispatchNoteSendRollbacks(s Saga) {
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

// compensateTradeTransaction is the trade_settlement reverse-walk compensator
// (task-205). A settlement moves goods in BOTH directions, so unlike the
// storage flows it can fail in a state where value has already changed hands:
// release A, release B, accept→B, accept→A fails leaves A's item soft-deleted
// and B holding both. compensateStorageOperation — where the per-action switch
// would otherwise route these steps — explicitly performs no rollback, so this
// arm is taken ahead of it in CompensateFailedStep.
//
// Mirrors compensateMtsOperation: dispatch the reverse-walk, then transition
// Compensating → Failed so the timeout backstop and this path cannot both emit.
func (c *CompensatorImpl) compensateTradeTransaction(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"failed_step":    failedStep.StepId(),
		"failed_action":  failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Info("Trade saga failing — dispatching reverse-walk compensation.")

	c.DispatchTradeTransactionRollbacks(s)

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

	reason := fmt.Sprintf("Trade settlement failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if err := EmitSagaFailed(c.l, c.ctx, s, DetermineErrorCode(s, failedStep), reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event after trade compensation.")
		return err
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"tenant_id":      c.t.Id().String(),
	}).Info("Trade reverse-walk compensation complete; saga terminated.")
	return nil
}

// DispatchTradeTransactionRollbacks reverse-walks the saga's COMPLETED steps and
// dispatches the inverse of each. This is the pure "dispatch" half — no
// lifecycle transitions, no event emission, no cache eviction. Callers own
// those. An error dispatching one inverse does not abort the chain.
//
// The walk is driven entirely by the recorded step list and each step's own
// payload; nothing is re-derived from the trade_settlement composite (which the
// expansion already replaced). Because it runs newest-first, accepts are undone
// before the matching releases are re-granted — the reverse of the forward
// order, so the recipient's copy is destroyed before the owner's is restored.
//
// Inverses:
//   - AwardMesos → AwardMesos with -Amount. The giver's negative deduction
//     re-credits and the receiver's positive credit debits, so the pair nets to
//     zero and the destroyed tax is never re-minted.
//   - AcceptToCharacter → RequestDestroyItem(templateId, quantity). The asset
//     the accept created has an id the orchestrator never learns —
//     compartment.AcceptedEventBody carries only a transactionId
//     (kafka/message/compartment/kafka.go) — so template+quantity is the only
//     available inverse. For stackables it is exact. For an equip it removes AN
//     instance of that template, which for a recipient who already owned one
//     could pick the wrong instance; the item COUNT is still conserved, and the
//     alternative is permanent loss.
//   - ReleaseFromTrade → RestoreTradeEscrow, un-soft-deleting the custody row.
//     Under escrow-at-staging (design §5A.7) a settlement releases from ESCROW,
//     not from a character, so the inverse is a custody restore rather than a
//     re-grant: the item goes back to being escrowed, and whichever teardown
//     follows returns it to its owner from there. That is strictly safer than
//     re-granting here, because it cannot race the accept that may already have
//     delivered the same item to the counterparty.
//
// Idempotency: each step's inverse is claimed via claimLateCompensation before
// dispatch — the same per-step, optimistic-version marker the late-success path
// uses. A second run of the walk (or a late-success delivery for the same step)
// finds the marker set and dispatches nothing, so no leg can double-credit. A
// claim that ERRORS (saga already evicted, version-conflict retries exhausted)
// means once-only cannot be guaranteed, so the inverse is skipped and logged at
// Error rather than risking a duplicate — the same at-most-once posture
// CompensateLateStep takes.
func (c *CompensatorImpl) DispatchTradeTransactionRollbacks(s Saga) {
	steps := s.Steps()
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Status() != Completed {
			continue
		}
		switch step.Action() {
		case AwardMesos:
			payload, ok := step.Payload().(AwardMesosPayload)
			if !ok {
				continue
			}
			if !c.claimTradeRollback(s, step) {
				continue
			}
			ch := channel.NewModel(payload.WorldId, payload.ChannelId)
			if err := c.charP.AwardMesosAndEmit(s.TransactionId(), ch, payload.CharacterId, payload.CharacterId, "SYSTEM", -payload.Amount, false); err != nil {
				c.l.WithError(err).WithFields(logrus.Fields{
					"transaction_id": s.TransactionId().String(),
					"step_id":        step.StepId(),
					"character_id":   payload.CharacterId,
					"amount":         payload.Amount,
				}).Error("Reverse-walk: trade AwardMesos reversal dispatch failed; continuing chain.")
			}
		case AcceptToCharacter:
			payload, ok := step.Payload().(AcceptToCharacterPayload)
			if !ok {
				continue
			}
			qty := payload.AssetData.Quantity
			if qty == 0 {
				qty = 1
			}
			if !c.claimTradeRollback(s, step) {
				continue
			}
			if err := c.compP.RequestDestroyItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, false); err != nil {
				c.l.WithError(err).WithFields(logrus.Fields{
					"transaction_id": s.TransactionId().String(),
					"step_id":        step.StepId(),
					"character_id":   payload.CharacterId,
					"template_id":    payload.TemplateId,
				}).Error("Reverse-walk: trade AcceptToCharacter → DestroyItem dispatch failed; continuing chain.")
			}
		case ReleaseFromTrade:
			payload, ok := step.Payload().(ReleaseFromTradePayload)
			if !ok {
				continue
			}
			if !c.claimTradeRollback(s, step) {
				continue
			}
			if err := c.tradeP.RestoreTradeEscrowAndEmit(s.TransactionId(), payload.EscrowId); err != nil {
				c.l.WithError(err).WithFields(logrus.Fields{
					"transaction_id": s.TransactionId().String(),
					"step_id":        step.StepId(),
					"escrow_id":      payload.EscrowId.String(),
				}).Error("Reverse-walk: trade ReleaseFromTrade → RestoreTradeEscrow dispatch failed; continuing chain.")
			}
		}
	}
}

// claimTradeRollback takes the once-only claim for one step's inverse. Returns
// false when the inverse was already dispatched (duplicate walk or a late
// success that already compensated) or when the claim could not be established
// at all — in both cases the caller must NOT dispatch.
func (c *CompensatorImpl) claimTradeRollback(s Saga, step Step[any]) bool {
	claimed, err := c.claimLateCompensation(s.TransactionId(), step.StepId())
	if err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"step_id":        step.StepId(),
			"step_action":    step.Action(),
			"tenant_id":      c.t.Id().String(),
			"reason":         "trade_rollback_claim_failed",
		}).Error("Reverse-walk: could not claim trade rollback; skipping inverse to preserve at-most-once.")
		return false
	}
	if !claimed {
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"step_id":        step.StepId(),
			"step_action":    step.Action(),
		}).Debug("Reverse-walk: trade rollback already claimed; inverse skipped.")
		return false
	}
	return true
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
		Owner:         p.Owner,
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
// DestroyAssetFromSlot is included: since task-128 its payload carries a
// TemplateId, so a late-successful destroy can be recreated via CreateItem
// (see the payload-level guard in CompensateLateStep and the
// DestroyAssetFromSlot case in dispatchLateInverse). A payload with
// TemplateId==0 (legacy producer) still has no recoverable quantity and is
// routed into the same absorb-only path as DestroyAsset+RemoveAll.
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
	DestroyAssetFromSlot:    {},
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

// tradeLateCompensableActions extends lateCompensableActions for TradeTransaction
// ONLY (task-205). These two actions are shared with the storage, cash-shop and
// MTS flows, so they are deliberately NOT added to the global map: registering
// them there would silently give every one of those saga types a destroy /
// custody-restore it does not have today.
//
// Why trade needs them at all: the settlement reverse-walk skips a step that is
// not yet Completed, so a settlement that times out with an accept IN FLIGHT
// re-grants the item to its owner and then the ACCEPTED event lands — the
// recipient keeps a copy and the owner has one too. Without a late inverse that
// is item DUPLICATION. Symmetrically a late-successful release is silent LOSS.
// This is the same class MTS closed by registering its three custody actions.
var tradeLateCompensableActions = map[Action]struct{}{
	AcceptToCharacter: {},
	ReleaseFromTrade:  {},
}

// isLateCompensable reports whether a late-successful step has a registered
// inverse. The base set applies to every saga; the trade extension applies only
// to TradeTransaction, so no other saga type's absorb behaviour changes.
func isLateCompensable(s Saga, action Action) bool {
	if _, ok := lateCompensableActions[action]; ok {
		return true
	}
	if s.SagaType() == TradeTransaction {
		_, ok := tradeLateCompensableActions[action]
		return ok
	}
	return false
}

func (c *CompensatorImpl) CompensateLateStep(s Saga, step Step[any]) (bool, error) {
	fields := logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"saga_type":      s.SagaType(),
		"step_id":        step.StepId(),
		"step_action":    step.Action(),
		"tenant_id":      c.t.Id().String(),
	}

	if !isLateCompensable(s, step.Action()) {
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

	// DestroyAssetFromSlot with TemplateId==0 is a legacy-producer payload:
	// there is nothing to recreate, so absorb-only (same shape as the
	// DestroyAsset+RemoveAll guard above) rather than awarding item 0.
	if step.Action() == DestroyAssetFromSlot {
		if payload, ok := step.Payload().(DestroyAssetFromSlotPayload); ok && payload.TemplateId == 0 {
			fields["reason"] = "late_effect_unrecoverable"
			c.l.WithFields(fields).Warn("Late-successful DestroyAssetFromSlot carries no TemplateId; destroyed item is not recoverable from the step payload, its effect is orphaned.")
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
	case DestroyAssetFromSlot:
		// TemplateId==0 is excluded upstream in CompensateLateStep (legacy
		// producer, no recoverable quantity), so only payloads carrying a
		// TemplateId reach here — recreate the destroyed book/item.
		payload, ok := step.Payload().(DestroyAssetFromSlotPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late DestroyAssetFromSlot compensation")
		}
		qty := payload.Quantity
		if qty == 0 {
			qty = 1
		}
		return c.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, time.Time{})
	case AwardMesos:
		payload, ok := step.Payload().(AwardMesosPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late AwardMesos compensation")
		}
		ch := channel.NewModel(payload.WorldId, payload.ChannelId)
		return c.charP.AwardMesosAndEmit(s.TransactionId(), ch, payload.CharacterId, payload.CharacterId, "SYSTEM", -payload.Amount, false)
	case AcceptToCharacter:
		// TradeTransaction only — isLateCompensable gates the other saga types
		// out. Re-asserted here so a future registration in the global map
		// cannot silently give another flow trade's semantics.
		if s.SagaType() != TradeTransaction {
			return fmt.Errorf("late AcceptToCharacter compensation is registered for trade settlements only, got saga type %s", s.SagaType())
		}
		payload, ok := step.Payload().(AcceptToCharacterPayload)
		if !ok {
			return fmt.Errorf("invalid payload for late AcceptToCharacter compensation")
		}
		qty := payload.AssetData.Quantity
		if qty == 0 {
			qty = 1
		}
		// The reverse-walk already re-granted this asset to its owner, so the
		// copy this late accept created in the recipient's inventory is the
		// duplicate and must go.
		return c.compP.RequestDestroyItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, false)
	case ReleaseFromTrade:
		// Both the settlement and the teardown unwind run as TradeTransaction —
		// they are two outcomes of one trade lifecycle and need the same set of
		// inverses, so DispatchTradeTransactionRollbacks covers both.
		if s.SagaType() != TradeTransaction {
			return fmt.Errorf("late ReleaseFromTrade compensation is registered for trade sagas only, got saga type %s", s.SagaType())
		}
		payload, ok := step.Payload().(ReleaseFromTradePayload)
		if !ok {
			return fmt.Errorf("invalid payload for late ReleaseFromTrade compensation")
		}
		// The reverse-walk skipped this step (it was not Completed then), so
		// nothing has undone the custody release. Un-soft-delete the row: the
		// item goes back to being escrowed, and the teardown that follows
		// returns it to its owner from there.
		return c.tradeP.RestoreTradeEscrowAndEmit(s.TransactionId(), payload.EscrowId)
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

// compensateTradeStaging reverse-walks a failed transfer_to_trade (task-205
// design §5A.4).
//
// The saga is two steps — release_from_character then accept_to_trade — and the
// dangerous ordering is the one that succeeds halfway: the asset has left the
// player's compartment and no escrow row exists to return it from. Without this
// walk the item is simply gone, which is a strictly worse outcome than the
// reserve-at-staging model this amendment replaced.
//
// Inverses:
//   - ReleaseFromCharacter → RequestAcceptAsset, re-granting to the original
//     owner using the snapshot carried on the AcceptToTrade step so scrolled
//     stats, cash ownership and expiry survive. This mirrors the MTS arm; the
//     snapshot is found by action rather than by an asset-id-suffixed step id
//     because a staging saga stages exactly one asset.
//   - AcceptToTrade → RemoveTradeEscrow, hard-deleting a row whose paired
//     release later failed. Without it a settlement or unwind would deliver an
//     item the owner still holds, minting a duplicate.
//
// Idempotency is the same claimLateCompensation marker the other reverse-walks
// take: a second walk, or a late success for a step already inverted, dispatches
// nothing.
func (c *CompensatorImpl) compensateTradeStaging(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"failed_step":    failedStep.StepId(),
		"failed_action":  failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Info("Trade staging saga failing — dispatching reverse-walk compensation.")

	c.DispatchTradeStagingRollbacks(s)

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

	reason := fmt.Sprintf("Trade staging failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if err := EmitSagaFailed(c.l, c.ctx, s, sagaMsg.ErrorCodeUnknown, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event after trade staging compensation.")
		return err
	}
	return nil
}

// DispatchTradeStagingRollbacks issues the inverse of every completed step of a
// failing trade-staging saga, newest first. See compensateTradeStaging for the
// per-action contract.
func (c *CompensatorImpl) DispatchTradeStagingRollbacks(s Saga) {
	// Locate the AcceptToTrade snapshot (if any) so a ReleaseFromCharacter
	// inverse can re-grant with the original equip stats.
	var escrowSnapshot *AcceptToTradePayload
	for _, step := range s.Steps() {
		if step.Action() != AcceptToTrade {
			continue
		}
		if p, ok := step.Payload().(AcceptToTradePayload); ok {
			pc := p
			escrowSnapshot = &pc
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
		case AwardMesos:
			// The MESO staging saga's only step. Without this arm a staging
			// saga carrying a completed award_mesos reverse-walked nothing, so
			// the debit stood with no escrow behind it — the meso twin of the
			// item destruction this walk exists to prevent. The late-completion
			// path (absorbLateTerminal) already carries this inverse verbatim;
			// only the reverse walk was missing it.
			payload, ok := step.Payload().(AwardMesosPayload)
			if !ok {
				continue
			}
			if !c.claimTradeRollback(s, step) {
				continue
			}
			ch := channel.NewModel(payload.WorldId, payload.ChannelId)
			if err := c.charP.AwardMesosAndEmit(s.TransactionId(), ch, payload.CharacterId, payload.CharacterId, "SYSTEM", -payload.Amount, false); err != nil {
				c.l.WithError(err).WithFields(logrus.Fields{
					"transaction_id": s.TransactionId().String(),
					"step_id":        step.StepId(),
					"character_id":   payload.CharacterId,
					"amount":         payload.Amount,
				}).Error("Reverse-walk: staging AwardMesos reversal dispatch failed; continuing chain.")
			}
		case AcceptToTrade:
			payload, ok := step.Payload().(AcceptToTradePayload)
			if !ok {
				continue
			}
			if !c.claimTradeRollback(s, step) {
				continue
			}
			if err := c.tradeP.RemoveTradeEscrowAndEmit(s.TransactionId(), payload.EscrowId); err != nil {
				c.l.WithError(err).WithFields(logrus.Fields{
					"transaction_id": s.TransactionId().String(),
					"step_id":        step.StepId(),
					"escrow_id":      payload.EscrowId.String(),
				}).Error("Reverse-walk: AcceptToTrade → RemoveTradeEscrow dispatch failed; continuing chain.")
			}
		case ReleaseFromCharacter:
			payload, ok := step.Payload().(ReleaseFromCharacterPayload)
			if !ok {
				continue
			}
			if escrowSnapshot == nil {
				c.l.WithFields(logrus.Fields{
					"transaction_id": s.TransactionId().String(),
					"step_id":        step.StepId(),
					"character_id":   payload.CharacterId,
				}).Error("Reverse-walk: staging ReleaseFromCharacter has no AcceptToTrade snapshot to re-grant; skipping.")
				continue
			}
			if !c.claimTradeRollback(s, step) {
				continue
			}
			assetData := assetDataFromSnapshot(escrowSnapshot.Snapshot)
			if err := c.compP.RequestAcceptAsset(s.TransactionId(), payload.CharacterId, payload.InventoryType, escrowSnapshot.Snapshot.TemplateId, assetData); err != nil {
				c.l.WithError(err).WithFields(logrus.Fields{
					"transaction_id": s.TransactionId().String(),
					"step_id":        step.StepId(),
					"character_id":   payload.CharacterId,
					"template_id":    escrowSnapshot.Snapshot.TemplateId,
				}).Error("Reverse-walk: staging ReleaseFromCharacter → AcceptToCharacter re-grant dispatch failed; continuing chain.")
			}
		}
	}
}
