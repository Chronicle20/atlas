package packet

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Encode func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte

func (e Encode) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	return e(l, ctx)
}

type Encoder interface {
	Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte
}
