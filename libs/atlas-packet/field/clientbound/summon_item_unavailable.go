package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SummonItemUnavailableWriter = "SummonItemUnavailable"

// SummonItemUnavailable is the clientbound CField::OnSummonItemInavailable packet.
// A single byte message/reason code shown when a summon item cannot be used.
type SummonItemUnavailable struct {
	message byte
}

func NewSummonItemUnavailable(message byte) SummonItemUnavailable {
	return SummonItemUnavailable{message: message}
}

func (m SummonItemUnavailable) Message() byte { return m.message }

func (m SummonItemUnavailable) Operation() string { return SummonItemUnavailableWriter }
func (m SummonItemUnavailable) String() string {
	return fmt.Sprintf("message [%d]", m.message)
}

func (m SummonItemUnavailable) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.message)
		return w.Bytes()
	}
}

func (m *SummonItemUnavailable) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.message = r.ReadByte()
	}
}
