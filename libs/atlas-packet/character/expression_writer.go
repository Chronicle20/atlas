package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterExpressionWriter = "CharacterExpression"

type CharacterExpressionW struct {
	characterId uint32
	expression  uint32
}

func NewCharacterExpressionW(characterId uint32, expression uint32) CharacterExpressionW {
	return CharacterExpressionW{characterId: characterId, expression: expression}
}

func (m CharacterExpressionW) CharacterId() uint32 { return m.characterId }
func (m CharacterExpressionW) Expression() uint32  { return m.expression }
func (m CharacterExpressionW) Operation() string   { return CharacterExpressionWriter }
func (m CharacterExpressionW) String() string {
	return fmt.Sprintf("characterId [%d], expression [%d]", m.characterId, m.expression)
}

func (m CharacterExpressionW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(m.expression)
		return w.Bytes()
	}
}

func (m *CharacterExpressionW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.expression = r.ReadUint32()
	}
}
