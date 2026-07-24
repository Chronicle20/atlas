package mock

import (
	"atlas-rps/saga"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// ProcessorMock is a test double for saga.Processor. The CreateFunc field is
// used when set; otherwise the method returns a nil error.
type ProcessorMock struct {
	CreateFunc func(s sharedsaga.Saga) error
}

var _ saga.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Create(s sharedsaga.Saga) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(s)
	}
	return nil
}
