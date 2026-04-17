package shop

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Entity struct {
	gorm.Model
	Id              uuid.UUID  `gorm:"type:uuid;primaryKey"`
	TenantId        uuid.UUID  `gorm:"type:uuid;not null;index"`
	TenantRegion    string     `gorm:"type:varchar(10);not null;default:''"`
	TenantMajor     uint16     `gorm:"not null;default:0"`
	TenantMinor     uint16     `gorm:"not null;default:0"`
	CharacterId     uint32     `gorm:"not null;index"`
	ShopType        byte       `gorm:"not null"`
	State           byte       `gorm:"not null"`
	Title           string     `gorm:"type:varchar(255);not null;default:''"`
	WorldId         world.Id   `gorm:"not null;default:0"`
	ChannelId       channel.Id `gorm:"not null;default:0"`
	MapId           uint32     `gorm:"not null;index"`
	InstanceId      uuid.UUID  `gorm:"type:uuid;not null;default:'00000000-0000-0000-0000-000000000000'"`
	X               int16      `gorm:"not null"`
	Y               int16      `gorm:"not null"`
	PermitItemId    uint32     `gorm:"not null"`
	ExpiresAt       *time.Time `gorm:"index"`
	ClosedAt        *time.Time
	CloseReason     byte   `gorm:"not null;default:0"`
	MesoBalance     uint32 `gorm:"not null;default:0"`
}

func (e *Entity) TableName() string {
	return "shops"
}

func Make(entity Entity) (Model, error) {
	return NewBuilder().
		SetId(entity.Id).
		SetCharacterId(entity.CharacterId).
		SetShopType(ShopType(entity.ShopType)).
		SetState(State(entity.State)).
		SetTitle(entity.Title).
		SetWorldId(entity.WorldId).
		SetChannelId(entity.ChannelId).
		SetMapId(entity.MapId).
		SetInstanceId(entity.InstanceId).
		SetX(entity.X).
		SetY(entity.Y).
		SetPermitItemId(entity.PermitItemId).
		SetCreatedAt(entity.CreatedAt).
		SetExpiresAt(entity.ExpiresAt).
		SetClosedAt(entity.ClosedAt).
		SetCloseReason(CloseReason(entity.CloseReason)).
		SetMesoBalance(entity.MesoBalance).
		Build()
}

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
