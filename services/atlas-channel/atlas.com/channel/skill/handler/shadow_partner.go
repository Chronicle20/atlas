package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/data/skill/effect"
	"atlas-channel/saga"
	"atlas-channel/socket/model"
	"context"
	"fmt"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	skill2 "github.com/Chronicle20/atlas-constants/skill"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func UseShadowPartner(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
		return func(m _map.Model, characterId uint32, info model.SkillUsageInfo, effect effect.Model) error {
			if effect.HPConsume() > 0 {
				_ = character.NewProcessor(l, ctx).ChangeHP(m, characterId, -int16(effect.HPConsume()))
			}
			if effect.MPConsume() > 0 {
				_ = character.NewProcessor(l, ctx).ChangeMP(m, characterId, -int16(effect.MPConsume()))
			}

			if effect.Cooldown() > 0 {
				_ = skill.NewProcessor(l, ctx).ApplyCooldown(m, skill2.Id(info.SkillId()), effect.Cooldown())(characterId)
			}

			if effect.ItemConsume() > 0 {
				quantity := effect.ItemConsumeAmount()
				if quantity == 0 {
					quantity = 1
				}

				now := time.Now()
				s := saga.Saga{
					TransactionId: uuid.New(),
					SagaType:      saga.InventoryTransaction,
					InitiatedBy:   "shadow_partner",
					Steps: []saga.Step[any]{
						{
							StepId:    fmt.Sprintf("shadow-partner-consume-%d-%d", characterId, info.SkillId()),
							Status:    saga.Pending,
							Action:    saga.DestroyAsset,
							CreatedAt: now,
							UpdatedAt: now,
							Payload: saga.DestroyAssetPayload{
								CharacterId: characterId,
								TemplateId:  effect.ItemConsume(),
								Quantity:    quantity,
								RemoveAll:   false,
							},
						},
					},
				}

				err := saga.NewProcessor(l, ctx).Create(s)
				if err != nil {
					l.WithError(err).Warn("Failed to create saga for summoning rock consumption.")
				}
			}

			_ = buff.NewProcessor(l, ctx).Apply(m, characterId, int32(info.SkillId()), effect.Duration(), effect.StatUps())(characterId)
			return nil
		}
	}
}
