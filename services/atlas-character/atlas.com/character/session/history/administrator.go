package history

import (
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// createSession creates a new session record for a character login
func createSession(db *gorm.DB, tenantId uuid.UUID, characterId uint32, worldId world.Id, channelId channel.Id) (Model, error) {
	e := &entity{
		TenantId:    tenantId,
		CharacterId: characterId,
		WorldId:     worldId,
		ChannelId:   channelId,
		LoginTime:   time.Now(),
		LogoutTime:  nil,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}

	return modelFromEntity(*e), nil
}

// closeSession closes an active session by setting the logout time
func closeSession(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
	now := time.Now()
	return db.Model(&entity{}).
		Where("tenant_id = ? AND character_id = ? AND logout_time IS NULL", tenantId, characterId).
		Update("logout_time", now).Error
}

// getActiveSession returns the current active session for a character, if any
func getActiveSession(db *gorm.DB, tenantId uuid.UUID, characterId uint32) (Model, error) {
	var e entity
	err := db.Where("tenant_id = ? AND character_id = ? AND logout_time IS NULL", tenantId, characterId).
		First(&e).Error
	if err != nil {
		return Model{}, err
	}
	return modelFromEntity(e), nil
}

// getSessionsSince returns all sessions for a character since the given time
func getSessionsSince(db *gorm.DB, tenantId uuid.UUID, characterId uint32, since time.Time) ([]Model, error) {
	var entities []entity
	// Get sessions that either:
	// 1. Started after 'since', OR
	// 2. Were still active (no logout) at 'since', OR
	// 3. Ended after 'since'
	err := db.Where("tenant_id = ? AND character_id = ? AND (login_time >= ? OR logout_time IS NULL OR logout_time >= ?)",
		tenantId, characterId, since, since).
		Order("login_time ASC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}

	models := make([]Model, len(entities))
	for i, e := range entities {
		models[i] = modelFromEntity(e)
	}
	return models, nil
}

// getSessionsInRange returns all sessions that overlap with the given time range
func getSessionsInRange(db *gorm.DB, tenantId uuid.UUID, characterId uint32, start, end time.Time) ([]Model, error) {
	var entities []entity
	// Get sessions that overlap with [start, end]:
	// Session overlaps if: login_time < end AND (logout_time IS NULL OR logout_time > start)
	err := db.Where("tenant_id = ? AND character_id = ? AND login_time < ? AND (logout_time IS NULL OR logout_time > ?)",
		tenantId, characterId, end, start).
		Order("login_time ASC").
		Find(&entities).Error
	if err != nil {
		return nil, err
	}

	models := make([]Model, len(entities))
	for i, e := range entities {
		models[i] = modelFromEntity(e)
	}
	return models, nil
}
