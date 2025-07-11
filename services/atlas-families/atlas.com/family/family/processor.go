package family

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor interface defines the core business logic operations
type Processor interface {
	AddJunior(db *gorm.DB, log logrus.FieldLogger) func(seniorId uint32, juniorId uint32) model.Provider[FamilyMember]
	RemoveMember(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Provider[[]FamilyMember]
	BreakLink(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Provider[[]FamilyMember]
	AwardRep(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, amount uint32, source string) model.Provider[FamilyMember]
	DeductRep(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, amount uint32, reason string) model.Provider[FamilyMember]
	ResetDailyRep(db *gorm.DB, log logrus.FieldLogger) func() model.Provider[int64]
	UpdateLocation(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, world byte, channel byte, mapId uint32) model.Provider[FamilyMember]
	UpdateLevel(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, level uint16) model.Provider[FamilyMember]
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
func (p *ProcessorImpl) AddJunior(db *gorm.DB, log logrus.FieldLogger) func(seniorId uint32, juniorId uint32) model.Provider[FamilyMember] {
	return func(seniorId uint32, juniorId uint32) model.Provider[FamilyMember] {
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
			seniorModel, err := GetByCharacterIdProvider(seniorId)(db)()
			if err != nil {
				if errors.Is(err, ErrMemberNotFound) {
					return FamilyMember{}, ErrSeniorNotFound
				}
				return FamilyMember{}, err
			}

			// Check if senior can add more juniors
			if !seniorModel.CanAddJunior() {
				return FamilyMember{}, ErrSeniorHasTooManyJuniors
			}

			// Get junior member
			juniorModel, err := GetByCharacterIdProvider(juniorId)(db)()
			if err != nil {
				if errors.Is(err, ErrMemberNotFound) {
					return FamilyMember{}, ErrJuniorNotFound
				}
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
			var result FamilyMember
			err = db.Transaction(func(tx *gorm.DB) error {
				// Update senior - add junior
				updatedSenior, err := seniorModel.Builder().
					AddJunior(juniorId).
					Touch().
					Build()
				if err != nil {
					return err
				}

				if _, err := UpdateProvider(updatedSenior)(tx)(); err != nil {
					return err
				}

				// Update junior - set senior
				updatedJunior, err := juniorModel.Builder().
					SetSeniorId(seniorId).
					Touch().
					Build()
				if err != nil {
					return err
				}

				if _, err := UpdateProvider(updatedJunior)(tx)(); err != nil {
					return err
				}

				result = updatedSenior
				return nil
			})
			
			return result, err
		}
	}
}

// RemoveMember removes a member from the family and handles cascade operations
func (p *ProcessorImpl) RemoveMember(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
	return func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"reason":      reason,
			}).Info("Removing member from family")

			// Get the member to remove
			memberModel, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return []FamilyMember{}, err
			}

			var updatedMembers []FamilyMember
			err = db.Transaction(func(tx *gorm.DB) error {
				// If member has a senior, remove from senior's junior list
				if memberModel.HasSenior() {
					if seniorModel, err := GetByCharacterIdProvider(*memberModel.SeniorId())(tx)(); err == nil {
						updatedSenior, err := seniorModel.Builder().
							RemoveJunior(characterId).
							Touch().
							Build()
						if err != nil {
							return err
						}

						if _, err := UpdateProvider(updatedSenior)(tx)(); err != nil {
							return err
						}
						updatedMembers = append(updatedMembers, updatedSenior)
					}
				}

				// If member has juniors, remove their senior reference
				if memberModel.HasJuniors() {
					for _, juniorId := range memberModel.JuniorIds() {
						if juniorModel, err := GetByCharacterIdProvider(juniorId)(tx)(); err == nil {
							updatedJunior, err := juniorModel.Builder().
								ClearSeniorId().
								Touch().
								Build()
							if err != nil {
								return err
							}

							if _, err := UpdateProvider(updatedJunior)(tx)(); err != nil {
								return err
							}
							updatedMembers = append(updatedMembers, updatedJunior)
						}
					}
				}

				// Remove the member
				if _, err := DeleteProvider(characterId)(tx)(); err != nil {
					return err
				}

				return nil
			})
			
			return updatedMembers, err
		}
	}
}

