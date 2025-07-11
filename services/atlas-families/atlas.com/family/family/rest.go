package family

import (
	"strconv"
	"time"

	"github.com/google/uuid"
)

// RestFamilyMember represents a family member in REST/JSON:API format
type RestFamilyMember struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	CharacterId uint32   `json:"characterId"`
	TenantId    string   `json:"tenantId"`
	SeniorId    *uint32  `json:"seniorId,omitempty"`
	JuniorIds   []uint32 `json:"juniorIds"`
	Rep         uint32   `json:"rep"`
	DailyRep    uint32   `json:"dailyRep"`
	Level       uint16   `json:"level"`
	World       byte     `json:"world"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

// GetID returns the ID for JSON:API compatibility
func (r RestFamilyMember) GetID() string {
	return r.ID
}

// GetType returns the type for JSON:API compatibility
func (r RestFamilyMember) GetType() string {
	return "familyMembers"
}

// RestFamilyTree represents a complete family tree in REST format
type RestFamilyTree struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"`
	Members []RestFamilyMember `json:"members"`
}

// GetID returns the ID for JSON:API compatibility
func (r RestFamilyTree) GetID() string {
	return r.ID
}

// GetType returns the type for JSON:API compatibility
func (r RestFamilyTree) GetType() string {
	return "familyTrees"
}

// RestReputation represents reputation information in REST format
type RestReputation struct {
	ID               string `json:"id"`
	Type             string `json:"type"`
	CharacterId      uint32 `json:"characterId"`
	AvailableRep     uint32 `json:"availableRep"`
	DailyRep         uint32 `json:"dailyRep"`
	DailyRepLimit    uint32 `json:"dailyRepLimit"`
	RemainingDailyRep uint32 `json:"remainingDailyRep"`
}

// GetID returns the ID for JSON:API compatibility
func (r RestReputation) GetID() string {
	return r.ID
}

// GetType returns the type for JSON:API compatibility
func (r RestReputation) GetType() string {
	return "reputation"
}

// RestLocation represents family member location in REST format
type RestLocation struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	CharacterId uint32 `json:"characterId"`
	World       byte   `json:"world"`
	Channel     *byte  `json:"channel,omitempty"`
	Map         uint32 `json:"map"`
	Online      bool   `json:"online"`
}

// GetID returns the ID for JSON:API compatibility
func (r RestLocation) GetID() string {
	return r.ID
}

// GetType returns the type for JSON:API compatibility
func (r RestLocation) GetType() string {
	return "locations"
}

// Transform converts a domain FamilyMember to REST representation
func Transform(fm FamilyMember) (RestFamilyMember, error) {
	// Copy junior IDs to avoid shared references
	juniorIds := make([]uint32, len(fm.JuniorIds()))
	copy(juniorIds, fm.JuniorIds())

	return RestFamilyMember{
		ID:          strconv.FormatUint(uint64(fm.Id()), 10),
		Type:        "familyMembers",
		CharacterId: fm.CharacterId(),
		TenantId:    fm.TenantId().String(),
		SeniorId:    fm.SeniorId(),
		JuniorIds:   juniorIds,
		Rep:         fm.Rep(),
		DailyRep:    fm.DailyRep(),
		Level:       fm.Level(),
		World:       fm.World(),
		CreatedAt:   fm.CreatedAt().Format(time.RFC3339),
		UpdatedAt:   fm.UpdatedAt().Format(time.RFC3339),
	}, nil
}

// Extract converts a REST FamilyMember back to domain model
func Extract(r RestFamilyMember) (FamilyMember, error) {
	// Parse ID
	id, err := strconv.ParseUint(r.ID, 10, 32)
	if err != nil {
		return FamilyMember{}, err
	}

	// Parse TenantId
	tenantId, err := uuid.Parse(r.TenantId)
	if err != nil {
		return FamilyMember{}, err
	}

	// Parse timestamps
	createdAt, err := time.Parse(time.RFC3339, r.CreatedAt)
	if err != nil {
		return FamilyMember{}, err
	}

	updatedAt, err := time.Parse(time.RFC3339, r.UpdatedAt)
	if err != nil {
		return FamilyMember{}, err
	}

	// Copy junior IDs to avoid shared references
	juniorIds := make([]uint32, len(r.JuniorIds))
	copy(juniorIds, r.JuniorIds)

	// Use the builder pattern to create the domain model
	builder := NewBuilder(r.CharacterId, tenantId, r.Level, r.World).
		SetId(uint32(id)).
		SetRep(r.Rep).
		SetDailyRep(r.DailyRep).
		SetCreatedAt(createdAt).
		SetUpdatedAt(updatedAt)

	// Set senior ID if present
	if r.SeniorId != nil {
		builder = builder.SetSeniorId(*r.SeniorId)
	}

	// Set junior IDs
	for _, juniorId := range juniorIds {
		builder = builder.AddJunior(juniorId)
	}

	return builder.Build()
}

