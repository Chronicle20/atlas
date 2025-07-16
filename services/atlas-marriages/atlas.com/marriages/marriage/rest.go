package marriage

import (
	"strconv"
	"time"
)

// RestMarriage represents the REST API model for marriage responses
type RestMarriage struct {
	ID               uint32           `json:"id"`
	CharacterId1     uint32           `json:"characterId1"`
	CharacterId2     uint32           `json:"characterId2"`
	Status           string           `json:"status"`
	ProposedAt       time.Time        `json:"proposedAt"`
	EngagedAt        *time.Time       `json:"engagedAt,omitempty"`
	MarriedAt        *time.Time       `json:"marriedAt,omitempty"`
	DivorcedAt       *time.Time       `json:"divorcedAt,omitempty"`
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`
	Partner          *RestPartner     `json:"partner,omitempty"`
	Ceremony         *RestCeremony    `json:"ceremony,omitempty"`
}

// RestPartner represents partner information in marriage response
type RestPartner struct {
	CharacterID uint32 `json:"characterId"`
	// Note: Character name and other details would be populated by external service calls
}

// RestCeremony represents ceremony information in marriage response
type RestCeremony struct {
	ID           uint32      `json:"id"`
	Status       string      `json:"status"`
	ScheduledAt  time.Time   `json:"scheduledAt"`
	StartedAt    *time.Time  `json:"startedAt,omitempty"`
	CompletedAt  *time.Time  `json:"completedAt,omitempty"`
	CancelledAt  *time.Time  `json:"cancelledAt,omitempty"`
	PostponedAt  *time.Time  `json:"postponedAt,omitempty"`
	InviteeCount int         `json:"inviteeCount"`
}

// RestProposal represents a proposal in REST API responses
type RestProposal struct {
	ID             uint32     `json:"id"`
	ProposerID     uint32     `json:"proposerId"`
	TargetID       uint32     `json:"targetId"`
	Status         string     `json:"status"`
	ProposedAt     time.Time  `json:"proposedAt"`
	RespondedAt    *time.Time `json:"respondedAt,omitempty"`
	ExpiresAt      time.Time  `json:"expiresAt"`
	RejectionCount uint32     `json:"rejectionCount"`
	CooldownUntil  *time.Time `json:"cooldownUntil,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// GetType returns the JSON:API resource type for marriage
func (rm RestMarriage) GetType() string {
	return "marriage"
}

// GetID returns the JSON:API resource ID for marriage
func (rm RestMarriage) GetID() string {
	return strconv.Itoa(int(rm.ID))
}

// GetType returns the JSON:API resource type for proposal
func (rp RestProposal) GetType() string {
	return "proposal"
}

// GetID returns the JSON:API resource ID for proposal
func (rp RestProposal) GetID() string {
	return strconv.Itoa(int(rp.ID))
}

// TransformMarriage converts a domain Marriage model to REST representation
func TransformMarriage(m Marriage) (RestMarriage, error) {
	return RestMarriage{
		ID:           m.Id(),
		CharacterId1: m.CharacterId1(),
		CharacterId2: m.CharacterId2(),
		Status:       m.Status().String(),
		ProposedAt:   m.ProposedAt(),
		EngagedAt:    m.EngagedAt(),
		MarriedAt:    m.MarriedAt(),
		DivorcedAt:   m.DivorcedAt(),
		CreatedAt:    m.CreatedAt(),
		UpdatedAt:    m.UpdatedAt(),
	}, nil
}

// TransformMarriageWithPartner converts a domain Marriage model to REST representation with partner info
func TransformMarriageWithPartner(m Marriage, characterId uint32) (RestMarriage, error) {
	marriage, err := TransformMarriage(m)
	if err != nil {
		return RestMarriage{}, err
	}

	// Add partner information
	partnerId, hasPartner := m.GetPartner(characterId)
	if hasPartner {
		marriage.Partner = &RestPartner{
			CharacterID: partnerId,
		}
	}

	return marriage, nil
}

// TransformMarriageWithCeremony converts a domain Marriage model to REST representation with ceremony info
func TransformMarriageWithCeremony(m Marriage, ceremony *Ceremony) (RestMarriage, error) {
	marriage, err := TransformMarriage(m)
	if err != nil {
		return RestMarriage{}, err
	}

	// Add ceremony information if available
	if ceremony != nil {
		marriage.Ceremony = &RestCeremony{
			ID:           ceremony.Id(),
			Status:       ceremony.Status().String(),
			ScheduledAt:  ceremony.ScheduledAt(),
			StartedAt:    ceremony.StartedAt(),
			CompletedAt:  ceremony.CompletedAt(),
			CancelledAt:  ceremony.CancelledAt(),
			PostponedAt:  ceremony.PostponedAt(),
			InviteeCount: ceremony.InviteeCount(),
		}
	}

	return marriage, nil
}

// TransformMarriageComplete converts a domain Marriage model to complete REST representation
func TransformMarriageComplete(m Marriage, characterId uint32, ceremony *Ceremony) (RestMarriage, error) {
	marriage, err := TransformMarriage(m)
	if err != nil {
		return RestMarriage{}, err
	}

	// Add partner information
	partnerId, hasPartner := m.GetPartner(characterId)
	if hasPartner {
		marriage.Partner = &RestPartner{
			CharacterID: partnerId,
		}
	}

	// Add ceremony information if available
	if ceremony != nil {
		marriage.Ceremony = &RestCeremony{
			ID:           ceremony.Id(),
			Status:       ceremony.Status().String(),
			ScheduledAt:  ceremony.ScheduledAt(),
			StartedAt:    ceremony.StartedAt(),
			CompletedAt:  ceremony.CompletedAt(),
			CancelledAt:  ceremony.CancelledAt(),
			PostponedAt:  ceremony.PostponedAt(),
			InviteeCount: ceremony.InviteeCount(),
		}
	}

	return marriage, nil
}

// TransformProposal converts a domain Proposal model to REST representation
func TransformProposal(p Proposal) (RestProposal, error) {
	return RestProposal{
		ID:             p.Id(),
		ProposerID:     p.ProposerId(),
		TargetID:       p.TargetId(),
		Status:         p.Status().String(),
		ProposedAt:     p.ProposedAt(),
		RespondedAt:    p.RespondedAt(),
		ExpiresAt:      p.ExpiresAt(),
		RejectionCount: p.RejectionCount(),
		CooldownUntil:  p.CooldownUntil(),
		CreatedAt:      p.CreatedAt(),
		UpdatedAt:      p.UpdatedAt(),
	}, nil
}

// TransformProposals converts a slice of domain Proposal models to REST representation
func TransformProposals(proposals []Proposal) ([]RestProposal, error) {
	restProposals := make([]RestProposal, 0, len(proposals))
	
	for _, proposal := range proposals {
		restProposal, err := TransformProposal(proposal)
		if err != nil {
			return nil, err
		}
		restProposals = append(restProposals, restProposal)
	}
	
	return restProposals, nil
}

// TransformMarriages converts a slice of domain Marriage models to REST representation
func TransformMarriages(marriages []Marriage) ([]RestMarriage, error) {
	restMarriages := make([]RestMarriage, 0, len(marriages))
	
	for _, marriage := range marriages {
		restMarriage, err := TransformMarriage(marriage)
		if err != nil {
			return nil, err
		}
		restMarriages = append(restMarriages, restMarriage)
	}
	
	return restMarriages, nil
}