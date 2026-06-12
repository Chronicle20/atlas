package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MonsterMovementWriter = "MoveMonster"

type Movement struct {
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

func NewMonsterMovement(uniqueId uint32, bNotForceLandingWhenDiscard bool, bNotChangeAction bool, bNextAttackPossible bool, bLeft int8, skillId int16, skillLevel int16, multiTargets model.MultiTargetForBall, randTimeForAreaAttack model.RandTimeForAreaAttack, movement model.Movement) Movement {
	return Movement{
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

func (m Movement) Operation() string { return MonsterMovementWriter }
func (m Movement) String() string {
	return fmt.Sprintf("uniqueId [%d], bLeft [%d], skillId [%d]", m.uniqueId, m.bLeft, m.skillId)
}

func (m Movement) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteBool(m.bNotForceLandingWhenDiscard)
		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" { // v87+ fields; v84..86 == v83 (off-by-one fix). delta §3.2
			w.WriteBool(m.bNotChangeAction)
		}
		w.WriteBool(m.bNextAttackPossible)
		w.WriteInt8(m.bLeft)
		w.WriteInt16(m.skillId)
		w.WriteInt16(m.skillLevel)
		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" { // v87+ fields; v84..86 == v83 (off-by-one fix). delta §3.2
			w.WriteByteArray(m.multiTargets.Encode(l, ctx)(options))
			w.WriteByteArray(m.randTimeForAreaAttack.Encode(l, ctx)(options))
		}
		w.WriteByteArray(m.movement.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *Movement) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.bNotForceLandingWhenDiscard = r.ReadBool()
		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" { // v87+ fields; v84..86 == v83 (off-by-one fix). delta §3.2
			m.bNotChangeAction = r.ReadBool()
		}
		m.bNextAttackPossible = r.ReadBool()
		m.bLeft = r.ReadInt8()
		m.skillId = r.ReadInt16()
		m.skillLevel = r.ReadInt16()
		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" { // v87+ fields; v84..86 == v83 (off-by-one fix). delta §3.2
			m.multiTargets.Decode(l, ctx)(r, options)
			m.randTimeForAreaAttack.Decode(l, ctx)(r, options)
		}
		m.movement.Decode(l, ctx)(r, options)
	}
}
