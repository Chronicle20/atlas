package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationMemoryGameFlipCard struct {
	first bool
	index byte
}

func (m OperationMemoryGameFlipCard) First() bool { return m.first }
func (m OperationMemoryGameFlipCard) Index() byte { return m.index }

func (m OperationMemoryGameFlipCard) Operation() string { return "OperationMemoryGameFlipCard" }

func (m OperationMemoryGameFlipCard) String() string {
	return fmt.Sprintf("first [%v] index [%d]", m.first, m.index)
}

func (m OperationMemoryGameFlipCard) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(m.first)
		w.WriteByte(m.index)
		return w.Bytes()
	}
}

func (m *OperationMemoryGameFlipCard) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.first = r.ReadBool()
		m.index = r.ReadByte()
	}
}
