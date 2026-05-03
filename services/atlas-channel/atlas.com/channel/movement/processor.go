package movement

import (
	dmap "atlas-channel/data/map"
	"atlas-channel/data/npc"
	movement2 "atlas-channel/kafka/message/movement"
	"atlas-channel/kafka/producer"
	_map2 "atlas-channel/map"
	"atlas-channel/monster"
	monsterinfo "atlas-channel/monster/information"
	"atlas-channel/pet"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	petpkt "github.com/Chronicle20/atlas/libs/atlas-packet/pet/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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

func (p *Processor) ForCharacter(f field.Model, characterId uint32, movement model.Movement) error {
	go func() {
		op := session.Announce(p.l)(p.ctx)(p.wp)(charpkt.CharacterMovementWriter)(charpkt.NewCharacterMovement(characterId, movement).Encode)
		err := _map2.NewProcessor(p.l, p.ctx).ForOtherSessionsInMap(f, characterId, op)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to move character [%d] for characters in map [%d].", characterId, f.MapId())
		}
	}()
	go func() {
		ms, err := model2.Fold(model2.FixedProvider(movement.Elements), summaryProvider(movement.StartX, movement.StartY, 0), folder)()
		if err != nil {
			return
		}
		err = producer.ProviderImpl(p.l)(p.ctx)(movement2.EnvCommandCharacterMovement)(CommandProducer(f, uint64(characterId), characterId, ms.X, ms.Y, ms.Fh, ms.Stance))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to issue movement command [%d].", characterId)
		}
	}()
	return nil
}

func (p *Processor) ForNPC(f field.Model, characterId uint32, objectId uint32, unk byte, unk2 byte, movement model.Movement) error {
	go func() {
		n, err := npc.NewProcessor(p.l, p.ctx).GetInMapByObjectId(f.MapId(), objectId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve npc moving.")
			return
		}
		op := session.Announce(p.l)(p.ctx)(p.wp)(npcpkt.NpcActionWriter)(npcpkt.NewNpcActionMove(objectId, unk, unk2, movement).Encode)
		err = p.sp.IfPresentByCharacterId(f.Channel())(characterId, op)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to move npc [%d] for character [%d].", n.Template(), characterId)
		}
		return
	}()
	return nil
}

func (p *Processor) ForPet(f field.Model, characterId uint32, petId uint32, movement model.Movement) error {
	go func() {
		// TODO look up pet.
		pe := pet.NewModelBuilder(petId, 0, 0, "").
			SetOwnerID(characterId).
			SetSlot(0).
			MustBuild()

		op := session.Announce(p.l)(p.ctx)(p.wp)(petpkt.PetMovementWriter)(petpkt.NewPetMovement(pe.OwnerId(), pe.Slot(), movement).Encode)
		err := _map2.NewProcessor(p.l, p.ctx).ForOtherSessionsInMap(f, characterId, op)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to move pet [%d] for characters in map [%d].", characterId, f.MapId())
		}
	}()
	go func() {
		ms, err := model2.Fold(model2.FixedProvider(movement.Elements), summaryProvider(movement.StartX, movement.StartY, 0), folder)()
		if err != nil {
			return
		}
		err = producer.ProviderImpl(p.l)(p.ctx)(movement2.EnvCommandPetMovement)(CommandProducer(f, uint64(petId), characterId, ms.X, ms.Y, ms.Fh, ms.Stance))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to issue movement command [%d].", characterId)
		}
	}()
	return nil
}

