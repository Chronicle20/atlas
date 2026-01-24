package saga

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
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
	AwardInventory         Action = "award_inventory"
	AwardExperience        Action = "award_experience"
	AwardLevel             Action = "award_level"
	AwardMesos             Action = "award_mesos"
	WarpToRandomPortal     Action = "warp_to_random_portal"
	WarpToPortal           Action = "warp_to_portal"
	DestroyAsset           Action = "destroy_asset"
	ChangeJob              Action = "change_job"
	CreateSkill            Action = "create_skill"
	UpdateSkill            Action = "update_skill"
	ValidateCharacterState Action = "validate_character_state"
	IncreaseBuddyCapacity  Action = "increase_buddy_capacity"
	GainCloseness          Action = "gain_closeness"
	ChangeHair             Action = "change_hair"
	ChangeFace             Action = "change_face"
	ChangeSkin             Action = "change_skin"
	SpawnMonster           Action = "spawn_monster"
	CompleteQuest          Action = "complete_quest"
	StartQuest             Action = "start_quest"
	SetQuestProgress       Action = "set_quest_progress"
	ApplyConsumableEffect  Action = "apply_consumable_effect"
	SendMessage            Action = "send_message"
	AwardFame              Action = "award_fame"
	ShowStorage            Action = "show_storage"
	SpawnReactorDrops      Action = "spawn_reactor_drops"

	// Portal-specific actions
	PlayPortalSound Action = "play_portal_sound"
	UpdateAreaInfo  Action = "update_area_info"
	ShowInfo        Action = "show_info"
	ShowInfoText    Action = "show_info_text"
	ShowHint        Action = "show_hint"
	BlockPortal     Action = "block_portal"
	UnblockPortal   Action = "unblock_portal"
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
	TemplateId uint32    `json:"templateId"`           // TemplateId of the item
	Quantity   uint32    `json:"quantity"`             // Quantity of the item
	Expiration time.Time `json:"expiration,omitempty"` // Expiration time for the item (zero value = no expiration)
}

// WarpToRandomPortalPayload represents the payload required to warp to a random portal within a specific field.
type WarpToRandomPortalPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId associated with the action
	FieldId     field.Id `json:"fieldId"`     // FieldId references the unique identifier of the field associated with the warp action.
}

// WarpToPortalPayload represents the payload required to warp a character to a specific portal in a field.
type WarpToPortalPayload struct {
	CharacterId uint32     `json:"characterId"`          // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`              // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`            // ChannelId associated with the action
	MapId       _map.Id    `json:"mapId"`                // MapId specifies the map to warp to
	PortalId    uint32     `json:"portalId"`             // PortalId specifies the unique identifier of the portal for the warp action.
	PortalName  string     `json:"portalName,omitempty"` // PortalName specifies the name of the portal (resolved to ID if provided).
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
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Amount      byte       `json:"amount"`      // Number of levels to award
}

// AwardMesosPayload represents the payload required to award mesos to a character.
type AwardMesosPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	ActorId     uint32     `json:"actorId"`     // ActorId identifies who is giving/taking the mesos
	ActorType   string     `json:"actorType"`   // ActorType identifies the type of actor (e.g., "SYSTEM", "NPC", "CHARACTER")
	Amount      int32      `json:"amount"`      // Amount of mesos to award (can be negative for deduction)
}

// DestroyAssetPayload represents the payload required to destroy an asset in a compartment.
type DestroyAssetPayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId associated with the action
	TemplateId  uint32 `json:"templateId"`  // TemplateId of the item to destroy
	Quantity    uint32 `json:"quantity"`    // Quantity of the item to destroy (ignored if RemoveAll is true)
	RemoveAll   bool   `json:"removeAll"`   // If true, remove all instances of the item regardless of Quantity
}

// ChangeJobPayload represents the payload required to change a character's job.
type ChangeJobPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	JobId       job.Id     `json:"jobId"`       // JobId to change to
}

// CreateSkillPayload represents the payload required to create a skill for a character.
type CreateSkillPayload struct {
	CharacterId uint32    `json:"characterId"` // CharacterId associated with the action
	SkillId     uint32    `json:"skillId"`     // SkillId to create
	Level       byte      `json:"level"`       // Skill level
	MasterLevel byte      `json:"masterLevel"` // Skill master level
	Expiration  time.Time `json:"expiration"`  // Skill expiration time
}

// UpdateSkillPayload represents the payload required to update a skill for a character.
type UpdateSkillPayload struct {
	CharacterId uint32    `json:"characterId"` // CharacterId associated with the action
	SkillId     uint32    `json:"skillId"`     // SkillId to update
	Level       byte      `json:"level"`       // New skill level
	MasterLevel byte      `json:"masterLevel"` // New skill master level
	Expiration  time.Time `json:"expiration"`  // New skill expiration time
}

// IncreaseBuddyCapacityPayload represents the payload required to increase a character's buddy list capacity.
type IncreaseBuddyCapacityPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Amount      byte       `json:"amount"`      // Amount to increase buddy capacity by
}

