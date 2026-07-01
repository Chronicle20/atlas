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
		// Per-mob anti-hack CRC. Present on the GMS legacy pre-83 client too:
		// v79 IDA-verified — TryDoingMeleeAttack (@0x8c2c57), TryDoingBodyAttack
		// (@0x8b77d3) and TryDoingMagicAttack (@0x8af1c4) all Encode4 the mob CRC
		// (sub_640131) as the final per-target field. The prior `>= 83` gate wrongly
		// dropped it for v79, desyncing the server decode by 4 bytes/target. Lowered
		// to `>= 79`; no in-range variant (v83..jms) changes (all already >= 83).
		if t.Region() == "GMS" && t.MajorVersion() >= 79 {
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
		// Symmetric with Decode: per-mob CRC present GMS v79+ (see Decode note).
		if t.Region() == "GMS" && t.MajorVersion() >= 79 {
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
