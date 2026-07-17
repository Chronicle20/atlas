package mock

import (
	"atlas-inventory/asset"
	"atlas-inventory/compartment"
	"atlas-inventory/kafka/message"
	dropMsg "atlas-inventory/kafka/message/drop"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	WithTransactionFunc               func(db *gorm.DB) *compartment.ProcessorImpl
	WithAssetProcessorFunc            func(ap asset.Processor) *compartment.ProcessorImpl
	ByIdProviderFunc                  func(id uuid.UUID) model.Provider[compartment.Model]
	GetByIdFunc                       func(id uuid.UUID) (compartment.Model, error)
	ByCharacterIdProviderFunc         func(characterId uint32) model.Provider[[]compartment.Model]
	GetByCharacterIdFunc              func(characterId uint32) ([]compartment.Model, error)
	ByCharacterAndTypeProviderFunc    func(characterId uint32) func(inventoryType inventory.Type) model.Provider[compartment.Model]
	GetByCharacterAndTypeFunc         func(characterId uint32) func(inventoryType inventory.Type) (compartment.Model, error)
	DecorateAssetFunc                 func(m compartment.Model) (compartment.Model, error)
	CreateFunc                        func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, capacity uint32) (compartment.Model, error)
	DeleteByModelFunc                 func(mb *message.Buffer) func(transactionId uuid.UUID, c compartment.Model) error
	EquipItemAndEmitFunc              func(transactionId uuid.UUID, characterId uint32, source int16, destination int16) error
	EquipItemFunc                     func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, source int16, destination int16) error
	RemoveEquipAndEmitFunc            func(transactionId uuid.UUID, characterId uint32, source int16, destination int16) error
	RemoveEquipFunc                   func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, source int16, destination int16) error
	MoveAndEmitFunc                   func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, source int16, destination int16) error
	MoveAndLockFunc                   func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, source int16, destination int16) error
	MoveFunc                          func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, source int16, destination int16) error
	IncreaseCapacityAndEmitFunc       func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, amount uint32) error
	IncreaseCapacityFunc              func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, amount uint32) error
	DropAndEmitFunc                   func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, f field.Model, x int16, y int16, source int16, quantity int16) error
	DropFunc                          func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, f field.Model, x int16, y int16, source int16, quantity int16) error
	RequestReserveAndEmitFunc         func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, reservationRequests []compartment.ReservationRequest) error
	RequestReserveFunc                func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, reservationRequests []compartment.ReservationRequest) error
	CancelReservationAndEmitFunc      func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16) error
	CancelReservationFunc             func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16) error
	ConsumeAssetAndEmitFunc           func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16) error
	ConsumeAssetFunc                  func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16) error
	DestroyAssetAndEmitFunc           func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error
	DestroyAssetFunc                  func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error
	ExpireAssetAndEmitFunc            func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, isCash bool, replaceItemId uint32, replaceMessage string) error
	ExpireAssetFunc                   func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, isCash bool, replaceItemId uint32, replaceMessage string) error
	CreateAssetAndEmitFunc            func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time, ownerId uint32, flag uint16, rechargeable uint64, useAverageStats bool) error
	CreateAssetAndLockFunc            func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time, ownerId uint32, flag uint16, rechargeable uint64, useAverageStats bool) error
	CreateAssetFunc                   func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time, ownerId uint32, flag uint16, rechargeable uint64, useAverageStats bool) error
	AttemptEquipmentPickUpAndEmitFunc func(transactionId uuid.UUID, f field.Model, characterId uint32, dropId uint32, templateId uint32, ed dropMsg.EquipmentData) error
	AttemptEquipmentPickUpFunc        func(mb *message.Buffer) func(transactionId uuid.UUID, f field.Model, characterId uint32, dropId uint32, templateId uint32, ed dropMsg.EquipmentData) error
	AttemptItemPickUpAndEmitFunc      func(transactionId uuid.UUID, f field.Model, characterId uint32, dropId uint32, templateId uint32, quantity uint32) error
	AttemptItemPickUpFunc             func(mb *message.Buffer) func(transactionId uuid.UUID, f field.Model, characterId uint32, dropId uint32, templateId uint32, quantity uint32) error
	RechargeAssetAndEmitFunc          func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error
	RechargeAssetFunc                 func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error
	MergeAndCompactAndEmitFunc        func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type) error
	CompactAndSortAndEmitFunc         func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type) error
	MergeAndCompactFunc               func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type) error
	AcceptAndEmitFunc                 func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, m asset.Model) error
	AcceptFunc                        func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, m asset.Model) error
	ReleaseAndEmitFunc                func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, assetId uint32, quantity uint32) error
	ReleaseFunc                       func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, assetId uint32, quantity uint32) error
	CompactAndSortFunc                func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type) error
	ModifyEquipmentAndEmitFunc        func(transactionId uuid.UUID, characterId uint32, assetId uint32, stats asset.Model) error
	ModifyEquipmentFunc               func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, assetId uint32, stats asset.Model) error
	ChangeTemplateAndEmitFunc         func(transactionId uuid.UUID, characterId uint32, petId uint32, newTemplateId uint32) error
	ChangeTemplateFunc                func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, petId uint32, newTemplateId uint32) error
}

