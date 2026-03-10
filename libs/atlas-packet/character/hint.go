package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterHintWriter = "CharacterHint"

type CharacterHint struct {
	hint    string
	width   uint16
	height  uint16
	atPoint bool
	x       int32
	y       int32
}

func NewCharacterHint(hint string, width uint16, height uint16, atPoint bool, x int32, y int32) CharacterHint {
	if width < 1 {
		width = uint16(len(hint)) * 10
		if width < 40 {
			width = 40
		}
	}
	if height < 5 {
		height = 5
	}
	return CharacterHint{hint: hint, width: width, height: height, atPoint: atPoint, x: x, y: y}
}

func (m CharacterHint) Hint() string      { return m.hint }
func (m CharacterHint) Width() uint16     { return m.width }
func (m CharacterHint) Height() uint16    { return m.height }
func (m CharacterHint) AtPoint() bool     { return m.atPoint }
func (m CharacterHint) X() int32          { return m.x }
func (m CharacterHint) Y() int32          { return m.y }
func (m CharacterHint) Operation() string { return CharacterHintWriter }
func (m CharacterHint) String() string    { return fmt.Sprintf("hint [%s]", m.hint) }

func (m CharacterHint) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.hint)
		w.WriteShort(m.width)
		w.WriteShort(m.height)
		w.WriteBool(!m.atPoint)
		if m.atPoint {
			w.WriteInt32(m.x)
			w.WriteInt32(m.y)
		}
		return w.Bytes()
	}
}

func (m *CharacterHint) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.hint = r.ReadAsciiString()
		m.width = r.ReadUint16()
		m.height = r.ReadUint16()
		notAtPoint := r.ReadBool()
		m.atPoint = !notAtPoint
		if m.atPoint {
			m.x = r.ReadInt32()
			m.y = r.ReadInt32()
		}
	}
}
