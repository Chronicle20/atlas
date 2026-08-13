package handler

import (
	cashData "atlas-channel/data/cash"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
)

// cashItemDataFunc is a test seam for the cash-item data lookup (package-var
// injection precedent: cashItemInSlotFunc in character_cash_item_use.go).
var cashItemDataFunc = func(l logrus.FieldLogger, ctx context.Context, itemId uint32) (cashData.RestModel, error) {
	return cashData.NewProcessor(l, ctx).GetById(itemId)
}

// buildMesoSackUseSaga assembles the meso_sack_use saga: destroy-first, exactly
// two steps. The sack is consumed by TEMPLATE, not by slot — the pre-branch
// guard in CharacterCashItemUseHandleFunc already proved the named CASH slot
// holds this template, the orchestrator's inverse for DestroyAsset is a plain
// RequestCreateItem, and a refund landing in the first free CASH slot matches
// every other refund path in the system.
func buildMesoSackUseSaga(transactionId uuid.UUID, now time.Time, characterId uint32, itemId item.Id, worldId world.Id, channelId channel.Id, amount int32) saga.Saga {
	return saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.MesoSackUse,
		InitiatedBy:   "CASH_ITEM_USE",
		Steps: []saga.Step{
			{
				StepId: "consume_meso_sack",
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
			{
				StepId: "award_mesos",
				Status: saga.Pending,
				Action: saga.AwardMesos,
				Payload: saga.AwardMesosPayload{
					CharacterId: characterId,
					WorldId:     worldId,
					ChannelId:   channelId,
					ActorId:     uint32(itemId),
					ActorType:   "ITEM",
					Amount:      amount,
					ShowEffect:  true,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
}

// handleMesoSackUse implements the CashSlotItemType 19 arm: classification 520
// meso sacks (5200000/5200001/5200002 on every version, plus the 5202xxx random
// family on v92/v95/JMS, which this pays at its flat info/meso value).
//
// The payout is resolved server-side from the WZ data keyed by the
// server-resolved template id — no client-supplied value influences it.
//
// Nothing is announced on the success path: the award's STAT_CHANGED{Meso} from
// atlas-character already carries ExclRequestSent, so that packet both renders
// the new balance and releases the client's exclusive-request gate, correctly
// ordered by construction.
func handleMesoSackUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id) {
	return func(s session.Model, itemId item.Id) {
		enableActions := func() {
			_ = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
		}

		cd, err := cashItemDataFunc(l, ctx, uint32(itemId))
		if err != nil {
			l.WithError(err).Warnf("Character [%d] used meso sack [%d] but its cash item data could not be resolved. Rejecting.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		// Fail closed. Zero covers both a Maple Point sack (5200009/5200010 carry
		// info/maplepoint and no info/meso) and a tenant whose WZ has not been
		// re-ingested since the Meso field was added. The ceiling covers the
		// int32 width of AwardMesosPayload.Amount: a larger value would wrap
		// negative and DEDUCT mesos. Never drop this guard.
		if cd.Meso == 0 {
			l.Warnf("Character [%d] used meso sack [%d] with no info/meso amount. Rejecting; nothing consumed.", s.CharacterId(), itemId)
			enableActions()
			return
		}
		if cd.Meso > uint32(math.MaxInt32) {
			l.Warnf("Character [%d] used meso sack [%d] whose amount [%d] exceeds the int32 award ceiling. Rejecting; nothing consumed.", s.CharacterId(), itemId, cd.Meso)
			enableActions()
			return
		}

		f := s.Field()
		l.Debugf("Character [%d] using meso sack [%d] for [%d] mesos.", s.CharacterId(), itemId, cd.Meso)
		_ = saga.NewProcessor(l, ctx).Create(buildMesoSackUseSaga(uuid.New(), time.Now(), s.CharacterId(), itemId, f.WorldId(), f.ChannelId(), int32(cd.Meso)))
	}
}
