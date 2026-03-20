package writer

import (
	"context"
	"errors"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type BodyFunc func(l logrus.FieldLogger, ctx context.Context) func(encoder packet.Encode) []byte

func MessageGetter(opWriter func(w *response.Writer), options map[string]interface{}) BodyFunc {
	return func(l logrus.FieldLogger, ctx context.Context) func(encoder packet.Encode) []byte {
		return func(encoder packet.Encode) []byte {
			w := response.NewWriter(l)
			opWriter(w)
			w.WriteByteArray(encoder(l, ctx)(options))
			return w.Bytes()
		}
	}
}

type Producer func(name string) (BodyFunc, error)

func ProducerGetter(wm map[string]BodyFunc) Producer {
	return func(name string) (BodyFunc, error) {
		if w, ok := wm[name]; ok {
			return w, nil
		}
		return nil, errors.New("writer not found")
	}
}
