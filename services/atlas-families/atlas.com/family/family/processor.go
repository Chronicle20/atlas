package family

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor interface defines the core business logic operations
type Processor interface {
	AddJunior(db *gorm.DB, log logrus.FieldLogger) func(seniorId uint32, juniorId uint32) model.Operator[FamilyMember]
	RemoveMember(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Operator[[]FamilyMember]
	BreakLink(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Operator[[]FamilyMember]
	AwardRep(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, amount uint32, source string) model.Operator[FamilyMember]
	DeductRep(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, amount uint32, reason string) model.Operator[FamilyMember]
	ResetDailyRep(db *gorm.DB, log logrus.FieldLogger) func() model.Operator[int64]
	UpdateLocation(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, world byte, channel byte, mapId uint32) model.Operator[FamilyMember]
	UpdateLevel(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, level uint16) model.Operator[FamilyMember]
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct{}

// NewProcessor creates a new processor instance
func NewProcessor() Processor {
	return &ProcessorImpl{}
}

// Business logic errors
var (
	ErrMemberNotFound         = errors.New("family member not found")
	ErrSeniorNotFound         = errors.New("senior member not found")
	ErrJuniorNotFound         = errors.New("junior member not found")
	ErrSeniorHasTooManyJuniors = errors.New("senior already has maximum number of juniors")
	ErrJuniorAlreadyLinked    = errors.New("junior is already linked to a senior")
	ErrLevelDifferenceTooLarge = errors.New("level difference exceeds maximum allowed")
	ErrNotOnSameMap           = errors.New("members must be on the same map to link")
	ErrInsufficientRep        = errors.New("insufficient reputation for operation")
	ErrRepCapExceeded         = errors.New("daily reputation cap exceeded")
	ErrCannotRemoveSelf       = errors.New("cannot remove self from family")
	ErrNoLinkToBreak          = errors.New("no family link exists to break")
)

// AddJunior adds a junior to a senior's family
func (p *ProcessorImpl) AddJunior(db *gorm.DB, log logrus.FieldLogger) func(seniorId uint32, juniorId uint32) model.Operator[FamilyMember] {
	return func(seniorId uint32, juniorId uint32) model.Operator[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"seniorId": seniorId,
				"juniorId": juniorId,
			}).Info("Adding junior to family")

			// Validate input
			if seniorId == juniorId {
				return FamilyMember{}, ErrSelfReference
			}

			// Get senior member
			var seniorEntity Entity
			if err := db.Where("character_id = ?", seniorId).First(&seniorEntity).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return FamilyMember{}, ErrSeniorNotFound
				}
				return FamilyMember{}, err
			}

			seniorModel, err := Make(seniorEntity)
			if err != nil {
				return FamilyMember{}, err
			}

			// Check if senior can add more juniors
			if !seniorModel.CanAddJunior() {
				return FamilyMember{}, ErrSeniorHasTooManyJuniors
			}

			// Get junior member
			var juniorEntity Entity
			if err := db.Where("character_id = ?", juniorId).First(&juniorEntity).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return FamilyMember{}, ErrJuniorNotFound
				}
				return FamilyMember{}, err
			}

			juniorModel, err := Make(juniorEntity)
			if err != nil {
				return FamilyMember{}, err
			}

			// Check if junior already has a senior
			if juniorModel.HasSenior() {
				return FamilyMember{}, ErrJuniorAlreadyLinked
			}

			// Validate level difference
			if !seniorModel.ValidateLevelDifference(juniorModel.Level()) {
				return FamilyMember{}, ErrLevelDifferenceTooLarge
			}

			// Validate same location
			if !seniorModel.IsSameLocation(juniorModel.World(), juniorModel.MapId()) {
				return FamilyMember{}, ErrNotOnSameMap
			}

			// Begin transaction
			return db.Transaction(func(tx *gorm.DB) (FamilyMember, error) {
				// Update senior - add junior
				updatedSenior := seniorModel.Builder().
					AddJunior(juniorId).
					Touch().
					Build()

				seniorEntity = ToEntity(updatedSenior)
				if err := tx.Save(&seniorEntity).Error; err != nil {
					return FamilyMember{}, err
				}

				// Update junior - set senior
				updatedJunior := juniorModel.Builder().
					SetSeniorId(seniorId).
					Touch().
					Build()

				juniorEntity = ToEntity(updatedJunior)
				if err := tx.Save(&juniorEntity).Error; err != nil {
					return FamilyMember{}, err
				}

				return updatedSenior, nil
			})
		}
	}
}

