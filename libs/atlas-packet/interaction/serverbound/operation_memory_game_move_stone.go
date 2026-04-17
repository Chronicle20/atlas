package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OperationMemoryGameMoveStone struct {
	point int64
	color byte
}

func (m OperationMemoryGameMoveStone) Point() int64 { return m.point }
func (m OperationMemoryGameMoveStone) Color() byte  { return m.color }

func (m OperationMemoryGameMoveStone) Operation() string { return "OperationMemoryGameMoveStone" }

func (m OperationMemoryGameMoveStone) String() string {
	return fmt.Sprintf("point [%d] color [%d]", m.point, m.color)
}

func (m OperationMemoryGameMoveStone) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt64(m.point)
		w.WriteByte(m.color)
		return w.Bytes()
	}
}

func (m *OperationMemoryGameMoveStone) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.point = r.ReadInt64()
		m.color = r.ReadByte()
	}
}
