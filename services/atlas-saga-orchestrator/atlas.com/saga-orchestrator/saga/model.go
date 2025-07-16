package saga

import (
	"atlas-saga-orchestrator/validation"
	"encoding/json"
	"fmt"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"time"
)

// Type the type of saga
type Type string

// Constants for different saga types
const (
	InventoryTransaction Type = "inventory_transaction"
	QuestReward          Type = "quest_reward"
	TradeTransaction     Type = "trade_transaction"
)

// Saga represents the entire saga transaction.
type Saga struct {
	TransactionId uuid.UUID   `json:"transactionId"` // Unique ID for the transaction
	SagaType      Type        `json:"sagaType"`      // Type of the saga (e.g., inventory_transaction)
	InitiatedBy   string      `json:"initiatedBy"`   // Who initiated the saga (e.g., NPC ID, user)
	Steps         []Step[any] `json:"steps"`         // List of steps in the saga
}

func (s *Saga) Failing() bool {
	for _, step := range s.Steps {
		if step.Status == Failed {
			return true
		}
	}
	return false
}

func (s *Saga) GetCurrentStep() (Step[any], bool) {
	for idx, step := range s.Steps {
		if step.Status == Pending {
			return s.Steps[idx], true
		}
	}
	return Step[any]{}, false
}

// FindFurthestCompletedStepIndex returns the index of the furthest completed step (last one with status "completed")
// Returns -1 if no completed step is found
func (s *Saga) FindFurthestCompletedStepIndex() int {
	furthestCompletedIndex := -1
	for i := len(s.Steps) - 1; i >= 0; i-- {
		if s.Steps[i].Status == Completed {
			furthestCompletedIndex = i
			break
		}
	}
	return furthestCompletedIndex
}

// FindEarliestPendingStepIndex returns the index of the earliest pending step (first one with status "pending")
// Returns -1 if no pending step is found
func (s *Saga) FindEarliestPendingStepIndex() int {
	earliestPendingIndex := -1
	for i := 0; i < len(s.Steps); i++ {
		if s.Steps[i].Status == Pending {
			earliestPendingIndex = i
			break
		}
	}
	return earliestPendingIndex
}

// SetStepStatus sets the status of a step at the given index
func (s *Saga) SetStepStatus(index int, status Status) {
	if index >= 0 && index < len(s.Steps) {
		s.Steps[index].Status = status
	}
}

// FindFailedStepIndex returns the index of the first failed step
// Returns -1 if no failed step is found
func (s *Saga) FindFailedStepIndex() int {
	for i := 0; i < len(s.Steps); i++ {
		if s.Steps[i].Status == Failed {
			return i
		}
	}
	return -1
}

type Status string

const (
	Pending   Status = "pending"
	Completed Status = "completed"
	Failed    Status = "failed"
)

// Define a custom type for Action
type Action string

// Constants for different actions
const (
	AwardInventory             Action = "award_inventory" // Deprecated: Use AwardAsset instead
	AwardAsset                 Action = "award_asset"     // Preferred over AwardInventory
	AwardExperience            Action = "award_experience"
	AwardLevel                 Action = "award_level"
	AwardMesos                 Action = "award_mesos"
	WarpToRandomPortal         Action = "warp_to_random_portal"
	WarpToPortal               Action = "warp_to_portal"
	DestroyAsset               Action = "destroy_asset"
	EquipAsset                 Action = "equip_asset"
	UnequipAsset               Action = "unequip_asset"
	ChangeJob                  Action = "change_job"
	CreateSkill                Action = "create_skill"
	UpdateSkill                Action = "update_skill"
	ValidateCharacterState     Action = "validate_character_state"
	RequestGuildName           Action = "request_guild_name"
	RequestGuildEmblem         Action = "request_guild_emblem"
	RequestGuildDisband        Action = "request_guild_disband"
	RequestGuildCapacityIncrease Action = "request_guild_capacity_increase"
	CreateInvite               Action = "create_invite"
	CreateCharacter            Action = "create_character"
)

// Step represents a single step within a saga.
type Step[T any] struct {
	StepId    string    `json:"stepId"`    // Unique ID for the step
	Status    Status    `json:"status"`    // Status of the step (e.g., pending, completed, failed)
	Action    Action    `json:"action"`    // The Action to be taken (e.g., validate_inventory, deduct_inventory)
	Payload   T         `json:"payload"`   // Data required for the action (specific to the action type)
	CreatedAt time.Time `json:"createdAt"` // Timestamp of when the step was created
	UpdatedAt time.Time `json:"updatedAt"` // Timestamp of the last update to the step
}

