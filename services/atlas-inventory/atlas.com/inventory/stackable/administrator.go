package stackable

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB, tenantId uuid.UUID, compartmentId uuid.UUID, quantity uint32, ownerId uint32, flag uint16, rechargeable uint64) (Model, error) {
	e := &Entity{
		TenantId:      tenantId,
		CompartmentId: compartmentId,
		Quantity:      quantity,
		OwnerId:       ownerId,
		Flag:          flag,
		Rechargeable:  rechargeable,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func updateQuantity(db *gorm.DB, tenantId uuid.UUID, id uint32, quantity uint32) error {
	return db.Model(&Entity{TenantId: tenantId, Id: id}).Select("Quantity").Updates(&Entity{Quantity: quantity}).Error
}

func deleteById(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	return db.Where(&Entity{TenantId: tenantId, Id: id}).Delete(&Entity{}).Error
}
