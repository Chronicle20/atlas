package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterKeyMapAutoMpWriter = "CharacterKeyMapAutoMp"

type CharacterKeyMapAutoMp struct {
	action int32
}

func NewCharacterKeyMapAutoMp(action int32) CharacterKeyMapAutoMp {
	return CharacterKeyMapAutoMp{action: action}
}

func (m CharacterKeyMapAutoMp) Action() int32    { return m.action }
func (m CharacterKeyMapAutoMp) Operation() string { return CharacterKeyMapAutoMpWriter }
func (m CharacterKeyMapAutoMp) String() string    { return fmt.Sprintf("action [%d]", m.action) }

func (m CharacterKeyMapAutoMp) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.action)
		return w.Bytes()
	}
}

func (m *CharacterKeyMapAutoMp) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.action = r.ReadInt32()
	}
}
