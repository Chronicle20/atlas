package model

import (
	"context"

	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type SpecialCashItem struct {
	sn       uint32
	modifier uint32
	info     byte
}

func (s *SpecialCashItem) Encoder(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(s.sn)
		w.WriteInt(s.modifier)
		w.WriteByte(s.info)
		return w.Bytes()
	}
}
