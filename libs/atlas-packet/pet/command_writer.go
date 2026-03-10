package pet

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetCommandResponseWriter = "PetCommandResponse"

type CommandResponse struct {
	ownerId   uint32
	slot      int8
	mode      byte
	animation byte
	success   bool
	balloon   bool
}

func NewPetCommandResponse(ownerId uint32, slot int8, animation byte, success bool, balloon bool) CommandResponse {
	return CommandResponse{ownerId: ownerId, slot: slot, mode: 0, animation: animation, success: success, balloon: balloon}
}

func NewPetFoodResponse(ownerId uint32, slot int8, animation byte, success bool, balloon bool) CommandResponse {
	return CommandResponse{ownerId: ownerId, slot: slot, mode: 1, animation: animation, success: success, balloon: balloon}
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
		w.WriteByte(m.animation)
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
		m.animation = r.ReadByte()
		m.success = r.ReadBool()
		m.balloon = r.ReadBool()
	}
}
