package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
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
	_ = tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(m.oid)
		w.WriteByte(0) // char level
		w.WriteByte(m.direction)
		w.WriteByte(byte(len(m.targets)))
		for _, t := range m.targets {
			w.WriteInt(t.monsterOid)
			w.WriteByte(6) // per Cosmic "who knows"
			w.WriteInt(t.damage)
		}
		return w.Bytes()
	}
}

func (m *SummonAttack) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	_ = tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.oid = r.ReadUint32()
		_ = r.ReadByte() // char level
		m.direction = r.ReadByte()
		count := int(r.ReadByte())
		m.targets = make([]SummonAttackTarget, 0, count)
		for i := 0; i < count; i++ {
			monsterOid := r.ReadUint32()
			_ = r.ReadByte() // per Cosmic "who knows" = 6
			damage := r.ReadUint32()
			m.targets = append(m.targets, SummonAttackTarget{monsterOid: monsterOid, damage: damage})
		}
	}
}
