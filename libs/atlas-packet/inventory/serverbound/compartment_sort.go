package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CompartmentSortRequestHandle = "CompartmentSort"

// CompartmentSortRequest - CField::SendCompartmentSortRequest
type CompartmentSortRequest struct {
	updateTime      uint32
	compartmentType byte
}

func (m CompartmentSortRequest) UpdateTime() uint32 {
	return m.updateTime
}

func (m CompartmentSortRequest) CompartmentType() byte {
	return m.compartmentType
}

func (m CompartmentSortRequest) Operation() string {
	return CompartmentSortRequestHandle
}

func (m CompartmentSortRequest) String() string {
	return fmt.Sprintf("updateTime [%d], compartmentType [%d]", m.updateTime, m.compartmentType)
}

func (m CompartmentSortRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteByte(m.compartmentType)
		return w.Bytes()
	}
}

func (m *CompartmentSortRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.compartmentType = r.ReadByte()
	}
}