func (p *Processor) ForMonster(f field.Model, characterId uint32, objectId uint32, moveId int16, skillPossible bool, skill int8, skillId int16, skillLevel int16, mt model.MultiTargetForBall, rt model.RandTimeForAreaAttack, movement model.Movement) error {
	mo, err := monster.NewProcessor(p.l, p.ctx).GetById(objectId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate monster [%d] moving.", objectId)
		return err
	}

	if f.WorldId() != mo.WorldId() || f.ChannelId() != mo.ChannelId() || f.MapId() != mo.MapId() {
		p.l.Errorf("Monster [%d] movement issued by [%d] does not have consistent map data.", objectId, characterId)
		return err
	}
	// Forecast the post-decrement MP for basic attacks (Cosmic compat — the
	// v83 client gates on the ack carrying decremented MP). For melee /
	// non-basic-attack actions, ackMp passes through unchanged.
	ackMp := uint16(mo.Mp())
	pos0, isBasicAttack := basicAttackPos(skill)
	if isBasicAttack {
		info, ierr := monsterinfo.NewProcessor(p.l, p.ctx).GetById(mo.MonsterId())
		if ierr != nil {
			p.l.WithError(ierr).Debugf("Unable to fetch attack info for monster template [%d]; ack uses unchanged MP.", mo.MonsterId())
		} else {
			ackMp = computeAckMp(ackMp, pos0, info.Attacks())
		}
	}
	go func() {
		// v83 protocol compat (per Cosmic MoveLifeHandler:144 +
		// PacketCreator.moveMonsterResponse): the wire-level "useSkills" bool
		// is actually the controller's aggro flag. The client uses it to
		// decide whether mob AI is active — without it, the client renders
		// the mob as idle, never sends rawActivity ∈ [24,41] (basic attack)
		// or [42,59] (skill confirm), and our authoritative-side handlers
		// never fire. Send aggro by default; OR-in the inbox prediction so a
		// queued skill cast still propagates if aggro is somehow false.
		useSkills := mo.ControllerHasAggro()
		var skillIdByte, skillLevelByte byte
		if d, hit := monster.GetNextSkillInbox().TakeAndClear(p.t, objectId); hit && !d.IsSentinel() {
			useSkills = true
			skillIdByte = d.SkillId
			skillLevelByte = d.SkillLevel
			p.l.Debugf("Inbox: serving predicted skill (%d,%d) into MoveMonsterAck for monster [%d].", skillIdByte, skillLevelByte, objectId)
		}
		op := session.Announce(p.l)(p.ctx)(p.wp)(monsterpkt.MonsterMovementAckWriter)(monsterpkt.NewMonsterMovementAck(objectId, moveId, ackMp, useSkills, skillIdByte, skillLevelByte).Encode)
		err = p.sp.IfPresentByCharacterId(f.Channel())(characterId, op)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to ack monster [%d] movement for character [%d].", objectId, characterId)
		}
	}()
	go func() {
		op := session.Announce(p.l)(p.ctx)(p.wp)(monsterpkt.MonsterMovementWriter)(monsterpkt.NewMonsterMovement(objectId, false, skillPossible, false, skill, skillId, skillLevel, mt, rt, movement).Encode)
		err = _map2.NewProcessor(p.l, p.ctx).ForOtherSessionsInMap(f, characterId, op)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to move monster [%d] for characters in map [%d].", objectId, f.MapId())
		}
	}()
	go func() {
		var ms summary
		ms, err = model2.Fold(model2.FixedProvider(movement.Elements), summaryProvider(movement.StartX, movement.StartY, 0), folder)()
		if err != nil {
			return
		}
		// Snap y to (foothold surface - 1) so the stored mob position is
		// always 1 px ABOVE the foothold surface. The controller's client
		// occasionally reports y at-or-below the slope surface (int16
		// truncation in its float→short conversion); when another client
		// (or the same client at map re-entry) receives the spawn packet
		// for this mob, the v83 client validates (x, y) against the
		// foothold and treats at-or-below positions as embedded-in-terrain,
		// dropping the mob through the foothold. Pre-snapping at the
		// channel boundary keeps the stored position above-surface so
		// spawn-packet validation always passes.
		//
		// Mirrors atlas-data/map/processor.go::snapToGround which does the
		// same -1 adjustment for fresh spawn-point positions; this covers
		// the post-movement path that snapToGround does not.
		ms.X, ms.Y = dmap.SnapMobPosition(p.l, p.ctx, f.MapId(), ms.X, ms.Y, ms.Fh)
		err = producer.ProviderImpl(p.l)(p.ctx)(movement2.EnvCommandMonsterMovement)(CommandProducer(f, uint64(objectId), characterId, ms.X, ms.Y, ms.Fh, ms.Stance))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to issue movement command [%d].", characterId)
		}
	}()
	if skillId > 0 {
		id, lvl, ok := narrowSkillBytes(skillId, skillLevel)
		if !ok {
			p.l.Warnf("Monster [%d] inbound skill out of range (id=%d level=%d); dropping.", objectId, skillId, skillLevel)
		} else {
			go func() {
				err := monster.NewProcessor(p.l, p.ctx).UseSkill(f, objectId, characterId, id, lvl)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to issue use skill command for monster [%d].", objectId)
				}
			}()
		}
	}
	if isBasicAttack {
		go func() {
			if err := monster.NewProcessor(p.l, p.ctx).UseBasicAttack(f, objectId, pos0); err != nil {
				p.l.WithError(err).Errorf("Unable to issue basic-attack command for monster [%d].", objectId)
			}
		}()
	}
	return nil
}

