package transfer

import (
	"atlas-compartment-transfer/kafka/message"
	compartment4 "atlas-compartment-transfer/kafka/message/cashshop/compartment"
	compartment2 "atlas-compartment-transfer/kafka/message/character/compartment"
	"atlas-compartment-transfer/kafka/message/compartment"
	compartment7 "atlas-compartment-transfer/kafka/message/storage/compartment"
	"atlas-compartment-transfer/kafka/producer"
	compartment5 "atlas-compartment-transfer/kafka/producer/cashshop/compartment"
	compartment3 "atlas-compartment-transfer/kafka/producer/character/compartment"
	compartment6 "atlas-compartment-transfer/kafka/producer/compartment"
	compartment8 "atlas-compartment-transfer/kafka/producer/storage/compartment"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"sync"
)

// TransactionStep represents the next step in the transfer saga
type TransactionStep func(mb *message.Buffer) error

// TransferInfo holds information about a transfer
type TransferInfo struct {
	Step              TransactionStep
	WorldId           byte
	CharacterId       uint32
	AccountId         uint32
	AssetId           uint32
	ToCompartmentId   uuid.UUID
	ToCompartmentType byte
	ToInventoryType   string
}

// TransactionCache is a singleton that holds the transaction cache
type TransactionCache struct {
	txCache   map[uuid.UUID]TransferInfo
	cacheLock sync.RWMutex
}

// instance is the singleton instance of TransactionCache
var instance *TransactionCache
var once sync.Once

// GetTransactionCache returns the singleton instance of TransactionCache
func GetTransactionCache() *TransactionCache {
	once.Do(func() {
		instance = &TransactionCache{
			txCache:   make(map[uuid.UUID]TransferInfo),
			cacheLock: sync.RWMutex{},
		}
	})
	return instance
}

// Store stores transfer information in the cache
func (tc *TransactionCache) Store(transactionId uuid.UUID, info TransferInfo) {
	tc.cacheLock.Lock()
	defer tc.cacheLock.Unlock()
	tc.txCache[transactionId] = info
}

// Get retrieves transfer information from the cache
func (tc *TransactionCache) Get(transactionId uuid.UUID) (TransferInfo, bool) {
	tc.cacheLock.RLock()
	defer tc.cacheLock.RUnlock()
	info, exists := tc.txCache[transactionId]
	return info, exists
}

// Delete removes a transaction step from the cache
func (tc *TransactionCache) Delete(transactionId uuid.UUID) {
	tc.cacheLock.Lock()
	defer tc.cacheLock.Unlock()
	delete(tc.txCache, transactionId)
}

// Processor defines the interface for the transfer processor
type Processor interface {
	Process(mb *message.Buffer) func(cmd compartment.TransferCommand) error
	ProcessAndEmit(cmd compartment.TransferCommand) error
	HandleAccepted(mb *message.Buffer) func(transactionId uuid.UUID) error
	HandleAcceptedAndEmit(transactionId uuid.UUID) error
	HandleReleased(mb *message.Buffer) func(transactionId uuid.UUID) error
	HandleReleasedAndEmit(transactionId uuid.UUID) error
	HandleError(mb *message.Buffer) func(transactionId uuid.UUID) error
	HandleErrorAndEmit(transactionId uuid.UUID) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l        logrus.FieldLogger
	ctx      context.Context
	producer producer.Provider
}

// NewProcessor creates a new processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:        l,
		ctx:      ctx,
		producer: producer.ProviderImpl(l)(ctx),
	}
}

// Process handles the transfer command
func (p *ProcessorImpl) Process(mb *message.Buffer) func(cmd compartment.TransferCommand) error {
	return func(cmd compartment.TransferCommand) error {
		p.l.Debugf("Initiating compartment transfer [%s] for character [%d].", cmd.TransactionId, cmd.CharacterId)

		// Step 1: Handle FromInventoryType
		var info TransferInfo
		nextStep := p.createReleaseStep(cmd)

		if cmd.ToInventoryType == compartment.InventoryTypeCharacter {
			p.l.Debugf("Informing [%s] inventory to receive that [%d] via transfer [%s].", cmd.ToInventoryType, cmd.AssetId, cmd.TransactionId)
			_ = mb.Put(compartment2.EnvCommandTopic, compartment3.AcceptCommandProvider(cmd.CharacterId, cmd.ToCompartmentType, cmd.TransactionId, cmd.ReferenceId))

			info = TransferInfo{
				Step:              nextStep,
				CharacterId:       cmd.CharacterId,
				AccountId:         cmd.AccountId,
				AssetId:           cmd.AssetId,
				ToCompartmentId:   cmd.ToCompartmentId,
				ToCompartmentType: cmd.ToCompartmentType,
				ToInventoryType:   cmd.ToInventoryType,
			}

		} else if cmd.ToInventoryType == compartment.InventoryTypeCashShop {
			p.l.Debugf("Informing [%s] inventory to receive that [%d] via transfer [%s].", cmd.ToInventoryType, cmd.AssetId, cmd.TransactionId)
			_ = mb.Put(compartment4.EnvCommandTopic, compartment5.AcceptCommandProvider(cmd.AccountId, cmd.ToCompartmentId, cmd.ToCompartmentType, cmd.TransactionId, cmd.ReferenceId))

			info = TransferInfo{
				Step:              nextStep,
				CharacterId:       cmd.CharacterId,
				AccountId:         cmd.AccountId,
				AssetId:           cmd.ReferenceId,
				ToCompartmentId:   cmd.ToCompartmentId,
				ToCompartmentType: cmd.ToCompartmentType,
				ToInventoryType:   cmd.ToInventoryType,
			}
		} else if cmd.ToInventoryType == compartment.InventoryTypeStorage {
			p.l.Debugf("Informing [%s] inventory to receive that [%d] via transfer [%s].", cmd.ToInventoryType, cmd.AssetId, cmd.TransactionId)
			_ = mb.Put(compartment7.EnvCommandTopic, compartment8.AcceptCommandProvider(cmd.WorldId, cmd.AccountId, cmd.TransactionId, cmd.Slot, cmd.TemplateId, cmd.ReferenceId, cmd.ReferenceType))

			info = TransferInfo{
				Step:              nextStep,
				WorldId:           cmd.WorldId,
				CharacterId:       cmd.CharacterId,
				AccountId:         cmd.AccountId,
				AssetId:           cmd.ReferenceId,
				ToCompartmentId:   cmd.ToCompartmentId,
				ToCompartmentType: cmd.ToCompartmentType,
				ToInventoryType:   cmd.ToInventoryType,
			}
		}

		// Store transaction and transfer info in cache
		GetTransactionCache().Store(cmd.TransactionId, info)
		return nil
	}
}

