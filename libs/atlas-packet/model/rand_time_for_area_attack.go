package model

import (
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type RandTimeForAreaAttack struct {
	Times []int32
}

func (m *RandTimeForAreaAttack) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		size := r.ReadUint32()
		m.Times = make([]int32, size)
		for i := 0; i < int(size); i++ {
			m.Times[i] = r.ReadInt32()
		}
	}
}

func (m *RandTimeForAreaAttack) Encoder(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(_ map[string]interface{}) []byte {
		w.WriteInt32(int32(len(m.Times)))
		for _, time := range m.Times {
			w.WriteInt32(time)
		}
		return w.Bytes()
	}
}
