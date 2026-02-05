package saga

import (
	"atlas-saga-orchestrator/validation"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/item"
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
	CharacterCreation    Type = "character_creation"
	StorageOperation     Type = "storage_operation"
	CharacterRespawn     Type = "character_respawn"
)

// Saga represents the entire saga transaction.
type Saga struct {
	transactionId uuid.UUID
	sagaType      Type
	initiatedBy   string
	steps         []Step[any]
}

// TransactionId returns the transaction ID
func (s Saga) TransactionId() uuid.UUID { return s.transactionId }

// SagaType returns the saga type
func (s Saga) SagaType() Type { return s.sagaType }

// InitiatedBy returns who initiated the saga
func (s Saga) InitiatedBy() string { return s.initiatedBy }

// Steps returns a copy of the steps slice
func (s Saga) Steps() []Step[any] {
	result := make([]Step[any], len(s.steps))
	copy(result, s.steps)
	return result
}

// StepAt returns the step at the given index
func (s Saga) StepAt(index int) (Step[any], bool) {
	if index < 0 || index >= len(s.steps) {
		return Step[any]{}, false
	}
	return s.steps[index], true
}

// StepCount returns the number of steps
func (s Saga) StepCount() int {
	return len(s.steps)
}

// MarshalJSON implements json.Marshaler for Saga
func (s Saga) MarshalJSON() ([]byte, error) {
	type alias struct {
		TransactionId uuid.UUID   `json:"transactionId"`
		SagaType      Type        `json:"sagaType"`
		InitiatedBy   string      `json:"initiatedBy"`
		Steps         []Step[any] `json:"steps"`
	}
	return json.Marshal(alias{
		TransactionId: s.transactionId,
		SagaType:      s.sagaType,
		InitiatedBy:   s.initiatedBy,
		Steps:         s.steps,
	})
}

