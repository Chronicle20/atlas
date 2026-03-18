package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterHealOverTimeHandle = "CharacterHealOverTimeHandle"

// HealOverTime - CUser::SendHealOverTime
type HealOverTime struct {
	updateTime uint32
	val        uint32
	hp         int16
	mp         int16
	unknown    byte
}

func (m HealOverTime) UpdateTime() uint32 {
	return m.updateTime
}

func (m HealOverTime) Val() uint32 {
	return m.val
}

func (m HealOverTime) HP() int16 {
	return m.hp
}

func (m HealOverTime) MP() int16 {
	return m.mp
}

func (m HealOverTime) Unknown() byte {
	return m.unknown
}

func (m HealOverTime) Operation() string {
	return CharacterHealOverTimeHandle
}

func (m HealOverTime) String() string {
	return fmt.Sprintf("updateTime [%d], val [%d], hp [%d], mp [%d], unknown [%d]", m.updateTime, m.val, m.hp, m.mp, m.unknown)
}

func (m HealOverTime) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt(m.val)
		w.WriteInt16(m.hp)
		w.WriteInt16(m.mp)
		if t.Region() == "GMS" && t.MajorVersion() <= 95 {
			w.WriteByte(m.unknown)
		}
		return w.Bytes()
	}
}

func (m *HealOverTime) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.val = r.ReadUint32()
		m.hp = r.ReadInt16()
		m.mp = r.ReadInt16()
		if t.Region() == "GMS" && t.MajorVersion() <= 95 {
			m.unknown = r.ReadByte()
		}
	}
}
