package storage

import (
	"context"
	"time"

	"atlas-saga-orchestrator/kafka/message"
	storage2 "atlas-saga-orchestrator/kafka/message/storage"
	storageCompartment "atlas-saga-orchestrator/kafka/message/storage/compartment"
	"atlas-saga-orchestrator/kafka/producer"

	"github.com/Chronicle20/atlas-constants/asset"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	DepositAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, slot int16, templateId uint32, expiration time.Time, referenceId uint32, referenceType string, referenceData storage2.ReferenceData) error
	Deposit(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, accountId uint32, slot int16, templateId uint32, expiration time.Time, referenceId uint32, referenceType string, referenceData storage2.ReferenceData) error
	WithdrawAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId asset.Id, quantity asset.Quantity) error
	Withdraw(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId asset.Id, quantity asset.Quantity) error
	UpdateMesosAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, mesos uint32, operation string) error
	UpdateMesos(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, accountId uint32, mesos uint32, operation string) error
	DepositRollbackAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId asset.Id) error
	DepositRollback(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId asset.Id) error
	ShowStorageAndEmit(transactionId uuid.UUID, ch channel.Model, characterId uint32, npcId uint32, accountId uint32) error
	ShowStorage(mb *message.Buffer) func(transactionId uuid.UUID, ch channel.Model, characterId uint32, npcId uint32, accountId uint32) error
	AcceptAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, characterId uint32, slot int16, templateId uint32, referenceId uint32, referenceType string, referenceData []byte, quantity asset.Quantity) error
	Accept(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, accountId uint32, characterId uint32, slot int16, templateId uint32, referenceId uint32, referenceType string, referenceData []byte, quantity asset.Quantity) error
	ReleaseAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, characterId uint32, assetId asset.Id, quantity asset.Quantity) error
	Release(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, accountId uint32, characterId uint32, assetId asset.Id, quantity asset.Quantity) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	p   producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		p:   producer.ProviderImpl(l)(ctx),
	}
}

func (p *ProcessorImpl) DepositAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, slot int16, templateId uint32, expiration time.Time, referenceId uint32, referenceType string, referenceData storage2.ReferenceData) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Deposit(mb)(transactionId, worldId, accountId, slot, templateId, expiration, referenceId, referenceType, referenceData)
	})
}

func (p *ProcessorImpl) Deposit(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, accountId uint32, slot int16, templateId uint32, expiration time.Time, referenceId uint32, referenceType string, referenceData storage2.ReferenceData) error {
	return func(transactionId uuid.UUID, worldId world.Id, accountId uint32, slot int16, templateId uint32, expiration time.Time, referenceId uint32, referenceType string, referenceData storage2.ReferenceData) error {
		return mb.Put(storage2.EnvCommandTopic, DepositCommandProvider(transactionId, worldId, accountId, slot, templateId, expiration, referenceId, referenceType, referenceData))
	}
}

func (p *ProcessorImpl) WithdrawAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId asset.Id, quantity asset.Quantity) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Withdraw(mb)(transactionId, worldId, accountId, assetId, quantity)
	})
}

func (p *ProcessorImpl) Withdraw(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId asset.Id, quantity asset.Quantity) error {
	return func(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId asset.Id, quantity asset.Quantity) error {
		return mb.Put(storage2.EnvCommandTopic, WithdrawCommandProvider(transactionId, worldId, accountId, assetId, quantity))
	}
}

func (p *ProcessorImpl) UpdateMesosAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, mesos uint32, operation string) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.UpdateMesos(mb)(transactionId, worldId, accountId, mesos, operation)
	})
}

func (p *ProcessorImpl) UpdateMesos(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, accountId uint32, mesos uint32, operation string) error {
	return func(transactionId uuid.UUID, worldId world.Id, accountId uint32, mesos uint32, operation string) error {
		return mb.Put(storage2.EnvCommandTopic, UpdateMesosCommandProvider(transactionId, worldId, accountId, mesos, operation))
	}
}

func (p *ProcessorImpl) DepositRollbackAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId asset.Id) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.DepositRollback(mb)(transactionId, worldId, accountId, assetId)
	})
}

func (p *ProcessorImpl) DepositRollback(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId asset.Id) error {
	return func(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId asset.Id) error {
		return mb.Put(storage2.EnvCommandTopic, DepositRollbackCommandProvider(transactionId, worldId, accountId, assetId))
	}
}

func (p *ProcessorImpl) ShowStorageAndEmit(transactionId uuid.UUID, ch channel.Model, characterId uint32, npcId uint32, accountId uint32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.ShowStorage(mb)(transactionId, ch, characterId, npcId, accountId)
	})
}

func (p *ProcessorImpl) ShowStorage(mb *message.Buffer) func(transactionId uuid.UUID, ch channel.Model, characterId uint32, npcId uint32, accountId uint32) error {
	return func(transactionId uuid.UUID, ch channel.Model, characterId uint32, npcId uint32, accountId uint32) error {
		return mb.Put(storage2.EnvCommandTopic, ShowStorageCommandProvider(transactionId, ch, characterId, npcId, accountId))
	}
}

func (p *ProcessorImpl) AcceptAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, characterId uint32, slot int16, templateId uint32, referenceId uint32, referenceType string, referenceData []byte, quantity asset.Quantity) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Accept(mb)(transactionId, worldId, accountId, characterId, slot, templateId, referenceId, referenceType, referenceData, quantity)
	})
}

func (p *ProcessorImpl) Accept(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, accountId uint32, characterId uint32, slot int16, templateId uint32, referenceId uint32, referenceType string, referenceData []byte, quantity asset.Quantity) error {
	return func(transactionId uuid.UUID, worldId world.Id, accountId uint32, characterId uint32, slot int16, templateId uint32, referenceId uint32, referenceType string, referenceData []byte, quantity asset.Quantity) error {
		return mb.Put(storageCompartment.EnvCommandTopic, AcceptCommandProvider(transactionId, worldId, accountId, characterId, slot, templateId, referenceId, referenceType, referenceData, quantity))
	}
}

func (p *ProcessorImpl) ReleaseAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, characterId uint32, assetId asset.Id, quantity asset.Quantity) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Release(mb)(transactionId, worldId, accountId, characterId, assetId, quantity)
	})
}

func (p *ProcessorImpl) Release(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, accountId uint32, characterId uint32, assetId asset.Id, quantity asset.Quantity) error {
	return func(transactionId uuid.UUID, worldId world.Id, accountId uint32, characterId uint32, assetId asset.Id, quantity asset.Quantity) error {
		return mb.Put(storageCompartment.EnvCommandTopic, ReleaseCommandProvider(transactionId, worldId, accountId, characterId, assetId, quantity))
	}
}
