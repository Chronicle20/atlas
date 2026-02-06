package commodities

import (
	"atlas-npc/database"
	"context"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func createCommodity(ctx context.Context, db *gorm.DB) func(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenTemplateId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error) {
	return func(npcId uint32, templateId uint32, mesoPrice uint32, discountRate byte, tokenTemplateId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error) {
		t := tenant.MustFromContext(ctx)
		id := uuid.New()
		entity := Entity{
			Id:              id,
			TenantId:        t.Id(),
			NpcId:           npcId,
			TemplateId:      templateId,
			MesoPrice:       mesoPrice,
			DiscountRate:    discountRate,
			TokenTemplateId: tokenTemplateId,
			TokenPrice:      tokenPrice,
			Period:          period,
			LevelLimit:      levelLimited,
		}

		if err := db.Create(&entity).Error; err != nil {
			return Model{}, err
		}

		return Make(entity)
	}
}

func updateCommodity(ctx context.Context, db *gorm.DB) func(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenTemplateId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error) {
	return func(id uuid.UUID, templateId uint32, mesoPrice uint32, discountRate byte, tokenTemplateId uint32, tokenPrice uint32, period uint32, levelLimited uint32) (Model, error) {
		t := tenant.MustFromContext(ctx)
		var entity Entity
		if err := db.Where(&Entity{Id: id, TenantId: t.Id()}).First(&entity).Error; err != nil {
			return Model{}, err
		}

		entity.TemplateId = templateId
		entity.MesoPrice = mesoPrice
		entity.DiscountRate = discountRate
		entity.TokenTemplateId = tokenTemplateId
		entity.TokenPrice = tokenPrice
		entity.Period = period
		entity.LevelLimit = levelLimited

		if err := db.Save(&entity).Error; err != nil {
			return Model{}, err
		}

		return Make(entity)
	}
}

func deleteCommodity(ctx context.Context, db *gorm.DB) func(id uuid.UUID) error {
	return func(id uuid.UUID) error {
		t := tenant.MustFromContext(ctx)
		return db.Where(&Entity{Id: id, TenantId: t.Id()}).Delete(&Entity{}).Error
	}
}

func deleteAllCommoditiesByNpcId(ctx context.Context, db *gorm.DB) func(npcId uint32) error {
	return func(npcId uint32) error {
		t := tenant.MustFromContext(ctx)
		return db.Where(&Entity{NpcId: npcId, TenantId: t.Id()}).Delete(&Entity{}).Error
	}
}

func deleteAllCommodities(ctx context.Context, db *gorm.DB) func() error {
	return func() error {
		t := tenant.MustFromContext(ctx)
		return db.Where(&Entity{TenantId: t.Id()}).Delete(&Entity{}).Error
	}
}

// DeleteAllCommoditiesForTenant deletes all commodities for a specific tenant and returns the count
func DeleteAllCommoditiesForTenant(db *gorm.DB, tenantId uuid.UUID) (int64, error) {
	result := db.Unscoped().Where("tenant_id = ?", tenantId).Delete(&Entity{})
	return result.RowsAffected, result.Error
}

// BulkCreateCommodities creates multiple commodities in a single transaction
func BulkCreateCommodities(db *gorm.DB, tenantId uuid.UUID, npcCommodities []Model) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		for _, c := range npcCommodities {
			entity := &Entity{
				Id:              uuid.New(),
				TenantId:        tenantId,
				NpcId:           c.NpcId(),
				TemplateId:      c.TemplateId(),
				MesoPrice:       c.MesoPrice(),
				DiscountRate:    c.DiscountRate(),
				TokenTemplateId: c.TokenTemplateId(),
				TokenPrice:      c.TokenPrice(),
				Period:          c.Period(),
				LevelLimit:      c.LevelLimit(),
			}
			if err := tx.Create(entity).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
