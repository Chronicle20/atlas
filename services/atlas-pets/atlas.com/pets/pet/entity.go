package pet

import (
	"atlas-pets/pet/exclude"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId   uuid.UUID        `gorm:"not null;"`
	OwnerId    uint32           `gorm:"not null;"`
	Id         uint32           `gorm:"primary_key;auto_increment"`
	CashId     uint64           `gorm:"not null"`
	TemplateId uint32           `gorm:"not null"`
	Name       string           `gorm:"size:13;not null"`
	Level      byte             `gorm:"not null;default:1"`
	Closeness  uint16           `gorm:"not null;default:0"`
	Fullness   byte             `gorm:"not null;default:100"`
	Expiration time.Time        `gorm:"not null;"`
	Slot       *int8            `gorm:"not null;default:-1"`
	Excludes   []exclude.Entity `gorm:"foreignkey:PetId"`
	Flag       uint16           `gorm:"not null;default:0"`
	PurchaseBy uint32           `gorm:"not null;default:0"`
	// ReviveTransactionId is the saga transaction of the last successful Water
	// of Life revive. It is what distinguishes a Kafka REDELIVERY (same id =>
	// replay, no second grant) from a genuine SECOND water used on an
	// already-revived pet (different id => reject and refund). Neither
	// "reject if alive" nor "no-op if alive" alone can tell those apart.
	ReviveTransactionId *uuid.UUID `gorm:"type:uuid"`
}

func (e Entity) TableName() string {
	return "pets"
}

// ToEntity is the inverse of Make: it projects the immutable Model into the
// GORM entity used for persistence. Id is deliberately left zero — it is
// assigned by the auto-increment on insert. Excludes are NOT carried across:
// they are owned by their own table and written through setExcludes, so
// populating the association here would make GORM cascade-insert them on
// every create.
func (m Model) ToEntity(tenantId uuid.UUID) Entity {
	s := m.slot
	return Entity{
		TenantId:            tenantId,
		OwnerId:             m.ownerId,
		CashId:              m.cashId,
		TemplateId:          m.templateId,
		Name:                m.name,
		Level:               m.level,
		Closeness:           m.closeness,
		Fullness:            m.fullness,
		Expiration:          m.expiration,
		Slot:                &s,
		Flag:                m.flag,
		PurchaseBy:          m.purchaseBy,
		ReviveTransactionId: m.reviveTransactionId,
	}
}

func Make(e Entity) (Model, error) {
	es, err := model.SliceMap(exclude.Make)(model.FixedProvider(e.Excludes))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}
	return NewModelBuilder(e.Id, e.CashId, e.TemplateId, e.Name, e.OwnerId).
		SetLevel(e.Level).
		SetCloseness(e.Closeness).
		SetFullness(e.Fullness).
		SetExpiration(e.Expiration).
		SetSlot(*e.Slot).
		SetExcludes(es).
		SetFlag(e.Flag).
		SetPurchaseBy(e.PurchaseBy).
		SetReviveTransactionId(e.ReviveTransactionId).
		Build()
}
