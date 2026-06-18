package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ZakumShrineWriter = "ZakumShrine"

// packet-audit:fname CField::OnZakumTimer
type ZakumShrine struct {
	state   byte
	seconds uint32
}

func NewZakumShrine(state byte, seconds uint32) ZakumShrine {
	return ZakumShrine{state: state, seconds: seconds}
}

func (m ZakumShrine) State() byte     { return m.state }
func (m ZakumShrine) Seconds() uint32 { return m.seconds }

func (m ZakumShrine) Operation() string { return ZakumShrineWriter }
func (m ZakumShrine) String() string {
	return fmt.Sprintf("state [%d] seconds [%d]", m.state, m.seconds)
}

func (m ZakumShrine) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.state)
		w.WriteInt(m.seconds)
		return w.Bytes()
	}
}

func (m *ZakumShrine) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.state = r.ReadByte()
		m.seconds = r.ReadUint32()
	}
}
