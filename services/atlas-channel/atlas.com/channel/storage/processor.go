package storage

import (
	"atlas-channel/asset"
	"atlas-channel/kafka/message/storage"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DefaultStorageCapacity is the default number of slots for new storage
const DefaultStorageCapacity byte = 4

type Processor interface {
	GetStorageData(accountId uint32, worldId byte) (StorageData, error)
	GetProjectionData(characterId uint32) (ProjectionData, error)
	Arrange(worldId byte, accountId uint32) error
	DepositMesos(worldId byte, accountId uint32, mesos uint32) error
	WithdrawMesos(worldId byte, accountId uint32, mesos uint32) error
	CloseStorage(characterId uint32) error
}

// ProjectionData holds projection data retrieved from storage service
type ProjectionData struct {
	CharacterId  uint32
	AccountId    uint32
	WorldId      byte
	Capacity     byte
	Mesos        uint32
	NpcId        uint32
	Compartments map[string][]asset.Model[any]
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
	// Fetch storage with assets included
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

	// Transform REST models to asset.Model (assets are now included in storage response)
	assets := make([]asset.Model[any], 0, len(storageModel.Assets))
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
	compartments := make(map[string][]asset.Model[any])
	for name, restAssets := range parsedCompartments {
		assets := make([]asset.Model[any], 0, len(restAssets))
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
func (p ProjectionData) GetAllAssetsFromProjection() []asset.Model[any] {
	var result []asset.Model[any]
	// Add assets from each compartment - they may overlap initially until filtered
	// Use equip compartment as primary since it starts with all assets
	if assets, ok := p.Compartments["equip"]; ok {
		result = append(result, assets...)
	}
	return result
}

// transformAsset converts an AssetRestModel to asset.Model
func transformAsset(a AssetRestModel) asset.Model[any] {
	refType := asset.ReferenceType(a.ReferenceType)
	invType := inventoryTypeFromTemplateId(a.TemplateId)

	// Build reference data from the ReferenceData field
	refData := buildReferenceDataFromRestModel(refType, a.ReferenceData)

	// Use ETC as fallback for unknown reference types
	if refType != asset.ReferenceTypeEquipable &&
		refType != asset.ReferenceTypeConsumable &&
		refType != asset.ReferenceTypeSetup &&
		refType != asset.ReferenceTypeEtc &&
		refType != asset.ReferenceTypeCash &&
		refType != asset.ReferenceTypePet {
		refType = asset.ReferenceTypeEtc
	}

	return asset.NewBuilder[any](a.Id, uuid.Nil, a.TemplateId, a.ReferenceId, refType).
		SetInventoryType(invType).
		SetSlot(a.Slot).
		SetExpiration(a.Expiration).
		SetReferenceData(refData).
		MustBuild()
}

// inventoryTypeFromTemplateId determines the inventory type from a template ID
func inventoryTypeFromTemplateId(templateId uint32) asset.InventoryType {
	category := templateId / 1000000
	switch category {
	case 1:
		return asset.InventoryTypeEquip
	case 2:
		return asset.InventoryTypeUse
	case 3:
		return asset.InventoryTypeSetup
	case 4:
		return asset.InventoryTypeEtc
	case 5:
		return asset.InventoryTypeCash
	default:
		return asset.InventoryTypeEtc
	}
}

// buildReferenceDataFromRestModel creates the appropriate reference data from REST model reference data
func buildReferenceDataFromRestModel(refType asset.ReferenceType, restData interface{}) any {
	if restData == nil {
		return nil
	}

	switch refType {
	case asset.ReferenceTypeEquipable:
		if ed, ok := restData.(EquipableRestData); ok {
			return asset.NewEquipableReferenceDataBuilder().
				SetOwnerId(ed.OwnerId).
				SetStrength(ed.Strength).
				SetDexterity(ed.Dexterity).
				SetIntelligence(ed.Intelligence).
				SetLuck(ed.Luck).
				SetHp(ed.Hp).
				SetMp(ed.Mp).
				SetWeaponAttack(ed.WeaponAttack).
				SetMagicAttack(ed.MagicAttack).
				SetWeaponDefense(ed.WeaponDefense).
				SetMagicDefense(ed.MagicDefense).
				SetAccuracy(ed.Accuracy).
				SetAvoidability(ed.Avoidability).
				SetHands(ed.Hands).
				SetSpeed(ed.Speed).
				SetJump(ed.Jump).
				SetSlots(ed.Slots).
				SetLocked(ed.Locked).
				SetSpikes(ed.Spikes).
				SetKarmaUsed(ed.KarmaUsed).
				SetCold(ed.Cold).
				SetCanBeTraded(ed.CanBeTraded).
				SetLevelType(ed.LevelType).
				SetLevel(ed.Level).
				SetExperience(ed.Experience).
				SetHammersApplied(ed.HammersApplied).
				Build()
		}
		return nil
	case asset.ReferenceTypeConsumable:
		if cd, ok := restData.(ConsumableRestData); ok {
			return asset.NewConsumableReferenceDataBuilder().
				SetQuantity(cd.Quantity).
				SetOwnerId(cd.OwnerId).
				SetFlag(cd.Flag).
				SetRechargeable(cd.Rechargeable).
				Build()
		}
		return nil
	case asset.ReferenceTypeSetup:
		if sd, ok := restData.(SetupRestData); ok {
			return asset.NewSetupReferenceDataBuilder().
				SetQuantity(sd.Quantity).
				SetOwnerId(sd.OwnerId).
				SetFlag(sd.Flag).
				Build()
		}
		return nil
	case asset.ReferenceTypeEtc:
		if ed, ok := restData.(EtcRestData); ok {
			return asset.NewEtcReferenceDataBuilder().
				SetQuantity(ed.Quantity).
				SetOwnerId(ed.OwnerId).
				SetFlag(ed.Flag).
				Build()
		}
		return nil
	case asset.ReferenceTypeCash:
		if cd, ok := restData.(CashRestData); ok {
			return asset.NewCashReferenceDataBuilder().
				SetQuantity(cd.Quantity).
				SetOwnerId(cd.OwnerId).
				SetFlag(cd.Flag).
				SetCashId(cd.CashId).
				Build()
		}
		return nil
	case asset.ReferenceTypePet:
		if pd, ok := restData.(PetRestData); ok {
			return asset.NewPetReferenceDataBuilder().
				SetOwnerId(pd.OwnerId).
				SetCashId(pd.CashId).
				SetFlag(pd.Flag).
				SetName(pd.Name).
				SetLevel(pd.Level).
				SetCloseness(pd.Closeness).
				SetFullness(pd.Fullness).
				SetSlot(pd.Slot).
				Build()
		}
		return nil
	default:
		// Default to ETC reference data
		if ed, ok := restData.(EtcRestData); ok {
			return asset.NewEtcReferenceDataBuilder().
				SetQuantity(ed.Quantity).
				SetOwnerId(ed.OwnerId).
				SetFlag(ed.Flag).
				Build()
		}
		return nil
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

// CloseStorage sends a CLOSE_STORAGE command to clear NPC context for a character
func (p *ProcessorImpl) CloseStorage(characterId uint32) error {
	p.l.Debugf("Sending CLOSE_STORAGE command for character [%d].", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(storage.EnvShowStorageCommandTopic)(CloseStorageCommandProvider(characterId))
}