// UnmarshalJSON implements json.Unmarshaler for Saga
func (s *Saga) UnmarshalJSON(data []byte) error {
	type alias struct {
		TransactionId uuid.UUID   `json:"transactionId"`
		SagaType      Type        `json:"sagaType"`
		InitiatedBy   string      `json:"initiatedBy"`
		Steps         []Step[any] `json:"steps"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	s.transactionId = a.TransactionId
	s.sagaType = a.SagaType
	s.initiatedBy = a.InitiatedBy
	s.steps = a.Steps
	return nil
}

// Failing returns true if any step has failed status
func (s Saga) Failing() bool {
	for _, step := range s.steps {
		if step.status == Failed {
			return true
		}
	}
	return false
}

// GetCurrentStep returns the first pending step
func (s Saga) GetCurrentStep() (Step[any], bool) {
	for _, step := range s.steps {
		if step.status == Pending {
			return step, true
		}
	}
	return Step[any]{}, false
}

// FindFurthestCompletedStepIndex returns the index of the furthest completed step (last one with status "completed")
// Returns -1 if no completed step is found
func (s Saga) FindFurthestCompletedStepIndex() int {
	furthestCompletedIndex := -1
	for i := len(s.steps) - 1; i >= 0; i-- {
		if s.steps[i].status == Completed {
			furthestCompletedIndex = i
			break
		}
	}
	return furthestCompletedIndex
}

// FindEarliestPendingStepIndex returns the index of the earliest pending step (first one with status "pending")
// Returns -1 if no pending step is found
func (s Saga) FindEarliestPendingStepIndex() int {
	for i := 0; i < len(s.steps); i++ {
		if s.steps[i].status == Pending {
			return i
		}
	}
	return -1
}

// FindFailedStepIndex returns the index of the first failed step
// Returns -1 if no failed step is found
func (s Saga) FindFailedStepIndex() int {
	for i := 0; i < len(s.steps); i++ {
		if s.steps[i].status == Failed {
			return i
		}
	}
	return -1
}

// ValidateStepOrdering ensures that the saga steps are in a valid order
// Returns true if the ordering is valid, false otherwise
func (s Saga) ValidateStepOrdering() bool {
	foundPending := false
	for i := 0; i < len(s.steps); i++ {
		if s.steps[i].status == Pending {
			foundPending = true
		} else if s.steps[i].status == Completed && foundPending {
			return false
		}
	}
	return true
}

// ValidateStateConsistency performs comprehensive state consistency validation
func (s Saga) ValidateStateConsistency() error {
	if !s.ValidateStepOrdering() {
		return fmt.Errorf("invalid step ordering detected")
	}

	stepIds := make(map[string]bool)
	for i, step := range s.steps {
		if stepIds[step.stepId] {
			return fmt.Errorf("duplicate step ID '%s' found at index %d", step.stepId, i)
		}
		stepIds[step.stepId] = true
	}

	for i, step := range s.steps {
		if step.status != Pending && step.status != Completed && step.status != Failed {
			return fmt.Errorf("invalid status '%s' at step index %d", step.status, i)
		}
	}

	for i, step := range s.steps {
		if step.action == "" {
			return fmt.Errorf("empty action at step index %d", i)
		}
	}

	if s.Failing() {
		failedCount := 0
		for _, step := range s.steps {
			if step.status == Failed {
				failedCount++
			}
		}
		if failedCount != 1 {
			return fmt.Errorf("saga is failing but has %d failed steps, expected exactly 1", failedCount)
		}
	}

	return nil
}

// GetStepCount returns the total number of steps in the saga
func (s Saga) GetStepCount() int {
	return len(s.steps)
}

// validateStateTransition validates if a step status transition is valid
func (s Saga) validateStateTransition(stepIndex int, newStatus Status) error {
	if stepIndex < 0 || stepIndex >= len(s.steps) {
		return fmt.Errorf("invalid step index: %d", stepIndex)
	}

	currentStep := s.steps[stepIndex]
	currentStatus := currentStep.status

	switch currentStatus {
	case Pending:
		if newStatus != Completed && newStatus != Failed {
			return fmt.Errorf("invalid transition from %s to %s", currentStatus, newStatus)
		}
	case Completed:
		if newStatus != Failed {
			return fmt.Errorf("invalid transition from %s to %s", currentStatus, newStatus)
		}
	case Failed:
		if newStatus != Pending {
			return fmt.Errorf("invalid transition from %s to %s", currentStatus, newStatus)
		}
	default:
		return fmt.Errorf("unknown status: %s", currentStatus)
	}

	return nil
}

// GetCompletedStepCount returns the number of completed steps in the saga
func (s Saga) GetCompletedStepCount() int {
	count := 0
	for _, step := range s.steps {
		if step.status == Completed {
			count++
		}
	}
	return count
}

// GetPendingStepCount returns the number of pending steps in the saga
func (s Saga) GetPendingStepCount() int {
	count := 0
	for _, step := range s.steps {
		if step.status == Pending {
			count++
		}
	}
	return count
}

// WithStepStatus returns a new Saga with the specified step's status updated
func (s Saga) WithStepStatus(index int, status Status) (Saga, error) {
	if index < 0 || index >= len(s.steps) {
		return Saga{}, fmt.Errorf("invalid step index: %d", index)
	}
	if err := s.validateStateTransition(index, status); err != nil {
		return Saga{}, err
	}

	newSteps := make([]Step[any], len(s.steps))
	copy(newSteps, s.steps)

	newSteps[index] = Step[any]{
		stepId:    s.steps[index].stepId,
		status:    status,
		action:    s.steps[index].action,
		payload:   s.steps[index].payload,
		createdAt: s.steps[index].createdAt,
		updatedAt: time.Now(),
	}

	return Saga{
		transactionId: s.transactionId,
		sagaType:      s.sagaType,
		initiatedBy:   s.initiatedBy,
		steps:         newSteps,
	}, nil
}

// WithStep returns a new Saga with the step added at the end
func (s Saga) WithStep(step Step[any]) (Saga, error) {
	for _, existingStep := range s.steps {
		if existingStep.stepId == step.stepId {
			return Saga{}, fmt.Errorf("step ID '%s' already exists in saga", step.stepId)
		}
	}

	newSteps := make([]Step[any], len(s.steps)+1)
	copy(newSteps, s.steps)
	newSteps[len(s.steps)] = step

	return Saga{
		transactionId: s.transactionId,
		sagaType:      s.sagaType,
		initiatedBy:   s.initiatedBy,
		steps:         newSteps,
	}, nil
}

// WithStepAfterIndex returns a new Saga with the step inserted after the specified index
func (s Saga) WithStepAfterIndex(index int, step Step[any]) (Saga, error) {
	if index < -1 || index >= len(s.steps) {
		return Saga{}, fmt.Errorf("invalid step index: %d", index)
	}

	for _, existingStep := range s.steps {
		if existingStep.stepId == step.stepId {
			return Saga{}, fmt.Errorf("step ID '%s' already exists in saga", step.stepId)
		}
	}

	insertIndex := index + 1
	newSteps := make([]Step[any], len(s.steps)+1)
	copy(newSteps[:insertIndex], s.steps[:insertIndex])
	newSteps[insertIndex] = step
	copy(newSteps[insertIndex+1:], s.steps[insertIndex:])

	return Saga{
		transactionId: s.transactionId,
		sagaType:      s.sagaType,
		initiatedBy:   s.initiatedBy,
		steps:         newSteps,
	}, nil
}

// WithSteps returns a new Saga with the steps replaced
func (s Saga) WithSteps(steps []Step[any]) Saga {
	newSteps := make([]Step[any], len(steps))
	copy(newSteps, steps)

	return Saga{
		transactionId: s.transactionId,
		sagaType:      s.sagaType,
		initiatedBy:   s.initiatedBy,
		steps:         newSteps,
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
	AwardInventory               Action = "award_inventory" // Deprecated: Use AwardAsset instead
	AwardAsset                   Action = "award_asset"     // Preferred over AwardInventory
	AwardExperience              Action = "award_experience"
	AwardLevel                   Action = "award_level"
	AwardMesos                   Action = "award_mesos"
	AwardCurrency                Action = "award_currency"
	WarpToRandomPortal           Action = "warp_to_random_portal"
	WarpToPortal                 Action = "warp_to_portal"
	DestroyAsset                 Action = "destroy_asset"
	DestroyAssetFromSlot         Action = "destroy_asset_from_slot"
	EquipAsset                   Action = "equip_asset"
	UnequipAsset                 Action = "unequip_asset"
	ChangeJob                    Action = "change_job"
	ChangeHair                   Action = "change_hair"
	ChangeFace                   Action = "change_face"
	ChangeSkin                   Action = "change_skin"
	CreateSkill                  Action = "create_skill"
	UpdateSkill                  Action = "update_skill"
	ValidateCharacterState       Action = "validate_character_state"
	RequestGuildName             Action = "request_guild_name"
	RequestGuildEmblem           Action = "request_guild_emblem"
	RequestGuildDisband          Action = "request_guild_disband"
	RequestGuildCapacityIncrease Action = "request_guild_capacity_increase"
	CreateInvite                 Action = "create_invite"
	CreateCharacter              Action = "create_character"
	CreateAndEquipAsset          Action = "create_and_equip_asset"
	IncreaseBuddyCapacity        Action = "increase_buddy_capacity"
	GainCloseness                Action = "gain_closeness"
	SpawnMonster                 Action = "spawn_monster"
	SpawnReactorDrops            Action = "spawn_reactor_drops"
	CompleteQuest                Action = "complete_quest"
	StartQuest                   Action = "start_quest"
	SetQuestProgress             Action = "set_quest_progress"
	ApplyConsumableEffect        Action = "apply_consumable_effect"
	SendMessage                  Action = "send_message"
	DepositToStorage             Action = "deposit_to_storage"
	UpdateStorageMesos           Action = "update_storage_mesos"
	AwardFame                    Action = "award_fame"
	ShowStorage                  Action = "show_storage"
	TransferToStorage            Action = "transfer_to_storage"     // High-level action expanded to accept_to_storage + release_from_character
	WithdrawFromStorage          Action = "withdraw_from_storage"   // High-level action expanded to accept_to_character + release_from_storage
	AcceptToStorage              Action = "accept_to_storage"       // Internal step (created by expansion)
	ReleaseFromCharacter         Action = "release_from_character"  // Internal step (created by expansion)
	AcceptToCharacter            Action = "accept_to_character"     // Internal step (created by expansion)
	ReleaseFromStorage           Action = "release_from_storage"    // Internal step (created by expansion)
	TransferToCashShop           Action = "transfer_to_cash_shop"   // High-level action expanded to accept_to_cash_shop + release_from_character
	WithdrawFromCashShop         Action = "withdraw_from_cash_shop" // High-level action expanded to accept_to_character + release_from_cash_shop
	AcceptToCashShop             Action = "accept_to_cash_shop"     // Internal step (created by expansion)
	ReleaseFromCashShop          Action = "release_from_cash_shop"  // Internal step (created by expansion)

	// Character stat actions
	SetHP            Action = "set_hp"             // Set character HP to an absolute value
	DeductExperience Action = "deduct_experience"  // Deduct experience from character (with floor at 0)
	CancelAllBuffs   Action = "cancel_all_buffs"   // Cancel all active buffs on character
	ResetStats       Action = "reset_stats"        // Reset character stats (for job advancement)

	// Portal-specific actions
	PlayPortalSound  Action = "play_portal_sound"  // Play portal sound effect to character
	ShowInfo         Action = "show_info"          // Show info/tutorial effect to character
	ShowInfoText     Action = "show_info_text"     // Show info text message to character
	UpdateAreaInfo   Action = "update_area_info"   // Update area info (quest record ex) for character
	ShowHint         Action = "show_hint"          // Show hint box to character
	ShowGuideHint    Action = "show_guide_hint"   // Show pre-defined guide hint by ID to character
	ShowIntro        Action = "show_intro"        // Show intro/direction effect to character (e.g., tutorial animations)
	BlockPortal      Action = "block_portal"       // Block a portal for a character (session-based)
	UnblockPortal    Action = "unblock_portal"     // Unblock a portal for a character
)

// Step represents a single step within a saga.
type Step[T any] struct {
	stepId    string
	status    Status
	action    Action
	payload   T
	createdAt time.Time
	updatedAt time.Time
}

// StepId returns the step ID
func (s Step[T]) StepId() string { return s.stepId }

// Status returns the step status
func (s Step[T]) Status() Status { return s.status }

// Action returns the step action
func (s Step[T]) Action() Action { return s.action }

// Payload returns the step payload
func (s Step[T]) Payload() T { return s.payload }

// CreatedAt returns the step creation time
func (s Step[T]) CreatedAt() time.Time { return s.createdAt }

// UpdatedAt returns the step update time
func (s Step[T]) UpdatedAt() time.Time { return s.updatedAt }

// MarshalJSON implements json.Marshaler for Step
func (s Step[T]) MarshalJSON() ([]byte, error) {
	type alias struct {
		StepId    string    `json:"stepId"`
		Status    Status    `json:"status"`
		Action    Action    `json:"action"`
		Payload   T         `json:"payload"`
		CreatedAt time.Time `json:"createdAt"`
		UpdatedAt time.Time `json:"updatedAt"`
	}
	return json.Marshal(alias{
		StepId:    s.stepId,
		Status:    s.status,
		Action:    s.action,
		Payload:   s.payload,
		CreatedAt: s.createdAt,
		UpdatedAt: s.updatedAt,
	})
}

// NewStep creates a new Step with the given values
func NewStep[T any](stepId string, status Status, action Action, payload T) Step[T] {
	now := time.Now()
	return Step[T]{
		stepId:    stepId,
		status:    status,
		action:    action,
		payload:   payload,
		createdAt: now,
		updatedAt: now,
	}
}

// NewStepWithTimestamps creates a new Step with explicit timestamps
func NewStepWithTimestamps[T any](stepId string, status Status, action Action, payload T, createdAt, updatedAt time.Time) Step[T] {
	return Step[T]{
		stepId:    stepId,
		status:    status,
		action:    action,
		payload:   payload,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
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
	Instance    uuid.UUID  `json:"instance"`             // Instance specifies the map instance UUID (uuid.Nil for non-instanced maps)
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

// AwardCurrencyPayload represents the payload required to award cash shop currency to a character.
type AwardCurrencyPayload struct {
	CharacterId  uint32 `json:"characterId"`  // CharacterId associated with the action
	AccountId    uint32 `json:"accountId"`    // AccountId that owns the wallet
	CurrencyType uint32 `json:"currencyType"` // CurrencyType: 1=credit, 2=points, 3=prepaid
	Amount       int32  `json:"amount"`       // Amount of currency to award (can be negative for deduction)
}

// DestroyAssetPayload represents the payload required to destroy an asset in a compartment.
type DestroyAssetPayload struct {
	CharacterId uint32 `json:"characterId"` // CharacterId associated with the action
	TemplateId  uint32 `json:"templateId"`  // TemplateId of the item to destroy
	Quantity    uint32 `json:"quantity"`    // Quantity of the item to destroy (ignored if RemoveAll is true)
	RemoveAll   bool   `json:"removeAll"`   // If true, remove all instances of the item regardless of Quantity
}

// DestroyAssetFromSlotPayload represents the payload required to destroy an asset from a specific inventory slot.
// Unlike DestroyAssetPayload which finds items by template ID, this targets a specific slot directly.
type DestroyAssetFromSlotPayload struct {
	CharacterId   uint32 `json:"characterId"`   // CharacterId associated with the action
	InventoryType byte   `json:"inventoryType"` // Type of inventory (1=equip, 2=use, 3=setup, 4=etc, 5=cash)
	Slot          int16  `json:"slot"`          // Slot to destroy from (negative for equipped slots, positive for inventory slots)
	Quantity      uint32 `json:"quantity"`      // Quantity to destroy (0 or 1 for equipment)
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
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	JobId       job.Id     `json:"jobId"`       // JobId to change to
}

// ChangeHairPayload represents the payload required to change a character's hair.
type ChangeHairPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	StyleId     uint32     `json:"styleId"`     // Hair style ID to change to (range: 30000-35000)
}

// ChangeFacePayload represents the payload required to change a character's face.
type ChangeFacePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	StyleId     uint32     `json:"styleId"`     // Face style ID to change to (range: 20000-25000)
}

// ChangeSkinPayload represents the payload required to change a character's skin.
type ChangeSkinPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	StyleId     byte       `json:"styleId"`     // Skin color ID to change to (range: 0-9)
}

// CreateSkillPayload represents the payload required to create a skill for a character.
type CreateSkillPayload struct {
	CharacterId uint32    `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id  `json:"worldId"`     // WorldId associated with the action
	SkillId     uint32    `json:"skillId"`     // SkillId to create
	Level       byte      `json:"level"`       // Skill level
	MasterLevel byte      `json:"masterLevel"` // Skill master level
	Expiration  time.Time `json:"expiration"`  // Skill expiration time
}

// UpdateSkillPayload represents the payload required to update a skill for a character.
type UpdateSkillPayload struct {
	CharacterId uint32    `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id  `json:"worldId"`     // WorldId associated with the action
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
	PetId  uint32 `json:"petId"`  // Pet ID to gain closeness with
	Amount uint16 `json:"amount"` // Amount of closeness to gain
}

// ValidateCharacterStatePayload represents the payload required to validate a character's state.
type ValidateCharacterStatePayload struct {
	CharacterId uint32                      `json:"characterId"` // CharacterId associated with the action
	Conditions  []validation.ConditionInput `json:"conditions"`  // Conditions to validate
}

// RequestGuildNamePayload represents the payload required to request a guild name.
type RequestGuildNamePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
}

// RequestGuildEmblemPayload represents the payload required to request a guild emblem change.
type RequestGuildEmblemPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
}

// RequestGuildDisbandPayload represents the payload required to request a guild disband.
type RequestGuildDisbandPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
}

// RequestGuildCapacityIncreasePayload represents the payload required to request a guild capacity increase.
type RequestGuildCapacityIncreasePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
}

// CreateInvitePayload represents the payload required to create an invitation.
type CreateInvitePayload struct {
	InviteType   string   `json:"inviteType"`   // Type of invitation (e.g., "GUILD", "PARTY", "BUDDY")
	OriginatorId uint32   `json:"originatorId"` // ID of the character sending the invitation
	TargetId     uint32   `json:"targetId"`     // ID of the character receiving the invitation
	ReferenceId  uint32   `json:"referenceId"`  // ID of the entity being invited to (e.g., guild ID, party ID)
	WorldId      world.Id `json:"worldId"`      // WorldId associated with the action
}

// CharacterCreatePayload represents the payload required to create a character.
// Note: this does not include any character attributes, as those are determined by the character service.
type CharacterCreatePayload struct {
	AccountId    uint32   `json:"accountId"` // AccountId associated with the action
	WorldId      world.Id `json:"worldId"`   // WorldId associated with the action
	Name         string   `json:"name"`      // Name of the character to create
	Gender       byte     `json:"gender"`
	Level        byte     `json:"level"`
	Strength     uint16   `json:"strength"`
	Dexterity    uint16   `json:"dexterity"`
	Intelligence uint16   `json:"intelligence"`
	Luck         uint16   `json:"luck"`
	JobId        job.Id   `json:"jobId"` // JobId to create the character with
	Hp           uint16   `json:"hp"`
	Mp           uint16   `json:"mp"`
	Face         uint32   `json:"face"`   // Face of the character
	Hair         uint32   `json:"hair"`   // Hair of the character
	Skin         byte     `json:"skin"`   // Skin of the character
	Top          uint32   `json:"top"`    // Top of the character
	Bottom       uint32   `json:"bottom"` // Bottom of the character
	Shoes        uint32   `json:"shoes"`  // Shoes of the character
	Weapon       uint32   `json:"weapon"` // Weapon of the character
	MapId        _map.Id  `json:"mapId"`  // Starting map ID for the character
}

// CreateAndEquipAssetPayload represents the payload required to create and equip an asset.
type CreateAndEquipAssetPayload struct {
	CharacterId uint32      `json:"characterId"` // CharacterId associated with the action
	Item        ItemPayload `json:"item"`        // Item to create and equip
}

type ExperienceDistributions struct {
	ExperienceType string `json:"experienceType"`
	Amount         uint32 `json:"amount"`
	Attr1          uint32 `json:"attr1"`
}

// SpawnMonsterPayload represents the payload required to spawn monsters.
// Note: Foothold (fh) is not included - it is resolved dynamically by the saga-orchestrator
// via atlas-data's foothold lookup endpoint.
type SpawnMonsterPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId initiating the spawn
	WorldId     world.Id   `json:"worldId"`     // WorldId for the spawn location
	ChannelId   channel.Id `json:"channelId"`   // ChannelId for the spawn location
	MapId       _map.Id    `json:"mapId"`       // MapId for the spawn location
	Instance    uuid.UUID  `json:"instance"`    // Instance UUID for the map instance
	MonsterId   uint32     `json:"monsterId"`   // Monster template ID to spawn
	X           int16      `json:"x"`           // X coordinate for spawn position
	Y           int16      `json:"y"`           // Y coordinate for spawn position
	Team        int8       `json:"team"`        // Team assignment (default 0)
	Count       int        `json:"count"`       // Number of monsters to spawn (default 1)
}

// SpawnReactorDropsPayload represents the payload required to spawn drops from a reactor.
// The saga-orchestrator fetches drop configuration from atlas-drop-information and calculates
// which items to drop based on chances, then spawns them via atlas-drops.
type SpawnReactorDropsPayload struct {
	CharacterId    uint32     `json:"characterId"`    // CharacterId who triggered the reactor
	WorldId        world.Id   `json:"worldId"`        // WorldId for the drop location
	ChannelId      channel.Id `json:"channelId"`      // ChannelId for the drop location
	MapId          _map.Id    `json:"mapId"`          // MapId for the drop location
	Instance       uuid.UUID  `json:"instance"`       // Instance identifier for the map
	ReactorId      uint32     `json:"reactorId"`      // Reactor template ID
	Classification string     `json:"classification"` // Reactor classification string
	X              int16      `json:"x"`              // X coordinate for drop position
	Y              int16      `json:"y"`              // Y coordinate for drop position
	DropType       string     `json:"dropType"`       // "drop" for simultaneous, "spray" for 200ms intervals
	Meso           bool       `json:"meso"`           // Whether meso drops are enabled
	MesoChance     uint32     `json:"mesoChance"`     // Chance for meso drop (1 = 100%)
	MesoMin        uint32     `json:"mesoMin"`        // Minimum meso amount
	MesoMax        uint32     `json:"mesoMax"`        // Maximum meso amount
	MinItems       uint32     `json:"minItems"`       // Minimum guaranteed drops (padded with meso)
}

// CompleteQuestPayload represents the payload required to complete a quest.
type CompleteQuestPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId completing the quest
	WorldId     world.Id `json:"worldId"`     // WorldId associated with the action
	QuestId     uint32   `json:"questId"`     // Quest ID to complete
	NpcId       uint32   `json:"npcId"`       // NPC ID granting completion (for rewards)
	Force       bool     `json:"force"`       // If true, skip requirement checks and just mark complete
}

// StartQuestPayload represents the payload required to start a quest.
type StartQuestPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId starting the quest
	WorldId     world.Id `json:"worldId"`     // WorldId associated with the action
	QuestId     uint32   `json:"questId"`     // Quest ID to start
	NpcId       uint32   `json:"npcId"`       // NPC ID initiating the quest
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
	CharacterId character.Id `json:"characterId"` // CharacterId to apply item effects to
	WorldId     world.Id     `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id   `json:"channelId"`   // ChannelId associated with the action
	ItemId      item.Id      `json:"itemId"`      // Consumable item ID whose effects should be applied
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

// PlayPortalSoundPayload represents the payload required to play the portal sound effect.
// This is a synchronous action that immediately completes after sending.
type PlayPortalSoundPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to play sound for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
}

// ShowInfoPayload represents the payload required to show an info/tutorial effect.
// This is a synchronous action that immediately completes after sending.
type ShowInfoPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show info for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Path        string     `json:"path"`        // Path to the info effect (e.g., "Effect/OnUserEff.img/RecoveryUp")
}

