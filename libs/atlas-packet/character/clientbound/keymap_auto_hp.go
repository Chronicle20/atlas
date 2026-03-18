package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterKeyMapAutoHpWriter = "CharacterKeyMapAutoHp"

type CharacterKeyMapAutoHp struct {
	action int32
}

func NewCharacterKeyMapAutoHp(action int32) CharacterKeyMapAutoHp {
	return CharacterKeyMapAutoHp{action: action}
}

func (m CharacterKeyMapAutoHp) Action() int32    { return m.action }
func (m CharacterKeyMapAutoHp) Operation() string { return CharacterKeyMapAutoHpWriter }
func (m CharacterKeyMapAutoHp) String() string    { return fmt.Sprintf("action [%d]", m.action) }

func (m CharacterKeyMapAutoHp) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.action)
		return w.Bytes()
	}
}

func (m *CharacterKeyMapAutoHp) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.action = r.ReadInt32()
	}
}