// RemoveMember removes a member from the family and handles cascade operations
func (p *ProcessorImpl) RemoveMember(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Operator[[]FamilyMember] {
	return func(characterId uint32, reason string) model.Operator[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"reason":      reason,
			}).Info("Removing member from family")

			// Get the member to remove
			var memberEntity Entity
			if err := db.Where("character_id = ?", characterId).First(&memberEntity).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return []FamilyMember{}, ErrMemberNotFound
				}
				return []FamilyMember{}, err
			}

			memberModel, err := Make(memberEntity)
			if err != nil {
				return []FamilyMember{}, err
			}

			return db.Transaction(func(tx *gorm.DB) ([]FamilyMember, error) {
				var updatedMembers []FamilyMember

				// If member has a senior, remove from senior's junior list
				if memberModel.HasSenior() {
					var seniorEntity Entity
					if err := tx.Where("character_id = ?", *memberModel.SeniorId()).First(&seniorEntity).Error; err == nil {
						seniorModel, err := Make(seniorEntity)
						if err == nil {
							updatedSenior := seniorModel.Builder().
								RemoveJunior(characterId).
								Touch().
								Build()

							seniorEntity = ToEntity(updatedSenior)
							if err := tx.Save(&seniorEntity).Error; err != nil {
								return []FamilyMember{}, err
							}
							updatedMembers = append(updatedMembers, updatedSenior)
						}
					}
				}

				// If member has juniors, remove their senior reference
				if memberModel.HasJuniors() {
					var juniorEntities []Entity
					if err := tx.Where("character_id IN ?", memberModel.JuniorIds()).Find(&juniorEntities).Error; err == nil {
						for _, juniorEntity := range juniorEntities {
							juniorModel, err := Make(juniorEntity)
							if err == nil {
								updatedJunior := juniorModel.Builder().
									ClearSeniorId().
									Touch().
									Build()

								juniorEntity = ToEntity(updatedJunior)
								if err := tx.Save(&juniorEntity).Error; err != nil {
									return []FamilyMember{}, err
								}
								updatedMembers = append(updatedMembers, updatedJunior)
							}
						}
					}
				}

				// Remove the member
				if err := tx.Delete(&memberEntity).Error; err != nil {
					return []FamilyMember{}, err
				}

				return updatedMembers, nil
			})
		}
	}
}

// BreakLink breaks the family link for a character
func (p *ProcessorImpl) BreakLink(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Operator[[]FamilyMember] {
	return func(characterId uint32, reason string) model.Operator[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"reason":      reason,
			}).Info("Breaking family link")

			// Get the member
			var memberEntity Entity
			if err := db.Where("character_id = ?", characterId).First(&memberEntity).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return []FamilyMember{}, ErrMemberNotFound
				}
				return []FamilyMember{}, err
			}

			memberModel, err := Make(memberEntity)
			if err != nil {
				return []FamilyMember{}, err
			}

			// Check if there's a link to break
			if !memberModel.HasSenior() && !memberModel.HasJuniors() {
				return []FamilyMember{}, ErrNoLinkToBreak
			}

			return db.Transaction(func(tx *gorm.DB) ([]FamilyMember, error) {
				var updatedMembers []FamilyMember

				// If member has a senior, remove from senior's junior list
				if memberModel.HasSenior() {
					var seniorEntity Entity
					if err := tx.Where("character_id = ?", *memberModel.SeniorId()).First(&seniorEntity).Error; err == nil {
						seniorModel, err := Make(seniorEntity)
						if err == nil {
							updatedSenior := seniorModel.Builder().
								RemoveJunior(characterId).
								Touch().
								Build()

							seniorEntity = ToEntity(updatedSenior)
							if err := tx.Save(&seniorEntity).Error; err != nil {
								return []FamilyMember{}, err
							}
							updatedMembers = append(updatedMembers, updatedSenior)
						}
					}

					// Clear member's senior reference
					updatedMember := memberModel.Builder().
						ClearSeniorId().
						Touch().
						Build()

					memberEntity = ToEntity(updatedMember)
					if err := tx.Save(&memberEntity).Error; err != nil {
						return []FamilyMember{}, err
					}
					updatedMembers = append(updatedMembers, updatedMember)
				}

				// If member has juniors, clear their senior reference
				if memberModel.HasJuniors() {
					var juniorEntities []Entity
					if err := tx.Where("character_id IN ?", memberModel.JuniorIds()).Find(&juniorEntities).Error; err == nil {
						for _, juniorEntity := range juniorEntities {
							juniorModel, err := Make(juniorEntity)
							if err == nil {
								updatedJunior := juniorModel.Builder().
									ClearSeniorId().
									Touch().
									Build()

								juniorEntity = ToEntity(updatedJunior)
								if err := tx.Save(&juniorEntity).Error; err != nil {
									return []FamilyMember{}, err
								}
								updatedMembers = append(updatedMembers, updatedJunior)
							}
						}
					}

					// Clear member's junior list
					updatedMember := memberModel.Builder().
						SetJuniorIds([]uint32{}).
						Touch().
						Build()

					memberEntity = ToEntity(updatedMember)
					if err := tx.Save(&memberEntity).Error; err != nil {
						return []FamilyMember{}, err
					}

					// Update the member in the result
					for i, member := range updatedMembers {
						if member.CharacterId() == characterId {
							updatedMembers[i] = updatedMember
							break
						}
					}
					if len(updatedMembers) == 0 || updatedMembers[len(updatedMembers)-1].CharacterId() != characterId {
						updatedMembers = append(updatedMembers, updatedMember)
					}
				}

				return updatedMembers, nil
			})
		}
	}
}