// ShowInfoTextPayload represents the payload required to show a text message to a character.
// This is a synchronous action that immediately completes after sending.
type ShowInfoTextPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show text for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Text        string     `json:"text"`        // Text message to display
}

// UpdateAreaInfoPayload represents the payload required to update area info (quest record ex).
// This is a synchronous action that immediately completes after sending.
type UpdateAreaInfoPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to update area info for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Area        uint16     `json:"area"`        // Area/info number (questId in the protocol)
	Info        string     `json:"info"`        // Info string to display
}

// ShowHintPayload represents the payload required to show a hint box to a character.
// This is a synchronous action that immediately completes after sending.
type ShowHintPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show hint to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Hint        string     `json:"hint"`        // Hint text to display
	Width       uint16     `json:"width"`       // Width of the hint box (0 for auto-calculation)
	Height      uint16     `json:"height"`      // Height of the hint box (0 for auto-calculation)
}

// ShowGuideHintPayload represents the payload required to show a pre-defined guide hint by ID.
// This is a synchronous action that immediately completes after sending.
// Used for guide hints like qm.guideHint(2) in quest scripts.
type ShowGuideHintPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show guide hint to
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	HintId      uint32     `json:"hintId"`      // Pre-defined hint ID (maps to client's guide hint system)
	Duration    uint32     `json:"duration"`    // Duration in milliseconds (default 7000ms if 0)
}

