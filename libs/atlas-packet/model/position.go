package model

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Position struct {
	x int32
	y int32
}

func NewPosition(x int32, y int32) Position {
	return Position{x: x, y: y}
}

func (m *Position) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.x = r.ReadInt32()
		m.y = r.ReadInt32()
	}
}

func (m *Position) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(_ map[string]interface{}) []byte {
		w.WriteInt32(m.x)
		w.WriteInt32(m.y)
		return w.Bytes()
	}
}

func (m *Position) X() int32 {
	return m.x
}

func (m *Position) Y() int32 {
	return m.y
}
