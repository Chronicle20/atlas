package handler

import (
	"atlas-channel/character"
	"atlas-channel/character/skill"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func processAttack(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(ai model2.AttackInfo) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(ai model2.AttackInfo) model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(ai model2.AttackInfo) model.Operator[session.Model] {
			return func(ai model2.AttackInfo) model.Operator[session.Model] {
				return func(s session.Model) error {
					c, err := character.GetByIdWithInventory(l)(ctx)(character.SkillModelDecorator(l)(ctx))(s.CharacterId())
					if err != nil {
						return err
					}

					if ai.SkillId() > 0 {
						// Process skill
						var sk skill.Model
						for _, tsk := range c.Skills() {
							if tsk.Id() == ai.SkillId() {
								sk = tsk
							}
						}
						if sk.Id() == 0 {
							l.Errorf("Character [%d] attempting to attack with skill [%d] which they do not own.", s.CharacterId(), ai.SkillId())
							return session.Destroy(l, ctx, session.GetRegistry())(s)
						}
					}

					for _, di := range ai.DamageInfo() {
						for _, d := range di.Damages() {
							err := monster.Damage(l)(ctx)(s.WorldId(), s.ChannelId(), di.MonsterId(), s.CharacterId(), d)
							if err != nil {
								l.WithError(err).Errorf("Unable to apply damage [%d] to monster [%d] from character [%d].", d, di.MonsterId(), s.CharacterId())
							}
						}
					}

					_ = _map.ForOtherSessionsInMap(l)(ctx)(s.WorldId(), s.ChannelId(), s.MapId(), s.CharacterId(), func(os session.Model) error {
						var writerName string
						var bodyProducer writer.BodyProducer
						if ai.AttackType() == model2.AttackTypeMelee {
							writerName = writer.CharacterAttackMelee
							bodyProducer = writer.CharacterAttackMeleeBody(l, t)(c, ai)
						} else if ai.AttackType() == model2.AttackTypeRanged {
							writerName = writer.CharacterAttackRanged
							bodyProducer = writer.CharacterAttackRangedBody(l, t)(c, ai)
						} else if ai.AttackType() == model2.AttackTypeMagic {
							writerName = writer.CharacterAttackMagic
							bodyProducer = writer.CharacterAttackMagicBody(l, t)(c, ai)
						} else if ai.AttackType() == model2.AttackTypeEnergy {
							writerName = writer.CharacterAttackEnergy
							bodyProducer = writer.CharacterAttackEnergyBody(l, t)(c, ai)
						} else {
							return errors.New("unhandled attack type")
						}

						err = session.Announce(l)(ctx)(wp)(writerName)(os, bodyProducer)
						if err != nil {
							l.WithError(err).Errorf("Unable to announce character [%d] is attacking to character [%d].", s.CharacterId(), os.CharacterId())
							return err
						}
						return nil
					})

					return nil
				}
			}
		}
	}
}
