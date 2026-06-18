package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SnowballHitWriter = "SnowballHit"

// packet-audit:fname CField_SnowBall::OnSnowBallHit
type SnowballHit struct {
	position byte
	damage   uint16
	distance uint16
}

func NewSnowballHit(position byte, damage uint16, distance uint16) SnowballHit {
	return SnowballHit{position: position, damage: damage, distance: distance}
}

func (m SnowballHit) Position() byte   { return m.position }
func (m SnowballHit) Damage() uint16   { return m.damage }
func (m SnowballHit) Distance() uint16 { return m.distance }

func (m SnowballHit) Operation() string { return SnowballHitWriter }
func (m SnowballHit) String() string {
	return fmt.Sprintf("position [%d] damage [%d] distance [%d]", m.position, m.damage, m.distance)
}

func (m SnowballHit) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.position)
		w.WriteShort(m.damage)
		w.WriteShort(m.distance)
		return w.Bytes()
	}
}

func (m *SnowballHit) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.position = r.ReadByte()
		m.damage = r.ReadUint16()
		m.distance = r.ReadUint16()
	}
}
