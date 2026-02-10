package storage

import (
	"atlas-channel/asset"
	"atlas-channel/kafka/message/storage"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DefaultStorageCapacity is the default number of slots for new storage
const DefaultStorageCapacity byte = 4

type Processor interface {
	GetStorageData(accountId uint32, worldId world.Id) (StorageData, error)
	GetProjectionData(characterId uint32) (ProjectionData, error)
	Arrange(worldId world.Id, accountId uint32) error
	DepositMesos(worldId world.Id, accountId uint32, mesos uint32) error
	WithdrawMesos(worldId world.Id, accountId uint32, mesos uint32) error
	CloseStorage(characterId uint32) error
}

// ProjectionData holds projection data retrieved from storage service
type ProjectionData struct {
	CharacterId  uint32
	AccountId    uint32
	WorldId      world.Id
	Capacity     byte
	Mesos        uint32
	NpcId        uint32
	Compartments map[string][]asset.Model
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
	Assets   []asset.Model
}

// GetStorageData fetches storage metadata and assets for an account
func (p *ProcessorImpl) GetStorageData(accountId uint32, worldId world.Id) (StorageData, error) {
	// Fetch storage with assets included
	storageModel, err := requestStorageByAccountAndWorld(accountId, worldId)(p.l, p.ctx)
	if err != nil {
		p.l.WithError(err).Debugf("Unable to get storage for account %d world %d, returning empty storage.", accountId, worldId)
		// Storage might not exist yet - return empty storage
		return StorageData{
			Capacity: DefaultStorageCapacity,
			Mesos:    0,
			Assets:   []asset.Model{},
		}, nil
	}

	// Transform REST models to asset.Model (assets are now included in storage response)
	assets := make([]asset.Model, 0, len(storageModel.Assets))
	for _, a := range storageModel.Assets {
		assets = append(assets, transformAsset(a))
	}

	return StorageData{
		Capacity: byte(storageModel.Capacity),
		Mesos:    storageModel.Mesos,
		Assets:   assets,
	}, nil
}

// GetProjectionData fetches projection data for a character from storage service
func (p *ProcessorImpl) GetProjectionData(characterId uint32) (ProjectionData, error) {
	// Fetch projection from storage service
	projModel, err := requestProjectionByCharacterId(characterId)(p.l, p.ctx)
	if err != nil {
		return ProjectionData{}, err
	}

	// Parse compartment assets from raw JSON
	parsedCompartments, err := projModel.ParseCompartmentAssets()
	if err != nil {
		return ProjectionData{}, err
	}

	// Transform compartments
	compartments := make(map[string][]asset.Model)
	for name, restAssets := range parsedCompartments {
		assets := make([]asset.Model, 0, len(restAssets))
		for _, a := range restAssets {
			assets = append(assets, transformAsset(a))
		}
		compartments[name] = assets
	}

	return ProjectionData{
		CharacterId:  projModel.CharacterId,
		AccountId:    projModel.AccountId,
		WorldId:      projModel.WorldId,
		Capacity:     byte(projModel.Capacity),
		Mesos:        projModel.Mesos,
		NpcId:        projModel.NpcId,
		Compartments: compartments,
	}, nil
}

// GetAllAssetsFromProjection returns all assets from a projection, sorted by inventory type
func (p ProjectionData) GetAllAssetsFromProjection() []asset.Model {
	var result []asset.Model
	// Add assets from each compartment - they may overlap initially until filtered
	// Use equip compartment as primary since it starts with all assets
	if assets, ok := p.Compartments["equip"]; ok {
		result = append(result, assets...)
	}
	return result
}

// transformAsset converts a flat AssetRestModel to asset.Model
func transformAsset(a AssetRestModel) asset.Model {
	return asset.NewModelBuilder(a.Id, uuid.Nil, a.TemplateId).
		SetSlot(a.Slot).
		SetExpiration(a.Expiration).
		SetQuantity(a.Quantity).
		SetOwnerId(a.OwnerId).
		SetFlag(a.Flag).
		SetRechargeable(a.Rechargeable).
		SetStrength(a.Strength).
		SetDexterity(a.Dexterity).
		SetIntelligence(a.Intelligence).
		SetLuck(a.Luck).
		SetHp(a.Hp).
		SetMp(a.Mp).
		SetWeaponAttack(a.WeaponAttack).
		SetMagicAttack(a.MagicAttack).
		SetWeaponDefense(a.WeaponDefense).
		SetMagicDefense(a.MagicDefense).
		SetAccuracy(a.Accuracy).
		SetAvoidability(a.Avoidability).
		SetHands(a.Hands).
		SetSpeed(a.Speed).
		SetJump(a.Jump).
		SetSlots(a.Slots).
		SetFlag(a.Flag).
		SetLevelType(a.LevelType).
		SetLevel(a.Level).
		SetExperience(a.Experience).
		SetHammersApplied(a.HammersApplied).
		SetCashId(a.CashId).
		SetCommodityId(a.CommodityId).
		SetPurchaseBy(a.PurchaseBy).
		SetPetId(a.PetId).
		MustBuild()
}

// Arrange sends an ARRANGE command to the storage service to merge and sort items
func (p *ProcessorImpl) Arrange(worldId world.Id, accountId uint32) error {
	p.l.Debugf("Sending ARRANGE command for storage account [%d] world [%d].", accountId, worldId)
	return producer.ProviderImpl(p.l)(p.ctx)(storage.EnvCommandTopic)(ArrangeCommandProvider(worldId, accountId, uuid.New()))
}

// DepositMesos sends an UPDATE_MESOS command to add mesos to storage
func (p *ProcessorImpl) DepositMesos(worldId world.Id, accountId uint32, mesos uint32) error {
	p.l.Debugf("Depositing [%d] mesos to storage account [%d] world [%d].", mesos, accountId, worldId)
	return producer.ProviderImpl(p.l)(p.ctx)(storage.EnvCommandTopic)(UpdateMesosCommandProvider(worldId, accountId, uuid.New(), mesos, storage.MesosOperationAdd))
}

// WithdrawMesos sends an UPDATE_MESOS command to withdraw mesos from storage
func (p *ProcessorImpl) WithdrawMesos(worldId world.Id, accountId uint32, mesos uint32) error {
	p.l.Debugf("Withdrawing [%d] mesos from storage account [%d] world [%d].", mesos, accountId, worldId)
	return producer.ProviderImpl(p.l)(p.ctx)(storage.EnvCommandTopic)(UpdateMesosCommandProvider(worldId, accountId, uuid.New(), mesos, storage.MesosOperationSubtract))
}

// CloseStorage sends a CLOSE_STORAGE command to clear NPC context for a character
func (p *ProcessorImpl) CloseStorage(characterId uint32) error {
	p.l.Debugf("Sending CLOSE_STORAGE command for character [%d].", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(storage.EnvCommandTopic)(CloseStorageCommandProvider(characterId))
}
