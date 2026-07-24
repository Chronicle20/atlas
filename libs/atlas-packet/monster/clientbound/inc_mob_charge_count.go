package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
//
// Legacy (pre-v83) wire note: INC_MOB_CHARGE_COUNT is a per-mob OnMobPacket case
// (op 232), so the v79 client consumes a leading uniqueId via CMobPool::OnMobPacket
// @0x646d46 (Decode4 @0x646d50 -> GetMob) BEFORE CMob::OnIncMobChargeCount reads
// chargeCount/attackReady. See legacyMobPoolPrefix.
//
// packet-audit:fname CMob::OnIncMobChargeCount
type IncMobChargeCount struct {
	uniqueId    uint32
	chargeCount int32
	attackReady int32
}

func NewIncMobChargeCount(uniqueId uint32, chargeCount int32, attackReady int32) IncMobChargeCount {
	return IncMobChargeCount{uniqueId: uniqueId, chargeCount: chargeCount, attackReady: attackReady}
}

func (m IncMobChargeCount) UniqueId() uint32   { return m.uniqueId }
func (m IncMobChargeCount) ChargeCount() int32 { return m.chargeCount }
func (m IncMobChargeCount) AttackReady() int32 { return m.attackReady }
func (m IncMobChargeCount) Operation() string  { return IncMobChargeCountWriter }
func (m IncMobChargeCount) String() string {
	return fmt.Sprintf("chargeCount [%d], attackReady [%d]", m.chargeCount, m.attackReady)
}

func (m IncMobChargeCount) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if legacyMobPoolPrefix(t) {
			w.WriteInt(m.uniqueId)
		}
		w.WriteInt32(m.chargeCount)
		w.WriteInt32(m.attackReady)
		return w.Bytes()
	}
}

func (m *IncMobChargeCount) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if legacyMobPoolPrefix(t) {
			m.uniqueId = r.ReadUint32()
		}
		m.chargeCount = r.ReadInt32()
		m.attackReady = r.ReadInt32()
	}
}