// GainClosenessPayload represents the payload required to gain closeness with a pet.
type GainClosenessPayload struct {
	PetId  uint32 `json:"petId"`  // PetId associated with the action
	Amount uint16 `json:"amount"` // Amount of closeness to gain
}

// ValidationConditionInput represents a condition for character state validation
// This is a simplified version for the shared library
type ValidationConditionInput struct {
	Type            string `json:"type"`
	Operator        string `json:"operator"`
	Value           int    `json:"value"`
	ReferenceId     uint32 `json:"referenceId,omitempty"`
	Step            string `json:"step,omitempty"`
	WorldId         byte   `json:"worldId,omitempty"`
	ChannelId       byte   `json:"channelId,omitempty"`
	IncludeEquipped bool   `json:"includeEquipped,omitempty"`
}

// ValidateCharacterStatePayload represents the payload required to validate a character's state.
type ValidateCharacterStatePayload struct {
	CharacterId uint32                     `json:"characterId"` // CharacterId associated with the action
	Conditions  []ValidationConditionInput `json:"conditions"`  // Conditions to validate
}

// ChangeHairPayload represents the payload required to change a character's hair.
type ChangeHairPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	StyleId     uint32     `json:"styleId"`     // Hair style ID to change to
}

// ChangeFacePayload represents the payload required to change a character's face.
type ChangeFacePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	StyleId     uint32     `json:"styleId"`     // Face style ID to change to
}

// ChangeSkinPayload represents the payload required to change a character's skin color.
type ChangeSkinPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	StyleId     byte       `json:"styleId"`     // Skin color ID to change to
}

// SpawnMonsterPayload represents the payload required to spawn monsters.
// Note: Foothold (fh) is resolved by saga-orchestrator via atlas-data lookup, not specified here.
type SpawnMonsterPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	MapId       uint32     `json:"mapId"`       // MapId where monsters should spawn
	MonsterId   uint32     `json:"monsterId"`   // MonsterId to spawn
	X           int16      `json:"x"`           // X coordinate for spawn
	Y           int16      `json:"y"`           // Y coordinate for spawn
	Team        int8       `json:"team"`        // Team assignment (optional, defaults to 0)
	Count       int        `json:"count"`       // Number of monsters to spawn (optional, defaults to 1)
}

// CompleteQuestPayload represents the payload required to complete a quest.
type CompleteQuestPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id `json:"worldId"`     // WorldId associated with the action
	QuestId     uint32   `json:"questId"`     // QuestId to complete
	NpcId       uint32   `json:"npcId"`       // NpcId that completed the quest
	Force       bool     `json:"force"`       // If true, skip requirement checks and just mark complete
}

// StartQuestPayload represents the payload required to start a quest.
// Note: This is currently a stub as no quest service exists yet.
type StartQuestPayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId associated with the action
	QuestId     uint32 `json:"questId"`     // QuestId to start
	NpcId       uint32 `json:"npcId"`       // NpcId that started the quest
}

// SetQuestProgressPayload represents the payload required to update quest progress.
type SetQuestProgressPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id `json:"worldId"`     // WorldId associated with the action
	QuestId     uint32   `json:"questId"`     // QuestId to update progress for
	InfoNumber  uint32   `json:"infoNumber"`  // Progress info number/step to update
	Progress    string   `json:"progress"`    // Progress value to set
}

// ApplyConsumableEffectPayload represents the payload required to apply consumable item effects to a character.
// This is used for NPC-initiated item usage where the item effects are applied
// without consuming from inventory (e.g., NPC buffs like Shinsoo's blessing).
type ApplyConsumableEffectPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to apply item effects to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	ItemId      uint32     `json:"itemId"`      // Consumable item ID whose effects should be applied
}

// SendMessagePayload represents the payload required to send a system message to a character.
// This is used for NPC-initiated messages like "You have acquired a Dragon Egg."
type SendMessagePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to send message to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	MessageType string     `json:"messageType"` // Message type: "NOTICE", "POP_UP", "PINK_TEXT", "BLUE_TEXT"
	Message     string     `json:"message"`     // The message text to display
}

// AwardFamePayload represents the payload required to award fame to a character.
type AwardFamePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to award fame to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Amount      int16      `json:"amount"`      // Amount of fame to award (can be negative)
}

// ShowStoragePayload represents the payload required to show the storage UI to a character.
// This is triggered by NPC interactions and sends a command to the channel service to display storage.
type ShowStoragePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show storage to
	NpcId       uint32     `json:"npcId"`       // NpcId of the storage keeper
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	AccountId   uint32     `json:"accountId"`   // AccountId that owns the storage
}

type ExperienceDistributions struct {
	ExperienceType string `json:"experienceType"`
	Amount         uint32 `json:"amount"`
	Attr1          uint32 `json:"attr1"`
}

// PlayPortalSoundPayload represents the payload for playing portal sound effect
type PlayPortalSoundPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to play sound for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
}