var _ compartment.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) WithTransaction(db *gorm.DB) *compartment.ProcessorImpl {
	if m.WithTransactionFunc != nil {
		return m.WithTransactionFunc(db)
	}
	return nil
}

func (m *ProcessorMock) WithAssetProcessor(ap asset.Processor) *compartment.ProcessorImpl {
	if m.WithAssetProcessorFunc != nil {
		return m.WithAssetProcessorFunc(ap)
	}
	return nil
}

func (m *ProcessorMock) ByIdProvider(id uuid.UUID) model.Provider[compartment.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return model.FixedProvider(compartment.Model{})
}

func (m *ProcessorMock) GetById(id uuid.UUID) (compartment.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return compartment.Model{}, nil
}

func (m *ProcessorMock) ByCharacterIdProvider(characterId uint32) model.Provider[[]compartment.Model] {
	if m.ByCharacterIdProviderFunc != nil {
		return m.ByCharacterIdProviderFunc(characterId)
	}
	return model.FixedProvider([]compartment.Model{})
}

func (m *ProcessorMock) GetByCharacterId(characterId uint32) ([]compartment.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId)
	}
	return []compartment.Model{}, nil
}

func (m *ProcessorMock) ByCharacterAndTypeProvider(characterId uint32) func(inventoryType inventory.Type) model.Provider[compartment.Model] {
	if m.ByCharacterAndTypeProviderFunc != nil {
		return m.ByCharacterAndTypeProviderFunc(characterId)
	}
	return func(inventoryType inventory.Type) model.Provider[compartment.Model] {
		return model.FixedProvider(compartment.Model{})
	}
}

func (m *ProcessorMock) GetByCharacterAndType(characterId uint32) func(inventoryType inventory.Type) (compartment.Model, error) {
	if m.GetByCharacterAndTypeFunc != nil {
		return m.GetByCharacterAndTypeFunc(characterId)
	}
	return func(inventoryType inventory.Type) (compartment.Model, error) {
		return compartment.Model{}, nil
	}
}

func (m *ProcessorMock) DecorateAsset(c compartment.Model) (compartment.Model, error) {
	if m.DecorateAssetFunc != nil {
		return m.DecorateAssetFunc(c)
	}
	return compartment.Model{}, nil
}

func (m *ProcessorMock) Create(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, capacity uint32) (compartment.Model, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, capacity uint32) (compartment.Model, error) {
		return compartment.Model{}, nil
	}
}

func (m *ProcessorMock) DeleteByModel(mb *message.Buffer) func(transactionId uuid.UUID, c compartment.Model) error {
	if m.DeleteByModelFunc != nil {
		return m.DeleteByModelFunc(mb)
	}
	return func(transactionId uuid.UUID, c compartment.Model) error {
		return nil
	}
}

