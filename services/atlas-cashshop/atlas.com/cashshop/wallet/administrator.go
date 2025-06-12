package wallet

import (
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func createEntity(db *gorm.DB, t tenant.Model, accountId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
	e := &Entity{
		TenantId:  t.Id(),
		AccountId: accountId,
		Credit:    credit,
		Points:    points,
		Prepaid:   prepaid,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func updateEntity(db *gorm.DB, t tenant.Model, accountId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
	var e Entity

	err := db.
		Where("tenant_id = ? AND account_id = ?", t.Id(), accountId).
		First(&e).Error
	if err != nil {
		return Model{}, err
	}

	e.Credit = credit
	e.Points = points
	e.Prepaid = prepaid

	err = db.Save(&e).Error
	if err != nil {
		return Model{}, err
	}

	return Make(e)
}

func deleteEntity(ctx context.Context) func(db *gorm.DB, tenantId uuid.UUID, accountId uint32) error {
	return func(db *gorm.DB, tenantId uuid.UUID, accountId uint32) error {
		return db.WithContext(ctx).Where("tenant_id = ? AND account_id = ?", tenantId, accountId).Delete(&Entity{}).Error
	}
}
