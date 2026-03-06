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

func getCode[E string](l logrus.FieldLogger) func(requester string, code E, codeProperty string, options map[string]interface{}) byte {
	return func(requester string, code E, codeProperty string, options map[string]interface{}) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options[codeProperty]; !ok {
			l.Errorf("Code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", code, requester)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", code, requester)
			return 99
		}

		res, ok := codes[string(code)].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", code, requester)
			return 99
		}
		return byte(res)
	}
}
