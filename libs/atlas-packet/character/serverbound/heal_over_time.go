package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterHealOverTimeHandle = "CharacterHealOverTimeHandle"

// HealOverTime - CWvsContext::SendStatChangeRequest (HEAL_OVER_TIME, the
// auto-recovery / sit-down heal request the client sends on a timer).
//
// Wire body across versions:
//
//	GMS v83/v87/v95 (CWvsContext::SendStatChangeRequest@0xa1e997/.../0x9f2a00):
//	    updateTime(4) + val(4) + hp(2) + mp(2) + option(1)
//	JMS v185 (CWvsContext::SendStatChangeRequestByItemOption@0xb054d6 — the
//	    opcode-0x54 sender called from CWvsContext::TryRecovery; the symbol name
//	    is misleading, ground truth is COutPacket(_, 0x54)):
//	    updateTime(4) + val(4) + hp(2) + mp(2) + option(1) + extra(4)
//	    where extra is a client validation dword (dword_CDA4F8). jms is the ONLY
//	    version that appends the trailing dword.
//
// The trailing option byte is present on GMS v83..v95 and on jms; later GMS
// builds (>95) drop it. jms additionally appends the validation dword.
type HealOverTime struct {
	updateTime uint32
	val        uint32
	hp         int16
	mp         int16
	unknown    byte
	extra      uint32
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

// Extra is the jms-only trailing validation dword (dword_CDA4F8); zero on GMS.
func (m HealOverTime) Extra() uint32 {
	return m.extra
}

func (m HealOverTime) Operation() string {
	return CharacterHealOverTimeHandle
}

func (m HealOverTime) String() string {
	return fmt.Sprintf("updateTime [%d], val [%d], hp [%d], mp [%d], unknown [%d], extra [%d]", m.updateTime, m.val, m.hp, m.mp, m.unknown, m.extra)
}

func (m HealOverTime) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt(m.val)
		w.WriteInt16(m.hp)
		w.WriteInt16(m.mp)
		if (t.Region() == "GMS" && t.MajorVersion() <= 95) || t.Region() == "JMS" {
			w.WriteByte(m.unknown)
		}
		if t.Region() == "JMS" {
			w.WriteInt(m.extra)
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
		if (t.Region() == "GMS" && t.MajorVersion() <= 95) || t.Region() == "JMS" {
			m.unknown = r.ReadByte()
		}
		if t.Region() == "JMS" {
			m.extra = r.ReadUint32()
		}
	}
}
