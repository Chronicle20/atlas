package marriage

import (
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// CreateProposal creates a new proposal in the database
func CreateProposal(db *gorm.DB, log logrus.FieldLogger) func(proposerId, targetId uint32, tenantId uuid.UUID) model.Provider[ProposalEntity] {
	return func(proposerId, targetId uint32, tenantId uuid.UUID) model.Provider[ProposalEntity] {
		return func() (ProposalEntity, error) {
			log.WithFields(logrus.Fields{
				"proposerId": proposerId,
				"targetId":   targetId,
				"tenantId":   tenantId,
			}).Debug("Creating proposal entity")

			// Create new proposal entity
			now := time.Now()
			entity := ProposalEntity{
				ProposerId:     proposerId,
				TargetId:       targetId,
				Status:         ProposalStatusPending,
				ProposedAt:     now,
				ExpiresAt:      now.Add(ProposalExpiryDuration),
				RejectionCount: 0,
				TenantId:       tenantId,
				CreatedAt:      now,
				UpdatedAt:      now,
			}

			if err := db.Create(&entity).Error; err != nil {
				return ProposalEntity{}, err
			}

			return entity, nil
		}
	}
}

// UpdateProposal updates an existing proposal in the database
func UpdateProposal(db *gorm.DB, log logrus.FieldLogger) func(proposal Proposal) model.Provider[ProposalEntity] {
	return func(proposal Proposal) model.Provider[ProposalEntity] {
		return func() (ProposalEntity, error) {
			log.WithField("proposalId", proposal.Id()).Debug("Updating proposal entity")

			entity := proposal.ToProposalEntity()
			if err := db.Save(&entity).Error; err != nil {
				return ProposalEntity{}, err
			}

			return entity, nil
		}
	}
}

// CreateMarriage creates a new marriage in the database
func CreateMarriage(db *gorm.DB, log logrus.FieldLogger) func(characterId1, characterId2 uint32, tenantId uuid.UUID) model.Provider[Entity] {
	return func(characterId1, characterId2 uint32, tenantId uuid.UUID) model.Provider[Entity] {
		return func() (Entity, error) {
			log.WithFields(logrus.Fields{
				"characterId1": characterId1,
				"characterId2": characterId2,
				"tenantId":     tenantId,
			}).Debug("Creating marriage entity")

			// Create new marriage entity
			now := time.Now()
			entity := Entity{
				CharacterId1: characterId1,
				CharacterId2: characterId2,
				Status:       StatusProposed,
				ProposedAt:   now,
				TenantId:     tenantId,
				CreatedAt:    now,
				UpdatedAt:    now,
			}

			if err := db.Create(&entity).Error; err != nil {
				return Entity{}, err
			}

			return entity, nil
		}
	}
}

// UpdateMarriage updates an existing marriage in the database
func UpdateMarriage(db *gorm.DB, log logrus.FieldLogger) func(marriage Marriage) model.Provider[Entity] {
	return func(marriage Marriage) model.Provider[Entity] {
		return func() (Entity, error) {
			log.WithField("marriageId", marriage.Id()).Debug("Updating marriage entity")

			entity := marriage.ToEntity()
			if err := db.Save(&entity).Error; err != nil {
				return Entity{}, err
			}

			return entity, nil
		}
	}
}

// CreateCeremony creates a new ceremony in the database
func CreateCeremony(db *gorm.DB, log logrus.FieldLogger) func(marriageId, characterId1, characterId2 uint32, tenantId uuid.UUID) model.Provider[CeremonyEntity] {
	return func(marriageId, characterId1, characterId2 uint32, tenantId uuid.UUID) model.Provider[CeremonyEntity] {
		return func() (CeremonyEntity, error) {
			log.WithFields(logrus.Fields{
				"marriageId":   marriageId,
				"characterId1": characterId1,
				"characterId2": characterId2,
				"tenantId":     tenantId,
			}).Debug("Creating ceremony entity")

			// Create new ceremony entity
			now := time.Now()
			inviteesJSON, err := inviteesToJSON([]uint32{})
			if err != nil {
				return CeremonyEntity{}, err
			}

			entity := CeremonyEntity{
				MarriageId:   marriageId,
				CharacterId1: characterId1,
				CharacterId2: characterId2,
				Status:       CeremonyStatusScheduled,
				ScheduledAt:  now,
				Invitees:     inviteesJSON,
				TenantId:     tenantId,
				CreatedAt:    now,
				UpdatedAt:    now,
			}

			if err := db.Create(&entity).Error; err != nil {
				return CeremonyEntity{}, err
			}

			return entity, nil
		}
	}
}

// UpdateCeremony updates an existing ceremony in the database
func UpdateCeremony(db *gorm.DB, log logrus.FieldLogger) func(ceremony Ceremony) model.Provider[CeremonyEntity] {
	return func(ceremony Ceremony) model.Provider[CeremonyEntity] {
		return func() (CeremonyEntity, error) {
			log.WithField("ceremonyId", ceremony.Id()).Debug("Updating ceremony entity")

			entity, err := ceremony.ToCeremonyEntity()
			if err != nil {
				return CeremonyEntity{}, err
			}

			if err := db.Save(&entity).Error; err != nil {
				return CeremonyEntity{}, err
			}

			return entity, nil
		}
	}
}