// AwardItemActionPayload represents the data needed to execute a specific action in a step.
type AwardItemActionPayload struct {
	CharacterId uint32      `json:"characterId"` // CharacterId associated with the action
	Item        ItemPayload `json:"item"`        // List of items involved in the action
}

// ItemPayload represents an individual item in a transaction, such as in inventory manipulation.
type ItemPayload struct {
	TemplateId uint32 `json:"templateId"` // TemplateId of the item
	Quantity   uint32 `json:"quantity"`   // Quantity of the item
}

// WarpToRandomPortalPayload represents the payload required to warp to a random portal within a specific field.
type WarpToRandomPortalPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId associated with the action
	FieldId     field.Id `json:"fieldId"`     // FieldId references the unique identifier of the field associated with the warp action.
}

// WarpToPortalPayload represents the payload required to warp a character to a specific portal in a field.
type WarpToPortalPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId associated with the action
	FieldId     field.Id `json:"fieldId"`     // FieldId references the unique identifier of the field associated with the warp action.
	PortalId    uint32   `json:"portalId"`    // PortalId specifies the unique identifier of the portal for the warp action.
}

// AwardExperiencePayload represents the payload required to award experience to a character.
type AwardExperiencePayload struct {
	CharacterId   uint32                    `json:"characterId"`   // CharacterId associated with the action
	WorldId       world.Id                  `json:"worldId"`       // WorldId associated with the action
	ChannelId     channel.Id                `json:"channelId"`     // ChannelId associated with the action
	Distributions []ExperienceDistributions `json:"distributions"` // List of experience distributions to award
}

// AwardLevelPayload represents the payload required to award levels to a character.
type AwardLevelPayload struct {
	CharacterId uint32      `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id    `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id  `json:"channelId"`   // ChannelId associated with the action
	Amount      byte        `json:"amount"`      // Number of levels to award
}

// AwardMesosPayload represents the payload required to award mesos to a character.
type AwardMesosPayload struct {
	CharacterId uint32      `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id    `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id  `json:"channelId"`   // ChannelId associated with the action
	ActorId     uint32      `json:"actorId"`     // ActorId identifies who is giving/taking the mesos
	ActorType   string      `json:"actorType"`   // ActorType identifies the type of actor (e.g., "SYSTEM", "NPC", "CHARACTER")
	Amount      int32       `json:"amount"`      // Amount of mesos to award (can be negative for deduction)
}

// DestroyAssetPayload represents the payload required to destroy an asset in a compartment.
type DestroyAssetPayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId associated with the action
	TemplateId  uint32 `json:"templateId"`  // TemplateId of the item to destroy
	Quantity    uint32 `json:"quantity"`    // Quantity of the item to destroy
}

// EquipAssetPayload represents the payload required to equip an asset from one inventory slot to an equipped slot.
type EquipAssetPayload struct {
	CharacterId   uint32 `json:"characterId"`   // CharacterId associated with the action
	InventoryType uint32 `json:"inventoryType"` // Type of inventory (e.g., equipment, consumables)
	Source        int16  `json:"source"`        // Source inventory slot (standard inventory slot)
	Destination   int16  `json:"destination"`   // Destination equipped slot (negative values for equipped slots)
}

// UnequipAssetPayload represents the payload required to unequip an asset from an equipped slot back to a standard inventory slot.
type UnequipAssetPayload struct {
	CharacterId   uint32 `json:"characterId"`   // CharacterId associated with the action
	InventoryType uint32 `json:"inventoryType"` // Type of inventory (e.g., equipment, consumables)
	Source        int16  `json:"source"`        // Source equipped slot (negative values for equipped slots)
	Destination   int16  `json:"destination"`   // Destination inventory slot (standard inventory slot)
}

// ChangeJobPayload represents the payload required to change a character's job.
type ChangeJobPayload struct {
	CharacterId uint32      `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id    `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id  `json:"channelId"`   // ChannelId associated with the action
	JobId       job.Id      `json:"jobId"`       // JobId to change to
}

// CreateSkillPayload represents the payload required to create a skill for a character.
type CreateSkillPayload struct {
	CharacterId  uint32    `json:"characterId"`  // CharacterId associated with the action
	SkillId      uint32    `json:"skillId"`      // SkillId to create
	Level        byte      `json:"level"`        // Skill level
	MasterLevel  byte      `json:"masterLevel"`  // Skill master level
	Expiration   time.Time `json:"expiration"`   // Skill expiration time
}

// UpdateSkillPayload represents the payload required to update a skill for a character.
type UpdateSkillPayload struct {
	CharacterId  uint32    `json:"characterId"`  // CharacterId associated with the action
	SkillId      uint32    `json:"skillId"`      // SkillId to update
	Level        byte      `json:"level"`        // New skill level
	MasterLevel  byte      `json:"masterLevel"`  // New skill master level
	Expiration   time.Time `json:"expiration"`   // New skill expiration time
}

