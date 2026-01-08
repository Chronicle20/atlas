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

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

// StorageData holds all the data needed to display storage UI
type StorageData struct {
	Capacity byte
	Mesos    uint32
	Assets   []asset.Model[any]
}

// GetStorageData fetches storage metadata and assets for an account
func (p *Processor) GetStorageData(accountId uint32, worldId byte) (StorageData, error) {
	// Fetch storage metadata
	storageModel, err := requestStorageByAccountAndWorld(accountId, worldId)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Debugf("Unable to get storage for account %d world %d, returning empty storage.", accountId, worldId)
		// Storage might not exist yet - return empty storage
		return StorageData{
			Capacity: 4, // Default capacity
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

	// Build the asset model with appropriate reference data
	switch refType {
	case asset.ReferenceTypeEquipable:
		// For equipables, we would need to fetch full equipment data
		// For now, create a minimal model with nil reference data
		return asset.NewBuilder[any](assetId, uuid.Nil, a.TemplateId, a.ReferenceId, refType).
			SetInventoryType(invType).
			SetSlot(a.Slot).
			SetExpiration(a.Expiration).
			SetReferenceData(nil).
			Build()

	case asset.ReferenceTypeConsumable:
		refData := asset.NewConsumableReferenceDataBuilder().
			SetQuantity(a.Quantity).
			SetOwnerId(a.OwnerId).
			SetFlag(a.Flag).
			Build()
		return asset.NewBuilder[any](assetId, uuid.Nil, a.TemplateId, a.ReferenceId, refType).
			SetInventoryType(invType).
			SetSlot(a.Slot).
			SetExpiration(a.Expiration).
			SetReferenceData(refData).
			Build()

	case asset.ReferenceTypeSetup:
		refData := asset.NewSetupReferenceDataBuilder().
			SetQuantity(a.Quantity).
			SetOwnerId(a.OwnerId).
			SetFlag(a.Flag).
			Build()
		return asset.NewBuilder[any](assetId, uuid.Nil, a.TemplateId, a.ReferenceId, refType).
			SetInventoryType(invType).
			SetSlot(a.Slot).
			SetExpiration(a.Expiration).
			SetReferenceData(refData).
			Build()

	case asset.ReferenceTypeEtc:
		refData := asset.NewEtcReferenceDataBuilder().
			SetQuantity(a.Quantity).
			SetOwnerId(a.OwnerId).
			SetFlag(a.Flag).
			Build()
		return asset.NewBuilder[any](assetId, uuid.Nil, a.TemplateId, a.ReferenceId, refType).
			SetInventoryType(invType).
			SetSlot(a.Slot).
			SetExpiration(a.Expiration).
			SetReferenceData(refData).
			Build()

	case asset.ReferenceTypeCash:
		refData := asset.NewCashReferenceDataBuilder().
			SetQuantity(a.Quantity).
			SetOwnerId(a.OwnerId).
			SetFlag(a.Flag).
			Build()
		return asset.NewBuilder[any](assetId, uuid.Nil, a.TemplateId, a.ReferenceId, refType).
			SetInventoryType(invType).
			SetSlot(a.Slot).
			SetExpiration(a.Expiration).
			SetReferenceData(refData).
			Build()

	default:
		// Default to ETC type
		refData := asset.NewEtcReferenceDataBuilder().
			SetQuantity(a.Quantity).
			SetOwnerId(a.OwnerId).
			SetFlag(a.Flag).
			Build()
		return asset.NewBuilder[any](assetId, uuid.Nil, a.TemplateId, a.ReferenceId, asset.ReferenceTypeEtc).
			SetInventoryType(invType).
			SetSlot(a.Slot).
			SetExpiration(a.Expiration).
			SetReferenceData(refData).
			Build()
	}
}

// Arrange sends an ARRANGE command to the storage service to merge and sort items
func (p *Processor) Arrange(worldId byte, accountId uint32) error {
	p.l.Debugf("Sending ARRANGE command for storage account [%d] world [%d].", accountId, worldId)
	return producer.ProviderImpl(p.l)(p.ctx)(storage.EnvCommandTopic)(ArrangeCommandProvider(worldId, accountId, uuid.New()))
}

// DepositMesos sends an UPDATE_MESOS command to add mesos to storage
func (p *Processor) DepositMesos(worldId byte, accountId uint32, mesos uint32) error {
	p.l.Debugf("Depositing [%d] mesos to storage account [%d] world [%d].", mesos, accountId, worldId)
	return producer.ProviderImpl(p.l)(p.ctx)(storage.EnvCommandTopic)(UpdateMesosCommandProvider(worldId, accountId, uuid.New(), mesos, storage.MesosOperationAdd))
}

// WithdrawMesos sends an UPDATE_MESOS command to withdraw mesos from storage
func (p *Processor) WithdrawMesos(worldId byte, accountId uint32, mesos uint32) error {
	p.l.Debugf("Withdrawing [%d] mesos from storage account [%d] world [%d].", mesos, accountId, worldId)
	return producer.ProviderImpl(p.l)(p.ctx)(storage.EnvCommandTopic)(UpdateMesosCommandProvider(worldId, accountId, uuid.New(), mesos, storage.MesosOperationSubtract))
}
