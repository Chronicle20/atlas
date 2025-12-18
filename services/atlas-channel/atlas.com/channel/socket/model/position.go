package model

import (
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Position struct {
	x int32
	y int32
}

func NewPosition(x int32, y int32) Position {
	return Position{x: x, y: y}
}

func (m *Position) Decode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(r *request.Reader) {
	return func(r *request.Reader) {
		m.x = r.ReadInt32()
		m.y = r.ReadInt32()
	}
}

func (m *Position) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt32(m.x)
		w.WriteInt32(m.y)
	}
}

func (m *Position) X() int32 {
	return m.x
}

func (m *Position) Y() int32 {
	return m.y
}
