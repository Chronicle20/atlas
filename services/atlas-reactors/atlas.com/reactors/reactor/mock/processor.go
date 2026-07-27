package mock

import (
	"atlas-reactors/reactor"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type ProcessorMock struct {
	GetByIdFunc           func(id uint32) (reactor.Model, error)
	GetInFieldFunc        func(f field.Model) ([]reactor.Model, error)
	CreateFunc            func(b *reactor.ModelBuilder) error
	DestroyInFieldFunc    func(f field.Model)
	TeardownFunc          func() func()
	DestroyAllFunc        func() error
	DestroyInTenantFunc   func(t tenant.Model) model.Operator[[]reactor.Model]
	DestroyFunc           func() model.Operator[reactor.Model]
	HitFunc               func(reactorId uint32, characterId uint32, skillId uint32) error
	TriggerFunc           func(r reactor.Model, characterId uint32)
	TriggerAndDestroyFunc func(r reactor.Model, characterId uint32) error
}

var _ reactor.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(id uint32) (reactor.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return reactor.Model{}, nil
}

func (m *ProcessorMock) GetInField(f field.Model) ([]reactor.Model, error) {
	if m.GetInFieldFunc != nil {
		return m.GetInFieldFunc(f)
	}
	return nil, nil
}

func (m *ProcessorMock) Create(b *reactor.ModelBuilder) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(b)
	}
	return nil
}

func (m *ProcessorMock) DestroyInField(f field.Model) {
	if m.DestroyInFieldFunc != nil {
		m.DestroyInFieldFunc(f)
	}
}

func (m *ProcessorMock) Teardown() func() {
	if m.TeardownFunc != nil {
		return m.TeardownFunc()
	}
	return func() {}
}

func (m *ProcessorMock) DestroyAll() error {
	if m.DestroyAllFunc != nil {
		return m.DestroyAllFunc()
	}
	return nil
}

func (m *ProcessorMock) DestroyInTenant(t tenant.Model) model.Operator[[]reactor.Model] {
	if m.DestroyInTenantFunc != nil {
		return m.DestroyInTenantFunc(t)
	}
	return func(models []reactor.Model) error {
		return nil
	}
}

func (m *ProcessorMock) Destroy() model.Operator[reactor.Model] {
	if m.DestroyFunc != nil {
		return m.DestroyFunc()
	}
	return func(r reactor.Model) error {
		return nil
	}
}

func (m *ProcessorMock) Hit(reactorId uint32, characterId uint32, skillId uint32) error {
	if m.HitFunc != nil {
		return m.HitFunc(reactorId, characterId, skillId)
	}
	return nil
}

func (m *ProcessorMock) Trigger(r reactor.Model, characterId uint32) {
	if m.TriggerFunc != nil {
		m.TriggerFunc(r, characterId)
	}
}

func (m *ProcessorMock) TriggerAndDestroy(r reactor.Model, characterId uint32) error {
	if m.TriggerAndDestroyFunc != nil {
		return m.TriggerAndDestroyFunc(r, characterId)
	}
	return nil
}