// ProcessAndEmit handles the transfer command and emits messages
func (p *ProcessorImpl) ProcessAndEmit(cmd compartment.TransferCommand) error {
	return message.Emit(p.producer)(func(mb *message.Buffer) error {
		return p.Process(mb)(cmd)
	})
}

// createReleaseStep creates a step function for releasing an asset
func (p *ProcessorImpl) createReleaseStep(cmd compartment.TransferCommand) TransactionStep {
	return func(mb *message.Buffer) error {
		if cmd.FromInventoryType == compartment.InventoryTypeCharacter {
			_ = mb.Put(compartment2.EnvCommandTopic, compartment3.ReleaseCommandProvider(cmd.CharacterId, cmd.FromCompartmentType, cmd.TransactionId, cmd.ReferenceId))
		} else if cmd.FromInventoryType == compartment.InventoryTypeCashShop {
			_ = mb.Put(compartment4.EnvCommandTopic, compartment5.ReleaseCommandProvider(cmd.AccountId, cmd.FromCompartmentId, cmd.FromCompartmentType, cmd.TransactionId, cmd.ReferenceId))
		} else if cmd.FromInventoryType == compartment.InventoryTypeStorage {
			_ = mb.Put(compartment7.EnvCommandTopic, compartment8.ReleaseCommandProvider(cmd.WorldId, cmd.AccountId, cmd.TransactionId, cmd.AssetId))
		}
		return nil
	}
}

// HandleAccepted handles the accepted status event
func (p *ProcessorImpl) HandleAccepted(mb *message.Buffer) func(transactionId uuid.UUID) error {
	return func(transactionId uuid.UUID) error {
		p.l.Debugf("Target compartment accepted transfer. Removing from original inventory. TransferId: [%s]", transactionId)

		// Get transfer info from cache
		info, exists := GetTransactionCache().Get(transactionId)

		if !exists {
			p.l.Warn("No transfer info found for transaction")
			return nil
		}

		// Execute next step
		err := info.Step(mb)
		if err != nil {
			p.l.WithError(err).Error("Failed to execute next step")
			return err
		}

		// Note: We no longer delete the transaction from the cache here
		// so that HandleReleased can access the transfer info

		return nil
	}
}

// HandleAcceptedAndEmit handles the accepted status event and emits messages
func (p *ProcessorImpl) HandleAcceptedAndEmit(transactionId uuid.UUID) error {
	return message.Emit(p.producer)(func(mb *message.Buffer) error {
		return p.HandleAccepted(mb)(transactionId)
	})
}

// HandleReleased handles the released status event
func (p *ProcessorImpl) HandleReleased(mb *message.Buffer) func(transactionId uuid.UUID) error {
	return func(transactionId uuid.UUID) error {
		p.l.Debugf("Asset released from original inventory. Transfer completed. TransferId: [%s]", transactionId)

		// Get transfer info from cache
		info, exists := GetTransactionCache().Get(transactionId)

		if !exists {
			p.l.Warn("No transfer info found for transaction")
			return nil
		}

		// Emit completed status event
		_ = mb.Put(compartment.EnvEventTopicStatus, compartment6.CompletedStatusEventProvider(
			info.CharacterId,
			transactionId,
			info.AccountId,
			info.AssetId,
			info.ToCompartmentId,
			info.ToCompartmentType,
			info.ToInventoryType,
		))

		// Remove transaction from cache
		GetTransactionCache().Delete(transactionId)

		return nil
	}
}

// HandleReleasedAndEmit handles the released status event and emits messages
func (p *ProcessorImpl) HandleReleasedAndEmit(transactionId uuid.UUID) error {
	return message.Emit(p.producer)(func(mb *message.Buffer) error {
		return p.HandleReleased(mb)(transactionId)
	})
}

// HandleError handles the error status event
func (p *ProcessorImpl) HandleError(mb *message.Buffer) func(transactionId uuid.UUID) error {
	return func(transactionId uuid.UUID) error {
		p.l.Debugf("Transfer failed. TransferId: [%s]", transactionId)

		// Get transfer info from cache
		_, exists := GetTransactionCache().Get(transactionId)

		// If no transfer info exists, just return
		if !exists {
			p.l.Warn("No transfer info found for transaction")
			return nil
		}

		// Remove transaction from cache
		GetTransactionCache().Delete(transactionId)

		// TODO: issue saga failed event
		return nil
	}
}

// HandleErrorAndEmit handles the error status event and emits messages
func (p *ProcessorImpl) HandleErrorAndEmit(transactionId uuid.UUID) error {
	return message.Emit(p.producer)(func(mb *message.Buffer) error {
		return p.HandleError(mb)(transactionId)
	})
}
