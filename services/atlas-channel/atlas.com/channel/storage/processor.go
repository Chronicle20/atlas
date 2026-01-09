package storage

import (
	"atlas-channel/asset"
	"atlas-channel/kafka/message/storage"
	"atlas-channel/kafka/producer"
	"context"
	"strconv"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DefaultStorageCapacity is the default number of slots for new storage
const DefaultStorageCapacity byte = 4

type Processor interface {
	GetStorageData(accountId uint32, worldId byte) (StorageData, error)
	Arrange(worldId byte, accountId uint32) error
	DepositMesos(worldId byte, accountId uint32, mesos uint32) error
	WithdrawMesos(worldId byte, accountId uint32, mesos uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

// StorageData holds all the data needed to display storage UI
type StorageData struct {
	Capacity byte
	Mesos    uint32
	Assets   []asset.Model[any]
}

// GetStorageData fetches storage metadata and assets for an account
func (p *ProcessorImpl) GetStorageData(accountId uint32, worldId byte) (StorageData, error) {
	// Fetch storage metadata
	storageModel, err := requestStorageByAccountAndWorld(accountId, worldId)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Debugf("Unable to get storage for account %d world %d, returning empty storage.", accountId, worldId)
		// Storage might not exist yet - return empty storage
		return StorageData{
			Capacity: DefaultStorageCapacity,
			Mesos:    0,
			Assets:   []asset.Model[any]{},
		}, nil
	}

	// Fetch assets
	assetModels, err := requestAssetsByAccountAndWorld(accountId, worldId)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to get assets for storage account %d world %d.", accountId, worldId)
		assetModels = []AssetRestModel{}
	}

	// Transform REST models to asset.Model
	assets := make([]asset.Model[any], 0, len(assetModels))
	for _, a := range assetModels {
		assetId, _ := strconv.ParseUint(a.Id, 10, 32)
		assets = append(assets, transformAsset(a, uint32(assetId)))
	}

	return StorageData{
		Capacity: byte(storageModel.Capacity),
		Mesos:    storageModel.Mesos,
		Assets:   assets,
	}, nil
}

// transformAsset converts an AssetRestModel to asset.Model
func transformAsset(a AssetRestModel, assetId uint32) asset.Model[any] {
	refType := asset.ReferenceType(a.ReferenceType)
	invType := asset.InventoryType(a.InventoryType)

	// Get reference data based on type
	refData := buildReferenceData(refType, a.Quantity, a.OwnerId, a.Flag)

	// Use ETC as fallback for unknown reference types
	if refType != asset.ReferenceTypeEquipable &&
		refType != asset.ReferenceTypeConsumable &&
		refType != asset.ReferenceTypeSetup &&
		refType != asset.ReferenceTypeEtc &&
		refType != asset.ReferenceTypeCash {
		refType = asset.ReferenceTypeEtc
	}

	return asset.NewBuilder[any](assetId, uuid.Nil, a.TemplateId, a.ReferenceId, refType).
		SetInventoryType(invType).
		SetSlot(a.Slot).
		SetExpiration(a.Expiration).
		SetReferenceData(refData).
		Build()
}

// buildReferenceData creates the appropriate reference data based on type
func buildReferenceData(refType asset.ReferenceType, quantity uint32, ownerId uint32, flag uint16) any {
	switch refType {
	case asset.ReferenceTypeEquipable:
		// Equipables don't have stackable reference data
		return nil
	case asset.ReferenceTypeConsumable:
		return asset.NewConsumableReferenceDataBuilder().
			SetQuantity(quantity).
			SetOwnerId(ownerId).
			SetFlag(flag).
			Build()
	case asset.ReferenceTypeSetup:
		return asset.NewSetupReferenceDataBuilder().
			SetQuantity(quantity).
			SetOwnerId(ownerId).
			SetFlag(flag).
			Build()
	case asset.ReferenceTypeCash:
		return asset.NewCashReferenceDataBuilder().
			SetQuantity(quantity).
			SetOwnerId(ownerId).
			SetFlag(flag).
			Build()
	default:
		// Default to ETC reference data
		return asset.NewEtcReferenceDataBuilder().
			SetQuantity(quantity).
			SetOwnerId(ownerId).
			SetFlag(flag).
			Build()
	}
}

// Arrange sends an ARRANGE command to the storage service to merge and sort items
func (p *ProcessorImpl) Arrange(worldId byte, accountId uint32) error {
	p.l.Debugf("Sending ARRANGE command for storage account [%d] world [%d].", accountId, worldId)
	return producer.ProviderImpl(p.l)(p.ctx)(storage.EnvCommandTopic)(ArrangeCommandProvider(worldId, accountId, uuid.New()))
}

// DepositMesos sends an UPDATE_MESOS command to add mesos to storage
func (p *ProcessorImpl) DepositMesos(worldId byte, accountId uint32, mesos uint32) error {
	p.l.Debugf("Depositing [%d] mesos to storage account [%d] world [%d].", mesos, accountId, worldId)
	return producer.ProviderImpl(p.l)(p.ctx)(storage.EnvCommandTopic)(UpdateMesosCommandProvider(worldId, accountId, uuid.New(), mesos, storage.MesosOperationAdd))
}

// WithdrawMesos sends an UPDATE_MESOS command to withdraw mesos from storage
func (p *ProcessorImpl) WithdrawMesos(worldId byte, accountId uint32, mesos uint32) error {
	p.l.Debugf("Withdrawing [%d] mesos from storage account [%d] world [%d].", mesos, accountId, worldId)
	return producer.ProviderImpl(p.l)(p.ctx)(storage.EnvCommandTopic)(UpdateMesosCommandProvider(worldId, accountId, uuid.New(), mesos, storage.MesosOperationSubtract))
}