// TransformTree converts a slice of domain FamilyMembers to REST family tree
func TransformTree(characterId uint32, members []FamilyMember) (RestFamilyTree, error) {
	restMembers := make([]RestFamilyMember, 0, len(members))
	
	for _, member := range members {
		restMember, err := Transform(member)
		if err != nil {
			return RestFamilyTree{}, err
		}
		restMembers = append(restMembers, restMember)
	}

	return RestFamilyTree{
		ID:      strconv.FormatUint(uint64(characterId), 10),
		Type:    "familyTrees",
		Members: restMembers,
	}, nil
}

// TransformFamilyTree is an alias for TransformTree to match resource expectations
func TransformFamilyTree(members []FamilyMember) (RestFamilyTree, error) {
	if len(members) == 0 {
		return RestFamilyTree{
			ID:      "0",
			Type:    "familyTrees",
			Members: []RestFamilyMember{},
		}, nil
	}
	
	// Use the first member's character ID as the tree ID
	return TransformTree(members[0].CharacterId(), members)
}

// TransformReputation converts a domain FamilyMember to REST reputation
func TransformReputation(fm FamilyMember) (RestReputation, error) {
	const dailyRepLimit = 5000
	remainingDailyRep := dailyRepLimit - fm.DailyRep()
	if remainingDailyRep < 0 {
		remainingDailyRep = 0
	}

	return RestReputation{
		ID:                strconv.FormatUint(uint64(fm.CharacterId()), 10),
		Type:              "reputation",
		CharacterId:       fm.CharacterId(),
		AvailableRep:      fm.Rep(),
		DailyRep:          fm.DailyRep(),
		DailyRepLimit:     dailyRepLimit,
		RemainingDailyRep: remainingDailyRep,
	}, nil
}

// TransformLocation converts a domain FamilyMember to REST location
func TransformLocation(fm FamilyMember, channel *byte, mapId uint32, online bool) (RestLocation, error) {
	return RestLocation{
		ID:          strconv.FormatUint(uint64(fm.CharacterId()), 10),
		Type:        "locations",
		CharacterId: fm.CharacterId(),
		World:       fm.World(),
		Channel:     channel,
		Map:         mapId,
		Online:      online,
	}, nil
}

// Request structures for JSON:API format

// AddJuniorRequest represents the request body for adding a junior
type AddJuniorRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			JuniorId uint32 `json:"juniorId" validate:"required"`
		} `json:"attributes"`
	} `json:"data"`
}

// BreakLinkRequest represents the request body for breaking a family link
type BreakLinkRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			Reason string `json:"reason,omitempty"`
		} `json:"attributes"`
	} `json:"data"`
}

// DeductRepRequest represents the request body for deducting reputation
type DeductRepRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			CharacterId uint32 `json:"characterId" validate:"required"`
			Amount      uint32 `json:"amount" validate:"required,min=1"`
			Reason      string `json:"reason" validate:"required"`
		} `json:"attributes"`
	} `json:"data"`
}

// ActivityRequest represents the request body for registering activity
type ActivityRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			CharacterId  uint32 `json:"characterId" validate:"required"`
			ActivityType string `json:"activityType" validate:"required,oneof=mob_kill expedition"`
			Amount       uint32 `json:"amount" validate:"required,min=1"`
		} `json:"attributes"`
	} `json:"data"`
}

// Note: These REST models are compatible with JSON:API standards but don't implement
// specific resource interfaces since the project uses api2go/jsonapi directly.