func (m *ProcessorMock) EquipItemAndEmit(transactionId uuid.UUID, characterId uint32, source int16, destination int16) error {
	if m.EquipItemAndEmitFunc != nil {
		return m.EquipItemAndEmitFunc(transactionId, characterId, source, destination)
	}
	return nil
}

func (m *ProcessorMock) EquipItem(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, source int16, destination int16) error {
	if m.EquipItemFunc != nil {
		return m.EquipItemFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, source int16, destination int16) error {
		return nil
	}
}

func (m *ProcessorMock) RemoveEquipAndEmit(transactionId uuid.UUID, characterId uint32, source int16, destination int16) error {
	if m.RemoveEquipAndEmitFunc != nil {
		return m.RemoveEquipAndEmitFunc(transactionId, characterId, source, destination)
	}
	return nil
}

func (m *ProcessorMock) RemoveEquip(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, source int16, destination int16) error {
	if m.RemoveEquipFunc != nil {
		return m.RemoveEquipFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, source int16, destination int16) error {
		return nil
	}
}

func (m *ProcessorMock) MoveAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, source int16, destination int16) error {
	if m.MoveAndEmitFunc != nil {
		return m.MoveAndEmitFunc(transactionId, characterId, inventoryType, source, destination)
	}
	return nil
}

func (m *ProcessorMock) MoveAndLock(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, source int16, destination int16) error {
	if m.MoveAndLockFunc != nil {
		return m.MoveAndLockFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, source int16, destination int16) error {
		return nil
	}
}

func (m *ProcessorMock) Move(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, source int16, destination int16) error {
	if m.MoveFunc != nil {
		return m.MoveFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, source int16, destination int16) error {
		return nil
	}
}

func (m *ProcessorMock) IncreaseCapacityAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, amount uint32) error {
	if m.IncreaseCapacityAndEmitFunc != nil {
		return m.IncreaseCapacityAndEmitFunc(transactionId, characterId, inventoryType, amount)
	}
	return nil
}

func (m *ProcessorMock) IncreaseCapacity(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, amount uint32) error {
	if m.IncreaseCapacityFunc != nil {
		return m.IncreaseCapacityFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, amount uint32) error {
		return nil
	}
}

func (m *ProcessorMock) DropAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, f field.Model, x int16, y int16, source int16, quantity int16) error {
	if m.DropAndEmitFunc != nil {
		return m.DropAndEmitFunc(transactionId, characterId, inventoryType, f, x, y, source, quantity)
	}
	return nil
}

func (m *ProcessorMock) Drop(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, f field.Model, x int16, y int16, source int16, quantity int16) error {
	if m.DropFunc != nil {
		return m.DropFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, f field.Model, x int16, y int16, source int16, quantity int16) error {
		return nil
	}
}

func (m *ProcessorMock) RequestReserveAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, reservationRequests []compartment.ReservationRequest) error {
	if m.RequestReserveAndEmitFunc != nil {
		return m.RequestReserveAndEmitFunc(transactionId, characterId, inventoryType, reservationRequests)
	}
	return nil
}

func (m *ProcessorMock) RequestReserve(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, reservationRequests []compartment.ReservationRequest) error {
	if m.RequestReserveFunc != nil {
		return m.RequestReserveFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, reservationRequests []compartment.ReservationRequest) error {
		return nil
	}
}

func (m *ProcessorMock) CancelReservationAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16) error {
	if m.CancelReservationAndEmitFunc != nil {
		return m.CancelReservationAndEmitFunc(transactionId, characterId, inventoryType, slot)
	}
	return nil
}

func (m *ProcessorMock) CancelReservation(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16) error {
	if m.CancelReservationFunc != nil {
		return m.CancelReservationFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16) error {
		return nil
	}
}

