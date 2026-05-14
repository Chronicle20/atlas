package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterExpressionWriter = "CharacterExpression"

type CharacterExpression struct {
	characterId  uint32
	expression   uint32
	duration     uint32
	byItemOption bool
}

func NewCharacterExpression(characterId uint32, expression uint32, duration uint32) CharacterExpression {
	return CharacterExpression{characterId: characterId, expression: expression, duration: duration}
}

func (m CharacterExpression) CharacterId() uint32 { return m.characterId }
func (m CharacterExpression) Expression() uint32  { return m.expression }
func (m CharacterExpression) Duration() uint32    { return m.duration }
func (m CharacterExpression) ByItemOption() bool  { return m.byItemOption }
func (m CharacterExpression) Operation() string   { return CharacterExpressionWriter }
func (m CharacterExpression) String() string {
	return fmt.Sprintf("characterId [%d], expression [%d], duration [%d]", m.characterId, m.expression, m.duration)
}

func (m CharacterExpression) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(m.expression)
		w.WriteInt(m.duration)
		w.WriteBool(m.byItemOption)
		return w.Bytes()
	}
}

func (m *CharacterExpression) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.expression = r.ReadUint32()
		m.duration = r.ReadUint32()
		m.byItemOption = r.ReadBool()
	}
}
