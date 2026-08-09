package coupon

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Entity is the coupons table. Code is stored NORMALIZED (trimmed, uppercased),
// so uniqueIndex on (tenant_id, code) IS the case-insensitive uniqueness
// guarantee — do not add a functional index over a raw value.
//
// Rewards is jsonb because the bundle is always read and written whole and is
// never queried by reward attribute.
type Entity struct {
	Id              uuid.UUID  `gorm:"primaryKey;type:uuid"`
	TenantId        uuid.UUID  `gorm:"not null;index;uniqueIndex:idx_coupons_tenant_code;index:idx_coupons_tenant_batch,priority:1"`
	BatchId         *uuid.UUID `gorm:"type:uuid;index:idx_coupons_tenant_batch,priority:2"`
	Code            string     `gorm:"not null;type:varchar(32);uniqueIndex:idx_coupons_tenant_code"`
	Description     string     `gorm:"type:text"`
	Active          bool       `gorm:"not null;default:true"`
	StartsAt        *time.Time
	ExpiresAt       *time.Time
	MaxUses         *uint32
	RedemptionCount uint32  `gorm:"not null;default:0"`
	Rewards         Rewards `gorm:"not null;type:jsonb"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (e Entity) TableName() string {
	return "coupons"
}

func (e *Entity) BeforeCreate(_ *gorm.DB) (err error) {
	if e.Id == uuid.Nil {
		e.Id = uuid.New()
	}
	return
}

func Make(e Entity) (Model, error) {
	batchId := uuid.Nil
	if e.BatchId != nil {
		batchId = *e.BatchId
	}
	return Model{
		id:              e.Id,
		batchId:         batchId,
		code:            e.Code,
		description:     e.Description,
		active:          e.Active,
		startsAt:        e.StartsAt,
		expiresAt:       e.ExpiresAt,
		maxUses:         e.MaxUses,
		redemptionCount: e.RedemptionCount,
		rewards:         e.Rewards,
		createdAt:       e.CreatedAt,
		updatedAt:       e.UpdatedAt,
	}, nil
}
