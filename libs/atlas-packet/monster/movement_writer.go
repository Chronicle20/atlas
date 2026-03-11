package monster

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MonsterMovementWriter = "MoveMonster"

type MovementW struct {
	uniqueId                    uint32
	bNotForceLandingWhenDiscard bool
	bNotChangeAction            bool
	bNextAttackPossible         bool
	bLeft                       int8
	skillId                     int16
	skillLevel                  int16
	multiTargets                model.MultiTargetForBall
	randTimeForAreaAttack       model.RandTimeForAreaAttack
	movement                    model.Movement
}

func NewMonsterMovementW(uniqueId uint32, bNotForceLandingWhenDiscard bool, bNotChangeAction bool, bNextAttackPossible bool, bLeft int8, skillId int16, skillLevel int16, multiTargets model.MultiTargetForBall, randTimeForAreaAttack model.RandTimeForAreaAttack, movement model.Movement) MovementW {
	return MovementW{
		uniqueId:                    uniqueId,
		bNotForceLandingWhenDiscard: bNotForceLandingWhenDiscard,
		bNotChangeAction:            bNotChangeAction,
		bNextAttackPossible:         bNextAttackPossible,
		bLeft:                       bLeft,
		skillId:                     skillId,
		skillLevel:                  skillLevel,
		multiTargets:                multiTargets,
		randTimeForAreaAttack:       randTimeForAreaAttack,
		movement:                    movement,
	}
}

func (m MovementW) Operation() string { return MonsterMovementWriter }
func (m MovementW) String() string {
	return fmt.Sprintf("uniqueId [%d], bLeft [%d], skillId [%d]", m.uniqueId, m.bLeft, m.skillId)
}

func (m MovementW) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteBool(m.bNotForceLandingWhenDiscard)
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			w.WriteBool(m.bNotChangeAction)
		}
		w.WriteBool(m.bNextAttackPossible)
		w.WriteInt8(m.bLeft)
		w.WriteInt16(m.skillId)
		w.WriteInt16(m.skillLevel)
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			w.WriteByteArray(m.multiTargets.Encode(l, ctx)(options))
			w.WriteByteArray(m.randTimeForAreaAttack.Encode(l, ctx)(options))
		}
		w.WriteByteArray(m.movement.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *MovementW) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.bNotForceLandingWhenDiscard = r.ReadBool()
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			m.bNotChangeAction = r.ReadBool()
		}
		m.bNextAttackPossible = r.ReadBool()
		m.bLeft = r.ReadInt8()
		m.skillId = r.ReadInt16()
		m.skillLevel = r.ReadInt16()
		if (t.Region() == "GMS" && t.MajorVersion() > 83) || t.Region() == "JMS" {
			m.multiTargets.Decode(l, ctx)(r, options)
			m.randTimeForAreaAttack.Decode(l, ctx)(r, options)
		}
		m.movement.Decode(l, ctx)(r, options)
	}
}
