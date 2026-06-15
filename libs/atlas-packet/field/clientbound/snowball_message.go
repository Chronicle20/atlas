package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SnowballMessageWriter = "SnowballMessage"

// packet-audit:fname CField_SnowBall::OnSnowBallMsg
type SnowballMessage struct {
	team    byte
	message byte
}

func NewSnowballMessage(team byte, message byte) SnowballMessage {
	return SnowballMessage{team: team, message: message}
}

func (m SnowballMessage) Team() byte    { return m.team }
func (m SnowballMessage) Message() byte { return m.message }

func (m SnowballMessage) Operation() string { return SnowballMessageWriter }
func (m SnowballMessage) String() string {
	return fmt.Sprintf("team [%d] message [%d]", m.team, m.message)
}

func (m SnowballMessage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.team)
		w.WriteByte(m.message)
		return w.Bytes()
	}
}

func (m *SnowballMessage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.team = r.ReadByte()
		m.message = r.ReadByte()
	}
}