// ShowIntroPayload represents the payload required to show an intro/direction effect to a character.
// This is a synchronous action that immediately completes after sending.
// Used for tutorial animations like "Effect/Direction1.img/aranTutorial/ClickPoleArm".
type ShowIntroPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to show intro for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Path        string     `json:"path"`        // Path to the intro effect (e.g., "Effect/Direction1.img/aranTutorial/ClickPoleArm")
}

// SetHPPayload represents the payload required to set a character's HP to an absolute value.
// This is an asynchronous action that completes when the character status event is received.
type SetHPPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to set HP for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Amount      uint16     `json:"amount"`      // Absolute HP value to set (clamped to 0..MaxHP)
}

// DeductExperiencePayload represents the payload required to deduct experience from a character.
// This is an asynchronous action that completes when the character status event is received.
// The experience deducted will not go below 0.
type DeductExperiencePayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to deduct experience from
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	Amount      uint32     `json:"amount"`      // Amount of experience to deduct (will not go below 0)
}

// CancelAllBuffsPayload represents the payload required to cancel all active buffs on a character.
// This is an asynchronous action that completes when the buff status events are received.
type CancelAllBuffsPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to cancel buffs for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
	MapId       _map.Id    `json:"mapId"`       // MapId associated with the action
	Instance    uuid.UUID  `json:"instance"`    // Instance associated with the action
}

