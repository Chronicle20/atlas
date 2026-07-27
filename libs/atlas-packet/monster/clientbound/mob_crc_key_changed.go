package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MobCrcKeyChangedWriter = "MobCrcKeyChanged"

// MobCrcKeyChanged is the clientbound MOB_CRC_KEY_CHANGED packet
// (CMobPool::OnMobCrcKeyChanged). The server pushes a refreshed mob-CRC key; the
// client stores it (m_dwMobCrcKey) and flags every live mob to re-checksum.
//
// Byte layout (IDA-verified, identical across all 5 versions — a single Decode4):
//   - crcKey : uint32 — the new mob CRC key (CInPacket::Decode4 → this->m_dwMobCrcKey)
//
// IDA basis: CMobPool::OnMobCrcKeyChanged — v83 @0x6797be, v87 @0x6b5399,
// v95 @0x657230 (m_dwMobCrcKey = CInPacket::Decode4(iPacket)). The mob-list
// re-checksum loop reads no further wire bytes; the only payload field is crcKey.
//
// packet-audit:fname CMobPool::OnMobCrcKeyChanged
type MobCrcKeyChanged struct {
	crcKey uint32
}

func NewMobCrcKeyChanged(crcKey uint32) MobCrcKeyChanged {
	return MobCrcKeyChanged{crcKey: crcKey}
}

func (m MobCrcKeyChanged) CrcKey() uint32    { return m.crcKey }
func (m MobCrcKeyChanged) Operation() string { return MobCrcKeyChangedWriter }
func (m MobCrcKeyChanged) String() string {
	return fmt.Sprintf("crcKey [%d]", m.crcKey)
}

func (m MobCrcKeyChanged) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.crcKey)
		return w.Bytes()
	}
}

func (m *MobCrcKeyChanged) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.crcKey = r.ReadUint32()
	}
}
