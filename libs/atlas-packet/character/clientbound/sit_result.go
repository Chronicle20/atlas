package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterSitResultWriter = "CharacterSitResult"

type CharacterSitResult struct {
	sitting bool
	chairId uint16
}

func NewCharacterSit(chairId uint16) CharacterSitResult {
	return CharacterSitResult{sitting: true, chairId: chairId}
}

func NewCharacterCancelSit() CharacterSitResult {
	return CharacterSitResult{sitting: false}
}

func (m CharacterSitResult) Sitting() bool    { return m.sitting }
func (m CharacterSitResult) ChairId() uint16  { return m.chairId }
func (m CharacterSitResult) Operation() string { return CharacterSitResultWriter }
func (m CharacterSitResult) String() string {
	if m.sitting {
		return fmt.Sprintf("sit chairId [%d]", m.chairId)
	}
	return "cancel sit"
}

func (m CharacterSitResult) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if m.sitting {
			w.WriteByte(1)
			w.WriteShort(m.chairId)
		} else {
			w.WriteByte(0)
		}
		return w.Bytes()
	}
}

func (m *CharacterSitResult) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		flag := r.ReadByte()
		if flag == 1 {
			m.sitting = true
			m.chairId = r.ReadUint16()
		} else {
			m.sitting = false
		}
	}
}
