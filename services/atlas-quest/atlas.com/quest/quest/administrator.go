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

func create(db *gorm.DB, t tenant.Model, characterId uint32, questId uint32) (Model, error) {
	e := &Entity{
		TenantId:    t.Id(),
		CharacterId: characterId,
		QuestId:     questId,
		State:       StateStarted,
		StartedAt:   time.Now(),
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func updateState(db *gorm.DB, tenantId uuid.UUID, id uint32, state State) error {
	entity, err := byIdEntityProvider(tenantId, id)(db)()
	if err != nil {
		return err
	}

	entity.State = state
	if state == StateCompleted {
		entity.CompletedAt = time.Now()
	}
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
