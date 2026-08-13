package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const PetCommandResponseWriter = "PetCommandResponse"

// modeCommand is the only arm of CPet::OnActionCommand that carries an action
// index on the wire. The mode-0 (command) arm indexes m_aInteraction with the
// byte that follows the mode; the mode-1 (food) arm instead scans
// m_aFoodReaction for the entry matching the pet's level and reads the success
// flag directly after the mode byte. Verified GMS v83 @0x7048ab and GMS v95
// @0x6a3930 — both take the level-scan path for mode 1 with no intervening
// Decode1. Encoding an index on the food arm shifts success and balloon one
// byte late, which makes a successfully fed pet play its actFail reaction.
const modeCommand = byte(0)

// packet-audit:fname CPet::OnActionCommand
type CommandResponse struct {
	ownerId   uint32
	slot      int8
	mode      byte
	animation byte
	success   bool
	balloon   bool
}

func NewPetCommandResponse(ownerId uint32, slot int8, animation byte, success bool, balloon bool) CommandResponse {
	return CommandResponse{ownerId: ownerId, slot: slot, mode: modeCommand, animation: animation, success: success, balloon: balloon}
}

func NewPetFoodResponse(ownerId uint32, slot int8, success bool, balloon bool) CommandResponse {
	return CommandResponse{ownerId: ownerId, slot: slot, mode: 1, success: success, balloon: balloon}
}

func (m CommandResponse) Operation() string { return PetCommandResponseWriter }
func (m CommandResponse) String() string {
	return fmt.Sprintf("ownerId [%d], slot [%d], mode [%d], success [%t]", m.ownerId, m.slot, m.mode, m.success)
}

func (m CommandResponse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		w.WriteInt8(m.slot)
		w.WriteByte(m.mode)
		if m.mode == modeCommand {
			w.WriteByte(m.animation)
		}
		w.WriteBool(m.success)
		w.WriteBool(m.balloon)
		return w.Bytes()
	}
}

func (m *CommandResponse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.slot = r.ReadInt8()
		m.mode = r.ReadByte()
		if m.mode == modeCommand {
			m.animation = r.ReadByte()
		}
		m.success = r.ReadBool()
		m.balloon = r.ReadBool()
	}
}
