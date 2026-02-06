package message

import (
	"atlas-messengers/kafka/producer"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

type Buffer struct {
	buffer map[string][]kafka.Message
}

func NewBuffer() *Buffer {
	return &Buffer{
		buffer: make(map[string][]kafka.Message),
	}
}

func (b *Buffer) Put(t string, p model.Provider[[]kafka.Message]) error {
	ms, err := p()
	if err != nil {
		return err
	}
	b.buffer[t] = append(b.buffer[t], ms...)
	return nil
}

func (b *Buffer) GetAll() map[string][]kafka.Message {
	return b.buffer
}

func Emit(p producer.Provider) func(f func(buf *Buffer) error) error {
	return func(f func(buf *Buffer) error) error {
		b := NewBuffer()
		err := f(b)
		if err != nil {
			return err
		}
		for t, ms := range b.GetAll() {
			err = p(t)(model.FixedProvider(ms))
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func EmitWithResult[M any, B any](p producer.Provider) func(func(*Buffer) func(B) (M, error)) func(B) (M, error) {
	return func(f func(*Buffer) func(B) (M, error)) func(B) (M, error) {
		return func(input B) (M, error) {
			var buf = NewBuffer()
			result, err := f(buf)(input)
			if err != nil {
				return result, err
			}
			for t, ms := range buf.GetAll() {
				if err = p(t)(model.FixedProvider(ms)); err != nil {
					return result, err
				}
			}
			return result, nil
		}
	}
}

// EmitAlways emits all buffered events regardless of whether the function succeeded or failed.
// This is useful when error events need to be emitted on failure paths.
func EmitAlways[M any, B any](p producer.Provider) func(func(*Buffer) func(B) (M, error)) func(B) (M, error) {
	return func(f func(*Buffer) func(B) (M, error)) func(B) (M, error) {
		return func(input B) (M, error) {
			var buf = NewBuffer()
			result, fnErr := f(buf)(input)
			// Always emit buffered events, even on error
			for t, ms := range buf.GetAll() {
				if emitErr := p(t)(model.FixedProvider(ms)); emitErr != nil {
					// If emission fails, return that error (original error is lost)
					return result, emitErr
				}
			}
			return result, fnErr
		}
	}
}

// EmitAlwaysNoResult emits all buffered events regardless of success/failure for functions that return only error.
func EmitAlwaysNoResult[B any](p producer.Provider) func(func(*Buffer) func(B) error) func(B) error {
	return func(f func(*Buffer) func(B) error) func(B) error {
		return func(input B) error {
			var buf = NewBuffer()
			fnErr := f(buf)(input)
			// Always emit buffered events, even on error
			for t, ms := range buf.GetAll() {
				if emitErr := p(t)(model.FixedProvider(ms)); emitErr != nil {
					return emitErr
				}
			}
			return fnErr
		}
	}
}
