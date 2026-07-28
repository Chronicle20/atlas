package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CoconutHandle = "Coconut"

// Coconut - CField_Coconut::BasicActionAttack
// Sent when the player attacks a coconut. Body: attack uint16, x uint16.
// packet-audit:fname CField_Coconut::BasicActionAttack#Coconut
type Coconut struct {
	attack uint16
	x      uint16
}

func NewCoconut(attack uint16, x uint16) Coconut {
	return Coconut{attack: attack, x: x}
}

func (m Coconut) Attack() uint16 { return m.attack }
func (m Coconut) X() uint16      { return m.x }

func (m Coconut) Operation() string {
	return CoconutHandle
}

func (m Coconut) String() string {
	return fmt.Sprintf("attack [%d], x [%d]", m.attack, m.x)
}

func (m Coconut) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteShort(m.attack)
		w.WriteShort(m.x)
		return w.Bytes()
	}
}

func (m *Coconut) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.attack = r.ReadUint16()
		m.x = r.ReadUint16()
	}
}
