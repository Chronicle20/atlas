package quest

import (
	"atlas-quest/quest/progress"
	"errors"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB, t tenant.Model, characterId uint32, questId uint32, expirationTime time.Time) (Model, error) {
	e := &Entity{
		TenantId:       t.Id(),
		CharacterId:    characterId,
		QuestId:        questId,
		State:          StateStarted,
		StartedAt:      time.Now(),
		ExpirationTime: expirationTime,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func restart(db *gorm.DB, tenantId uuid.UUID, id uint32, expirationTime time.Time) (Model, error) {
	entity, err := byIdEntityProvider(tenantId, id)(db)()
	if err != nil {
		return Model{}, err
	}

	// Clear progress for restarting
	if err := db.Where("quest_status_id = ?", entity.ID).Delete(&progress.Entity{}).Error; err != nil {
		return Model{}, fmt.Errorf("failed to delete progress: %w", err)
	}

	entity.State = StateStarted
	entity.StartedAt = time.Now()
	entity.CompletedAt = time.Time{} // Clear completion time
	entity.ExpirationTime = expirationTime
	entity.Progress = nil

	if err := db.Save(&entity).Error; err != nil {
		return Model{}, err
	}
	return Make(entity)
}

func completeQuest(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	entity, err := byIdEntityProvider(tenantId, id)(db)()
	if err != nil {
		return err
	}

	entity.State = StateCompleted
	entity.CompletedAt = time.Now()
	entity.CompletedCount++
	return db.Save(&entity).Error
}

func forfeitQuest(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	entity, err := byIdEntityProvider(tenantId, id)(db)()
	if err != nil {
		return err
	}

	// Clear progress
	if err := db.Where("quest_status_id = ?", entity.ID).Delete(&progress.Entity{}).Error; err != nil {
		return fmt.Errorf("failed to delete progress: %w", err)
	}

	// Reset state and increment forfeit count
	entity.State = StateNotStarted
	entity.ForfeitCount++
	entity.Progress = nil
	entity.ExpirationTime = time.Time{}
	return db.Save(&entity).Error
}

func setProgress(db *gorm.DB, tenantId uuid.UUID, id uint32, infoNumber uint32, progressValue string) error {
	entity, err := byIdEntityProvider(tenantId, id)(db)()
	if err != nil {
		return err
	}

	// Find existing progress or create new
	var found bool
	for i := range entity.Progress {
		if entity.Progress[i].InfoNumber == infoNumber {
			entity.Progress[i].Progress = progressValue
			found = true
			err = db.Save(&entity.Progress[i]).Error
			if err != nil {
				return err
			}
			break
		}
	}

	if !found {
		pe := progress.Entity{
			QuestStatusId: entity.ID,
			InfoNumber:    infoNumber,
			Progress:      progressValue,
		}
		err = db.Create(&pe).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func deleteWithProgress(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	entity, err := byIdEntityProvider(tenantId, id)(db)()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("failed to find entity: %w", err)
	}

	// Delete associated progress
	if err := db.Where("quest_status_id = ?", entity.ID).Delete(&progress.Entity{}).Error; err != nil {
		return fmt.Errorf("failed to delete progress: %w", err)
	}

	// Delete the entity
	if err := db.Delete(&entity).Error; err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	return nil
}

func deleteByCharacterIdWithProgress(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
	var entities []Entity
	if err := db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).Find(&entities).Error; err != nil {
		return fmt.Errorf("failed to find entities: %w", err)
	}

	for _, entity := range entities {
		if err := db.Where("quest_status_id = ?", entity.ID).Delete(&progress.Entity{}).Error; err != nil {
			return fmt.Errorf("failed to delete progress: %w", err)
		}
		if err := db.Delete(&entity).Error; err != nil {
			return fmt.Errorf("failed to delete entity: %w", err)
		}
	}

	return nil
}

// initializeProgress creates initial progress entries for mob kills and map visits
// mobIds is a list of mob IDs to track (for kill requirements)
// mapIds is a list of map IDs to track (for medal/fieldEnter requirements)
func initializeProgress(db *gorm.DB, questStatusId uint32, mobIds []uint32, mapIds []uint32) error {
	// Initialize mob progress entries (with "000" for 0 kills)
	for _, mobId := range mobIds {
		pe := progress.Entity{
			QuestStatusId: questStatusId,
			InfoNumber:    mobId,
			Progress:      "000",
		}
		if err := db.Create(&pe).Error; err != nil {
			return fmt.Errorf("failed to create mob progress for %d: %w", mobId, err)
		}
	}

	// Initialize map progress entries (with "0" for not visited)
	for _, mapId := range mapIds {
		pe := progress.Entity{
			QuestStatusId: questStatusId,
			InfoNumber:    mapId,
			Progress:      "0",
		}
		if err := db.Create(&pe).Error; err != nil {
			return fmt.Errorf("failed to create map progress for %d: %w", mapId, err)
		}
	}

	return nil
}