// AwardRep awards reputation to a character
func (p *ProcessorImpl) AwardRep(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, amount uint32, source string) model.Operator[FamilyMember] {
	return func(characterId uint32, amount uint32, source string) model.Operator[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"amount":      amount,
				"source":      source,
			}).Info("Awarding reputation")

			// Get the member
			var memberEntity Entity
			if err := db.Where("character_id = ?", characterId).First(&memberEntity).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return FamilyMember{}, ErrMemberNotFound
				}
				return FamilyMember{}, err
			}

			memberModel, err := Make(memberEntity)
			if err != nil {
				return FamilyMember{}, err
			}

			// Check if member can receive more rep today
			if !memberModel.CanReceiveRep(amount) {
				return FamilyMember{}, ErrRepCapExceeded
			}

			// Update member with new rep
			updatedMember := memberModel.Builder().
				AddRep(amount).
				AddDailyRep(amount).
				Touch().
				Build()

			memberEntity = ToEntity(updatedMember)
			if err := db.Save(&memberEntity).Error; err != nil {
				return FamilyMember{}, err
			}

			return updatedMember, nil
		}
	}
}

// DeductRep deducts reputation from a character
func (p *ProcessorImpl) DeductRep(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, amount uint32, reason string) model.Operator[FamilyMember] {
	return func(characterId uint32, amount uint32, reason string) model.Operator[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"amount":      amount,
				"reason":      reason,
			}).Info("Deducting reputation")

			// Get the member
			var memberEntity Entity
			if err := db.Where("character_id = ?", characterId).First(&memberEntity).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return FamilyMember{}, ErrMemberNotFound
				}
				return FamilyMember{}, err
			}

			memberModel, err := Make(memberEntity)
			if err != nil {
				return FamilyMember{}, err
			}

			// Check if member has enough rep
			if memberModel.Rep() < amount {
				return FamilyMember{}, ErrInsufficientRep
			}

			// Update member with deducted rep
			updatedMember := memberModel.Builder().
				SubtractRep(amount).
				Touch().
				Build()

			memberEntity = ToEntity(updatedMember)
			if err := db.Save(&memberEntity).Error; err != nil {
				return FamilyMember{}, err
			}

			return updatedMember, nil
		}
	}
}

// ResetDailyRep resets daily reputation for all members
func (p *ProcessorImpl) ResetDailyRep(db *gorm.DB, log logrus.FieldLogger) func() model.Operator[int64] {
	return func() model.Operator[int64] {
		return func() (int64, error) {
			log.Info("Resetting daily reputation for all members")

			result := db.Model(&Entity{}).
				Where("daily_rep > 0").
				Updates(map[string]interface{}{
					"daily_rep":  0,
					"updated_at": time.Now(),
				})

			if result.Error != nil {
				return 0, result.Error
			}

			return result.RowsAffected, nil
		}
	}
}

// UpdateLocation updates a member's location information
func (p *ProcessorImpl) UpdateLocation(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, world byte, channel byte, mapId uint32) model.Operator[FamilyMember] {
	return func(characterId uint32, world byte, channel byte, mapId uint32) model.Operator[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"world":       world,
				"channel":     channel,
				"mapId":       mapId,
			}).Info("Updating member location")

			// Get the member
			var memberEntity Entity
			if err := db.Where("character_id = ?", characterId).First(&memberEntity).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return FamilyMember{}, ErrMemberNotFound
				}
				return FamilyMember{}, err
			}

			memberModel, err := Make(memberEntity)
			if err != nil {
				return FamilyMember{}, err
			}

			// Update member location
			updatedMember := memberModel.Builder().
				SetWorld(world).
				SetChannel(channel).
				SetMapId(mapId).
				Touch().
				Build()

			memberEntity = ToEntity(updatedMember)
			if err := db.Save(&memberEntity).Error; err != nil {
				return FamilyMember{}, err
			}

			return updatedMember, nil
		}
	}
}

// UpdateLevel updates a member's level
func (p *ProcessorImpl) UpdateLevel(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, level uint16) model.Operator[FamilyMember] {
	return func(characterId uint32, level uint16) model.Operator[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"level":       level,
			}).Info("Updating member level")

			// Validate level
			if err := ValidateLevel(level); err != nil {
				return FamilyMember{}, err
			}

			// Get the member
			var memberEntity Entity
			if err := db.Where("character_id = ?", characterId).First(&memberEntity).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return FamilyMember{}, ErrMemberNotFound
				}
				return FamilyMember{}, err
			}

			memberModel, err := Make(memberEntity)
			if err != nil {
				return FamilyMember{}, err
			}

			// Update member level
			updatedMember := memberModel.Builder().
				SetLevel(level).
				Touch().
				Build()

			memberEntity = ToEntity(updatedMember)
			if err := db.Save(&memberEntity).Error; err != nil {
				return FamilyMember{}, err
			}

			return updatedMember, nil
		}
	}
}