package writer

import (
	"errors"
)

type BodyProducer func(op uint16) []byte

type HeaderFunc func(op uint16) BodyFunc

type BodyFunc func(rem BodyProducer) []byte

func MessageGetter(op uint16) BodyFunc {
	return func(rem BodyProducer) []byte {
		return rem(op)
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
