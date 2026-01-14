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
	e := NewEntityBuilder().
		SetTenantId(t.Id()).
		SetCharacterId(characterId).
		SetQuestId(questId).
		SetState(StateStarted).
		SetStartedAt(time.Now()).
		SetExpirationTime(expirationTime).
		Build()

	err := db.Create(&e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(e)
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

	updated := CloneEntity(entity).
		SetState(StateStarted).
		SetStartedAt(time.Now()).
		SetCompletedAt(time.Time{}).
		SetExpirationTime(expirationTime).
		SetProgress(nil).
		Build()

	if err := db.Save(&updated).Error; err != nil {
		return Model{}, err
	}
	return Make(updated)
}

func completeQuest(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	entity, err := byIdEntityProvider(tenantId, id)(db)()
	if err != nil {
		return err
	}

	updated := CloneEntity(entity).
		SetState(StateCompleted).
		SetCompletedAt(time.Now()).
		SetCompletedCount(entity.CompletedCount + 1).
		Build()

	return db.Save(&updated).Error
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
	updated := CloneEntity(entity).
		SetState(StateNotStarted).
		SetForfeitCount(entity.ForfeitCount + 1).
		SetProgress(nil).
		SetExpirationTime(time.Time{}).
		Build()

	return db.Save(&updated).Error
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
			updated := progress.CloneEntity(entity.Progress[i]).
				SetProgress(progressValue).
				Build()
			found = true
			err = db.Save(&updated).Error
			if err != nil {
				return err
			}
			break
		}
	}

	if !found {
		pe := progress.NewEntityBuilder().
			SetTenantId(tenantId).
			SetQuestStatusId(entity.ID).
			SetInfoNumber(infoNumber).
			SetProgress(progressValue).
			Build()
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
func initializeProgress(db *gorm.DB, tenantId uuid.UUID, questStatusId uint32, mobIds []uint32, mapIds []uint32) error {
	// Initialize mob progress entries (with "000" for 0 kills)
	for _, mobId := range mobIds {
		pe := progress.NewEntityBuilder().
			SetTenantId(tenantId).
			SetQuestStatusId(questStatusId).
			SetInfoNumber(mobId).
			SetProgress("000").
			Build()
		if err := db.Create(&pe).Error; err != nil {
			return fmt.Errorf("failed to create mob progress for %d: %w", mobId, err)
		}
	}

	// Initialize map progress entries (with "0" for not visited)
	for _, mapId := range mapIds {
		pe := progress.NewEntityBuilder().
			SetTenantId(tenantId).
			SetQuestStatusId(questStatusId).
			SetInfoNumber(mapId).
			SetProgress("0").
			Build()
		if err := db.Create(&pe).Error; err != nil {
			return fmt.Errorf("failed to create map progress for %d: %w", mapId, err)
		}
	}

	return nil
}
