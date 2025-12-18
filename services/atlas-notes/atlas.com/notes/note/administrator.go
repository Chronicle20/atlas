package note

import (
	"atlas-notes/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// createNote creates a new note in the database
func createNote(db *gorm.DB) func(tenantId uuid.UUID) func(note Model) (Model, error) {
	return func(tenantId uuid.UUID) func(note Model) (Model, error) {
		return func(note Model) (Model, error) {
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
	}
}

// updateNote updates an existing note in the database
func updateNote(db *gorm.DB) func(tenantId uuid.UUID) func(note Model) (Model, error) {
	return func(tenantId uuid.UUID) func(note Model) (Model, error) {
		return func(note Model) (Model, error) {
			entity := MakeEntity(tenantId, note)

			err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
				return tx.Where("tenant_id = ? AND id = ?", tenantId, note.Id()).Updates(&entity).Error
			})
			if err != nil {
				return Model{}, err
			}

			entity, err = getByIdProvider(tenantId)(note.Id())(db)()
			if err != nil {
				return Model{}, err
			}
			return Make(entity)
		}
	}
}

// deleteNote deletes a note from the database
func deleteNote(db *gorm.DB) func(tenantId uuid.UUID) func(id uint32) error {
	return func(tenantId uuid.UUID) func(id uint32) error {
		return func(id uint32) error {
			return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
				return tx.Where("tenant_id = ? AND id = ?", tenantId, id).Delete(&Entity{}).Error
			})
		}
	}
}

// deleteAllNotes deletes all notes for a character from the database
func deleteAllNotes(db *gorm.DB) func(tenantId uuid.UUID) func(characterId uint32) error {
	return func(tenantId uuid.UUID) func(characterId uint32) error {
		return func(characterId uint32) error {
			return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
				return tx.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).Delete(&Entity{}).Error
			})
		}
	}
}