// ResetStatsPayload represents the payload required to reset a character's stats.
// This is used during job advancement to reset AP distribution.
type ResetStatsPayload struct {
	CharacterId uint32     `json:"characterId"` // CharacterId to reset stats for
	WorldId     world.Id   `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id `json:"channelId"`   // ChannelId associated with the action
}

// BlockPortalPayload represents the payload required to block a portal for a character.
// This is a synchronous action that immediately completes after sending the event.
// The portal will remain blocked for the character until they logout or it is explicitly unblocked.
type BlockPortalPayload struct {
	CharacterId uint32  `json:"characterId"` // CharacterId to block the portal for
	MapId       _map.Id `json:"mapId"`       // MapId where the portal is located
	PortalId    uint32  `json:"portalId"`    // PortalId to block
}

// UnblockPortalPayload represents the payload required to unblock a portal for a character.
// This is a synchronous action that immediately completes after sending the event.
type UnblockPortalPayload struct {
	CharacterId uint32  `json:"characterId"` // CharacterId to unblock the portal for
	MapId       _map.Id `json:"mapId"`       // MapId where the portal is located
	PortalId    uint32  `json:"portalId"`    // PortalId to unblock
}

// DepositToStoragePayload represents the payload required to deposit an item to account storage.
type DepositToStoragePayload struct {
	CharacterId   uint32    `json:"characterId"`   // CharacterId initiating the deposit
	AccountId     uint32    `json:"accountId"`     // AccountId that owns the storage
	WorldId       world.Id  `json:"worldId"`       // WorldId for the storage (storage is world-scoped)
	Slot          int16     `json:"slot"`          // Target slot in storage
	TemplateId    uint32    `json:"templateId"`    // Item template ID
	ReferenceId   uint32    `json:"referenceId"`   // Reference ID for the item data (external service ID)
	ReferenceType string    `json:"referenceType"` // Type of reference: "equipable", "consumable", "setup", "etc", "cash", "pet"
	Expiration    time.Time `json:"expiration"`    // Item expiration time
	Quantity      uint32    `json:"quantity"`      // Quantity (for stackables)
	OwnerId       uint32    `json:"ownerId"`       // Owner ID (for stackables)
	Flag          uint16    `json:"flag"`          // Item flag (for stackables)
}

// UpdateStorageMesosPayload represents the payload required to update mesos in account storage.
type UpdateStorageMesosPayload struct {
	CharacterId uint32   `json:"characterId"` // CharacterId initiating the update
	AccountId   uint32   `json:"accountId"`   // AccountId that owns the storage
	WorldId     world.Id `json:"worldId"`     // WorldId for the storage
	Operation   string   `json:"operation"`   // Operation: "SET", "ADD", "SUBTRACT"
	Mesos       uint32   `json:"mesos"`       // Mesos amount
}

// AwardFamePayload represents the payload required to award fame to a character.
// This is used for NPC/quest-initiated fame rewards (e.g., qm.gainFame() in scripts).
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

// TransferToStoragePayload represents a high-level transfer from character inventory to storage
// This step is expanded by saga-orchestrator into accept_to_storage + release_from_character
type TransferToStoragePayload struct {
	TransactionId       uuid.UUID `json:"transactionId"`       // Saga transaction ID
	CharacterId         uint32    `json:"characterId"`         // Character initiating the transfer
	WorldId             world.Id  `json:"worldId"`             // World ID
	AccountId           uint32    `json:"accountId"`           // Account ID (storage owner)
	SourceSlot          int16     `json:"sourceSlot"`          // Slot in character inventory
	SourceInventoryType byte      `json:"sourceInventoryType"` // Character inventory type (equip, use, etc.)
	Quantity            uint32    `json:"quantity"`            // Quantity to transfer (0 = all)
}

// WithdrawFromStoragePayload represents a high-level withdrawal from storage to character inventory
// This step is expanded by saga-orchestrator into accept_to_character + release_from_storage
type WithdrawFromStoragePayload struct {
	TransactionId uuid.UUID `json:"transactionId"` // Saga transaction ID
	CharacterId   uint32    `json:"characterId"`   // Character receiving the item
	WorldId       world.Id  `json:"worldId"`       // World ID
	AccountId     uint32    `json:"accountId"`     // Account ID (storage owner)
	SourceSlot    int16     `json:"sourceSlot"`    // Slot in storage
	InventoryType byte      `json:"inventoryType"` // Target character inventory type
	Quantity      uint32    `json:"quantity"`      // Quantity to withdraw (0 = all)
}

// AcceptToStoragePayload represents the payload for the accept_to_storage action (internal step)
// This is created by saga-orchestrator expansion with all asset data pre-populated
type AcceptToStoragePayload struct {
	TransactionId uuid.UUID       `json:"transactionId"` // Saga transaction ID
	WorldId       world.Id        `json:"worldId"`       // World ID
	AccountId     uint32          `json:"accountId"`     // Account ID
	CharacterId   uint32          `json:"characterId"`   // Character initiating the transfer
	TemplateId    uint32          `json:"templateId"`    // Item template ID
	ReferenceId   uint32          `json:"referenceId"`   // Reference ID
	ReferenceType string          `json:"referenceType"` // Reference type
	ReferenceData json.RawMessage `json:"referenceData"` // Asset-specific data
	Quantity      uint32          `json:"quantity"`      // Quantity to accept (0 = all from source)
}

// ReleaseFromCharacterPayload represents the payload for the release_from_character action (internal step)
// This is created by saga-orchestrator expansion with asset ID pre-populated
type ReleaseFromCharacterPayload struct {
	TransactionId uuid.UUID `json:"transactionId"` // Saga transaction ID
	CharacterId   uint32    `json:"characterId"`   // Character ID
	InventoryType byte      `json:"inventoryType"` // Inventory type (equip, use, etc.)
	AssetId       uint32    `json:"assetId"`       // Asset ID to release (populated during expansion)
	Quantity      uint32    `json:"quantity"`      // Quantity to release (0 = all)
}

// AcceptToCharacterPayload represents the payload for the accept_to_character action (internal step)
// This is created by saga-orchestrator expansion with all asset data pre-populated
type AcceptToCharacterPayload struct {
	TransactionId uuid.UUID       `json:"transactionId"` // Saga transaction ID
	CharacterId   uint32          `json:"characterId"`   // Character ID
	InventoryType byte            `json:"inventoryType"` // Inventory type (equip, use, etc.)
	TemplateId    uint32          `json:"templateId"`    // Item template ID
	ReferenceId   uint32          `json:"referenceId"`   // Reference ID
	ReferenceType string          `json:"referenceType"` // Reference type
	ReferenceData json.RawMessage `json:"referenceData"` // Asset-specific data
	Quantity      uint32          `json:"quantity"`      // Quantity to accept (0 = all from source)
}

// ReleaseFromStoragePayload represents the payload for the release_from_storage action (internal step)
// This is created by saga-orchestrator expansion with asset ID pre-populated
type ReleaseFromStoragePayload struct {
	TransactionId uuid.UUID `json:"transactionId"` // Saga transaction ID
	WorldId       world.Id  `json:"worldId"`       // World ID
	AccountId     uint32    `json:"accountId"`     // Account ID
	CharacterId   uint32    `json:"characterId"`   // Character receiving the item
	AssetId       uint32    `json:"assetId"`       // Asset ID to release (populated during expansion)
	Quantity      uint32    `json:"quantity"`      // Quantity to release (0 = all)
}

// TransferToCashShopPayload represents a high-level transfer from character inventory to cash shop
// This step is expanded by saga-orchestrator into accept_to_cash_shop + release_from_character
type TransferToCashShopPayload struct {
	TransactionId       uuid.UUID `json:"transactionId"`       // Saga transaction ID
	CharacterId         uint32    `json:"characterId"`         // Character initiating the transfer
	AccountId           uint32    `json:"accountId"`           // Account ID (cash shop owner)
	CashId              int64     `json:"cashId"`              // Cash serial number of the item to transfer
	SourceInventoryType byte      `json:"sourceInventoryType"` // Character inventory type (equip, use, etc.)
	CompartmentType     byte      `json:"compartmentType"`     // Cash shop compartment type (1=Explorer, 2=Cygnus, 3=Legend)
}

// WithdrawFromCashShopPayload represents a high-level withdrawal from cash shop to character inventory
// This step is expanded by saga-orchestrator into accept_to_character + release_from_cash_shop
type WithdrawFromCashShopPayload struct {
	TransactionId   uuid.UUID `json:"transactionId"`   // Saga transaction ID
	CharacterId     uint32    `json:"characterId"`     // Character receiving the item
	AccountId       uint32    `json:"accountId"`       // Account ID (cash shop owner)
	CashId          uint64    `json:"cashId"`          // Cash serial number of the item to withdraw
	CompartmentType byte      `json:"compartmentType"` // Cash shop compartment type (1=Explorer, 2=Cygnus, 3=Legend)
	InventoryType   byte      `json:"inventoryType"`   // Target character inventory type
}

// AcceptToCashShopPayload represents the payload for the accept_to_cash_shop action (internal step)
// This is created by saga-orchestrator expansion with all asset data pre-populated
type AcceptToCashShopPayload struct {
	TransactionId   uuid.UUID       `json:"transactionId"`   // Saga transaction ID
	CharacterId     uint32          `json:"characterId"`     // Character ID for session lookup
	AccountId       uint32          `json:"accountId"`       // Account ID
	CompartmentId   uuid.UUID       `json:"compartmentId"`   // Cash shop compartment ID
	CompartmentType byte            `json:"compartmentType"` // Compartment type (1=Explorer, 2=Cygnus, 3=Legend)
	CashId          int64           `json:"cashId"`          // Preserved CashId from source item
	TemplateId      uint32          `json:"templateId"`      // Item template ID
	ReferenceId     uint32          `json:"referenceId"`     // Reference ID
	ReferenceType   string          `json:"referenceType"`   // Reference type
	ReferenceData   json.RawMessage `json:"referenceData"`   // Asset-specific data
}

// ReleaseFromCashShopPayload represents the payload for the release_from_cash_shop action (internal step)
// This is created by saga-orchestrator expansion with asset ID pre-populated
type ReleaseFromCashShopPayload struct {
	TransactionId   uuid.UUID `json:"transactionId"`   // Saga transaction ID
	CharacterId     uint32    `json:"characterId"`     // Character ID for session lookup
	AccountId       uint32    `json:"accountId"`       // Account ID
	CompartmentId   uuid.UUID `json:"compartmentId"`   // Cash shop compartment ID
	CompartmentType byte      `json:"compartmentType"` // Compartment type (1=Explorer, 2=Cygnus, 3=Legend)
	AssetId         uint32    `json:"assetId"`         // Cash item ID to release (populated during expansion)
	CashId          int64     `json:"cashId"`          // CashId for client notification correlation
	TemplateId      uint32    `json:"templateId"`      // Item template ID for client notification
}

// Custom UnmarshalJSON for Step[T] to handle the generics
func (s *Step[T]) UnmarshalJSON(data []byte) error {
	// First unmarshal to get the action type
	var actionOnly struct {
		StepId    string          `json:"stepId"`
		Status    Status          `json:"status"`
		Action    Action          `json:"action"`
		CreatedAt time.Time       `json:"createdAt"`
		UpdatedAt time.Time       `json:"updatedAt"`
		Payload   json.RawMessage `json:"payload"`
	}

	if err := json.Unmarshal(data, &actionOnly); err != nil {
		return err
	}

	s.stepId = actionOnly.StepId
	s.status = actionOnly.Status
	s.action = actionOnly.Action
	s.createdAt = actionOnly.CreatedAt
	s.updatedAt = actionOnly.UpdatedAt

	// Now handle the Payload field based on the Action type
	switch s.action {
	case AwardInventory, AwardAsset:
		var payload AwardItemActionPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case AwardExperience:
		var payload AwardExperiencePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case AwardLevel:
		var payload AwardLevelPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case AwardMesos:
		var payload AwardMesosPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case AwardCurrency:
		var payload AwardCurrencyPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case WarpToRandomPortal:
		var payload WarpToRandomPortalPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case WarpToPortal:
		var payload WarpToPortalPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case DestroyAsset:
		var payload DestroyAssetPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case DestroyAssetFromSlot:
		var payload DestroyAssetFromSlotPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case EquipAsset:
		var payload EquipAssetPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case UnequipAsset:
		var payload UnequipAssetPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ChangeJob:
		var payload ChangeJobPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ChangeHair:
		var payload ChangeHairPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ChangeFace:
		var payload ChangeFacePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ChangeSkin:
		var payload ChangeSkinPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case CreateSkill:
		var payload CreateSkillPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case UpdateSkill:
		var payload UpdateSkillPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case CreateInvite:
		var payload CreateInvitePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case CreateCharacter:
		var payload CharacterCreatePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case CreateAndEquipAsset:
		var payload CreateAndEquipAssetPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case IncreaseBuddyCapacity:
		var payload IncreaseBuddyCapacityPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case SpawnMonster:
		var payload SpawnMonsterPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case SpawnReactorDrops:
		var payload SpawnReactorDropsPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case CompleteQuest:
		var payload CompleteQuestPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case StartQuest:
		var payload StartQuestPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case SetQuestProgress:
		var payload SetQuestProgressPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ApplyConsumableEffect:
		var payload ApplyConsumableEffectPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case SendMessage:
		var payload SendMessagePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case PlayPortalSound:
		var payload PlayPortalSoundPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ShowInfo:
		var payload ShowInfoPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ShowInfoText:
		var payload ShowInfoTextPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case UpdateAreaInfo:
		var payload UpdateAreaInfoPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ShowHint:
		var payload ShowHintPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ShowGuideHint:
		var payload ShowGuideHintPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ShowIntro:
		var payload ShowIntroPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case SetHP:
		var payload SetHPPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case CancelAllBuffs:
		var payload CancelAllBuffsPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ResetStats:
		var payload ResetStatsPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case BlockPortal:
		var payload BlockPortalPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case UnblockPortal:
		var payload UnblockPortalPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case DepositToStorage:
		var payload DepositToStoragePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case TransferToStorage:
		var payload TransferToStoragePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case WithdrawFromStorage:
		var payload WithdrawFromStoragePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case UpdateStorageMesos:
		var payload UpdateStorageMesosPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case AwardFame:
		var payload AwardFamePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ShowStorage:
		var payload ShowStoragePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case AcceptToStorage:
		var payload AcceptToStoragePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ReleaseFromCharacter:
		var payload ReleaseFromCharacterPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case AcceptToCharacter:
		var payload AcceptToCharacterPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ReleaseFromStorage:
		var payload ReleaseFromStoragePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ValidateCharacterState:
		var payload ValidateCharacterStatePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case GainCloseness:
		var payload GainClosenessPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case TransferToCashShop:
		var payload TransferToCashShopPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case WithdrawFromCashShop:
		var payload WithdrawFromCashShopPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case AcceptToCashShop:
		var payload AcceptToCashShopPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case ReleaseFromCashShop:
		var payload ReleaseFromCashShopPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	default:
		return fmt.Errorf("unknown action: %s", s.action)
	}

	return nil
}

// Error definitions for builder validation
var (
	ErrEmptyTransactionId = errors.New("transaction ID is required")
	ErrEmptySagaType      = errors.New("saga type is required")
)
