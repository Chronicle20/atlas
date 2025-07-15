package marriage

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// GetProposalByIdProvider retrieves a proposal by ID
func GetProposalByIdProvider(db *gorm.DB, log logrus.FieldLogger) func(proposalId uint32, tenantId uuid.UUID) model.Provider[Proposal] {
	return func(proposalId uint32, tenantId uuid.UUID) model.Provider[Proposal] {
		return func() (Proposal, error) {
			log.WithFields(logrus.Fields{
				"proposalId": proposalId,
				"tenantId":   tenantId,
			}).Debug("Retrieving proposal by ID")

			var entity ProposalEntity
			err := db.Where("id = ? AND tenant_id = ?", proposalId, tenantId).First(&entity).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return Proposal{}, errors.New("proposal not found")
				}
				return Proposal{}, err
			}

			return MakeProposal(entity)
		}
	}
}

// GetActiveProposalProvider retrieves an active proposal between two characters
func GetActiveProposalProvider(db *gorm.DB, log logrus.FieldLogger) func(proposerId, targetId uint32, tenantId uuid.UUID) model.Provider[*Proposal] {
	return func(proposerId, targetId uint32, tenantId uuid.UUID) model.Provider[*Proposal] {
		return func() (*Proposal, error) {
			log.WithFields(logrus.Fields{
				"proposerId": proposerId,
				"targetId":   targetId,
				"tenantId":   tenantId,
			}).Debug("Retrieving active proposal")

			var entity ProposalEntity
			err := db.Where("proposer_id = ? AND target_id = ? AND tenant_id = ? AND status = ?",
				proposerId, targetId, tenantId, ProposalStatusPending).
				First(&entity).Error

			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, nil
				}
				return nil, err
			}

			proposal, err := MakeProposal(entity)
			if err != nil {
				return nil, err
			}

			// Check if proposal has expired
			if proposal.IsExpired() {
				return nil, nil
			}

			return &proposal, nil
		}
	}
}

// GetPendingProposalsByCharacterProvider retrieves all pending proposals for a character
func GetPendingProposalsByCharacterProvider(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, tenantId uuid.UUID) model.Provider[[]Proposal] {
	return func(characterId uint32, tenantId uuid.UUID) model.Provider[[]Proposal] {
		return func() ([]Proposal, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"tenantId":    tenantId,
			}).Debug("Retrieving pending proposals for character")

			var entities []ProposalEntity
			err := db.Where("(proposer_id = ? OR target_id = ?) AND tenant_id = ? AND status = ?",
				characterId, characterId, tenantId, ProposalStatusPending).
				Order("created_at DESC").
				Find(&entities).Error

			if err != nil {
				return nil, err
			}

			proposals := make([]Proposal, 0, len(entities))
			for _, entity := range entities {
				proposal, err := MakeProposal(entity)
				if err != nil {
					return nil, err
				}

				// Only include non-expired proposals
				if !proposal.IsExpired() {
					proposals = append(proposals, proposal)
				}
			}

			return proposals, nil
		}
	}
}

// GetProposalHistoryProvider retrieves the history of proposals between two characters
func GetProposalHistoryProvider(db *gorm.DB, log logrus.FieldLogger) func(proposerId, targetId uint32, tenantId uuid.UUID) model.Provider[[]Proposal] {
	return func(proposerId, targetId uint32, tenantId uuid.UUID) model.Provider[[]Proposal] {
		return func() ([]Proposal, error) {
			log.WithFields(logrus.Fields{
				"proposerId": proposerId,
				"targetId":   targetId,
				"tenantId":   tenantId,
			}).Debug("Retrieving proposal history")

			var entities []ProposalEntity
			err := db.Where("proposer_id = ? AND target_id = ? AND tenant_id = ?",
				proposerId, targetId, tenantId).
				Order("created_at DESC").
				Find(&entities).Error

			if err != nil {
				return nil, err
			}

			proposals := make([]Proposal, 0, len(entities))
			for _, entity := range entities {
				proposal, err := MakeProposal(entity)
				if err != nil {
					return nil, err
				}
				proposals = append(proposals, proposal)
			}

			return proposals, nil
		}
	}
}

