package cash

import (
	"atlas-cashshop/cash/wishlist"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	TenantId    uuid.UUID         `gorm:"not null"`
	CharacterId uint32            `gorm:"primaryKey;autoIncrement:false;not null"`
	Credit      uint32            `gorm:"not null;default=0"`
	Points      uint32            `gorm:"not null;default=0"`
	Prepaid     uint32            `gorm:"not null;default=0"`
	Wishlist    []wishlist.Entity `gorm:"foreignKey:CharacterId"`
}

func (e Entity) TableName() string {
	return "characters"
}

func Make(e Entity) (Model, error) {
	wl := make([]wishlist.Model, 0)
	for _, w := range e.Wishlist {
		wl = append(wl, wishlist.NewModel(w.Id, w.SerialNumber))
	}

	return Model{
		characterId: e.CharacterId,
		credit:      e.Credit,
		points:      e.Points,
		prepaid:     e.Prepaid,
		wishlist:    wl,
	}, nil
}
