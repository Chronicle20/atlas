package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CompartmentMergeRequestHandle = "CompartmentMerge"

// CompartmentMergeRequest - CField::SendCompartmentMergeRequest
type CompartmentMergeRequest struct {
	updateTime      uint32
	compartmentType byte
}

func (m CompartmentMergeRequest) UpdateTime() uint32 {
	return m.updateTime
}

func (m CompartmentMergeRequest) CompartmentType() byte {
	return m.compartmentType
}

func (m CompartmentMergeRequest) Operation() string {
	return CompartmentMergeRequestHandle
}

func (m CompartmentMergeRequest) String() string {
	return fmt.Sprintf("updateTime [%d], compartmentType [%d]", m.updateTime, m.compartmentType)
}

func (m CompartmentMergeRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteByte(m.compartmentType)
		return w.Bytes()
	}
}

func (m *CompartmentMergeRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.compartmentType = r.ReadByte()
	}
}