// BreakLink breaks the family link for a character
func (p *ProcessorImpl) BreakLink(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
	return func(characterId uint32, reason string) model.Provider[[]FamilyMember] {
		return func() ([]FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"reason":      reason,
			}).Info("Breaking family link")

			// Get the member
			memberModel, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return []FamilyMember{}, err
			}

			// Check if there's a link to break
			if !memberModel.HasSenior() && !memberModel.HasJuniors() {
				return []FamilyMember{}, ErrNoLinkToBreak
			}

			var updatedMembers []FamilyMember
			err = db.Transaction(func(tx *gorm.DB) error {
				// If member has a senior, remove from senior's junior list
				if memberModel.HasSenior() {
					var seniorEntity Entity
					if err := tx.Where("character_id = ?", *memberModel.SeniorId()).First(&seniorEntity).Error; err == nil {
						seniorModel, err := Make(seniorEntity)
						if err == nil {
							updatedSenior, err := seniorModel.Builder().
								RemoveJunior(characterId).
								Touch().
								Build()
							if err != nil {
								return err
							}

							seniorEntity = ToEntity(updatedSenior)
							if err := tx.Save(&seniorEntity).Error; err != nil {
								return err
							}
							updatedMembers = append(updatedMembers, updatedSenior)
						}
					}

					// Clear member's senior reference
					updatedMember, err := memberModel.Builder().
						ClearSeniorId().
						Touch().
						Build()
					if err != nil {
						return err
					}

					if _, err := UpdateProvider(updatedMember)(tx)(); err != nil {
						return err
					}
					updatedMembers = append(updatedMembers, updatedMember)
				}

				// If member has juniors, clear their senior reference
				if memberModel.HasJuniors() {
					for _, juniorId := range memberModel.JuniorIds() {
						if juniorModel, err := GetByCharacterIdProvider(juniorId)(tx)(); err == nil {
							updatedJunior, err := juniorModel.Builder().
								ClearSeniorId().
								Touch().
								Build()
							if err != nil {
								return err
							}

							if _, err := UpdateProvider(updatedJunior)(tx)(); err != nil {
								return err
							}
							updatedMembers = append(updatedMembers, updatedJunior)
						}
					}

					// Clear member's junior list
					updatedMember, err := memberModel.Builder().
						SetJuniorIds([]uint32{}).
						Touch().
						Build()
					if err != nil {
						return err
					}

					if _, err := UpdateProvider(updatedMember)(tx)(); err != nil {
						return err
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

				return nil
			})
			
			return updatedMembers, err
		}
	}
}

// AwardRep awards reputation to a character
func (p *ProcessorImpl) AwardRep(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, amount uint32, source string) model.Provider[FamilyMember] {
	return func(characterId uint32, amount uint32, source string) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"amount":      amount,
				"source":      source,
			}).Info("Awarding reputation")

			// Get the member
			memberModel, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return FamilyMember{}, err
			}

			// Check if member can receive more rep today
			if !memberModel.CanReceiveRep(amount) {
				return FamilyMember{}, ErrRepCapExceeded
			}

			// Update member with new rep
			updatedMember, err := memberModel.Builder().
				AddRep(amount).
				AddDailyRep(amount).
				Touch().
				Build()
			if err != nil {
				return FamilyMember{}, err
			}

			if _, err := UpdateProvider(updatedMember)(db)(); err != nil {
				return FamilyMember{}, err
			}

			return updatedMember, nil
		}
	}
}

// DeductRep deducts reputation from a character
func (p *ProcessorImpl) DeductRep(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, amount uint32, reason string) model.Provider[FamilyMember] {
	return func(characterId uint32, amount uint32, reason string) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"amount":      amount,
				"reason":      reason,
			}).Info("Deducting reputation")

			// Get the member
			memberModel, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return FamilyMember{}, err
			}

			// Check if member has enough rep
			if memberModel.Rep() < amount {
				return FamilyMember{}, ErrInsufficientRep
			}

			// Update member with deducted rep
			updatedMember, err := memberModel.Builder().
				SubtractRep(amount).
				Touch().
				Build()
			if err != nil {
				return FamilyMember{}, err
			}

			if _, err := UpdateProvider(updatedMember)(db)(); err != nil {
				return FamilyMember{}, err
			}

			return updatedMember, nil
		}
	}
}

// ResetDailyRep resets daily reputation for all members
func (p *ProcessorImpl) ResetDailyRep(db *gorm.DB, log logrus.FieldLogger) func() model.Provider[int64] {
	return func() model.Provider[int64] {
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
func (p *ProcessorImpl) UpdateLocation(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, world byte, channel byte, mapId uint32) model.Provider[FamilyMember] {
	return func(characterId uint32, world byte, channel byte, mapId uint32) model.Provider[FamilyMember] {
		return func() (FamilyMember, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"world":       world,
				"channel":     channel,
				"mapId":       mapId,
			}).Info("Updating member location")

			// Get the member
			memberModel, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return FamilyMember{}, err
			}

			// Update member location
			updatedMember, err := memberModel.Builder().
				SetWorld(world).
				SetChannel(channel).
				SetMapId(mapId).
				Touch().
				Build()
			if err != nil {
				return FamilyMember{}, err
			}

			if _, err := UpdateProvider(updatedMember)(db)(); err != nil {
				return FamilyMember{}, err
			}

			return updatedMember, nil
		}
	}
}

// UpdateLevel updates a member's level
func (p *ProcessorImpl) UpdateLevel(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, level uint16) model.Provider[FamilyMember] {
	return func(characterId uint32, level uint16) model.Provider[FamilyMember] {
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
			memberModel, err := GetByCharacterIdProvider(characterId)(db)()
			if err != nil {
				return FamilyMember{}, err
			}

			// Update member level
			updatedMember, err := memberModel.Builder().
				SetLevel(level).
				Touch().
				Build()
			if err != nil {
				return FamilyMember{}, err
			}

			if _, err := UpdateProvider(updatedMember)(db)(); err != nil {
				return FamilyMember{}, err
			}

			return updatedMember, nil
		}
	}
}