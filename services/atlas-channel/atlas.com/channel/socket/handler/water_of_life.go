package handler

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/data/cash"
	"atlas-channel/pet"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	petsb "github.com/Chronicle20/atlas/libs/atlas-packet/pet/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// Rejection texts. All three live here so the synchronous path below and the
// asynchronous REVIVE_FAILED path (kafka/consumer/pet) cannot drift.
const (
	waterOfLifeNoItemMessage   = "You do not have a Water of Life."
	waterOfLifeNoDollMessage   = "You have no pet that has dried up."
	waterOfLifeNoEffectMessage = "The Water of Life has no effect."
	// WaterOfLifeFailedMessage is announced when atlas-pets rejects the revive
	// after the item was already consumed; the saga refunds it.
	WaterOfLifeFailedMessage = "The Water of Life had no effect. It has been returned to you."
)

// WaterOfLifeHandleFunc handles CWvsContext::SendWaterOfLife.
//
// The packet body is EMPTY, so the server derives every operand itself: the
// held Water of Life and the most-recently-dried-up pet. It is a top-level
// opcode handler, NOT an arm of CharacterCashItemUseHandleFunc -- the client
// reaches SendWaterOfLife from SendEtcCashItemUseRequest (gms_v83 @0xa1dc5b),
// a sibling of SendCashSlotItemUseRequest and SendConsumeCashItemUseRequest,
// so the cash-item-use dispatcher never observes this item.
//
// NO EnableActions on any path. SendConsumeCashItemUseRequest (@0xa0ea6f) is
// the ONLY caller of CWvsContext::SetExclRequestSent (@0xa0ebbc); the CASH-tab
// double-click path gates on CanSendExclRequest(500) (CDraggableItem::
// OnDoubleClicked @0x4efdf7) but never latches. Sending an unlock here would be
// inert, and a lie in the code about what the client is doing.
func WaterOfLifeHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := petsb.WaterOfLife{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		reject := func(text string) {
			_ = session.Announce(l)(ctx)(wp)(charcb.CharacterStatusMessageWriter)(charpkt.CharacterStatusMessageOperationSystemMessageBody(text))(s)
		}

		cp := character.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.InventoryDecorator)(s.CharacterId())
		if err != nil {
			l.WithError(err).Warnf("Character [%d] unable to resolve inventory for a Water of Life use.", s.CharacterId())
			reject(waterOfLifeNoEffectMessage)
			return
		}
		sourceTemplateId, ok := findWaterOfLife(c.Inventory().Cash().Assets())
		if !ok {
			l.Warnf("Character [%d] used a Water of Life while holding none.", s.CharacterId())
			reject(waterOfLifeNoItemMessage)
			return
		}

		ps, err := pet.NewProcessor(l, ctx).GetByOwner(s.CharacterId())
		if err != nil {
			l.WithError(err).Warnf("Character [%d] unable to resolve pets for a Water of Life use.", s.CharacterId())
			reject(waterOfLifeNoEffectMessage)
			return
		}
		target, ok := selectRevivableTarget(ps, time.Now())
		if !ok {
			l.Warnf("Character [%d] used a Water of Life with no dried-up pet.", s.CharacterId())
			reject(waterOfLifeNoDollMessage)
			return
		}

		// Pre-flight only (FR-8.3): the authoritative derivation happens in
		// atlas-pets and is independently re-bounded in atlas-inventory. This
		// read exists so a WZ data error costs the player nothing -- once the
		// saga starts, the item is gone.
		cd, err := cash.NewProcessor(l, ctx).GetById(sourceTemplateId)
		if err != nil {
			l.WithError(err).Warnf("Character [%d] unable to resolve cash data for Water of Life [%d].", s.CharacterId(), sourceTemplateId)
			reject(waterOfLifeNoEffectMessage)
			return
		}
		if cd.Life == 0 {
			l.Errorf("Water of Life [%d] has no info/life; refusing to consume it for character [%d].", sourceTemplateId, s.CharacterId())
			reject(waterOfLifeNoEffectMessage)
			return
		}

		l.Infof("Character [%d] reviving pet [%d] with Water of Life [%d] (life [%d] days).", s.CharacterId(), target.Id(), sourceTemplateId, cd.Life)

		transactionId := uuid.New()
		now := time.Now()
		_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
			TransactionId: transactionId,
			SagaType:      saga.PetRevive,
			InitiatedBy:   "WATER_OF_LIFE",
			Steps: []saga.Step{
				{
					StepId: "destroy_water_of_life",
					Status: saga.Pending,
					Action: saga.DestroyAsset,
					Payload: saga.DestroyAssetPayload{
						CharacterId: s.CharacterId(),
						TemplateId:  sourceTemplateId,
						Quantity:    1,
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					StepId: "revive_pet",
					Status: saga.Pending,
					Action: saga.RevivePet,
					Payload: saga.RevivePetPayload{
						CharacterId:      s.CharacterId(),
						PetId:            target.Id(),
						SourceTemplateId: sourceTemplateId,
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		})
	}
}

// findWaterOfLife returns the template id of the character's Water of Life,
// resolved by CLASSIFICATION (518) so every present and future template
// qualifies. The lowest slot wins, so the choice is reproducible regardless of
// the backing slice's order. Only existence matters downstream:
// DestroyAssetPayload takes {CharacterId, TemplateId, Quantity} and resolves
// the slot itself.
func findWaterOfLife(assets []asset.Model) (uint32, bool) {
	var (
		best     uint32
		bestSlot int16
		found    bool
	)
	for _, a := range assets {
		if item.GetClassification(item.Id(a.TemplateId())) != item.ClassificationWaterOfLife {
			continue
		}
		if !found || a.Slot() < bestSlot {
			best, bestSlot, found = a.TemplateId(), a.Slot(), true
		}
	}
	return best, found
}

// selectRevivableTarget picks the MOST-RECENTLY-expired pet: among pets whose
// expiration is strictly in the past, the greatest (latest) expiration wins,
// with the lowest pet id breaking ties. The tie-break is not an edge case --
// two pets bought in one transaction share an expiration timestamp, and the
// operation must be reproducible. A zero expiration is a permanent pet, never
// a doll.
func selectRevivableTarget(ps []pet.Model, now time.Time) (pet.Model, bool) {
	candidates := make([]pet.Model, 0, len(ps))
	for _, p := range ps {
		if p.Expiration().IsZero() {
			continue
		}
		if now.After(p.Expiration()) {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return pet.Model{}, false
	}
	sort.Slice(candidates, func(i, j int) bool {
		if !candidates[i].Expiration().Equal(candidates[j].Expiration()) {
			return candidates[i].Expiration().After(candidates[j].Expiration())
		}
		return candidates[i].Id() < candidates[j].Id()
	})
	return candidates[0], true
}