func (m *ProcessorMock) ConsumeAssetAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16) error {
	if m.ConsumeAssetAndEmitFunc != nil {
		return m.ConsumeAssetAndEmitFunc(transactionId, characterId, inventoryType, slot)
	}
	return nil
}

func (m *ProcessorMock) ConsumeAsset(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16) error {
	if m.ConsumeAssetFunc != nil {
		return m.ConsumeAssetFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16) error {
		return nil
	}
}

func (m *ProcessorMock) DestroyAssetAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
	if m.DestroyAssetAndEmitFunc != nil {
		return m.DestroyAssetAndEmitFunc(transactionId, characterId, inventoryType, slot, quantity)
	}
	return nil
}

func (m *ProcessorMock) DestroyAsset(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
	if m.DestroyAssetFunc != nil {
		return m.DestroyAssetFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
		return nil
	}
}

func (m *ProcessorMock) ExpireAssetAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, isCash bool, replaceItemId uint32, replaceMessage string) error {
	if m.ExpireAssetAndEmitFunc != nil {
		return m.ExpireAssetAndEmitFunc(transactionId, characterId, inventoryType, slot, isCash, replaceItemId, replaceMessage)
	}
	return nil
}

func (m *ProcessorMock) ExpireAsset(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, isCash bool, replaceItemId uint32, replaceMessage string) error {
	if m.ExpireAssetFunc != nil {
		return m.ExpireAssetFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, isCash bool, replaceItemId uint32, replaceMessage string) error {
		return nil
	}
}

func (m *ProcessorMock) CreateAssetAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time, ownerId uint32, flag uint16, rechargeable uint64, useAverageStats bool) error {
	if m.CreateAssetAndEmitFunc != nil {
		return m.CreateAssetAndEmitFunc(transactionId, characterId, inventoryType, templateId, quantity, expiration, ownerId, flag, rechargeable, useAverageStats)
	}
	return nil
}

func (m *ProcessorMock) CreateAssetAndLock(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time, ownerId uint32, flag uint16, rechargeable uint64, useAverageStats bool) error {
	if m.CreateAssetAndLockFunc != nil {
		return m.CreateAssetAndLockFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time, ownerId uint32, flag uint16, rechargeable uint64, useAverageStats bool) error {
		return nil
	}
}

func (m *ProcessorMock) CreateAsset(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time, ownerId uint32, flag uint16, rechargeable uint64, useAverageStats bool) error {
	if m.CreateAssetFunc != nil {
		return m.CreateAssetFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time, ownerId uint32, flag uint16, rechargeable uint64, useAverageStats bool) error {
		return nil
	}
}

func (m *ProcessorMock) AttemptEquipmentPickUpAndEmit(transactionId uuid.UUID, f field.Model, characterId uint32, dropId uint32, templateId uint32, ed dropMsg.EquipmentData) error {
	if m.AttemptEquipmentPickUpAndEmitFunc != nil {
		return m.AttemptEquipmentPickUpAndEmitFunc(transactionId, f, characterId, dropId, templateId, ed)
	}
	return nil
}

func (m *ProcessorMock) AttemptEquipmentPickUp(mb *message.Buffer) func(transactionId uuid.UUID, f field.Model, characterId uint32, dropId uint32, templateId uint32, ed dropMsg.EquipmentData) error {
	if m.AttemptEquipmentPickUpFunc != nil {
		return m.AttemptEquipmentPickUpFunc(mb)
	}
	return func(transactionId uuid.UUID, f field.Model, characterId uint32, dropId uint32, templateId uint32, ed dropMsg.EquipmentData) error {
		return nil
	}
}

func (m *ProcessorMock) AttemptItemPickUpAndEmit(transactionId uuid.UUID, f field.Model, characterId uint32, dropId uint32, templateId uint32, quantity uint32) error {
	if m.AttemptItemPickUpAndEmitFunc != nil {
		return m.AttemptItemPickUpAndEmitFunc(transactionId, f, characterId, dropId, templateId, quantity)
	}
	return nil
}

