package status

import "github.com/google/uuid"

// State represents quest status
type State byte

const (
	StateNotStarted State = 0
	StateStarted    State = 1
	StateCompleted  State = 2
)

// RestModel represents the quest status response from atlas-quest service
type RestModel struct {
	Id          string    `json:"-"`
	TenantId    uuid.UUID `json:"-"`
	CharacterId uint32    `json:"characterId"`
	QuestId     uint32    `json:"questId"`
	State       State     `json:"state"`
}

// GetName returns the JSON:API type name
func (r RestModel) GetName() string {
	return "quests"
}

// GetID returns the JSON:API resource ID
func (r RestModel) GetID() string {
	return r.Id
}

// SetID sets the JSON:API resource ID
func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// IsNotStarted returns true if the quest has not been started
func (r RestModel) IsNotStarted() bool {
	return r.State == StateNotStarted
}

// IsStarted returns true if the quest is in progress
func (r RestModel) IsStarted() bool {
	return r.State == StateStarted
}

// IsCompleted returns true if the quest has been completed
func (r RestModel) IsCompleted() bool {
	return r.State == StateCompleted
}
