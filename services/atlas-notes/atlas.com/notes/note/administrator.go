package note

import (
	database "github.com/Chronicle20/atlas-database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func createNote(db *gorm.DB, tenantId uuid.UUID, note Model) (Model, error) {
	entity := MakeEntity(tenantId, note)
	entity.ID = 0

	err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Create(&entity).Error
	})
	if err != nil {
		return Model{}, err
	}

	return Make(entity)
}

func updateNote(db *gorm.DB, tenantId uuid.UUID, note Model) (Model, error) {
	entity := MakeEntity(tenantId, note)

	err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Where("id = ?", note.Id()).Updates(&entity).Error
	})
	if err != nil {
		return Model{}, err
	}

	entity, err = getByIdProvider(note.Id())(db)()
	if err != nil {
		return Model{}, err
	}
	return Make(entity)
}

func deleteNote(db *gorm.DB, id uint32) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Where("id = ?", id).Delete(&Entity{}).Error
	})
}

func deleteAllNotes(db *gorm.DB, characterId uint32) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Where("character_id = ?", characterId).Delete(&Entity{}).Error
	})
}
