package movement

import (
	"atlas-channel/data/npc"
	movement2 "atlas-channel/kafka/message/movement"
	"atlas-channel/kafka/producer"
	_map2 "atlas-channel/map"
	"atlas-channel/monster"
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

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	wp  writer.Producer
	t   tenant.Model
	sp  *session.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		wp:  wp,
		t:   tenant.MustFromContext(ctx),
		sp:  session.NewProcessor(l, ctx),
	}
	return p
}

func (p *Processor) ForCharacter(m _map.Model, characterId uint32, movement model.Movement) error {
	go func() {
		op := session.Announce(p.l)(p.ctx)(p.wp)(writer.CharacterMovement)(writer.CharacterMovementBody(p.l, tenant.MustFromContext(p.ctx))(characterId, movement))
		err := _map2.NewProcessor(p.l, p.ctx).ForOtherSessionsInMap(m, characterId, op)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to move character [%d] for characters in map [%d].", characterId, m.MapId())
		}
	}()
	go func() {
		ms, err := model2.Fold(model2.FixedProvider(movement.Elements), summaryProvider(movement.StartX, movement.StartY, 0), folder)()
		if err != nil {
			return
		}
		err = producer.ProviderImpl(p.l)(p.ctx)(movement2.EnvCommandCharacterMovement)(CommandProducer(m, uint64(characterId), characterId, ms.X, ms.Y, ms.Stance))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to issue movement command [%d].", characterId)
		}
	}()
	return nil
}

func (p *Processor) ForNPC(m _map.Model, characterId uint32, objectId uint32, unk byte, unk2 byte, movement model.Movement) error {
	go func() {
		n, err := npc.NewProcessor(p.l, p.ctx).GetInMapByObjectId(m.MapId(), objectId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve npc moving.")
			return
		}
		op := session.Announce(p.l)(p.ctx)(p.wp)(writer.NPCAction)(writer.NPCActionMoveBody(p.l, p.t)(objectId, unk, unk2, movement))
		err = p.sp.IfPresentByCharacterId(m.WorldId(), m.ChannelId())(characterId, op)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to move npc [%d] for character [%d].", n.Template(), characterId)
		}
		return
	}()
	return nil
}

func (p *Processor) ForPet(m _map.Model, characterId uint32, petId uint32, movement model.Movement) error {
	go func() {
		// TODO look up pet.
		pe := pet.NewModelBuilder(petId, 0, 0, "").
			SetOwnerID(characterId).
			SetSlot(0).
			MustBuild()

		op := session.Announce(p.l)(p.ctx)(p.wp)(writer.PetMovement)(writer.PetMovementBody(p.l, p.t)(pe, movement))
		err := _map2.NewProcessor(p.l, p.ctx).ForOtherSessionsInMap(m, characterId, op)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to move pet [%d] for characters in map [%d].", characterId, m.MapId())
		}
	}()
	go func() {
		ms, err := model2.Fold(model2.FixedProvider(movement.Elements), summaryProvider(movement.StartX, movement.StartY, 0), folder)()
		if err != nil {
			return
		}
		err = producer.ProviderImpl(p.l)(p.ctx)(movement2.EnvCommandPetMovement)(CommandProducer(m, uint64(petId), characterId, ms.X, ms.Y, ms.Stance))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to issue movement command [%d].", characterId)
		}
	}()
	return nil
}

func (p *Processor) ForMonster(m _map.Model, characterId uint32, objectId uint32, moveId int16, skillPossible bool, skill int8, skillId int16, skillLevel int16, mt model.MultiTargetForBall, rt model.RandTimeForAreaAttack, movement model.Movement) error {
	mo, err := monster.NewProcessor(p.l, p.ctx).GetById(objectId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate monster [%d] moving.", objectId)
		return err
	}

	if m.WorldId() != mo.WorldId() || m.ChannelId() != mo.ChannelId() || m.MapId() != mo.MapId() {
		p.l.Errorf("Monster [%d] movement issued by [%d] does not have consistent map data.", objectId, characterId)
		return err
	}
	go func() {
		op := session.Announce(p.l)(p.ctx)(p.wp)(writer.MoveMonsterAck)(writer.MoveMonsterAckBody(p.l, p.t)(objectId, moveId, uint16(mo.MP()), false, 0, 0))
		err = p.sp.IfPresentByCharacterId(m.WorldId(), m.ChannelId())(characterId, op)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to ack monster [%d] movement for character [%d].", objectId, characterId)
		}
	}()
	go func() {
		op := session.Announce(p.l)(p.ctx)(p.wp)(writer.MoveMonster)(writer.MoveMonsterBody(p.l, p.t)(objectId, false, skillPossible, false, skill, skillId, skillLevel, mt, rt, movement))
		err = _map2.NewProcessor(p.l, p.ctx).ForOtherSessionsInMap(m, characterId, op)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to move monster [%d] for characters in map [%d].", objectId, m.MapId())
		}
	}()
	go func() {
		var ms summary
		ms, err = model2.Fold(model2.FixedProvider(movement.Elements), summaryProvider(movement.StartX, movement.StartY, 0), folder)()
		if err != nil {
			return
		}
		err = producer.ProviderImpl(p.l)(p.ctx)(movement2.EnvCommandMonsterMovement)(CommandProducer(m, uint64(objectId), characterId, ms.X, ms.Y, ms.Stance))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to issue movement command [%d].", characterId)
		}
	}()
	return nil
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
