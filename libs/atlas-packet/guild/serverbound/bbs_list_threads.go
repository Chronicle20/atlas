package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type BBSListThreads struct {
	startIndex uint32
}

func (m BBSListThreads) StartIndex() uint32 {
	return m.startIndex
}

func (m BBSListThreads) Operation() string {
	return "BBSListThreads"
}

func (m BBSListThreads) String() string {
	return fmt.Sprintf("startIndex [%d]", m.startIndex)
}

func (m BBSListThreads) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.startIndex)
		return w.Bytes()
	}
}

func (m *BBSListThreads) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.startIndex = r.ReadUint32()
	}
}
