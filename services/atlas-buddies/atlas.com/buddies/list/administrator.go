package list

import (
	"atlas-buddies/buddy"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB, tenantId uuid.UUID, characterId uint32, capacity byte) (Model, error) {
	e := &Entity{
		TenantId:    tenantId,
		CharacterId: characterId,
		Capacity:    capacity,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func addPendingBuddy(db *gorm.DB, characterId uint32, targetId uint32, targetName string, group string) error {
	return addBuddy(db, characterId, targetId, targetName, group, true)
}

func addBuddy(db *gorm.DB, characterId uint32, targetId uint32, targetName string, group string, pending bool) error {
	e, err := byCharacterIdEntityProvider(characterId)(db)()
	if err != nil {
		return err
	}

	nb := buddy.Entity{
		CharacterId:   targetId,
		ListId:        e.Id,
		Group:         group,
		CharacterName: targetName,
		ChannelId:     -1,
		Pending:       pending,
	}
	return db.Create(&nb).Error
}

func removeBuddy(db *gorm.DB, characterId uint32, targetId uint32) error {
	e, err := byCharacterIdEntityProvider(characterId)(db)()
	if err != nil {
		return err
	}

	var rb buddy.Entity
	for _, b := range e.Buddies {
		if b.CharacterId == targetId {
			rb = b
			break
		}
	}

	if rb.ListId == uuid.Nil {
		return gorm.ErrRecordNotFound
	}

	return db.Delete(&rb).Error
}

func updateBuddyChannel(db *gorm.DB, characterId uint32, targetId uint32, channelId int8) (bool, error) {
	bbl, err := byCharacterIdEntityProvider(targetId)(db)()
	if err != nil {
		return false, err
	}

	var meAsBuddy *buddy.Entity
	for _, pm := range bbl.Buddies {
		if pm.CharacterId == characterId {
			meAsBuddy = &pm
		}
	}
	if meAsBuddy == nil {
		return false, nil
	}
	meAsBuddy.ChannelId = channelId

	err = db.Save(meAsBuddy).Error
	if err != nil {
		return false, err
	}
	return true, nil
}

func updateBuddyShopStatus(db *gorm.DB, characterId uint32, targetId uint32, inShop bool) (bool, error) {
	bbl, err := byCharacterIdEntityProvider(targetId)(db)()
	if err != nil {
		return false, err
	}

	var meAsBuddy *buddy.Entity
	for _, pm := range bbl.Buddies {
		if pm.CharacterId == characterId {
			meAsBuddy = &pm
		}
	}
	if meAsBuddy == nil {
		return false, nil
	}
	meAsBuddy.InShop = inShop

	err = db.Save(meAsBuddy).Error
	if err != nil {
		return false, err
	}
	return true, nil
}

// buddyNameUpdate reports one owner's buddy-list row that updateBuddyName
// actually renamed, carrying what the caller needs to emit BUDDY_UPDATED to
// that owner (list/processor.go:610-645's UpdateBuddyChannel emit pattern).
type buddyNameUpdate struct {
	OwnerId   uint32
	Group     string
	ChannelId int8
	InShop    bool
}

// updateBuddyName renames every buddy-list row that NAMES targetId — rows
// whose own character_id column equals targetId, never the id of whichever
// owner happens to hold it — to name. A naive implementation that instead
// scoped the update to "targetId's own list" (treating character_id as the
// list owner, as list.Entity's field of the same name does) would silently
// rename targetId's OWN buddies instead of targetId's name in every OTHER
// owner's list — the exact wrong-column bug this function exists to avoid.
//
// buddy.Entity's sole primary key is character_id (buddy/entity.go:24), a
// pre-existing defect (surfaced separately, not fixed by this task) that
// limits the table to at most one such row across all owners today; the
// query still matches on the buddy's column so it stays correct if that
// defect is ever fixed and multiple owners come to hold the same buddy.
// Rows already at name are left untouched and excluded from the returned
// slice — that is what keeps a redelivered NAME_CHANGED event from emitting
// BUDDY_UPDATED a second time.
func updateBuddyName(db *gorm.DB, targetId uint32, name string) ([]buddyNameUpdate, error) {
	var rows []buddy.Entity
	if err := db.Where("character_id = ?", targetId).Find(&rows).Error; err != nil {
		return nil, err
	}

	var updates []buddyNameUpdate
	for _, row := range rows {
		if row.CharacterName == name {
			continue
		}
		row.CharacterName = name
		if err := db.Save(&row).Error; err != nil {
			return nil, err
		}

		var le Entity
		if err := db.Where("id = ?", row.ListId).First(&le).Error; err != nil {
			return nil, err
		}
		updates = append(updates, buddyNameUpdate{OwnerId: le.CharacterId, Group: row.Group, ChannelId: row.ChannelId, InShop: row.InShop})
	}
	return updates, nil
}

func deleteEntityWithBuddies(db *gorm.DB, characterId uint32) error {
	var entity Entity

	// Step 1: Find the Entity
	if err := db.
		Where("character_id = ?", characterId).
		First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // No-op if not found
		}
		return fmt.Errorf("failed to find entity: %w", err)
	}

	// Step 2: Delete associated Buddies
	if err := db.
		Where("list_id = ?", entity.Id).
		Delete(&buddy.Entity{}).Error; err != nil {
		return fmt.Errorf("failed to delete buddies: %w", err)
	}

	// Step 3: Delete the Entity
	if err := db.Delete(&entity).Error; err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	return nil
}

// updateCapacity increases the buddy list capacity for a character.
// This function validates that the new capacity is greater than the current capacity
// before performing the database update.
//
// Parameters:
//   - db: Database transaction or connection
//   - characterId: ID of the character whose capacity should be increased
//   - newCapacity: The new capacity value (must be > current capacity)
//
// Returns:
//   - error: nil on success, or an error if validation fails or database operation fails
//
// Validation Rules:
//   - newCapacity must be strictly greater than the current capacity
//   - Character must exist in the database
//
// Error Conditions:
//   - Returns "INVALID_CAPACITY" error if newCapacity <= currentCapacity
//   - Returns database error if character not found or save operation fails
func updateCapacity(db *gorm.DB, characterId uint32, newCapacity byte) error {
	// Get the current entity to validate capacity
	entity, err := byCharacterIdEntityProvider(characterId)(db)()
	if err != nil {
		return err
	}

	// Validate that new capacity is greater than current capacity
	if newCapacity <= entity.Capacity {
		return errors.New("INVALID_CAPACITY")
	}

	// Update the capacity
	entity.Capacity = newCapacity
	return db.Save(&entity).Error
}
