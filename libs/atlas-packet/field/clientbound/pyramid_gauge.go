package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const PyramidGaugeWriter = "PyramidGauge"

// packet-audit:fname CField_Massacre::OnMassacreIncGauge
type PyramidGauge struct {
	gauge uint32
}

func NewPyramidGauge(gauge uint32) PyramidGauge {
	return PyramidGauge{gauge: gauge}
}

func (m PyramidGauge) Gauge() uint32 { return m.gauge }

func (m PyramidGauge) Operation() string { return PyramidGaugeWriter }
func (m PyramidGauge) String() string {
	return fmt.Sprintf("gauge [%d]", m.gauge)
}

func (m PyramidGauge) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.gauge)
		return w.Bytes()
	}
}

func (m *PyramidGauge) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.gauge = r.ReadUint32()
	}
}