// UpdateAreaInfoPayload represents the payload for updating a player's area info (quest record ex)
// Used for quest-related area tracking (e.g., qm.updateAreaInfo() in scripts)
type UpdateAreaInfoPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to update area info for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Area        uint16     `json:"area"`        // Area/info number (questId in the protocol)
	Info        string     `json:"info"`        // Info string to display
}

// ShowInfoPayload represents the payload for showing an info/tutorial effect to a player
// Used for tutorial messages and visual effects (e.g., qm.showInfo() in scripts)
type ShowInfoPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show info to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Path        string     `json:"path"`        // Path to the info effect (e.g., "Effect/OnUserEff.img/RecoveryUp")
}

// ShowInfoTextPayload represents the payload for showing a text message to a player
// Used for tutorial/info text messages (e.g., qm.showInfoText() in scripts)
type ShowInfoTextPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show text to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Text        string     `json:"text"`        // Text message to display
}

// ShowHintPayload represents the payload for showing a hint box to a player
// Used for displaying hint messages (e.g., qm.showHint() in scripts)
type ShowHintPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show hint to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Hint        string     `json:"hint"`        // Hint text to display
	Width       uint16     `json:"width"`       // Width of the hint box (0 for auto-calculation)
	Height      uint16     `json:"height"`      // Height of the hint box (0 for auto-calculation)
}

// BlockPortalPayload represents the payload for blocking a portal for a character
// This is a synchronous action that immediately completes after sending the command.
// The portal will remain blocked for the character until they logout or it is explicitly unblocked.
type BlockPortalPayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId to block the portal for
	MapId       uint32 `json:"mapId"`       // MapId where the portal is located
	PortalId    uint32 `json:"portalId"`    // PortalId to block
}

// UnblockPortalPayload represents the payload for unblocking a portal for a character
// This is a synchronous action that immediately completes after sending the command.
type UnblockPortalPayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId to unblock the portal for
	MapId       uint32 `json:"mapId"`       // MapId where the portal is located
	PortalId    uint32 `json:"portalId"`    // PortalId to unblock
}

// SpawnReactorDropsPayload represents the payload for spawning drops from a reactor.
// saga-orchestrator will fetch drop configuration from atlas-drop-information
// and spawn drops via atlas-drops service.
type SpawnReactorDropsPayload struct {
	CharacterId    uint32     `json:"characterId"`    // Character who triggered the reactor
	WorldId        world.Id   `json:"worldId"`        // WorldId for drop spawning
	ChannelId      channel.Id `json:"channelId"`      // ChannelId for drop spawning
	MapId          uint32     `json:"mapId"`          // MapId where drops should spawn
	ReactorId      uint32     `json:"reactorId"`      // ReactorId for fetching drop configuration
	Classification string     `json:"classification"` // Reactor classification string
	X              int16      `json:"x"`              // Reactor X position (drop origin)
	Y              int16      `json:"y"`              // Reactor Y position (drop origin)
	DropType       string     `json:"dropType"`       // "drop" (simultaneous) or "spray" (200ms intervals)
	Meso           bool       `json:"meso"`           // Whether meso drops are enabled
	MesoChance     uint32     `json:"mesoChance"`     // Meso drop probability (1/chance)
	MesoMin        uint32     `json:"mesoMin"`        // Minimum meso amount per drop
	MesoMax        uint32     `json:"mesoMax"`        // Maximum meso amount per drop
	MinItems       uint32     `json:"minItems"`       // Minimum guaranteed drops (padded with meso)
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
	case AwardInventory:
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
	case IncreaseBuddyCapacity:
		var payload IncreaseBuddyCapacityPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case GainCloseness:
		var payload GainClosenessPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case ValidateCharacterState:
		var payload ValidateCharacterStatePayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case SpawnMonster:
		var payload SpawnMonsterPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case CompleteQuest:
		var payload CompleteQuestPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case StartQuest:
		var payload StartQuestPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case SetQuestProgress:
		var payload SetQuestProgressPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case ApplyConsumableEffect:
		var payload ApplyConsumableEffectPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case SendMessage:
		var payload SendMessagePayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case AwardFame:
		var payload AwardFamePayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case ShowStorage:
		var payload ShowStoragePayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case ChangeHair:
		var payload ChangeHairPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case ChangeFace:
		var payload ChangeFacePayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case ChangeSkin:
		var payload ChangeSkinPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case PlayPortalSound:
		var payload PlayPortalSoundPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case UpdateAreaInfo:
		var payload UpdateAreaInfoPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case ShowInfo:
		var payload ShowInfoPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case ShowInfoText:
		var payload ShowInfoTextPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case ShowHint:
		var payload ShowHintPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case BlockPortal:
		var payload BlockPortalPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case UnblockPortal:
		var payload UnblockPortalPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	case SpawnReactorDrops:
		var payload SpawnReactorDropsPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
	default:
		return fmt.Errorf("unknown action: %s", s.Action)
	}

	return nil
}

// Processor is the interface for saga operations
type Processor interface {
	// Create creates a new saga
	Create(s Saga) error
}
