package model

import (
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type MiniGameRecord struct {
	Unknown uint32
	Wins    uint32
	Ties    uint32
	Losses  uint32
	Points  uint32
}

func (m *MiniGameRecord) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt(m.Unknown)
		w.WriteInt(m.Wins)
		w.WriteInt(m.Ties)
		w.WriteInt(m.Losses)
		w.WriteInt(m.Points)
	}
}
