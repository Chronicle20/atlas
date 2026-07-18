package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MonsterBombHandle = "MonsterBomb"

// MonsterBomb is the serverbound MONSTER_BOMB packet
// (CMob::TryFirstSelfDestruction): the controller reports that a self-destructing
// mob's first-attack body-rect intersected the local user, requesting the server
// detonate it.
//
// Byte layout (IDA-verified, identical across all 5 versions — a single Encode4):
//   - mobId : uint32 — secured mob id of the self-destructing mob
//     (_ZtlSecureFuse(m_dwMobID, m_dwMobID_CS) / GetMobID(this))
//
// IDA basis: CMob::TryFirstSelfDestruction — v83 @0x66e636 (opcode 0xC1),
// v87 @0x6a95bd, v95 @0x640ee0 (opcode 0xE8):
//
//	COutPacket(op); Encode4(GetMobID(this)); SendPacket — exactly one wire field.
//
// In v84 the sender is unnamed in the IDB (no anchor symbol); the wire shape is
// v83-identical, so v84 takes this same codec (route lands; evidence inherits v83).
//
// packet-audit:fname CMob::TryFirstSelfDestruction
type MonsterBomb struct {
	mobId uint32
}

func (m MonsterBomb) MobId() uint32     { return m.mobId }
func (m MonsterBomb) Operation() string { return MonsterBombHandle }
func (m MonsterBomb) String() string {
	return fmt.Sprintf("mobId [%d]", m.mobId)
}

func (m MonsterBomb) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.mobId)
		return w.Bytes()
	}
}

func (m *MonsterBomb) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mobId = r.ReadUint32()
	}
}
