package model

import (
	"context"

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

func (m *MultiTargetForBall) Encoder(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(int32(len(m.Targets)))
		for _, target := range m.Targets {
			w.WriteByteArray(target.Encoder(l, ctx)(options))
		}
		return w.Bytes()
	}
}