// GetLastProposalByProposerProvider retrieves the last proposal made by a proposer
func GetLastProposalByProposerProvider(db *gorm.DB, log logrus.FieldLogger) func(proposerId uint32, tenantId uuid.UUID) model.Provider[*Proposal] {
	return func(proposerId uint32, tenantId uuid.UUID) model.Provider[*Proposal] {
		return func() (*Proposal, error) {
			log.WithFields(logrus.Fields{
				"proposerId": proposerId,
				"tenantId":   tenantId,
			}).Debug("Retrieving last proposal by proposer")

			var entity ProposalEntity
			err := db.Where("proposer_id = ? AND tenant_id = ?", proposerId, tenantId).
				Order("created_at DESC").
				First(&entity).Error

			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, nil
				}
				return nil, err
			}

			proposal, err := MakeProposal(entity)
			if err != nil {
				return nil, err
			}

			return &proposal, nil
		}
	}
}

// GetLastProposalToTargetProvider retrieves the last proposal made to a specific target
func GetLastProposalToTargetProvider(db *gorm.DB, log logrus.FieldLogger) func(proposerId, targetId uint32, tenantId uuid.UUID) model.Provider[*Proposal] {
	return func(proposerId, targetId uint32, tenantId uuid.UUID) model.Provider[*Proposal] {
		return func() (*Proposal, error) {
			log.WithFields(logrus.Fields{
				"proposerId": proposerId,
				"targetId":   targetId,
				"tenantId":   tenantId,
			}).Debug("Retrieving last proposal to target")

			var entity ProposalEntity
			err := db.Where("proposer_id = ? AND target_id = ? AND tenant_id = ?", proposerId, targetId, tenantId).
				Order("created_at DESC").
				First(&entity).Error

			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, nil
				}
				return nil, err
			}

			proposal, err := MakeProposal(entity)
			if err != nil {
				return nil, err
			}

			return &proposal, nil
		}
	}
}

// GetActiveMarriageByCharacterProvider retrieves active marriage for a character
func GetActiveMarriageByCharacterProvider(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, tenantId uuid.UUID) model.Provider[*Marriage] {
	return func(characterId uint32, tenantId uuid.UUID) model.Provider[*Marriage] {
		return func() (*Marriage, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"tenantId":    tenantId,
			}).Debug("Retrieving active marriage for character")

			var entity Entity
			err := db.Where("(character_id1 = ? OR character_id2 = ?) AND tenant_id = ? AND status IN (?)",
				characterId, characterId, tenantId, []MarriageStatus{StatusProposed, StatusEngaged, StatusMarried}).
				First(&entity).Error

			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, nil
				}
				return nil, err
			}

			marriage, err := Make(entity)
			if err != nil {
				return nil, err
			}

			return &marriage, nil
		}
	}
}

// GetMarriageByIdProvider retrieves a marriage by ID
func GetMarriageByIdProvider(db *gorm.DB, log logrus.FieldLogger) func(marriageId uint32, tenantId uuid.UUID) model.Provider[Marriage] {
	return func(marriageId uint32, tenantId uuid.UUID) model.Provider[Marriage] {
		return func() (Marriage, error) {
			log.WithFields(logrus.Fields{
				"marriageId": marriageId,
				"tenantId":   tenantId,
			}).Debug("Retrieving marriage by ID")

			var entity Entity
			err := db.Where("id = ? AND tenant_id = ?", marriageId, tenantId).First(&entity).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return Marriage{}, errors.New("marriage not found")
				}
				return Marriage{}, err
			}

			return Make(entity)
		}
	}
}

// GetMarriageHistoryByCharacterProvider retrieves marriage history for a character
func GetMarriageHistoryByCharacterProvider(db *gorm.DB, log logrus.FieldLogger) func(characterId uint32, tenantId uuid.UUID) model.Provider[[]Marriage] {
	return func(characterId uint32, tenantId uuid.UUID) model.Provider[[]Marriage] {
		return func() ([]Marriage, error) {
			log.WithFields(logrus.Fields{
				"characterId": characterId,
				"tenantId":    tenantId,
			}).Debug("Retrieving marriage history for character")

			var entities []Entity
			err := db.Where("(character_id1 = ? OR character_id2 = ?) AND tenant_id = ?",
				characterId, characterId, tenantId).
				Order("created_at DESC").
				Find(&entities).Error

			if err != nil {
				return nil, err
			}

			marriages := make([]Marriage, 0, len(entities))
			for _, entity := range entities {
				marriage, err := Make(entity)
				if err != nil {
					return nil, err
				}
				marriages = append(marriages, marriage)
			}

			return marriages, nil
		}
	}
}

