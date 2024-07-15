package writer

import (
	"errors"
)

type BodyProducer func(op uint16, options map[string]interface{}) []byte

type HeaderFunc func(op uint16, options map[string]interface{}) BodyFunc

type BodyFunc func(rem BodyProducer) []byte

func MessageGetter(op uint16, options map[string]interface{}) BodyFunc {
	return func(rem BodyProducer) []byte {
		return rem(op, options)
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
