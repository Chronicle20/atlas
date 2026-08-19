package ring

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Entity is one HALF of a ring pair. Two rows share a PairId; both are
// inserted in the same transaction as the two cash assets and the wallet
// debit, so a partial pair is not persistable (FR-RING-4).
//
// State rather than delete-only: a later task that breaks, expires, or
// un-equips a ring needs somewhere to record that without losing the history
// (FR-RING-9).
type Entity struct {
	Id                 uuid.UUID `gorm:"primaryKey;not null"`
	TenantId           uuid.UUID `gorm:"not null;index"`
	PairId             uuid.UUID `gorm:"not null;index"`
	CharacterId        uint32    `gorm:"not null;index"`
	PartnerCharacterId uint32    `gorm:"not null"`
	AssetId            uint32    `gorm:"not null"`
	ItemTemplateId     uint32    `gorm:"not null"`
	RingType           string    `gorm:"not null"`
	State              string    `gorm:"not null"`
	CreatedAt          time.Time `gorm:"not null"`
}

func (e Entity) TableName() string { return "cash_rings" }

func Make(e Entity) (Model, error) {
	return Model{
		id:                 e.Id,
		pairId:             e.PairId,
		characterId:        e.CharacterId,
		partnerCharacterId: e.PartnerCharacterId,
		assetId:            e.AssetId,
		itemTemplateId:     e.ItemTemplateId,
		ringType:           Type(e.RingType),
		state:              State(e.State),
		createdAt:          e.CreatedAt,
	}, nil
}

// ToEntity transforms a Model into an Entity for persistence.
func (m Model) ToEntity(tenantId uuid.UUID) Entity {
	return Entity{
		Id:                 m.id,
		TenantId:           tenantId,
		PairId:             m.pairId,
		CharacterId:        m.characterId,
		PartnerCharacterId: m.partnerCharacterId,
		AssetId:            m.assetId,
		ItemTemplateId:     m.itemTemplateId,
		RingType:           string(m.ringType),
		State:              string(m.state),
		CreatedAt:          m.createdAt,
	}
}
