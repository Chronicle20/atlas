package model

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// perMobCrc reports whether each per-target DamageInfo entry ends with the
// anti-hack CRC word. GMS v61+ (IDA-verified per version, see Decode) and
// jms v185 (live-capture derived — the jms sender's encode tail is
// code-flow-virtualized; see attackDrBlocks in attack_info.go and
// docs/tasks/fix-jms185-attack-decode/diagnosis.md).
func perMobCrc(t tenant.Model) bool {
	return (t.Region() == "GMS" && t.MajorVersion() >= 61) || t.Region() == "JMS"
}

func NewDamageInfo(hits byte) *DamageInfo {
	return &DamageInfo{hits: hits}
}

// NewMesoExplosionDamageInfo constructs a DamageInfo for the meso-explosion
// attack variant (skill 4211006): the wire entry carries a 1-byte damage-line
// count in place of the standard 2-byte delay, so hits is unused (task-150
// design §2.1/§2.2).
func NewMesoExplosionDamageInfo() *DamageInfo {
	return &DamageInfo{mesoExplosion: true}
}

type DamageInfo struct {
	hits                byte
	mesoExplosion       bool
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
		if m.mesoExplosion {
			count := r.ReadByte()
			for range count {
				m.damages = append(m.damages, r.ReadUint32())
			}
		} else {
			m.delay = r.ReadUint16()
			for range m.hits {
				m.damages = append(m.damages, r.ReadUint32())
			}
		}
		// Per-mob anti-hack CRC. Present on the GMS legacy pre-83 client too:
		// v79 IDA-verified — TryDoingMeleeAttack (@0x8c2c57), TryDoingBodyAttack
		// (@0x8b77d3) and TryDoingMagicAttack (@0x8af1c4) all Encode4 the mob CRC
		// (sub_640131) as the final per-target field. The v72 melee sender
		// (sub_85DDD2 @0x85fb50, Encode4 sub_61F8A5) writes it too, and the v61
		// melee sender (sub_7A45F1 @0x7a5f14, Encode4 sub_5CF2AF) writes it as the
		// final per-target field as well, so the field predates v72 — lowered from
		// `>= 72` to `>= 61`.
		//
		// jms v185 writes it too: in the live 79-byte melee capture the four
		// bytes after the damage line (`66 1d a1 07`) precede a characterX/Y
		// pair that decodes to the caster's real position, and dropping them
		// leaves the packet four bytes unconsumed. See perMobCrc.
		if perMobCrc(t) {
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
		if m.mesoExplosion {
			w.WriteByte(byte(len(m.damages)))
		} else {
			w.WriteShort(m.delay)
		}
		for _, d := range m.damages {
			w.WriteInt(d)
		}
		// Symmetric with Decode: per-mob CRC present GMS v61+ and jms v185
		// (see Decode note).
		if perMobCrc(t) {
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
