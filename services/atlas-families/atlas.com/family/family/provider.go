package family

import (
	database "github.com/Chronicle20/atlas-database"
	"errors"

	"github.com/Chronicle20/atlas-model/model"
	"gorm.io/gorm"
)

// GetByCharacterIdProvider returns a provider for finding a family member by character ID
func GetByCharacterIdProvider(characterId uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var entity Entity
		if err := db.Where("character_id = ?", characterId).First(&entity).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrorProvider[Entity](ErrMemberNotFound)
			}
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(entity)
	}
}

// GetByIdProvider returns a provider for finding a family member by ID
func GetByIdProvider(id uint32) database.EntityProvider[Entity] {
	return func(db *gorm.DB) model.Provider[Entity] {
		var entity Entity
		if err := db.Where("id = ?", id).First(&entity).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrorProvider[Entity](ErrMemberNotFound)
			}
			return model.ErrorProvider[Entity](err)
		}
		return model.FixedProvider(entity)
	}
}

// GetBySeniorIdProvider returns a provider for finding all juniors of a senior
func GetBySeniorIdProvider(seniorId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		var entities []Entity
		if err := db.Where("senior_id = ?", seniorId).Find(&entities).Error; err != nil {
			return model.ErrorProvider[[]Entity](err)
		}
		return model.FixedProvider(entities)
	}
}

// GetFamilyTreeProvider returns a provider for getting a complete family tree starting from a character
func GetFamilyTreeProvider(characterId uint32) database.EntityProvider[[]Entity] {
	return func(db *gorm.DB) model.Provider[[]Entity] {
		// Get the starting member
		member, err := GetByCharacterIdProvider(characterId)(db)()
		if err != nil {
			return model.ErrorProvider[[]Entity](err)
		}

		var familyMembers []Entity
		familyMembers = append(familyMembers, member)

		// Get all family members related to this character
		// This includes: senior, juniors, and siblings (other juniors of the same senior)
		relatedIds := make(map[uint32]bool)
		relatedIds[characterId] = true

		// Add senior if exists
		if member.SeniorId != nil {
			seniorId := *member.SeniorId
			if !relatedIds[seniorId] {
				senior, err := GetByCharacterIdProvider(seniorId)(db)()
				if err == nil {
					familyMembers = append(familyMembers, senior)
					relatedIds[seniorId] = true
				}
			}
		}

		// Add juniors if exist
		if len(member.JuniorIds) > 0 {
			juniors, err := GetBySeniorIdProvider(characterId)(db)()
			if err == nil {
				for _, junior := range juniors {
					if !relatedIds[junior.CharacterId] {
						familyMembers = append(familyMembers, junior)
						relatedIds[junior.CharacterId] = true
					}
				}
			}
		}

		// Add siblings (other juniors of the same senior)
		if member.SeniorId != nil {
			siblings, err := GetBySeniorIdProvider(*member.SeniorId)(db)()
			if err == nil {
				for _, sibling := range siblings {
					if !relatedIds[sibling.CharacterId] {
						familyMembers = append(familyMembers, sibling)
						relatedIds[sibling.CharacterId] = true
					}
				}
			}
		}
		return model.FixedProvider(familyMembers)
	}
}

// ExistsProvider returns a provider for checking if a family member exists by character ID
func ExistsProvider(characterId uint32) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		return func() (bool, error) {
			var count int64
			if err := db.Model(&Entity{}).Where("character_id = ?", characterId).Count(&count).Error; err != nil {
				return false, err
			}
			return count > 0, nil
		}
	}
}
