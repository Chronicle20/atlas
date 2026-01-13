package mock

import (
	"atlas-inventory/inventory"
	"atlas-inventory/kafka/message"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProcessorImpl struct {
	WithTransactionFn     func(db *gorm.DB) inventory.Processor
	GetByCharacterIdFn    func(characterId uint32) (inventory.Model, error)
	ByCharacterIdProviderFn func(characterId uint32) model.Provider[inventory.Model]
	CreateAndEmitFn       func(transactionId uuid.UUID, characterId uint32) (inventory.Model, error)
	CreateFn              func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) (inventory.Model, error)
	DeleteAndEmitFn       func(transactionId uuid.UUID, characterId uint32) error
	DeleteFn              func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) error
}

func (p *ProcessorImpl) WithTransaction(db *gorm.DB) inventory.Processor {
	return p.WithTransactionFn(db)
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (inventory.Model, error) {
	return p.GetByCharacterIdFn(characterId)
}

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[inventory.Model] {
	return p.ByCharacterIdProviderFn(characterId)
}

func (p *ProcessorImpl) CreateAndEmit(transactionId uuid.UUID, characterId uint32) (inventory.Model, error) {
	return p.CreateAndEmitFn(transactionId, characterId)
}

func (p *ProcessorImpl) Create(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) (inventory.Model, error) {
	return p.CreateFn(mb)
}

func (p *ProcessorImpl) DeleteAndEmit(transactionId uuid.UUID, characterId uint32) error {
	return p.DeleteAndEmitFn(transactionId, characterId)
}

func (p *ProcessorImpl) Delete(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) error {
	return p.DeleteFn(mb)
}
