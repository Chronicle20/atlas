package asset

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId      uuid.UUID      `gorm:"not null"`
	Id            uint32         `gorm:"primaryKey;autoIncrement;not null"`
	CompartmentId uuid.UUID      `gorm:"not null"`
	Slot          int16          `gorm:"not null"`
	TemplateId    uint32         `gorm:"not null"`
	Expiration    time.Time      `gorm:"not null"`
	CreatedAt     time.Time      `gorm:"not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
	// stackable fields
	Quantity     uint32
	OwnerId      uint32
	Flag         uint16
	Rechargeable uint64
	// equipment fields
	Strength       uint16
	Dexterity      uint16
	Intelligence   uint16
	Luck           uint16
	Hp             uint16
	Mp             uint16
	WeaponAttack   uint16
	MagicAttack    uint16
	WeaponDefense  uint16
	MagicDefense   uint16
	Accuracy       uint16
	Avoidability   uint16
	Hands          uint16
	Speed          uint16
	Jump           uint16
	Slots          uint16
	Locked         bool
	Spikes         bool
	KarmaUsed      bool
	Cold           bool
	CanBeTraded    bool
	LevelType      byte
	Level          byte
	Experience     uint32
	HammersApplied uint32
	EquippedSince  *time.Time
	// cash fields
	CashId      int64
	CommodityId uint32
	PurchaseBy  uint32
	// pet reference
	PetId uint32
}

func (e Entity) TableName() string {
	return "assets"
}

func Make(e Entity) (Model, error) {
	return Model{
		id:             e.Id,
		compartmentId:  e.CompartmentId,
		slot:           e.Slot,
		templateId:     e.TemplateId,
		expiration:     e.Expiration,
		createdAt:      e.CreatedAt,
		quantity:       e.Quantity,
		ownerId:        e.OwnerId,
		flag:           e.Flag,
		rechargeable:   e.Rechargeable,
		strength:       e.Strength,
		dexterity:      e.Dexterity,
		intelligence:   e.Intelligence,
		luck:           e.Luck,
		hp:             e.Hp,
		mp:             e.Mp,
		weaponAttack:   e.WeaponAttack,
		magicAttack:    e.MagicAttack,
		weaponDefense:  e.WeaponDefense,
		magicDefense:   e.MagicDefense,
		accuracy:       e.Accuracy,
		avoidability:   e.Avoidability,
		hands:          e.Hands,
		speed:          e.Speed,
		jump:           e.Jump,
		slots:          e.Slots,
		locked:         e.Locked,
		spikes:         e.Spikes,
		karmaUsed:      e.KarmaUsed,
		cold:           e.Cold,
		canBeTraded:    e.CanBeTraded,
		levelType:      e.LevelType,
		level:          e.Level,
		experience:     e.Experience,
		hammersApplied: e.HammersApplied,
		equippedSince:  e.EquippedSince,
		cashId:         e.CashId,
		commodityId:    e.CommodityId,
		purchaseBy:     e.PurchaseBy,
		petId:          e.PetId,
	}, nil
}