// GetCeremonyByIdProvider retrieves a ceremony by ID
func GetCeremonyByIdProvider(db *gorm.DB, log logrus.FieldLogger) func(ceremonyId uint32, tenantId uuid.UUID) model.Provider[Ceremony] {
	return func(ceremonyId uint32, tenantId uuid.UUID) model.Provider[Ceremony] {
		return func() (Ceremony, error) {
			log.WithFields(logrus.Fields{
				"ceremonyId": ceremonyId,
				"tenantId":   tenantId,
			}).Debug("Retrieving ceremony by ID")

			var entity CeremonyEntity
			err := db.Where("id = ? AND tenant_id = ?", ceremonyId, tenantId).First(&entity).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return Ceremony{}, errors.New("ceremony not found")
				}
				return Ceremony{}, err
			}

			return MakeCeremony(entity)
		}
	}
}

// GetCeremonyByMarriageIdProvider retrieves a ceremony by marriage ID
func GetCeremonyByMarriageIdProvider(db *gorm.DB, log logrus.FieldLogger) func(marriageId uint32, tenantId uuid.UUID) model.Provider[*Ceremony] {
	return func(marriageId uint32, tenantId uuid.UUID) model.Provider[*Ceremony] {
		return func() (*Ceremony, error) {
			log.WithFields(logrus.Fields{
				"marriageId": marriageId,
				"tenantId":   tenantId,
			}).Debug("Retrieving ceremony by marriage ID")

			var entity CeremonyEntity
			err := db.Where("marriage_id = ? AND tenant_id = ?", marriageId, tenantId).First(&entity).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, nil
				}
				return nil, err
			}

			ceremony, err := MakeCeremony(entity)
			if err != nil {
				return nil, err
			}

			return &ceremony, nil
		}
	}
}

// CheckGlobalCooldownProvider checks if a character is in global cooldown
func CheckGlobalCooldownProvider(db *gorm.DB, log logrus.FieldLogger) func(proposerId uint32, tenantId uuid.UUID) model.Provider[bool] {
	return func(proposerId uint32, tenantId uuid.UUID) model.Provider[bool] {
		return func() (bool, error) {
			log.WithFields(logrus.Fields{
				"proposerId": proposerId,
				"tenantId":   tenantId,
			}).Debug("Checking global cooldown")

			lastProposalProvider := GetLastProposalByProposerProvider(db, log)(proposerId, tenantId)
			lastProposal, err := lastProposalProvider()
			if err != nil {
				return false, err
			}

			if lastProposal == nil {
				return true, nil // No previous proposals
			}

			// Check if global cooldown period has passed
			cooldownEnd := lastProposal.CreatedAt().Add(GlobalCooldownDuration)
			return time.Now().After(cooldownEnd), nil
		}
	}
}

// CheckPerTargetCooldownProvider checks if a character is in per-target cooldown
func CheckPerTargetCooldownProvider(db *gorm.DB, log logrus.FieldLogger) func(proposerId, targetId uint32, tenantId uuid.UUID) model.Provider[bool] {
	return func(proposerId, targetId uint32, tenantId uuid.UUID) model.Provider[bool] {
		return func() (bool, error) {
			log.WithFields(logrus.Fields{
				"proposerId": proposerId,
				"targetId":   targetId,
				"tenantId":   tenantId,
			}).Debug("Checking per-target cooldown")

			lastProposalProvider := GetLastProposalToTargetProvider(db, log)(proposerId, targetId, tenantId)
			lastProposal, err := lastProposalProvider()
			if err != nil {
				return false, err
			}

			if lastProposal == nil {
				return true, nil // No previous proposals to this target
			}

			// If last proposal was rejected, check cooldown
			if lastProposal.Status() == ProposalStatusRejected && lastProposal.CooldownUntil() != nil {
				return time.Now().After(*lastProposal.CooldownUntil()), nil
			}

			// If last proposal expired, apply initial cooldown
			if lastProposal.Status() == ProposalStatusExpired {
				cooldownEnd := lastProposal.UpdatedAt().Add(InitialPerTargetCooldown)
				return time.Now().After(cooldownEnd), nil
			}

			return true, nil
		}
	}
}