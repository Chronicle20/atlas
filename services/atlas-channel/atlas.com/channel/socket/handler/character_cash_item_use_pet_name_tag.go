package handler

import (
	"atlas-channel/pet"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	petconst "github.com/Chronicle20/atlas/libs/atlas-constants/pet"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
)

// petsForOwnerFunc is a test seam for the pet lookup (package-var injection
// precedent: cashItemInSlotFunc and cashItemDataFunc in this package).
var petsForOwnerFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]pet.Model, error) {
	return pet.NewProcessor(l, ctx).GetByOwner(characterId)
}

// buildPetNameTagUseSaga assembles the pet_name_tag_use saga: RENAME FIRST,
// consume second. This is the deliberate inverse of meso_sack_use's
// consume-then-award ordering (PRD FR-7.2) — a rename that fails must not cost
// the player a cash item. PreviousName rides the payload so the compensator can
// revert the name if the consume step later fails (FR-7.4); atlas-channel
// already read the pet to resolve the target, so capturing it is free.
//
// The tag is consumed by TEMPLATE, not by slot: the pre-branch guard in
// CharacterCashItemUseHandleFunc already proved the named CASH slot holds this
// template, and the orchestrator's inverse for DestroyAsset is a plain
// RequestCreateItem into the first free CASH slot.
func buildPetNameTagUseSaga(transactionId uuid.UUID, now time.Time, characterId uint32, itemId item.Id, petId uint32, name string, previousName string) saga.Saga {
	return saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.PetNameTagUse,
		InitiatedBy:   "CASH_ITEM_USE",
		Steps: []saga.Step{
			{
				StepId: "rename_pet",
				Status: saga.Pending,
				Action: saga.RenamePet,
				Payload: saga.RenamePetPayload{
					CharacterId:  characterId,
					PetId:        petId,
					Name:         name,
					PreviousName: previousName,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				StepId: "consume_pet_name_tag",
				Status: saga.Pending,
				Action: saga.DestroyAsset,
				Payload: saga.DestroyAssetPayload{
					CharacterId: characterId,
					TemplateId:  uint32(itemId),
					Quantity:    1,
					RemoveAll:   false,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
}

// handlePetNameTagUse implements the CashSlotItemType 17 arm: classification 517
// Pet Name Tags (5170000).
//
// WHICH PET. The request carries no pet identifier — the case-17 arm of
// CWvsContext::SendConsumeCashItemUseRequest performs exactly one Encode
// (EncodeStr @0xa0bcb5) and never calls the pet-picker SetUtilDlgEx_Pet
// (@0x9acb27). The server must resolve the target itself, and the rule is the
// character's LEAD pet (Slot() == 0). That is not an Atlas policy invention: the
// client's own arm calls sub_46D2D5(this, 0) @0xa0ba47, which resolves the CASH
// item backing pet-locker index 0 — Atlas's slot 0 (design OQ-4, PRD FR-3.1).
// The two therefore agree by derivation, not by coincidence.
//
// When locker index 0 is empty the unmodified client abandons the arm before any
// dialog and sends nothing (cmp esi,ebx / jz def_A0A6E6 @0xa0ba4e), so the
// no-lead-pet rejection below is a crafted-packet path — rare, but it must still
// fail closed.
//
// Nothing here warps, so a plain enable-actions StatChanged is the correct
// unlock on every rejection path (reference_exclrequest_unlock_contract).
func handlePetNameTagUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id, name string) {
	return func(s session.Model, itemId item.Id, name string) {
		enableActions := func() {
			_ = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
		}
		reject := func(msg string) {
			_ = session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePinkTextBody("", "", msg))(s)
			enableActions()
		}

		ps, err := petsForOwnerFunc(l, ctx, s.CharacterId())
		if err != nil {
			l.WithError(err).Warnf("Character [%d] used pet name tag [%d] but their pets could not be resolved. Rejecting; nothing consumed.", s.CharacterId(), itemId)
			reject("You are unable to use this item right now.")
			return
		}

		var target pet.Model
		var found bool
		for _, p := range ps {
			if p.Slot() == 0 {
				target = p
				found = true
				break
			}
		}
		if !found {
			l.Warnf("Character [%d] used pet name tag [%d] with no lead pet. Rejecting; nothing consumed.", s.CharacterId(), itemId)
			reject("You must have a pet out to use this.")
			return
		}
		// Belt and braces over a processor that already filters by owner.
		if target.OwnerId() != s.CharacterId() {
			l.Warnf("Character [%d] used pet name tag [%d] but resolved pet [%d] is owned by [%d]. Rejecting; nothing consumed.", s.CharacterId(), itemId, target.Id(), target.OwnerId())
			reject("You are unable to use this item right now.")
			return
		}

		normalized := petconst.NormalizeName(name)
		if verr := petconst.ValidateName(normalized); verr != nil {
			l.WithError(verr).Warnf("Character [%d] used pet name tag [%d] on pet [%d] with an invalid name [%s]. Rejecting; nothing consumed.", s.CharacterId(), itemId, target.Id(), name)
			reject("That name cannot be used.")
			return
		}
		if normalized == target.Name() {
			l.Warnf("Character [%d] used pet name tag [%d] on pet [%d] without changing the name. Rejecting; nothing consumed.", s.CharacterId(), itemId, target.Id())
			reject("That is already your pet's name.")
			return
		}

		l.Debugf("Character [%d] renaming pet [%d] from [%s] to [%s] with tag [%d].", s.CharacterId(), target.Id(), target.Name(), normalized, itemId)
		_ = saga.NewProcessor(l, ctx).Create(buildPetNameTagUseSaga(uuid.New(), time.Now(), s.CharacterId(), itemId, target.Id(), normalized, target.Name()))
	}
}
