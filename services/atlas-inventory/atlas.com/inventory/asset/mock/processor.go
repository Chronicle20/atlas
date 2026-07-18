package mock

import (
	"atlas-inventory/asset"
	"atlas-inventory/data/consumable"
	"atlas-inventory/kafka/message"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	WithTransactionFunc              func(tx *gorm.DB) asset.Processor
	ConsumableProcessorFunc          func() consumable.Processor
	WithConsumableProcessorFunc      func(conp consumable.Processor) asset.Processor
	ByCompartmentIdProviderFunc      func(compartmentId uuid.UUID) model.Provider[[]asset.Model]
	GetByCompartmentIdFunc           func(compartmentId uuid.UUID) ([]asset.Model, error)
	ByCompartmentIdPagedProviderFunc func(compartmentId uuid.UUID, page model.Page) model.Provider[model.Paged[asset.Model]]
	GetBySlotFunc                    func(compartmentId uuid.UUID, slot int16) (asset.Model, error)
	BySlotProviderFunc               func(compartmentId uuid.UUID) func(slot int16) model.Provider[asset.Model]
	ByIdProviderFunc                 func(id uint32) model.Provider[asset.Model]
	GetByIdFunc                      func(id uint32) (asset.Model, error)
	GetSlotMaxFunc                   func(templateId uint32) (uint32, error)
	DeleteFunc                       func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a asset.Model) error
	ExpireFunc                       func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, isCash bool, replaceItemId uint32, replaceMessage string) func(a asset.Model) error
	DropFunc                         func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a asset.Model) error
	UpdateSlotFunc                   func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, ap model.Provider[asset.Model], sp model.Provider[int16]) error
	UpdateQuantityFunc               func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, a asset.Model, quantity uint32) error
	UpdateEquipmentStatsFunc         func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, assetId uint32, stats asset.Model) error
	ChangeTemplateAndEmitFunc        func(transactionId uuid.UUID, characterId uint32, assetId uint32, newTemplateId uint32) error
	ChangeTemplateFunc               func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, assetId uint32, newTemplateId uint32) error
	UpdateOwnerFunc                  func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a asset.Model, owner string) error
	ApplyLockFunc                    func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a asset.Model, expiration time.Time) error
	ClearLockFunc                    func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a asset.Model) error
	DeleteAndEmitFunc                func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, assetId uint32) error
	CreateFunc                       func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, templateId uint32, slot int16, opts asset.CreateOptions) (asset.Model, error)
	CreateFromModelFunc              func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, m asset.Model) (asset.Model, error)
	AcceptFunc                       func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, slot int16, m asset.Model) (asset.Model, error)
	ReleaseFunc                      func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a asset.Model) error
}

var _ asset.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) WithTransaction(tx *gorm.DB) asset.Processor {
	if m.WithTransactionFunc != nil {
		return m.WithTransactionFunc(tx)
	}
	return m
}

func (m *ProcessorMock) ConsumableProcessor() consumable.Processor {
	if m.ConsumableProcessorFunc != nil {
		return m.ConsumableProcessorFunc()
	}
	return nil
}

func (m *ProcessorMock) WithConsumableProcessor(conp consumable.Processor) asset.Processor {
	if m.WithConsumableProcessorFunc != nil {
		return m.WithConsumableProcessorFunc(conp)
	}
	return m
}

func (m *ProcessorMock) ByCompartmentIdProvider(compartmentId uuid.UUID) model.Provider[[]asset.Model] {
	if m.ByCompartmentIdProviderFunc != nil {
		return m.ByCompartmentIdProviderFunc(compartmentId)
	}
	return model.FixedProvider([]asset.Model{})
}

func (m *ProcessorMock) GetByCompartmentId(compartmentId uuid.UUID) ([]asset.Model, error) {
	if m.GetByCompartmentIdFunc != nil {
		return m.GetByCompartmentIdFunc(compartmentId)
	}
	return []asset.Model{}, nil
}

func (m *ProcessorMock) ByCompartmentIdPagedProvider(compartmentId uuid.UUID, page model.Page) model.Provider[model.Paged[asset.Model]] {
	if m.ByCompartmentIdPagedProviderFunc != nil {
		return m.ByCompartmentIdPagedProviderFunc(compartmentId, page)
	}
	return model.FixedProvider(model.Paged[asset.Model]{Page: page})
}

func (m *ProcessorMock) GetBySlot(compartmentId uuid.UUID, slot int16) (asset.Model, error) {
	if m.GetBySlotFunc != nil {
		return m.GetBySlotFunc(compartmentId, slot)
	}
	return asset.Model{}, nil
}

func (m *ProcessorMock) BySlotProvider(compartmentId uuid.UUID) func(slot int16) model.Provider[asset.Model] {
	if m.BySlotProviderFunc != nil {
		return m.BySlotProviderFunc(compartmentId)
	}
	return func(slot int16) model.Provider[asset.Model] {
		return model.FixedProvider(asset.Model{})
	}
}

