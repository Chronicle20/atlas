package serverbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MobCrcKeyChangedReplyHandle = "MobCrcKeyChangedReply"

// MobCrcKeyChangedReply is the serverbound MOB_CRC_KEY_CHANGED_REPLY packet, the
// client's acknowledgement that it processed a MOB_CRC_KEY_CHANGED push.
//
// Byte layout (IDA-verified): EMPTY payload — the reply carries the opcode only.
// CMobPool::OnMobCrcKeyChanged (v83 @0x6797be, v87 @0x6b5399, v95 @0x657230) builds
// the reply COutPacket (v83 opcode 0xA4, v87 0xAE, v95 0xBE) and immediately
// SendPacket()s it with no Encode* calls — there are zero wire fields.
// packet-audit:fname CMobPool::OnMobCrcKeyChanged
type MobCrcKeyChangedReply struct {
}

func (m MobCrcKeyChangedReply) Operation() string { return MobCrcKeyChangedReplyHandle }
func (m MobCrcKeyChangedReply) String() string    { return "" }

func (m MobCrcKeyChangedReply) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		// empty payload — opcode only (no Encode* calls in the send site)
		return w.Bytes()
	}
}

func (m *MobCrcKeyChangedReply) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// empty payload — nothing to read
	}
}