// ValidateCharacterStatePayload represents the payload required to validate a character's state.
type ValidateCharacterStatePayload struct {
	CharacterId uint32                  `json:"characterId"` // CharacterId associated with the action
	Conditions  []validation.ConditionInput `json:"conditions"`  // Conditions to validate
}

// RequestGuildNamePayload represents the payload required to request a guild name.
type RequestGuildNamePayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId associated with the action
	WorldId     byte   `json:"worldId"`     // WorldId associated with the action
	ChannelId   byte   `json:"channelId"`   // ChannelId associated with the action
}

// RequestGuildEmblemPayload represents the payload required to request a guild emblem change.
type RequestGuildEmblemPayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId associated with the action
	WorldId     byte   `json:"worldId"`     // WorldId associated with the action
	ChannelId   byte   `json:"channelId"`   // ChannelId associated with the action
}

// RequestGuildDisbandPayload represents the payload required to request a guild disband.
type RequestGuildDisbandPayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId associated with the action
	WorldId     byte   `json:"worldId"`     // WorldId associated with the action
	ChannelId   byte   `json:"channelId"`   // ChannelId associated with the action
}

// RequestGuildCapacityIncreasePayload represents the payload required to request a guild capacity increase.
type RequestGuildCapacityIncreasePayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId associated with the action
	WorldId     byte   `json:"worldId"`     // WorldId associated with the action
	ChannelId   byte   `json:"channelId"`   // ChannelId associated with the action
}

// CreateInvitePayload represents the payload required to create an invitation.
type CreateInvitePayload struct {
	InviteType   string `json:"inviteType"`   // Type of invitation (e.g., "GUILD", "PARTY", "BUDDY")
	OriginatorId uint32 `json:"originatorId"` // ID of the character sending the invitation
	TargetId     uint32 `json:"targetId"`     // ID of the character receiving the invitation
	ReferenceId  uint32 `json:"referenceId"`  // ID of the entity being invited to (e.g., guild ID, party ID)
	WorldId      byte   `json:"worldId"`      // WorldId associated with the action
}

// CharacterCreatePayload represents the payload required to create a character.
// Note: this does not include any character attributes, as those are determined by the character service.
type CharacterCreatePayload struct {
	AccountId uint32 `json:"accountId"` // AccountId associated with the action
	Name      string `json:"name"`      // Name of the character to create
	WorldId   byte   `json:"worldId"`   // WorldId associated with the action
	ChannelId byte   `json:"channelId"` // ChannelId associated with the action
	JobId     uint32 `json:"jobId"`     // JobId to create the character with
	Face      uint32 `json:"face"`      // Face of the character
	Hair      uint32 `json:"hair"`      // Hair of the character
	HairColor uint32 `json:"hairColor"` // HairColor of the character
	Skin      uint32 `json:"skin"`      // Skin of the character
	Top       uint32 `json:"top"`       // Top of the character
	Bottom    uint32 `json:"bottom"`    // Bottom of the character
	Shoes     uint32 `json:"shoes"`     // Shoes of the character
	Weapon    uint32 `json:"weapon"`    // Weapon of the character
}

type ExperienceDistributions struct {
	ExperienceType string `json:"experienceType"`
	Amount         uint32 `json:"amount"`
	Attr1          uint32 `json:"attr1"`
}

// Custom UnmarshalJSON for Step[T] to handle the generics
func (s *Step[T]) UnmarshalJSON(data []byte) error {
	type Alias Step[T] // Alias to avoid recursion
	aux := &struct {
		Payload json.RawMessage `json:"payload"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	// Unmarshal the generic part (excluding Payload first)
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Now handle the Payload field based on the Action type (you can customize this)
	switch s.Action {
	case AwardInventory, AwardAsset: // Handle both action types the same way
		var payload AwardItemActionPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case AwardExperience:
		var payload AwardExperiencePayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case AwardLevel:
		var payload AwardLevelPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case AwardMesos:
		var payload AwardMesosPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case WarpToRandomPortal:
		var payload WarpToRandomPortalPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case WarpToPortal:
		var payload WarpToPortalPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case DestroyAsset:
		var payload DestroyAssetPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case EquipAsset:
		var payload EquipAssetPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case UnequipAsset:
		var payload UnequipAssetPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case ChangeJob:
		var payload ChangeJobPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case CreateSkill:
		var payload CreateSkillPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case UpdateSkill:
		var payload UpdateSkillPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case CreateInvite:
		var payload CreateInvitePayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case CreateCharacter:
		var payload CharacterCreatePayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	default:
		return fmt.Errorf("unknown action: %s", s.Action)
	}

	return nil
}