func (m *ProcessorMock) AttemptItemPickUp(mb *message.Buffer) func(transactionId uuid.UUID, f field.Model, characterId uint32, dropId uint32, templateId uint32, quantity uint32) error {
	if m.AttemptItemPickUpFunc != nil {
		return m.AttemptItemPickUpFunc(mb)
	}
	return func(transactionId uuid.UUID, f field.Model, characterId uint32, dropId uint32, templateId uint32, quantity uint32) error {
		return nil
	}
}

func (m *ProcessorMock) RechargeAssetAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
	if m.RechargeAssetAndEmitFunc != nil {
		return m.RechargeAssetAndEmitFunc(transactionId, characterId, inventoryType, slot, quantity)
	}
	return nil
}

func (m *ProcessorMock) RechargeAsset(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
	if m.RechargeAssetFunc != nil {
		return m.RechargeAssetFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
		return nil
	}
}

func (m *ProcessorMock) MergeAndCompactAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type) error {
	if m.MergeAndCompactAndEmitFunc != nil {
		return m.MergeAndCompactAndEmitFunc(transactionId, characterId, inventoryType)
	}
	return nil
}

func (m *ProcessorMock) CompactAndSortAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type) error {
	if m.CompactAndSortAndEmitFunc != nil {
		return m.CompactAndSortAndEmitFunc(transactionId, characterId, inventoryType)
	}
	return nil
}

func (m *ProcessorMock) MergeAndCompact(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type) error {
	if m.MergeAndCompactFunc != nil {
		return m.MergeAndCompactFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type) error {
		return nil
	}
}

func (m *ProcessorMock) AcceptAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, a asset.Model) error {
	if m.AcceptAndEmitFunc != nil {
		return m.AcceptAndEmitFunc(transactionId, characterId, inventoryType, a)
	}
	return nil
}

func (m *ProcessorMock) Accept(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, a asset.Model) error {
	if m.AcceptFunc != nil {
		return m.AcceptFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, a asset.Model) error {
		return nil
	}
}

func (m *ProcessorMock) ReleaseAndEmit(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, assetId uint32, quantity uint32) error {
	if m.ReleaseAndEmitFunc != nil {
		return m.ReleaseAndEmitFunc(transactionId, characterId, inventoryType, assetId, quantity)
	}
	return nil
}

func (m *ProcessorMock) Release(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, assetId uint32, quantity uint32) error {
	if m.ReleaseFunc != nil {
		return m.ReleaseFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, assetId uint32, quantity uint32) error {
		return nil
	}
}

func (m *ProcessorMock) CompactAndSort(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type) error {
	if m.CompactAndSortFunc != nil {
		return m.CompactAndSortFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type) error {
		return nil
	}
}

func (m *ProcessorMock) ModifyEquipmentAndEmit(transactionId uuid.UUID, characterId uint32, assetId uint32, stats asset.Model) error {
	if m.ModifyEquipmentAndEmitFunc != nil {
		return m.ModifyEquipmentAndEmitFunc(transactionId, characterId, assetId, stats)
	}
	return nil
}

func (m *ProcessorMock) ModifyEquipment(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, assetId uint32, stats asset.Model) error {
	if m.ModifyEquipmentFunc != nil {
		return m.ModifyEquipmentFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, assetId uint32, stats asset.Model) error {
		return nil
	}
}

func (m *ProcessorMock) ChangeTemplateAndEmit(transactionId uuid.UUID, characterId uint32, petId uint32, newTemplateId uint32) error {
	if m.ChangeTemplateAndEmitFunc != nil {
		return m.ChangeTemplateAndEmitFunc(transactionId, characterId, petId, newTemplateId)
	}
	return nil
}

func (m *ProcessorMock) ChangeTemplate(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, petId uint32, newTemplateId uint32) error {
	if m.ChangeTemplateFunc != nil {
		return m.ChangeTemplateFunc(mb)
	}
	return func(transactionId uuid.UUID, characterId uint32, petId uint32, newTemplateId uint32) error {
		return nil
	}
}
