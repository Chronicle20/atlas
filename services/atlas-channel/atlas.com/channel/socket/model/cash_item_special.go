package model

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type SpecialCashItem struct {
	sn       uint32
	modifier uint32
	info     byte
}

func (s *SpecialCashItem) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt(s.sn)
		w.WriteInt(s.modifier)
		w.WriteByte(s.info)
	}
}
