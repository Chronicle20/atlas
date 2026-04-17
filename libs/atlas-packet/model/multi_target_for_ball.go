package model

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type MultiTargetForBall struct {
	Targets []Position
}

func (m *MultiTargetForBall) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		size := r.ReadUint32()
		m.Targets = make([]Position, size)
		for i := 0; i < int(size); i++ {
			p := Position{}
			p.Decode(l, ctx)(r, options)
			m.Targets[i] = p
		}
	}
}

func (m *MultiTargetForBall) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(int32(len(m.Targets)))
		for _, target := range m.Targets {
			w.WriteByteArray(target.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}
