package ring

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Half is one side of a pair before it has a pair id.
type Half struct {
	CharacterId    uint32
	AssetId        uint32
	ItemTemplateId uint32
}

// CreatePair inserts BOTH halves of a pair in one call. It is called inside
// the ring purchase transaction; a partial pair must never be persistable
// (FR-RING-4).
//
// Both rows are written with a single db.Create call over a slice, which GORM
// sends as one multi-row INSERT -- so the statement lands or neither half
// does, even if the caller's handle is not itself already inside a
// transaction.
func CreatePair(db *gorm.DB, tenantId uuid.UUID, ringType Type, a Half, b Half) (uuid.UUID, error) {
	pairId := uuid.New()
	now := time.Now()

	rows := []Entity{
		{
			Id:                 uuid.New(),
			TenantId:           tenantId,
			PairId:             pairId,
			CharacterId:        a.CharacterId,
			PartnerCharacterId: b.CharacterId,
			AssetId:            a.AssetId,
			ItemTemplateId:     a.ItemTemplateId,
			RingType:           string(ringType),
			State:              string(StateActive),
			CreatedAt:          now,
		},
		{
			Id:                 uuid.New(),
			TenantId:           tenantId,
			PairId:             pairId,
			CharacterId:        b.CharacterId,
			PartnerCharacterId: a.CharacterId,
			AssetId:            b.AssetId,
			ItemTemplateId:     b.ItemTemplateId,
			RingType:           string(ringType),
			State:              string(StateActive),
			CreatedAt:          now,
		},
	}

	if err := db.Create(&rows).Error; err != nil {
		return uuid.Nil, err
	}
	return pairId, nil
}
