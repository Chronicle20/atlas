package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SnowballHandle = "Snowball"

// Snowball - CField_SnowBall::BasicActionAttack
// Sent when the player attacks a snowball. Body: attack byte, damage uint16, x uint16.
type Snowball struct {
	attack byte
	damage uint16
	x      uint16
}

func NewSnowball(attack byte, damage uint16, x uint16) Snowball {
	return Snowball{attack: attack, damage: damage, x: x}
}

func (m Snowball) Attack() byte   { return m.attack }
func (m Snowball) Damage() uint16 { return m.damage }
func (m Snowball) X() uint16      { return m.x }

func (m Snowball) Operation() string {
	return SnowballHandle
}

func (m Snowball) String() string {
	return fmt.Sprintf("attack [%d], damage [%d], x [%d]", m.attack, m.damage, m.x)
}

func (m Snowball) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.attack)
		w.WriteShort(m.damage)
		w.WriteShort(m.x)
		return w.Bytes()
	}
}

func (m *Snowball) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.attack = r.ReadByte()
		m.damage = r.ReadUint16()
		m.x = r.ReadUint16()
	}
}
