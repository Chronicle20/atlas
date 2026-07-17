package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const SummonAttackWriter = "SummonAttack"

// SummonAttackTarget is one damaged monster in a SummonAttack packet.
type SummonAttackTarget struct {
	monsterOid uint32
	damage     uint32
}

func NewSummonAttackTarget(monsterOid, damage uint32) SummonAttackTarget {
	return SummonAttackTarget{monsterOid: monsterOid, damage: damage}
}

func (t SummonAttackTarget) MonsterOid() uint32 { return t.monsterOid }
func (t SummonAttackTarget) Damage() uint32     { return t.damage }

// SummonAttack is the server -> client SUMMON_ATTACK packet, encoded per Cosmic
// PacketCreator.summonAttack (PacketCreator.java:2307-2322): int cid, int
// summonOid, byte 0 (char level), byte direction, byte count, then per target
// {int monsterOid, byte 6, int damage}.
type SummonAttack struct {
	characterId uint32
	oid         uint32
	direction   byte
	targets     []SummonAttackTarget
}

func NewSummonAttack(characterId, oid uint32, direction byte, targets []SummonAttackTarget) SummonAttack {
	return SummonAttack{
		characterId: characterId,
		oid:         oid,
		direction:   direction,
		targets:     targets,
	}
}

func (m SummonAttack) CharacterId() uint32           { return m.characterId }
func (m SummonAttack) Oid() uint32                   { return m.oid }
func (m SummonAttack) Direction() byte               { return m.direction }
func (m SummonAttack) Targets() []SummonAttackTarget { return m.targets }
func (m SummonAttack) Operation() string             { return SummonAttackWriter }

func (m SummonAttack) String() string {
	return fmt.Sprintf("characterId [%d], oid [%d], direction [%d], targets [%d]", m.characterId, m.oid, m.direction, len(m.targets))
}

func (m SummonAttack) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		// oid: present on ALL versions. cid is read upstream by
		// CUserPool::OnUserCommonPacket; CSummonedPool::OnPacket@0x938dd7 then does
		// one Decode4 = the oid before OnAttack. Wire = cid + oid + body (the old
		// "no oid pre-95" reading missed the upstream cid read). The trailing flag
		// byte below remains a genuine v95-only addition.
		w.WriteInt(m.oid)
		// char level: present on GMS v83+ and JMS v185, ABSENT on GMS v79. The v79
		// attack reader CSummonedPool::OnAttack (sub_71CFE9@0x71cfe9) reads the
		// action byte FIRST (Decode1@0x71d06f → bLeft|direction), then count
		// (Decode1@0x71d08b) — no leading charLevel byte, where v83+ read charLevel
		// then the action byte. Writing it on v79 shifts direction/count by one.
		if t.MajorAtLeast(83) {
			w.WriteByte(0) // char level
		}
		w.WriteByte(m.direction)
		w.WriteByte(byte(len(m.targets)))
		for _, t := range m.targets {
			w.WriteInt(t.monsterOid)
			w.WriteByte(6) // per Cosmic "who knows"
			w.WriteInt(t.damage)
		}
		// v95+ DELTA (gated >= 95, GMS only): the v95 client summon-attack
		// reader CSummoned::OnAttack@0x753340 decodes one trailing byte after
		// the target loop (Decode1@0x7534e1, value discarded). v87's reader
		// CSummonedPool::OnAttack@0x7f904c has NO trailing read, so this is a
		// v95-specific addition — gated >=95 like the SummonSpawn avatar-look
		// byte. See summon-packet-delta.md §3.4 / CLAUDE.md v84-off-by-one note.
		if t.IsRegion("GMS") && t.MajorAtLeast(95) {
			w.WriteByte(0) // trailing flag byte (consumed/discarded by client)
		}
		return w.Bytes()
	}
}

func (m *SummonAttack) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.oid = r.ReadUint32() // present on all versions (see Encode)
		if t.MajorAtLeast(83) {
			_ = r.ReadByte() // char level — absent on GMS v79 (see Encode)
		}
		m.direction = r.ReadByte()
		count := int(r.ReadByte())
		m.targets = make([]SummonAttackTarget, 0, count)
		for i := 0; i < count; i++ {
			monsterOid := r.ReadUint32()
			_ = r.ReadByte() // per Cosmic "who knows" = 6
			damage := r.ReadUint32()
			m.targets = append(m.targets, SummonAttackTarget{monsterOid: monsterOid, damage: damage})
		}
		// v95+ DELTA (mirror of Encode): consume the trailing flag byte.
		if t.IsRegion("GMS") && t.MajorAtLeast(95) {
			_ = r.ReadByte() // trailing flag byte
		}
	}
}
