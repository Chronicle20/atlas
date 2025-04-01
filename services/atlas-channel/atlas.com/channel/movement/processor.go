package movement

import (
	movement2 "atlas-channel/kafka/message/movement"
	"atlas-channel/kafka/producer"
	movement3 "atlas-channel/kafka/producer/movement"
	_map2 "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/npc"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	model2 "github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func ForCharacter(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(m _map.Model, characterId uint32, movement model.Movement) error {
	return func(ctx context.Context) func(wp writer.Producer) func(m _map.Model, characterId uint32, movement model.Movement) error {
		return func(wp writer.Producer) func(m _map.Model, characterId uint32, movement model.Movement) error {
			return func(m _map.Model, characterId uint32, movement model.Movement) error {
				go func() {
					op := session.Announce(l)(ctx)(wp)(writer.CharacterMovement)(writer.CharacterMovementBody(l, tenant.MustFromContext(ctx))(characterId, movement))
					err := _map2.ForOtherSessionsInMap(l)(ctx)(m, characterId, op)
					if err != nil {
						l.WithError(err).Errorf("Unable to move character [%d] for characters in map [%d].", characterId, m.MapId())
					}
				}()
				go func() {
					ms, err := model2.Fold(model2.FixedProvider(movement.Elements), summaryProvider(movement.StartX, movement.StartY, 0), folder)()
					if err != nil {
						return
					}
					err = producer.ProviderImpl(l)(ctx)(movement2.EnvCommandCharacterMovement)(movement3.CommandProducer(m, uint64(characterId), characterId, ms.X, ms.Y, ms.Stance))
					if err != nil {
						l.WithError(err).Errorf("Unable to issue movement command [%d].", characterId)
					}
				}()
				return nil
			}
		}
	}
}

