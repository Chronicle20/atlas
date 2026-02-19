package asset

import (
	"context"
	"errors"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	return &Processor{
		l:   l,
		ctx: ctx,
		db:  db,
	}
}

func (p *Processor) GetAssetById(assetId uint32) (Model, error) {
	return GetById(p.db.WithContext(p.ctx))(assetId)
}

func (p *Processor) GetAssetsByStorageId(storageId uuid.UUID) ([]Model, error) {
	return GetByStorageId(p.db.WithContext(p.ctx))(storageId)
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

func (p *Processor) GetOrCreateStorageId(worldId world.Id, accountId uint32) (uuid.UUID, error) {
	t := tenant.MustFromContext(p.ctx)

	var storageEntity StorageEntity
	err := p.db.WithContext(p.ctx).Where("world_id = ? AND account_id = ?", byte(worldId), accountId).
		First(&storageEntity).Error

	if err == nil {
		return storageEntity.Id, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		storageEntity = StorageEntity{
			TenantId:  t.Id(),
			Id:        uuid.New(),
			WorldId:   byte(worldId),
			AccountId: accountId,
			Capacity:  4,
			Mesos:     0,
		}
		createErr := p.db.WithContext(p.ctx).Create(&storageEntity).Error
		if createErr != nil {
			return uuid.Nil, createErr
		}
		return storageEntity.Id, nil
	}

	return uuid.Nil, err
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:             m.Id(),
		Slot:           m.Slot(),
		TemplateId:     m.TemplateId(),
		Expiration:     m.Expiration(),
		Quantity:       m.quantity,
		OwnerId:        m.OwnerId(),
		Flag:           m.Flag(),
		Rechargeable:   m.Rechargeable(),
		Strength:       m.Strength(),
		Dexterity:      m.Dexterity(),
		Intelligence:   m.Intelligence(),
		Luck:           m.Luck(),
		Hp:             m.Hp(),
		Mp:             m.Mp(),
		WeaponAttack:   m.WeaponAttack(),
		MagicAttack:    m.MagicAttack(),
		WeaponDefense:  m.WeaponDefense(),
		MagicDefense:   m.MagicDefense(),
		Accuracy:       m.Accuracy(),
		Avoidability:   m.Avoidability(),
		Hands:          m.Hands(),
		Speed:          m.Speed(),
		Jump:           m.Jump(),
		Slots:          m.Slots(),
		LevelType:      m.LevelType(),
		Level:          m.Level(),
		Experience:     m.Experience(),
		HammersApplied: m.HammersApplied(),
		CashId:         m.CashId(),
		CommodityId:    m.CommodityId(),
		PurchaseBy:     m.PurchaseBy(),
		PetId:          m.PetId(),
	}, nil
}

func TransformAll(models []Model) ([]RestModel, error) {
	result := make([]RestModel, 0, len(models))
	for _, m := range models {
		rm, err := Transform(m)
		if err != nil {
			return nil, err
		}
		result = append(result, rm)
	}
	return result, nil
}
