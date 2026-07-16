package asset

import (
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetAssetById(assetId uint32) (Model, error)
	GetAssetsByStorageId(storageId uuid.UUID) ([]Model, error)
	GetOrCreateStorageId(worldId world.Id, accountId uint32) (uuid.UUID, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// WithTransaction returns a clone of the processor bound to the transaction handle.
func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) *ProcessorImpl {
	return &ProcessorImpl{l: p.l, ctx: p.ctx, db: tx}
}

func (p *ProcessorImpl) GetAssetById(assetId uint32) (Model, error) {
	return GetById(p.db.WithContext(p.ctx))(assetId)
}

func (p *ProcessorImpl) GetAssetsByStorageId(storageId uuid.UUID) ([]Model, error) {
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

func (p *ProcessorImpl) GetOrCreateStorageId(worldId world.Id, accountId uint32) (uuid.UUID, error) {
	t := tenant.MustFromContext(p.ctx)

	var id uuid.UUID
	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		var storageEntity StorageEntity
		err := tx.Where("world_id = ? AND account_id = ?", byte(worldId), accountId).
			First(&storageEntity).Error
		if err == nil {
			id = storageEntity.Id
			return nil
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
			if createErr := tx.Create(&storageEntity).Error; createErr != nil {
				return createErr
			}
			id = storageEntity.Id
			return nil
		}
		return err
	})
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
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
