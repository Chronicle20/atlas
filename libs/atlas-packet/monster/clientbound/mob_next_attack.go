package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MobNextAttackWriter = "MobNextAttack"

// MobNextAttack is the clientbound MOB_NEXT_ATTACK packet (CMob::OnNextAttack):
// the server tells the client a mob should evaluate/queue its next attack against
// a target; the single int is the attack/target selector forwarded to the mob's
// attack-range check.
//
// Byte layout (IDA-verified, a single Decode4):
//   - attackId : int32 — Decode4; gates IsTargetInAttackRange / GenerateMovePath
//
// IDA basis: CMob::OnNextAttack — v95 @0x6528a0 (`v3 = Decode4(iPacket); if
// (IsActive && v3 > 0) { IsTargetInAttackRange(...); GenerateMovePath(...) }`).
// v95-only: the v95 dispatcher CMobPool::OnMobPacket @0x6570b0 case 308 routes here;
// no other in-scope version (v83/v84/v87/jms) has a NextAttack dispatcher case.
//
// packet-audit:fname CMob::OnNextAttack
type MobNextAttack struct {
	attackId int32
}

func NewMobNextAttack(attackId int32) MobNextAttack {
	return MobNextAttack{attackId: attackId}
}

func (m MobNextAttack) AttackId() int32   { return m.attackId }
func (m MobNextAttack) Operation() string { return MobNextAttackWriter }
func (m MobNextAttack) String() string {
	return fmt.Sprintf("attackId [%d]", m.attackId)
}

func (m MobNextAttack) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.attackId)
		return w.Bytes()
	}
}

func (m *MobNextAttack) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.attackId = r.ReadInt32()
	}
}
