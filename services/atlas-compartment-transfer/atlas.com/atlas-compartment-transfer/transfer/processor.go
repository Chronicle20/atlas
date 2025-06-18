package transfer

import (
	"atlas-compartment-transfer/kafka/message"
	compartment4 "atlas-compartment-transfer/kafka/message/cashshop/compartment"
	compartment2 "atlas-compartment-transfer/kafka/message/character/compartment"
	"atlas-compartment-transfer/kafka/message/compartment"
	"atlas-compartment-transfer/kafka/producer"
	compartment5 "atlas-compartment-transfer/kafka/producer/cashshop/compartment"
	compartment3 "atlas-compartment-transfer/kafka/producer/character/compartment"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"sync"
)

// TransactionStep represents the next step in the transfer saga
type TransactionStep func(mb *message.Buffer) error

// TransactionCache is a singleton that holds the transaction cache
type TransactionCache struct {
	txCache   map[uuid.UUID]TransactionStep
	cacheLock sync.RWMutex
}

// instance is the singleton instance of TransactionCache
var instance *TransactionCache
var once sync.Once

// GetTransactionCache returns the singleton instance of TransactionCache
func GetTransactionCache() *TransactionCache {
	once.Do(func() {
		instance = &TransactionCache{
			txCache:   make(map[uuid.UUID]TransactionStep),
			cacheLock: sync.RWMutex{},
		}
	})
	return instance
}

// Store stores a transaction step in the cache
func (tc *TransactionCache) Store(transactionId uuid.UUID, step TransactionStep) {
	tc.cacheLock.Lock()
	defer tc.cacheLock.Unlock()
	tc.txCache[transactionId] = step
}

// Get retrieves a transaction step from the cache
func (tc *TransactionCache) Get(transactionId uuid.UUID) (TransactionStep, bool) {
	tc.cacheLock.RLock()
	defer tc.cacheLock.RUnlock()
	step, exists := tc.txCache[transactionId]
	return step, exists
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
		if cmd.ToInventoryType == compartment.InventoryTypeCharacter {
			p.l.Debugf("Informing [%s] inventory to receive that [%d] via transfer [%s].", cmd.ToInventoryType, cmd.AssetId, cmd.TransactionId)
			_ = mb.Put(compartment2.EnvCommandTopic, compartment3.AcceptCommandProvider(cmd.CharacterId, cmd.ToCompartmentType, cmd.TransactionId, cmd.ReferenceId))
		} else if cmd.ToInventoryType == compartment.InventoryTypeCashShop {
			p.l.Debugf("Informing [%s] inventory to receive that [%d] via transfer [%s].", cmd.ToInventoryType, cmd.AssetId, cmd.TransactionId)
			_ = mb.Put(compartment4.EnvCommandTopic, compartment5.AcceptCommandProvider(cmd.AccountId, cmd.ToCompartmentId, cmd.ToCompartmentType, cmd.TransactionId, cmd.ReferenceId))
		}

		// Create next step function for release
		nextStep := p.createReleaseStep(cmd)

		// Store transaction and next step in cache
		GetTransactionCache().Store(cmd.TransactionId, nextStep)
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
		}
		return nil
	}
}

// HandleAccepted handles the accepted status event
func (p *ProcessorImpl) HandleAccepted(mb *message.Buffer) func(transactionId uuid.UUID) error {
	return func(transactionId uuid.UUID) error {
		p.l.Debugf("Target compartment accepted transfer. Removing from original inventory. TransferId: [%s]", transactionId)

		// Get next step from cache
		nextStep, exists := GetTransactionCache().Get(transactionId)

		if !exists {
			p.l.Warn("No next step found for transaction")
			return nil
		}

		// Execute next step
		err := nextStep(mb)
		if err != nil {
			p.l.WithError(err).Error("Failed to execute next step")
			return err
		}

		// Remove transaction from cache
		GetTransactionCache().Delete(transactionId)

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
		// TODO emit saga concluded event
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
