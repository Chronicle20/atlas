package handler

import (
	character2 "atlas-channel/character"
	dataskill "atlas-channel/data/skill"
	"atlas-channel/pointreset"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
)

// handlePointResetItemUse implements the CashSlotItemType 23/24 arm: AP Reset
// (5050000) and SP Reset (5050001-5050004). AP-vs-SP is decided by item id —
// the 23/24 type distinction is never used for dispatch (design §2.4).
func handlePointResetItemUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id, p cashsb.ItemUsePointReset) {
	return func(s session.Model, itemId item.Id, p cashsb.ItemUsePointReset) {
		enableActions := func() {
			_ = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
		}
		rejectWithMessage := func(msg string) {
			_ = session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePinkTextBody("", "", msg))(s)
			enableActions()
		}

		cp := character2.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.SkillModelDecorator)(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to load character [%d] for point reset.", s.CharacterId())
			enableActions()
			return
		}

		// FR-4: a dead character cannot use either item (enable-actions only,
		// no pink text — Cosmic parity).
		if c.Hp() == 0 {
			l.Warnf("Character [%d] attempted point reset [%d] while dead.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		f := s.Field()
		now := time.Now()

		buildSaga := func(transferStep saga.Step) saga.Saga {
			return saga.Saga{
				TransactionId: uuid.New(),
				SagaType:      saga.PointReset,
				InitiatedBy:   "CASH_ITEM_USE",
				Steps: []saga.Step{
					{
						StepId: "consume_point_reset_item",
						Status: saga.Pending,
						Action: saga.DestroyAsset,
						Payload: saga.DestroyAssetPayload{
							CharacterId: s.CharacterId(),
							TemplateId:  uint32(itemId),
							Quantity:    1,
							RemoveAll:   false,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					transferStep,
				},
			}
		}

		if itemId == pointreset.ApResetItemId {
			to, okTo := pointreset.AbilityFromWireFlag(p.To())
			from, okFrom := pointreset.AbilityFromWireFlag(p.From())
			if !okTo || !okFrom {
				l.Warnf("Character [%d] sent AP reset with invalid stat flags to [%d] from [%d].", s.CharacterId(), p.To(), p.From())
				enableActions()
				return
			}
			if ve := pointreset.ValidateApTransfer(c, from, to); ve != nil {
				l.Warnf("Character [%d] AP reset pre-validation rejected: [%s] detail [%s].", s.CharacterId(), ve.Code, ve.Detail)
				rejectWithMessage(pointreset.ErrorMessage(ve.Code, ve.Detail))
				return
			}
			_ = saga.NewProcessor(l, ctx).Create(buildSaga(saga.Step{
				StepId: "transfer_point",
				Status: saga.Pending,
				Action: saga.TransferAP,
				Payload: saga.TransferAPPayload{
					CharacterId: s.CharacterId(),
					WorldId:     f.WorldId(),
					ChannelId:   f.ChannelId(),
					From:        from,
					To:          to,
				},
				CreatedAt: now,
				UpdatedAt: now,
			}))
			return
		}

		if tier, ok := pointreset.SpResetTier(itemId); ok {
			toId := skill2.Id(p.To())
			fromId := skill2.Id(p.From())

			// Game-data max level for the target (non-4th-job cap); also
			// confirms the skill exists in game data.
			ds, err := dataskill.NewProcessor(l, ctx).GetById(uint32(toId))
			if err != nil {
				l.WithError(err).Warnf("Character [%d] SP reset target [%d] not found in game data.", s.CharacterId(), toId)
				rejectWithMessage(pointreset.ErrorMessage(pointreset.ErrorCodeInvalidTarget, ""))
				return
			}
			targetMaxLevel := byte(len(ds.Effects()))

			if ve := pointreset.ValidateSpTransfer(c, fromId, toId, tier, targetMaxLevel); ve != nil {
				l.Warnf("Character [%d] SP reset pre-validation rejected: [%s] from [%d] to [%d] tier [%d].", s.CharacterId(), ve.Code, fromId, toId, tier)
				rejectWithMessage(pointreset.ErrorMessage(ve.Code, ve.Detail))
				return
			}
			_ = saga.NewProcessor(l, ctx).Create(buildSaga(saga.Step{
				StepId: "transfer_point",
				Status: saga.Pending,
				Action: saga.TransferSP,
				Payload: saga.TransferSPPayload{
					CharacterId:    s.CharacterId(),
					WorldId:        f.WorldId(),
					ChannelId:      f.ChannelId(),
					JobId:          c.JobId(),
					FromSkillId:    fromId,
					ToSkillId:      toId,
					ItemTier:       tier,
					TargetMaxLevel: targetMaxLevel,
				},
				CreatedAt: now,
				UpdatedAt: now,
			}))
			return
		}

		// A 505x classification id that is neither 5050000 nor 5050001-4 is
		// impossible from a legit client.
		l.Warnf("Character [%d] attempted point reset with unexpected item [%d].", s.CharacterId(), itemId)
		enableActions()
	}
}
