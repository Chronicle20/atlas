package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const IncMobChargeCountWriter = "IncMobChargeCount"

// IncMobChargeCount is the clientbound INC_MOB_CHARGE_COUNT packet
// (CMob::OnIncMobChargeCount): the server updates a mob's charge counter and
// attack-ready flag (used by charge-up bosses).
//
// Byte layout (IDA-verified, identical across versions — two Decode4):
//   - chargeCount : int32 — this->m_nMobChargeCount = Decode4
//   - attackReady : int32 — this->m_bAttackReady   = Decode4
//
// IDA basis: CMob::OnIncMobChargeCount — v83 @0x6710fc, v84 @0x687655,
// v87 @0x6ac230, v95 @0x63d500 (`m_nMobChargeCount = Decode4; m_bAttackReady =
// Decode4`). jms has NO INC_MOB_CHARGE_COUNT dispatcher case (CMobPool::OnMobPacket
// @0x6f8732 has no such case) → version-absent there.
type IncMobChargeCount struct {
	chargeCount int32
	attackReady int32
}

func NewIncMobChargeCount(chargeCount int32, attackReady int32) IncMobChargeCount {
	return IncMobChargeCount{chargeCount: chargeCount, attackReady: attackReady}
}

func (m IncMobChargeCount) ChargeCount() int32 { return m.chargeCount }
func (m IncMobChargeCount) AttackReady() int32 { return m.attackReady }
func (m IncMobChargeCount) Operation() string  { return IncMobChargeCountWriter }
func (m IncMobChargeCount) String() string {
	return fmt.Sprintf("chargeCount [%d], attackReady [%d]", m.chargeCount, m.attackReady)
}

func (m IncMobChargeCount) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.chargeCount)
		w.WriteInt32(m.attackReady)
		return w.Bytes()
	}
}

func (m *IncMobChargeCount) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.chargeCount = r.ReadInt32()
		m.attackReady = r.ReadInt32()
	}
}
