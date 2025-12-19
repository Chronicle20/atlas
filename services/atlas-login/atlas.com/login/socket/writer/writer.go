package writer

import (
	"errors"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type OpWriterProducer[E uint8 | uint16] func(op E) OpWriter

type OpWriter func(w *response.Writer)

type BodyProducer func(w *response.Writer, options map[string]interface{}) []byte

type BodyFunc func(l logrus.FieldLogger) func(rem BodyProducer) []byte

type HeaderFunc func(opWriter OpWriter, options map[string]interface{}) BodyFunc

func MessageGetter(opWriter OpWriter, options map[string]interface{}) BodyFunc {
	return func(l logrus.FieldLogger) func(rem BodyProducer) []byte {
		return func(rem BodyProducer) []byte {
			w := response.NewWriter(l)
			opWriter(w)
			return rem(w, options)
		}
	}
}

type Producer func(l logrus.FieldLogger, name string) (BodyFunc, error)

func ProducerGetter(wm map[string]BodyFunc) Producer {
	return func(l logrus.FieldLogger, name string) (BodyFunc, error) {
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