func (m *ProcessorMock) ByIdProvider(id uint32) model.Provider[asset.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return model.FixedProvider(asset.Model{})
}

func (m *ProcessorMock) GetById(id uint32) (asset.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return asset.Model{}, nil
}

func (m *ProcessorMock) GetSlotMax(templateId uint32) (uint32, error) {
	if m.GetSlotMaxFunc != nil {
		return m.GetSlotMaxFunc(templateId)
	}
	return 0, nil
}

func (m *ProcessorMock) Delete(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a asset.Model) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a asset.Model) error {
		return func(a asset.Model) error {
			return nil
		}
	}
}

func (m *ProcessorMock) Expire(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, isCash bool, replaceItemId uint32, replaceMessage string) func(a asset.Model) error {
	if m.ExpireFunc != nil {
		return m.ExpireFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, isCash bool, replaceItemId uint32, replaceMessage string) func(a asset.Model) error {
		return func(a asset.Model) error {
			return nil
		}
	}
}

func (m *ProcessorMock) Drop(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a asset.Model) error {
	if m.DropFunc != nil {
		return m.DropFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a asset.Model) error {
		return func(a asset.Model) error {
			return nil
		}
	}
}

func (m *ProcessorMock) UpdateSlot(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, ap model.Provider[asset.Model], sp model.Provider[int16]) error {
	if m.UpdateSlotFunc != nil {
		return m.UpdateSlotFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, ap model.Provider[asset.Model], sp model.Provider[int16]) error {
		return nil
	}
}

func (m *ProcessorMock) UpdateQuantity(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, a asset.Model, quantity uint32) error {
	if m.UpdateQuantityFunc != nil {
		return m.UpdateQuantityFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, a asset.Model, quantity uint32) error {
		return nil
	}
}

func (m *ProcessorMock) UpdateEquipmentStats(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, assetId uint32, stats asset.Model) error {
	if m.UpdateEquipmentStatsFunc != nil {
		return m.UpdateEquipmentStatsFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, assetId uint32, stats asset.Model) error {
		return nil
	}
}

func (m *ProcessorMock) ChangeTemplateAndEmit(transactionId uuid.UUID, characterId uint32, assetId uint32, newTemplateId uint32) error {
	if m.ChangeTemplateAndEmitFunc != nil {
		return m.ChangeTemplateAndEmitFunc(transactionId, characterId, assetId, newTemplateId)
	}
	return nil
}

func (m *ProcessorMock) ChangeTemplate(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, assetId uint32, newTemplateId uint32) error {
	if m.ChangeTemplateFunc != nil {
		return m.ChangeTemplateFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, assetId uint32, newTemplateId uint32) error {
		return nil
	}
}

func (m *ProcessorMock) UpdateOwner(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a asset.Model, owner string) error {
	if m.UpdateOwnerFunc != nil {
		return m.UpdateOwnerFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32) func(a asset.Model, owner string) error {
		return func(a asset.Model, owner string) error {
			return nil
		}
	}
}

func (m *ProcessorMock) ApplyLock(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a asset.Model, expiration time.Time) error {
	if m.ApplyLockFunc != nil {
		return m.ApplyLockFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32) func(a asset.Model, expiration time.Time) error {
		return func(a asset.Model, expiration time.Time) error {
			return nil
		}
	}
}

func (m *ProcessorMock) ClearLock(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) func(a asset.Model) error {
	if m.ClearLockFunc != nil {
		return m.ClearLockFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32) func(a asset.Model) error {
		return func(a asset.Model) error {
			return nil
		}
	}
}

func (m *ProcessorMock) DeleteAndEmit(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, assetId uint32) error {
	if m.DeleteAndEmitFunc != nil {
		return m.DeleteAndEmitFunc(transactionId, characterId, compartmentId, assetId)
	}
	return nil
}

func (m *ProcessorMock) Create(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, templateId uint32, slot int16, opts asset.CreateOptions) (asset.Model, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, templateId uint32, slot int16, opts asset.CreateOptions) (asset.Model, error) {
		return asset.Model{}, nil
	}
}

func (m *ProcessorMock) CreateFromModel(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, m2 asset.Model) (asset.Model, error) {
	if m.CreateFromModelFunc != nil {
		return m.CreateFromModelFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, m2 asset.Model) (asset.Model, error) {
		return asset.Model{}, nil
	}
}

func (m *ProcessorMock) Accept(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, slot int16, m2 asset.Model) (asset.Model, error) {
	if m.AcceptFunc != nil {
		return m.AcceptFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, slot int16, m2 asset.Model) (asset.Model, error) {
		return asset.Model{}, nil
	}
}

func (m *ProcessorMock) Release(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a asset.Model) error {
	if m.ReleaseFunc != nil {
		return m.ReleaseFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID) func(a asset.Model) error {
		return func(a asset.Model) error {
			return nil
		}
	}
}
