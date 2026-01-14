package asset

import (
	"atlas-storage/equipable"
	"atlas-storage/pet"
	"atlas-storage/stackable"
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor struct {
	l                  logrus.FieldLogger
	ctx                context.Context
	db                 *gorm.DB
	equipableProcessor equipable.Processor
	petProcessor       pet.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	return &Processor{
		l:                  l,
		ctx:                ctx,
		db:                 db,
		equipableProcessor: equipable.NewProcessor(l, ctx),
		petProcessor:       pet.NewProcessor(l, ctx),
	}
}

// GetAssetById retrieves an asset by ID
func (p *Processor) GetAssetById(assetId uint32) (Model[any], error) {
	t := tenant.MustFromContext(p.ctx)
	return GetById(p.l, p.db, t.Id())(assetId)
}

// GetAssetsByStorageId retrieves all assets for a storage
func (p *Processor) GetAssetsByStorageId(storageId uuid.UUID) ([]Model[any], error) {
	t := tenant.MustFromContext(p.ctx)
	return GetByStorageId(p.l, p.db, t.Id())(storageId)
}

// StorageEntity is a minimal storage entity for cross-package queries
type StorageEntity struct {
	TenantId  uuid.UUID `gorm:"not null;uniqueIndex:idx_tenant_world_account"`
	Id        uuid.UUID `gorm:"primaryKey;type:uuid"`
	WorldId   byte      `gorm:"not null;uniqueIndex:idx_tenant_world_account"`
	AccountId uint32    `gorm:"not null;uniqueIndex:idx_tenant_world_account"`
	Capacity  uint32    `gorm:"not null;default:4"`
	Mesos     uint32    `gorm:"not null;default:0"`
}

func (StorageEntity) TableName() string {
	return "storages"
}

// GetOrCreateStorageId retrieves or creates a storage and returns its ID
func (p *Processor) GetOrCreateStorageId(worldId byte, accountId uint32) (uuid.UUID, error) {
	t := tenant.MustFromContext(p.ctx)

	// Try to get existing storage
	var storage StorageEntity
	err := p.db.Where("tenant_id = ? AND world_id = ? AND account_id = ?", t.Id(), worldId, accountId).
		First(&storage).Error

	if err == nil {
		return storage.Id, nil
	}

	// Storage not found, create it
	if err == gorm.ErrRecordNotFound {
		storage = StorageEntity{
			TenantId:  t.Id(),
			Id:        uuid.New(),
			WorldId:   worldId,
			AccountId: accountId,
			Capacity:  4,
			Mesos:     0,
		}
		createErr := p.db.Create(&storage).Error
		if createErr != nil {
			return uuid.Nil, createErr
		}
		return storage.Id, nil
	}

	return uuid.Nil, err
}

// DecorateAsset adds reference data to an asset based on its type
func (p *Processor) DecorateAsset(m Model[any]) (Model[any], error) {
	switch m.ReferenceType() {
	case ReferenceTypeEquipable:
		return p.DecorateEquipable(m)
	case ReferenceTypeCashEquipable:
		return p.DecorateCashEquipable(m)
	case ReferenceTypeConsumable, ReferenceTypeSetup, ReferenceTypeEtc:
		return p.DecorateStackable(m)
	case ReferenceTypePet:
		return p.DecoratePet(m)
	default:
		return m, nil
	}
}

// DecorateEquipable loads equipable data from atlas-equipables service
func (p *Processor) DecorateEquipable(m Model[any]) (Model[any], error) {
	e, err := p.equipableProcessor.ByIdProvider(m.ReferenceId())()
	if err != nil {
		p.l.WithError(err).Warnf("Failed to load equipable data for reference id %d", m.ReferenceId())
		return m, err
	}
	return Clone(m).SetReferenceData(e).MustBuild(), nil
}

// DecorateCashEquipable loads cash equipable data
func (p *Processor) DecorateCashEquipable(m Model[any]) (Model[any], error) {
	// Cash equipables don't have additional reference data in storage service
	// They would be stored elsewhere or have minimal data
	return m, nil
}

// DecorateStackable loads stackable data from local stackable table
func (p *Processor) DecorateStackable(m Model[any]) (Model[any], error) {
	s, err := stackable.GetByAssetId(p.l, p.db)(m.Id())
	if err != nil {
		p.l.WithError(err).Warnf("Failed to load stackable data for asset id %d", m.Id())
		return m, err
	}
	return Clone(m).SetReferenceData(s).MustBuild(), nil
}

// DecoratePet loads pet data from atlas-pets service
func (p *Processor) DecoratePet(m Model[any]) (Model[any], error) {
	pe, err := p.petProcessor.ByIdProvider(m.ReferenceId())()
	if err != nil {
		p.l.WithError(err).Warnf("Failed to load pet data for reference id %d", m.ReferenceId())
		return m, err
	}
	return Clone(m).SetReferenceData(pe).MustBuild(), nil
}

// TransformToBaseRestModel converts a decorated Model to BaseRestModel with full reference data
func TransformToBaseRestModel(m Model[any]) (BaseRestModel, error) {
	brm := BaseRestModel{
		Id:            m.Id(),
		Slot:          m.Slot(),
		TemplateId:    m.TemplateId(),
		Expiration:    m.Expiration(),
		ReferenceId:   m.ReferenceId(),
		ReferenceType: string(m.ReferenceType()),
	}

	switch m.ReferenceType() {
	case ReferenceTypeEquipable:
		if em, ok := m.ReferenceData().(equipable.Model); ok {
			brm.ReferenceData = EquipableRestData{
				BaseData: BaseData{
					OwnerId: em.OwnerId(),
				},
				StatisticRestData: StatisticRestData{
					Strength:      em.Strength(),
					Dexterity:     em.Dexterity(),
					Intelligence:  em.Intelligence(),
					Luck:          em.Luck(),
					Hp:            em.Hp(),
					Mp:            em.Mp(),
					WeaponAttack:  em.WeaponAttack(),
					MagicAttack:   em.MagicAttack(),
					WeaponDefense: em.WeaponDefense(),
					MagicDefense:  em.MagicDefense(),
					Accuracy:      em.Accuracy(),
					Avoidability:  em.Avoidability(),
					Hands:         em.Hands(),
					Speed:         em.Speed(),
					Jump:          em.Jump(),
				},
				Slots:          em.Slots(),
				Locked:         em.Locked(),
				Spikes:         em.Spikes(),
				KarmaUsed:      em.KarmaUsed(),
				Cold:           em.Cold(),
				CanBeTraded:    em.CanBeTraded(),
				LevelType:      em.LevelType(),
				Level:          em.Level(),
				Experience:     em.Experience(),
				HammersApplied: em.HammersApplied(),
			}
		}
	case ReferenceTypeConsumable:
		if sm, ok := m.ReferenceData().(stackable.Model); ok {
			brm.ReferenceData = ConsumableRestData{
				BaseData: BaseData{
					OwnerId: sm.OwnerId(),
				},
				StackableRestData: StackableRestData{
					Quantity: sm.Quantity(),
				},
				Flag:         sm.Flag(),
				Rechargeable: 0, // Storage doesn't track rechargeable
			}
		}
	case ReferenceTypeSetup:
		if sm, ok := m.ReferenceData().(stackable.Model); ok {
			brm.ReferenceData = SetupRestData{
				BaseData: BaseData{
					OwnerId: sm.OwnerId(),
				},
				StackableRestData: StackableRestData{
					Quantity: sm.Quantity(),
				},
				Flag: sm.Flag(),
			}
		}
	case ReferenceTypeEtc:
		if sm, ok := m.ReferenceData().(stackable.Model); ok {
			brm.ReferenceData = EtcRestData{
				BaseData: BaseData{
					OwnerId: sm.OwnerId(),
				},
				StackableRestData: StackableRestData{
					Quantity: sm.Quantity(),
				},
				Flag: sm.Flag(),
			}
		}
	case ReferenceTypePet:
		if pm, ok := m.ReferenceData().(pet.Model); ok {
			brm.ReferenceData = PetRestData{
				BaseData: BaseData{
					OwnerId: pm.OwnerId(),
				},
				CashBaseRestData: CashBaseRestData{
					CashId: pm.CashId(),
				},
				Flag:        pm.Flag(),
				PurchasedBy: pm.PurchasedBy(),
				Name:        pm.Name(),
				Level:       pm.Level(),
				Closeness:   pm.Closeness(),
				Fullness:    pm.Fullness(),
				Slot:        pm.Slot(),
			}
		}
	}

	return brm, nil
}

// TransformAllToBaseRestModel converts multiple decorated Models to BaseRestModels
func TransformAllToBaseRestModel(models []Model[any]) ([]BaseRestModel, error) {
	result := make([]BaseRestModel, 0, len(models))
	for _, m := range models {
		brm, err := TransformToBaseRestModel(m)
		if err != nil {
			return nil, err
		}
		result = append(result, brm)
	}
	return result, nil
}

// DecorateAll decorates multiple assets with reference data
func (p *Processor) DecorateAll(assets []Model[any]) ([]Model[any], error) {
	result := make([]Model[any], 0, len(assets))
	for _, a := range assets {
		decorated, err := p.DecorateAsset(a)
		if err != nil {
			p.l.WithError(err).Warnf("Failed to decorate asset %d, using undecorated version", a.Id())
			result = append(result, a)
			continue
		}
		result = append(result, decorated)
	}
	return result, nil
}

// GetByStorageIdDecorated retrieves and decorates all assets for a storage
func (p *Processor) GetByStorageIdDecorated(tenantId uuid.UUID, storageId uuid.UUID) ([]Model[any], error) {
	assets, err := GetByStorageId(p.l, p.db, tenantId)(storageId)
	if err != nil {
		return nil, err
	}
	return p.DecorateAll(assets)
}
