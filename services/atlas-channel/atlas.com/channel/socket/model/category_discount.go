package model

import (
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type CategoryDiscount struct {
	category     byte
	categorySub  byte
	discountRate byte
}

func (s *CategoryDiscount) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteByte(s.category)
		w.WriteByte(s.categorySub)
		w.WriteByte(s.discountRate)
	}
}
