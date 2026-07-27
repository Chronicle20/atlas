package packet

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

type Decode func(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{})

func (d Decode) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return d(l, ctx)
}

type Decoder interface {
	Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{})
}
