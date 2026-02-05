package mock

import (
	"atlas-transports/channel"

	channel2 "github.com/Chronicle20/atlas-constants/channel"
)

// Compile-time interface compliance check
var _ channel.Processor = (*ProcessorMock)(nil)

// ProcessorMock is a mock implementation of the channel.Processor interface
type ProcessorMock struct {
	RegisterFunc   func(ch channel2.Model) error
	UnregisterFunc func(ch channel2.Model) error
	GetAllFunc     func() []channel2.Model
}

// Register is a mock implementation
func (m *ProcessorMock) Register(ch channel2.Model) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ch)
	}
	return nil
}

// Unregister is a mock implementation
func (m *ProcessorMock) Unregister(ch channel2.Model) error {
	if m.UnregisterFunc != nil {
		return m.UnregisterFunc(ch)
	}
	return nil
}

// GetAll is a mock implementation
func (m *ProcessorMock) GetAll() []channel2.Model {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return []channel2.Model{}
}
