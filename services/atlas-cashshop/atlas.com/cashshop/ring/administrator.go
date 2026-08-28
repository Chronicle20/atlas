package ring

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Half is one side of a pair before it has a pair id.
type Half struct {
	CharacterId uint32
	AssetId     uint32
	// CashId is this half's own asset's cash id at purchase time (the
	// caller reads it off the asset.Model returned by astP.Create /
	// CreateGift). Persisted on Entity so enrich (processor.go) does not
	// need to re-resolve it once the ring leaves the locker -- see
	// docs/tasks/task-269-ring-pair-behavior/bug-ring-cash-id-resolves-to-zero.md.
	CashId         int64
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

	ma, err := NewBuilder().
		SetPairId(pairId).
		SetCharacterId(a.CharacterId).
		SetPartnerCharacterId(b.CharacterId).
		SetAssetId(a.AssetId).
		SetCashId(a.CashId).
		SetItemTemplateId(a.ItemTemplateId).
		SetType(ringType).
		SetState(StateActive).
		SetCreatedAt(now).
		Build()
	if err != nil {
		return uuid.Nil, err
	}

	mb, err := NewBuilder().
		SetPairId(pairId).
		SetCharacterId(b.CharacterId).
		SetPartnerCharacterId(a.CharacterId).
		SetAssetId(b.AssetId).
		SetCashId(b.CashId).
		SetItemTemplateId(b.ItemTemplateId).
		SetType(ringType).
		SetState(StateActive).
		SetCreatedAt(now).
		Build()
	if err != nil {
		return uuid.Nil, err
	}

	rows := []Entity{ma.ToEntity(tenantId), mb.ToEntity(tenantId)}

	if err := db.Create(&rows).Error; err != nil {
		return uuid.Nil, err
	}
	return pairId, nil
}
