package handler

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	petsb "github.com/Chronicle20/atlas/libs/atlas-packet/pet/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// PetDestroyItemHandleFunc handles DESTROY_PET_ITEM_REQUEST — the other arm of
// CWvsContext::SendActivatePetRequest (GMS v95 @0x9f6980, named).
//
// The client sends it when the pet it just double-clicked is BOTH dried up
// (GW_ItemSlotPet::IsDead) and marked WZ noRevive (CItemInfo::IsNoRevive): a
// pet no Water of Life can bring back, so the client asks the server to destroy
// the item and offers the Cash Shop. A dried-up pet that IS revivable takes the
// other path — a "The time has run out so it can't move." notice with no packet
// at all — which is why this handler never competes with the Water of Life flow.
//
// The client latches its exclusive-request lock (SetExclRequestSent(1)) right
// after the send, so every rejection path here MUST unlock. On the success path
// the saga's own inventory packets carry the release, matching the other
// destroy-first handlers.
func PetDestroyItemHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := petsb.DestroyItem{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		enableActions := func() { _ = session.EnableActions(l)(ctx)(wp)(s) }

		cp := character.NewProcessor(l, ctx)
		// PetAssetEnrichmentDecorator supplies both operands this handler
		// matches on: the serial the client keys by, and the dead date.
		c, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator)(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to load character [%d] for a pet item destroy.", s.CharacterId())
			enableActions()
			return
		}

		a, ok := findPetBySerialNumber(c.Inventory().Cash().Assets(), p.CashItemSerialNumber())
		if !ok {
			l.Warnf("Character [%d] asked to destroy pet serial [%d], which they do not hold.", s.CharacterId(), p.CashItemSerialNumber())
			enableActions()
			return
		}

		// The client only reaches this packet for a dried-up pet. Re-check
		// server-side: the request names an item to destroy, and a client that
		// sent it for a live pet would otherwise delete a working pet.
		if a.PetDeadDate().IsZero() || time.Now().Before(a.PetDeadDate()) {
			l.Warnf("Character [%d] asked to destroy pet [%d] in slot [%d], which has not dried up.", s.CharacterId(), a.PetId(), a.Slot())
			enableActions()
			return
		}

		l.Infof("Character [%d] destroying dried-up pet [%d] (template [%d], slot [%d]).", s.CharacterId(), a.PetId(), a.TemplateId(), a.Slot())

		if err = saga.NewProcessor(l, ctx).Create(buildDestroyPetItemSaga(uuid.New(), time.Now(), s.CharacterId(), a.Slot(), a.TemplateId())); err != nil {
			l.WithError(err).Errorf("Unable to create the destroy saga for pet [%d] of character [%d].", a.PetId(), s.CharacterId())
			enableActions()
			return
		}
	}
}

// buildDestroyPetItemSaga is the one-step destroy. The step targets a SLOT, not
// a template: a character may hold several pets of one template and only the
// clicked one is dead. Deleting the asset is the whole cascade — atlas-pets
// drops the pet record on the asset-deleted event.
func buildDestroyPetItemSaga(transactionId uuid.UUID, now time.Time, characterId uint32, slot int16, templateId uint32) saga.Saga {
	return saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.InventoryTransaction,
		InitiatedBy:   "DESTROY_PET_ITEM",
		Steps: []saga.Step{
			{
				StepId: "destroy_dead_pet_item",
				Status: saga.Pending,
				Action: saga.DestroyAssetFromSlot,
				Payload: saga.DestroyAssetFromSlotPayload{
					CharacterId:   characterId,
					InventoryType: byte(inventory.TypeValueCash),
					Slot:          slot,
					Quantity:      1,
					ShowEffect:    false,
					TemplateId:    templateId,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
}

// findPetBySerialNumber resolves the cash asset the client named by
// liCashItemSN. The fallback to the pet id mirrors
// packet/model.Asset.PetSerialNumber: an asset whose pet carries no serial goes
// out on the wire keyed by its pet id, so that is what comes back.
func findPetBySerialNumber(assets []asset.Model, sn uint64) (asset.Model, bool) {
	for _, a := range assets {
		if !a.IsPet() {
			continue
		}
		effective := a.PetSerialNumber()
		if effective == 0 {
			effective = uint64(a.PetId())
		}
		if effective == sn {
			return a, true
		}
	}
	return asset.Model{}, false
}
