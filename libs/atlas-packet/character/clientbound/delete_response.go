package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const DeleteCharacterResponseWriter = "DeleteCharacterResponse"

type DeleteCharacterResponse struct {
	characterId uint32
	code        byte
}

func NewDeleteCharacterResponse(characterId uint32, code byte) DeleteCharacterResponse {
	return DeleteCharacterResponse{characterId: characterId, code: code}
}

func (m DeleteCharacterResponse) CharacterId() uint32 { return m.characterId }
func (m DeleteCharacterResponse) Code() byte          { return m.code }
func (m DeleteCharacterResponse) Operation() string   { return DeleteCharacterResponseWriter }
func (m DeleteCharacterResponse) String() string {
	return fmt.Sprintf("characterId [%d], code [%d]", m.characterId, m.code)
}

func (m DeleteCharacterResponse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByte(m.code)
		return w.Bytes()
	}
}

func (m *DeleteCharacterResponse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.code = r.ReadByte()
	}
}
