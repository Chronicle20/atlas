package model

import (
	"context"

	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type CategoryDiscount struct {
	category     byte
	categorySub  byte
	discountRate byte
}

func (s *CategoryDiscount) Encoder(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(s.category)
		w.WriteByte(s.categorySub)
		w.WriteByte(s.discountRate)
		return w.Bytes()
	}
}
