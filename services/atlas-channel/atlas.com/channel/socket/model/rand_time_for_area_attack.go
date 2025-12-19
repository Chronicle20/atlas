package model

import (
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type RandTimeForAreaAttack struct {
	Times []int32
}

func (m *RandTimeForAreaAttack) Decode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		size := r.ReadUint32()
		m.Times = make([]int32, size)
		for i := 0; i < int(size); i++ {
			m.Times[i] = r.ReadInt32()
		}
	}
}

func (m *RandTimeForAreaAttack) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt32(int32(len(m.Times)))
		for _, time := range m.Times {
			w.WriteInt32(time)
		}
	}
}
