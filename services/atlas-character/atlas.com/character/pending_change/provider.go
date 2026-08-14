package pending_change

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/google/uuid"
)

func modelFromEntity(e entity) (Model, error) {
	b := NewBuilder().
		SetId(e.Id).
		SetCharacterId(e.CharacterId).
		SetType(e.Type).
		SetStatus(e.Status).
		SetSourceWorldId(e.SourceWorldId).
		SetReason(e.Reason).
		SetTransactionId(e.TransactionId).
		SetCreatedAt(e.CreatedAt).
		SetExpiresAt(e.ExpiresAt).
		SetResolvedAt(e.ResolvedAt).
		SetNotifiedAt(e.NotifiedAt)

	if e.RequestedName != nil {
		b.SetRequestedName(*e.RequestedName)
	}
	if e.DestinationWorldId != nil {
		b.SetDestinationWorldId(*e.DestinationWorldId)
	}
	if e.AssetId != nil {
		b.SetAssetId(*e.AssetId)
	}

	return b.Build(), nil
}

func getById(db *gorm.DB, tenantId uuid.UUID, id uuid.UUID) (Model, error) {
	var e entity
	err := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Model{}, ErrNotFound
		}
		return Model{}, err
	}
	return modelFromEntity(e)
}

func getByCharacterId(db *gorm.DB, tenantId uuid.UUID, characterId uint32) ([]Model, error) {
	var es []entity
	if err := db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).Find(&es).Error; err != nil {
		return nil, err
	}
	ms := make([]Model, 0, len(es))
	for _, e := range es {
		m, err := modelFromEntity(e)
		if err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	return ms, nil
}

func getPendingByNameLower(db *gorm.DB, tenantId uuid.UUID, nameLower string) (Model, error) {
	var e entity
	err := db.Where("tenant_id = ? AND status = ? AND type = ? AND requested_name_lower = ?", tenantId, StatusPending, TypeNameChange, nameLower).First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Model{}, ErrNotFound
		}
		return Model{}, err
	}
	return modelFromEntity(e)
}

func getExpired(db *gorm.DB, tenantId uuid.UUID, now time.Time) ([]Model, error) {
	var es []entity
	if err := db.Where("tenant_id = ? AND status = ? AND expires_at < ?", tenantId, StatusPending, now).Find(&es).Error; err != nil {
		return nil, err
	}
	ms := make([]Model, 0, len(es))
	for _, e := range es {
		m, err := modelFromEntity(e)
		if err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	return ms, nil
}

func getResolvedUnnotified(db *gorm.DB, tenantId uuid.UUID, characterId uint32) ([]Model, error) {
	var es []entity
	if err := db.Where("tenant_id = ? AND character_id = ? AND resolved_at IS NOT NULL AND notified_at IS NULL", tenantId, characterId).Find(&es).Error; err != nil {
		return nil, err
	}
	ms := make([]Model, 0, len(es))
	for _, e := range es {
		m, err := modelFromEntity(e)
		if err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	return ms, nil
}

func getPendingByCharacterId(db *gorm.DB, tenantId uuid.UUID, characterId uint32) ([]Model, error) {
	var es []entity
	if err := db.Where("tenant_id = ? AND character_id = ? AND status = ?", tenantId, characterId, StatusPending).Find(&es).Error; err != nil {
		return nil, err
	}
	ms := make([]Model, 0, len(es))
	for _, e := range es {
		m, err := modelFromEntity(e)
		if err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	return ms, nil
}