func ForNPC(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(m _map.Model, characterId uint32, objectId uint32, unk byte, unk2 byte, movement model.Movement) error {
	return func(ctx context.Context) func(wp writer.Producer) func(m _map.Model, characterId uint32, objectId uint32, unk byte, unk2 byte, movement model.Movement) error {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(m _map.Model, characterId uint32, objectId uint32, unk byte, unk2 byte, movement model.Movement) error {
			return func(m _map.Model, characterId uint32, objectId uint32, unk byte, unk2 byte, movement model.Movement) error {
				go func() {
					n, err := npc.GetInMapByObjectId(l)(ctx)(m.MapId(), objectId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve npc moving.")
						return
					}
					op := session.Announce(l)(ctx)(wp)(writer.NPCAction)(writer.NPCActionMoveBody(l, t)(objectId, unk, unk2, movement))
					err = session.IfPresentByCharacterId(t, m.WorldId(), m.ChannelId())(characterId, op)
					if err != nil {
						l.WithError(err).Errorf("Unable to move npc [%d] for character [%d].", n.Template(), characterId)
					}
					return
				}()
				return nil
			}
		}
	}
}

func ForPet(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(m _map.Model, characterId uint32, petId uint64, movement model.Movement) error {
	return func(ctx context.Context) func(wp writer.Producer) func(m _map.Model, characterId uint32, petId uint64, movement model.Movement) error {
		return func(wp writer.Producer) func(m _map.Model, characterId uint32, petId uint64, movement model.Movement) error {
			return func(m _map.Model, characterId uint32, petId uint64, movement model.Movement) error {
				go func() {
					// TODO look up pet.
					p := pet.NewModelBuilder(petId, 0, 0, "").
						SetOwnerID(characterId).
						SetSlot(0).
						Build()

					op := session.Announce(l)(ctx)(wp)(writer.PetMovement)(writer.PetMovementBody(l, tenant.MustFromContext(ctx))(p, movement))
					err := _map2.ForOtherSessionsInMap(l)(ctx)(m, characterId, op)
					if err != nil {
						l.WithError(err).Errorf("Unable to move pet [%d] for characters in map [%d].", characterId, m.MapId())
					}
				}()
				go func() {
					ms, err := model2.Fold(model2.FixedProvider(movement.Elements), summaryProvider(movement.StartX, movement.StartY, 0), folder)()
					if err != nil {
						return
					}
					err = producer.ProviderImpl(l)(ctx)(movement2.EnvCommandPetMovement)(movement3.CommandProducer(m, petId, characterId, ms.X, ms.Y, ms.Stance))
					if err != nil {
						l.WithError(err).Errorf("Unable to issue movement command [%d].", characterId)
					}
				}()
				return nil
			}
		}
	}
}

func ForMonster(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(m _map.Model, characterId uint32, objectId uint32, moveId int16, skillPossible bool, skill int8, skillId int16, skillLevel int16, mt model.MultiTargetForBall, rt model.RandTimeForAreaAttack, movement model.Movement) error {
	return func(ctx context.Context) func(wp writer.Producer) func(m _map.Model, characterId uint32, objectId uint32, moveId int16, skillPossible bool, skill int8, skillId int16, skillLevel int16, mt model.MultiTargetForBall, rt model.RandTimeForAreaAttack, movement model.Movement) error {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func(m _map.Model, characterId uint32, objectId uint32, moveId int16, skillPossible bool, skill int8, skillId int16, skillLevel int16, mt model.MultiTargetForBall, rt model.RandTimeForAreaAttack, movement model.Movement) error {
			return func(m _map.Model, characterId uint32, objectId uint32, moveId int16, skillPossible bool, skill int8, skillId int16, skillLevel int16, mt model.MultiTargetForBall, rt model.RandTimeForAreaAttack, movement model.Movement) error {
				mo, err := monster.GetById(l)(ctx)(objectId)
				if err != nil {
					l.WithError(err).Errorf("Unable to locate monster [%d] moving.", objectId)
					return err
				}

				if m.WorldId() != mo.WorldId() || m.ChannelId() != mo.ChannelId() || m.MapId() != mo.MapId() {
					l.Errorf("Monster [%d] movement issued by [%d] does not have consistent map data.", objectId, characterId)
					return err
				}
				go func() {
					op := session.Announce(l)(ctx)(wp)(writer.MoveMonsterAck)(writer.MoveMonsterAckBody(l, t)(objectId, moveId, uint16(mo.MP()), false, 0, 0))
					err = session.IfPresentByCharacterId(t, m.WorldId(), m.ChannelId())(characterId, op)
					if err != nil {
						l.WithError(err).Errorf("Unable to ack monster [%d] movement for character [%d].", objectId, characterId)
					}
				}()
				go func() {
					op := session.Announce(l)(ctx)(wp)(writer.MoveMonster)(writer.MoveMonsterBody(l, tenant.MustFromContext(ctx))(objectId, false, skillPossible, false, skill, skillId, skillLevel, mt, rt, movement))
					err = _map2.ForOtherSessionsInMap(l)(ctx)(m, characterId, op)
					if err != nil {
						l.WithError(err).Errorf("Unable to move monster [%d] for characters in map [%d].", objectId, m.MapId())
					}
				}()
				go func() {
					var ms summary
					ms, err = model2.Fold(model2.FixedProvider(movement.Elements), summaryProvider(movement.StartX, movement.StartY, 0), folder)()
					if err != nil {
						return
					}
					err = producer.ProviderImpl(l)(ctx)(movement2.EnvCommandMonsterMovement)(movement3.CommandProducer(m, uint64(objectId), characterId, ms.X, ms.Y, ms.Stance))
					if err != nil {
						l.WithError(err).Errorf("Unable to issue movement command [%d].", characterId)
					}
				}()
				return nil
			}
		}
	}
}

type summary struct {
	X      int16
	Y      int16
	Stance byte
}

func summaryProvider(x int16, y int16, stance byte) model2.Provider[summary] {
	return func() (summary, error) {
		return summary{
			X:      x,
			Y:      y,
			Stance: stance,
		}, nil
	}
}

const (
	TypeNormal        = "NORMAL"
	TypeTeleport      = "TELEPORT"
	TypeStartFallDown = "START_FALL_DOWN"
	TypeFlyingBlock   = "FLYING_BLOCK"
	TypeJump          = "JUMP"
	TypeStatChange    = "STAT_CHANGE"
)

func folder(s summary, e model.EncoderDecoder) (summary, error) {
	return foldMovementSummary(s, e)
}

func foldMovementSummary(s summary, e interface{}) (summary, error) {
	ms := summary{X: s.X, Y: s.Y, Stance: s.Stance}

	switch v := e.(type) {
	case *model.NormalElement:
		ms.X = v.X
		ms.Y = v.Y
		ms.Stance = v.BMoveAction
		return ms, nil
	case model.JumpElement:
		ms.Stance = v.BMoveAction
		return ms, nil
	case model.TeleportElement:
		ms.Stance = v.BMoveAction
		return ms, nil
	case model.StartFallDownElement:
		ms.Stance = v.BMoveAction
		return ms, nil
	default:
		return ms, nil
	}
}
