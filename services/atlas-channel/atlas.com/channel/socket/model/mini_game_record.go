package model

import (
	"context"

	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type MiniGameRecord struct {
	Unknown uint32
	Wins    uint32
	Ties    uint32
	Losses  uint32
	Points  uint32
}

func (m *MiniGameRecord) Encoder(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.Unknown)
		w.WriteInt(m.Wins)
		w.WriteInt(m.Ties)
		w.WriteInt(m.Losses)
		w.WriteInt(m.Points)
		return w.Bytes()
	}
}
