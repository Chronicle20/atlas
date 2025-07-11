package family

import (
	"atlas-family/database"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetByCharacterIdProvider returns a provider for finding a family member by character ID
func GetByCharacterIdProvider(characterId uint32) database.EntityProvider[FamilyMember] {
	return func(db *gorm.DB) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			var entity Entity
			if err := db.Where("character_id = ?", characterId).First(&entity).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return FamilyMember{}, ErrMemberNotFound
				}
				return FamilyMember{}, err
			}
			return Make(entity)
		}
	}
}

// GetByIdProvider returns a provider for finding a family member by ID
func GetByIdProvider(id uint32) database.EntityProvider[FamilyMember] {
	return func(db *gorm.DB) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			var entity Entity
			if err := db.Where("id = ?", id).First(&entity).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return FamilyMember{}, ErrMemberNotFound
				}
				return FamilyMember{}, err
			}
			return Make(entity)
		}
	}
}

// GetByTenantIdProvider returns a provider for finding all family members by tenant ID
func GetByTenantIdProvider(tenantId uuid.UUID) database.EntityProvider[[]FamilyMember] {
	return func(db *gorm.DB) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			var entities []Entity
			if err := db.Where("tenant_id = ?", tenantId).Find(&entities).Error; err != nil {
				return []FamilyMember{}, err
			}
			return model.SliceMap(Make)(model.FixedProvider(entities))(model.ParallelMap())()
		}
	}
}

// GetBySeniorIdProvider returns a provider for finding all juniors of a senior
func GetBySeniorIdProvider(seniorId uint32) database.EntityProvider[[]FamilyMember] {
	return func(db *gorm.DB) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			var entities []Entity
			if err := db.Where("senior_id = ?", seniorId).Find(&entities).Error; err != nil {
				return []FamilyMember{}, err
			}
			return model.SliceMap(Make)(model.FixedProvider(entities))(model.ParallelMap())()
		}
	}
}

// GetFamilyTreeProvider returns a provider for getting a complete family tree starting from a character
func GetFamilyTreeProvider(characterId uint32) database.EntityProvider[[]FamilyMember] {
	return func(db *gorm.DB) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			// Get the starting member
			member, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return []FamilyMember{}, err
			}

			var familyMembers []FamilyMember
			familyMembers = append(familyMembers, member)

			// Get all family members related to this character
			// This includes: senior, juniors, and siblings (other juniors of the same senior)
			relatedIds := make(map[uint32]bool)
			relatedIds[characterId] = true

			// Add senior if exists
			if member.HasSenior() {
				seniorId := *member.SeniorId()
				if !relatedIds[seniorId] {
					senior, err := GetByCharacterIdProvider(seniorId)(db)()
					if err == nil {
						familyMembers = append(familyMembers, senior)
						relatedIds[seniorId] = true
					}
				}
			}

			// Add juniors if exist
			if member.HasJuniors() {
				juniors, err := GetBySeniorIdProvider(characterId)(db)()
				if err == nil {
					for _, junior := range juniors {
						if !relatedIds[junior.CharacterId()] {
							familyMembers = append(familyMembers, junior)
							relatedIds[junior.CharacterId()] = true
						}
					}
				}
			}

			// Add siblings (other juniors of the same senior)
			if member.HasSenior() {
				siblings, err := GetBySeniorIdProvider(*member.SeniorId())(db)()
				if err == nil {
					for _, sibling := range siblings {
						if !relatedIds[sibling.CharacterId()] {
							familyMembers = append(familyMembers, sibling)
							relatedIds[sibling.CharacterId()] = true
						}
					}
				}
			}

			return familyMembers, nil
		}
	}
}

// GetByWorldAndMapProvider returns a provider for finding family members on a specific world and map
func GetByWorldAndMapProvider(world byte, mapId uint32) database.EntityProvider[[]FamilyMember] {
	return func(db *gorm.DB) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			var entities []Entity
			if err := db.Where("world = ? AND map_id = ?", world, mapId).Find(&entities).Error; err != nil {
				return []FamilyMember{}, err
			}
			return model.SliceMap(Make)(model.FixedProvider(entities))(model.ParallelMap())()
		}
	}
}

// GetActiveMembers returns a provider for finding all family members with recent activity
func GetActiveMembersProvider() database.EntityProvider[[]FamilyMember] {
	return func(db *gorm.DB) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			var entities []Entity
			if err := db.Where("daily_rep > 0").Find(&entities).Error; err != nil {
				return []FamilyMember{}, err
			}
			return model.SliceMap(Make)(model.FixedProvider(entities))(model.ParallelMap())()
		}
	}
}

// GetMembersNeedingResetProvider returns a provider for finding members that need daily rep reset
func GetMembersNeedingResetProvider() database.EntityProvider[[]FamilyMember] {
	return func(db *gorm.DB) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			var entities []Entity
			if err := db.Where("daily_rep > 0").Find(&entities).Error; err != nil {
				return []FamilyMember{}, err
			}
			return model.SliceMap(Make)(model.FixedProvider(entities))(model.ParallelMap())()
		}
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

// CreateProvider returns a provider for creating a new family member
func CreateProvider(member FamilyMember) database.EntityProvider[FamilyMember] {
	return func(db *gorm.DB) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			entity := ToEntity(member)
			if err := db.Create(&entity).Error; err != nil {
				return FamilyMember{}, err
			}
			return Make(entity)
		}
	}
}

// UpdateProvider returns a provider for updating an existing family member
func UpdateProvider(member FamilyMember) database.EntityProvider[FamilyMember] {
	return func(db *gorm.DB) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			entity := ToEntity(member)
			if err := db.Save(&entity).Error; err != nil {
				return FamilyMember{}, err
			}
			return Make(entity)
		}
	}
}

// DeleteProvider returns a provider for deleting a family member by character ID
func DeleteProvider(characterId uint32) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		return func() (bool, error) {
			result := db.Where("character_id = ?", characterId).Delete(&Entity{})
			if result.Error != nil {
				return false, result.Error
			}
			return result.RowsAffected > 0, nil
		}
	}
}