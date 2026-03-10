package chat

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const GeneralChatWriter = "CharacterChatGeneral"

type GeneralChat struct {
	characterId uint32
	gm          bool
	message     string
	show        bool
}

func NewGeneralChat(characterId uint32, gm bool, message string, show bool) GeneralChat {
	return GeneralChat{characterId: characterId, gm: gm, message: message, show: show}
}

func (m GeneralChat) CharacterId() uint32 { return m.characterId }
func (m GeneralChat) Gm() bool            { return m.gm }
func (m GeneralChat) Message() string      { return m.message }
func (m GeneralChat) Show() bool           { return m.show }

func (m GeneralChat) Operation() string { return GeneralChatWriter }
func (m GeneralChat) String() string {
	return fmt.Sprintf("general chat from [%d]", m.characterId)
}

func (m GeneralChat) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteBool(m.gm)
		w.WriteAsciiString(m.message)
		w.WriteBool(m.show)
		return w.Bytes()
	}
}

func (m *GeneralChat) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.gm = r.ReadBool()
		m.message = r.ReadAsciiString()
		m.show = r.ReadBool()
	}
}