type summary struct {
	X      int16
	Y      int16
	Fh     int16
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

func folder(s summary, e model.MovementCodec) (summary, error) {
	return foldMovementSummary(s, e)
}

func foldMovementSummary(s summary, e interface{}) (summary, error) {
	ms := summary{X: s.X, Y: s.Y, Fh: s.Fh, Stance: s.Stance}

	// Fh is preserved across mid-air frames (Jump, StartFallDown, etc.) — those
	// frames carry no resting foothold. Only NormalElement and TeleportElement
	// land the mob on a foothold; we copy v.Fh from those, but only when
	// non-zero so we don't trample the spawn-time fh during a fall sequence
	// where the client transmits Fh=0 for "no anchor yet".
	switch v := e.(type) {
	case *model.NormalElement:
		ms.X = v.X
		ms.Y = v.Y
		ms.Stance = v.BMoveAction
		if v.Fh != 0 {
			ms.Fh = v.Fh
		}
		return ms, nil
	case model.JumpElement:
		ms.Stance = v.BMoveAction
		return ms, nil
	case model.TeleportElement:
		ms.Stance = v.BMoveAction
		if v.Fh != 0 {
			ms.Fh = v.Fh
		}
		return ms, nil
	case model.StartFallDownElement:
		ms.Stance = v.BMoveAction
		return ms, nil
	default:
		return ms, nil
	}
}

// narrowSkillBytes narrows the inbound MoveLife skill values from int16 to
// byte. Returns ok=false on negative or out-of-range values; the caller
// should drop the skill cast in that case.
func narrowSkillBytes(skillId int16, skillLevel int16) (byte, byte, bool) {
	if skillId < 0 || skillId > 255 || skillLevel < 0 || skillLevel > 255 {
		return 0, 0, false
	}
	return byte(skillId), byte(skillLevel), true
}


// computeAckMp returns the MP value to advertise in MoveMonsterAck for a
// basic-attack action. It looks up the attack-position's conMP in atks
// (matching the 1-indexed information.AttackInfo.Pos by adding 1 to the
// 0-indexed wire attackPos) and subtracts it from currentMp, clamping to
// zero on underflow. When no matching attack info is present (or atks is
// nil), currentMp passes through unchanged — that matches melee mobs that
// have no info subdir.
func computeAckMp(currentMp uint16, attackPos uint8, atks []monsterinfo.AttackInfo) uint16 {
	wantPos := attackPos + 1
	for _, a := range atks {
		if a.Pos != wantPos {
			continue
		}
		if a.ConMP <= 0 {
			return currentMp
		}
		if uint16(a.ConMP) >= currentMp {
			return 0
		}
		return currentMp - uint16(a.ConMP)
	}
	return currentMp
}
