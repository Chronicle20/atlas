// Package message is a minimal stand-in for a service-local kafka/message
// package (see e.g. services/atlas-fame/atlas.com/fame/kafka/message), used
// only to give the emitArgInTx fixture the same
// message.Emit(producer.ProviderImpl(...))(func(mb *Buffer) error {...})
// call shape as the real atlas-monster-book bug.
package message

type Buffer struct{}

func Emit(p interface{}) func(f func(mb *Buffer) error) error {
	return func(f func(mb *Buffer) error) error {
		return f(&Buffer{})
	}
}
