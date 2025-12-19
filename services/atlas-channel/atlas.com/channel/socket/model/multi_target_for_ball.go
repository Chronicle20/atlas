package model

import (
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type MultiTargetForBall struct {
	Targets []Position
}

func (m *MultiTargetForBall) Decode(l logrus.FieldLogger, t tenant.Model, ops map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		size := r.ReadUint32()
		m.Targets = make([]Position, size)
		for i := 0; i < int(size); i++ {
			p := Position{}
			p.Decode(l, t, ops)(r)
			m.Targets[i] = p
		}
	}
}

func (m *MultiTargetForBall) Encode(l logrus.FieldLogger, t tenant.Model, ops map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt32(int32(len(m.Targets)))
		for _, target := range m.Targets {
			target.Encode(l, t, ops)(w)
		}
	}
}
