package pet

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetChatWriter = "PetChat"

type ChatW struct {
	ownerId uint32
	slot    int8
	nType   byte
	nAction byte
	message string
	balloon bool
}

func NewPetChatW(ownerId uint32, slot int8, nType byte, nAction byte, message string, balloon bool) ChatW {
	return ChatW{ownerId: ownerId, slot: slot, nType: nType, nAction: nAction, message: message, balloon: balloon}
}

func (m ChatW) Operation() string { return PetChatWriter }
func (m ChatW) String() string {
	return fmt.Sprintf("ownerId [%d], slot [%d], message [%s]", m.ownerId, m.slot, m.message)
}

func (m ChatW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		w.WriteInt8(m.slot)
		w.WriteByte(m.nType)
		w.WriteByte(m.nAction)
		w.WriteAsciiString(m.message)
		w.WriteBool(m.balloon)
		return w.Bytes()
	}
}

func (m *ChatW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.slot = r.ReadInt8()
		m.nType = r.ReadByte()
		m.nAction = r.ReadByte()
		m.message = r.ReadAsciiString()
		m.balloon = r.ReadBool()
	}
}
