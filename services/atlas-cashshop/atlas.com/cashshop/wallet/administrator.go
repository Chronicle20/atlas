package wallet

import (
	"github.com/Chronicle20/atlas-tenant"
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

func updateEntity(db *gorm.DB, accountId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
	var e Entity

	err := db.
		Where("account_id = ?", accountId).
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

func deleteEntity(db *gorm.DB, accountId uint32) error {
	return db.Where("account_id = ?", accountId).Delete(&Entity{}).Error
}
