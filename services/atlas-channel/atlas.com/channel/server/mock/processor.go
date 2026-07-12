package mock

import (
	"atlas-channel/server"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type ProcessorMock struct {
	RegisterFunc func(t tenant.Model, ch channel.Model, ipAddress string, port int) server.Model
	GetAllFunc   func() []server.Model
}

var _ server.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Register(t tenant.Model, ch channel.Model, ipAddress string, port int) server.Model {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(t, ch, ipAddress, port)
	}
	return server.Model{}
}

func (m *ProcessorMock) GetAll() []server.Model {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return nil
}
