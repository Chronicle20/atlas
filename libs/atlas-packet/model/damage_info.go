package model

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func NewDamageInfo(hits byte) *DamageInfo {
	return &DamageInfo{hits: hits}
}

type DamageInfo struct {
	hits                byte
	monsterId           uint32
	hitAction           byte
	forceAction         byte
	frameIdx            byte
	calcDamageStatIndex byte
	hitPositionX        uint16
	hitPositionY        uint16
	previousPositionX   uint16
	previousPositionY   uint16
	delay               uint16
	damages             []uint32
	crc                 uint32
}

func (m *DamageInfo) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.monsterId = r.ReadUint32()
		m.hitAction = r.ReadByte()
		m.forceAction = r.ReadByte()
		m.frameIdx = r.ReadByte()
		m.calcDamageStatIndex = r.ReadByte()
		m.hitPositionX = r.ReadUint16()
		m.hitPositionY = r.ReadUint16()
		m.previousPositionX = r.ReadUint16()
		m.previousPositionY = r.ReadUint16()
		m.delay = r.ReadUint16()
		for range m.hits {
			m.damages = append(m.damages, r.ReadUint32())
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 83 {
			m.crc = r.ReadUint32()
		}
	}
}

// Encode is the symmetric mirror of Decode (client->server damage entry). Kept
// field-for-field in sync with Decode so AttackInfo round-trips across versions.
func (m *DamageInfo) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(m.monsterId)
		w.WriteByte(m.hitAction)
		w.WriteByte(m.forceAction)
		w.WriteByte(m.frameIdx)
		w.WriteByte(m.calcDamageStatIndex)
		w.WriteShort(m.hitPositionX)
		w.WriteShort(m.hitPositionY)
		w.WriteShort(m.previousPositionX)
		w.WriteShort(m.previousPositionY)
		w.WriteShort(m.delay)
		for _, d := range m.damages {
			w.WriteInt(d)
		}
		if t.Region() == "GMS" && t.MajorVersion() >= 83 {
			w.WriteInt(m.crc)
		}
		return w.Bytes()
	}
}

func (m *DamageInfo) Damages() []uint32 {
	return m.damages
}

func (m *DamageInfo) MonsterId() uint32 {
	return m.monsterId
}

func (m *DamageInfo) HitAction() byte {
	return m.hitAction
}

// Builder methods for constructing DamageInfo in the server-send path.

func (m *DamageInfo) SetMonsterId(monsterId uint32) *DamageInfo {
	m.monsterId = monsterId
	return m
}

func (m *DamageInfo) SetHitAction(hitAction byte) *DamageInfo {
	m.hitAction = hitAction
	return m
}

func (m *DamageInfo) SetDamages(damages []uint32) *DamageInfo {
	m.damages = damages
	return m
}
