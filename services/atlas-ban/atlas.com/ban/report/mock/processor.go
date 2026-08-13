package mock

import (
	"atlas-ban/kafka/message"
	report2 "atlas-ban/kafka/message/report"
	"atlas-ban/report"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

var _ report.Processor = (*ProcessorMock)(nil)

type ProcessorMock struct {
	CreateFromCommandFunc        func(buf *message.Buffer) func(c report2.CreateCommandBody) error
	CreateFromCommandAndEmitFunc func(c report2.CreateCommandBody) error
	UpdateStatusFunc             func(reportId uuid.UUID, status report.Status) (report.Model, error)
	GetByIdFunc                  func(reportId uuid.UUID) (report.Model, error)
	ByIdProviderFunc             func(reportId uuid.UUID) model.Provider[report.Model]
	GetByTenantFunc              func() ([]report.Model, error)
	GetByStatusFunc              func(status report.Status) ([]report.Model, error)
}

func (m *ProcessorMock) CreateFromCommand(buf *message.Buffer) func(c report2.CreateCommandBody) error {
	if m.CreateFromCommandFunc != nil {
		return m.CreateFromCommandFunc(buf)
	}
	return func(report2.CreateCommandBody) error { return nil }
}

func (m *ProcessorMock) CreateFromCommandAndEmit(c report2.CreateCommandBody) error {
	if m.CreateFromCommandAndEmitFunc != nil {
		return m.CreateFromCommandAndEmitFunc(c)
	}
	return nil
}

func (m *ProcessorMock) UpdateStatus(reportId uuid.UUID, status report.Status) (report.Model, error) {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(reportId, status)
	}
	return report.Model{}, nil
}

func (m *ProcessorMock) GetById(reportId uuid.UUID) (report.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(reportId)
	}
	return report.Model{}, nil
}

func (m *ProcessorMock) ByIdProvider(reportId uuid.UUID) model.Provider[report.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(reportId)
	}
	return model.FixedProvider(report.Model{})
}

func (m *ProcessorMock) GetByTenant() ([]report.Model, error) {
	if m.GetByTenantFunc != nil {
		return m.GetByTenantFunc()
	}
	return nil, nil
}

func (m *ProcessorMock) GetByStatus(status report.Status) ([]report.Model, error) {
	if m.GetByStatusFunc != nil {
		return m.GetByStatusFunc(status)
	}
	return nil, nil
}
