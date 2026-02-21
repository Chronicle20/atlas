package saga

import (
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	"atlas-saga-orchestrator/validation"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-constants/world"
	sharedsaga "github.com/Chronicle20/atlas-saga"
	"github.com/google/uuid"
)

// ============================================================
// Re-exported types from shared library
// ============================================================

// Type, Action, and Status are re-exported from the shared saga library.
type (
	Type   = sharedsaga.Type
	Action = sharedsaga.Action
	Status = sharedsaga.Status
)

// Saga type constants
const (
	InventoryTransaction = sharedsaga.InventoryTransaction
	QuestReward          = sharedsaga.QuestReward
	TradeTransaction     = sharedsaga.TradeTransaction
	CharacterCreation    = sharedsaga.CharacterCreation
	StorageOperation     = sharedsaga.StorageOperation
	CharacterRespawn     = sharedsaga.CharacterRespawn
	GachaponTransaction  = sharedsaga.GachaponTransaction
)

// Status constants
const (
	Pending   = sharedsaga.Pending
	Completed = sharedsaga.Completed
	Failed    = sharedsaga.Failed
)

// Action constants
const (
	// Core inventory/asset actions
	AwardAsset           = sharedsaga.AwardAsset
	AwardExperience      = sharedsaga.AwardExperience
	AwardLevel           = sharedsaga.AwardLevel
	AwardMesos           = sharedsaga.AwardMesos
	AwardCurrency        = sharedsaga.AwardCurrency
	AwardFame            = sharedsaga.AwardFame
	DestroyAsset         = sharedsaga.DestroyAsset
	DestroyAssetFromSlot = sharedsaga.DestroyAssetFromSlot
	EquipAsset           = sharedsaga.EquipAsset
	UnequipAsset         = sharedsaga.UnequipAsset
	CreateAndEquipAsset  = sharedsaga.CreateAndEquipAsset

	// Warp actions
	WarpToRandomPortal  = sharedsaga.WarpToRandomPortal
	WarpToPortal        = sharedsaga.WarpToPortal
	WarpToSavedLocation = sharedsaga.WarpToSavedLocation
	SaveLocation        = sharedsaga.SaveLocation

	// Character state actions
	ChangeJob              = sharedsaga.ChangeJob
	ChangeHair             = sharedsaga.ChangeHair
	ChangeFace             = sharedsaga.ChangeFace
	ChangeSkin             = sharedsaga.ChangeSkin
	SetHP                  = sharedsaga.SetHP
	DeductExperience       = sharedsaga.DeductExperience
	CancelAllBuffs         = sharedsaga.CancelAllBuffs
	ResetStats             = sharedsaga.ResetStats
	ValidateCharacterState = sharedsaga.ValidateCharacterState
	IncreaseBuddyCapacity  = sharedsaga.IncreaseBuddyCapacity
	GainCloseness          = sharedsaga.GainCloseness

	// Skill actions
	CreateSkill = sharedsaga.CreateSkill
	UpdateSkill = sharedsaga.UpdateSkill

	// Quest actions
	CompleteQuest    = sharedsaga.CompleteQuest
	StartQuest       = sharedsaga.StartQuest
	SetQuestProgress = sharedsaga.SetQuestProgress

	// Consumable effect actions
	ApplyConsumableEffect  = sharedsaga.ApplyConsumableEffect
	CancelConsumableEffect = sharedsaga.CancelConsumableEffect

	// Message actions
	SendMessage = sharedsaga.SendMessage

	// UI/visual effect actions
	FieldEffect     = sharedsaga.FieldEffect
	UiLock          = sharedsaga.UiLock
	PlayPortalSound = sharedsaga.PlayPortalSound
	UpdateAreaInfo  = sharedsaga.UpdateAreaInfo
	ShowInfo        = sharedsaga.ShowInfo
	ShowInfoText    = sharedsaga.ShowInfoText
	ShowIntro       = sharedsaga.ShowIntro
	ShowHint        = sharedsaga.ShowHint
	ShowGuideHint   = sharedsaga.ShowGuideHint
	BlockPortal     = sharedsaga.BlockPortal
	UnblockPortal   = sharedsaga.UnblockPortal

	// Spawn actions
	SpawnMonster      = sharedsaga.SpawnMonster
	SpawnReactorDrops = sharedsaga.SpawnReactorDrops

	// Storage actions
	ShowStorage          = sharedsaga.ShowStorage
	DepositToStorage     = sharedsaga.DepositToStorage
	UpdateStorageMesos   = sharedsaga.UpdateStorageMesos
	TransferToStorage    = sharedsaga.TransferToStorage
	WithdrawFromStorage  = sharedsaga.WithdrawFromStorage
	AcceptToStorage      = sharedsaga.AcceptToStorage
	ReleaseFromCharacter = sharedsaga.ReleaseFromCharacter
	AcceptToCharacter    = sharedsaga.AcceptToCharacter
	ReleaseFromStorage   = sharedsaga.ReleaseFromStorage

	// Cash shop actions
	TransferToCashShop   = sharedsaga.TransferToCashShop
	WithdrawFromCashShop = sharedsaga.WithdrawFromCashShop
	AcceptToCashShop     = sharedsaga.AcceptToCashShop
	ReleaseFromCashShop  = sharedsaga.ReleaseFromCashShop

	// Guild actions
	RequestGuildName             = sharedsaga.RequestGuildName
	RequestGuildEmblem           = sharedsaga.RequestGuildEmblem
	RequestGuildDisband          = sharedsaga.RequestGuildDisband
	RequestGuildCapacityIncrease = sharedsaga.RequestGuildCapacityIncrease
	CreateInvite                 = sharedsaga.CreateInvite

	// Character creation actions
	CreateCharacter = sharedsaga.CreateCharacter

	// Transport actions
	StartInstanceTransport = sharedsaga.StartInstanceTransport

	// Gachapon actions
	SelectGachaponReward = sharedsaga.SelectGachaponReward
	EmitGachaponWin      = sharedsaga.EmitGachaponWin

	// Party quest actions
	RegisterPartyQuest         = sharedsaga.RegisterPartyQuest
	WarpPartyQuestMembersToMap = sharedsaga.WarpPartyQuestMembersToMap
	LeavePartyQuest            = sharedsaga.LeavePartyQuest
	EnterPartyQuestBonus       = sharedsaga.EnterPartyQuestBonus

	// Party quest reactor orchestration actions
	UpdatePqCustomData  = sharedsaga.UpdatePqCustomData
	HitReactor          = sharedsaga.HitReactor
	BroadcastPqMessage  = sharedsaga.BroadcastPqMessage
	StageClearAttemptPq = sharedsaga.StageClearAttemptPq

	// Field effect actions
	FieldEffectWeather = sharedsaga.FieldEffectWeather
)

// Re-exported payload types from shared library
type (
	AwardItemActionPayload               = sharedsaga.AwardItemActionPayload
	ItemPayload                          = sharedsaga.ItemPayload
	WarpToRandomPortalPayload            = sharedsaga.WarpToRandomPortalPayload
	WarpToPortalPayload                  = sharedsaga.WarpToPortalPayload
	AwardExperiencePayload               = sharedsaga.AwardExperiencePayload
	AwardLevelPayload                    = sharedsaga.AwardLevelPayload
	AwardMesosPayload                    = sharedsaga.AwardMesosPayload
	AwardCurrencyPayload                 = sharedsaga.AwardCurrencyPayload
	AwardFamePayload                     = sharedsaga.AwardFamePayload
	DestroyAssetPayload                  = sharedsaga.DestroyAssetPayload
	DestroyAssetFromSlotPayload          = sharedsaga.DestroyAssetFromSlotPayload
	EquipAssetPayload                    = sharedsaga.EquipAssetPayload
	UnequipAssetPayload                  = sharedsaga.UnequipAssetPayload
	CreateAndEquipAssetPayload           = sharedsaga.CreateAndEquipAssetPayload
	ChangeJobPayload                     = sharedsaga.ChangeJobPayload
	ChangeHairPayload                    = sharedsaga.ChangeHairPayload
	ChangeFacePayload                    = sharedsaga.ChangeFacePayload
	ChangeSkinPayload                    = sharedsaga.ChangeSkinPayload
	SetHPPayload                         = sharedsaga.SetHPPayload
	DeductExperiencePayload              = sharedsaga.DeductExperiencePayload
	CancelAllBuffsPayload                = sharedsaga.CancelAllBuffsPayload
	ResetStatsPayload                    = sharedsaga.ResetStatsPayload
	IncreaseBuddyCapacityPayload         = sharedsaga.IncreaseBuddyCapacityPayload
	GainClosenessPayload                 = sharedsaga.GainClosenessPayload
	CompleteQuestPayload                 = sharedsaga.CompleteQuestPayload
	StartQuestPayload                    = sharedsaga.StartQuestPayload
	SetQuestProgressPayload              = sharedsaga.SetQuestProgressPayload
	SendMessagePayload                   = sharedsaga.SendMessagePayload
	FieldEffectPayload                   = sharedsaga.FieldEffectPayload
	UiLockPayload                        = sharedsaga.UiLockPayload
	PlayPortalSoundPayload               = sharedsaga.PlayPortalSoundPayload
	ShowInfoPayload                      = sharedsaga.ShowInfoPayload
	ShowInfoTextPayload                  = sharedsaga.ShowInfoTextPayload
	UpdateAreaInfoPayload                = sharedsaga.UpdateAreaInfoPayload
	ShowHintPayload                      = sharedsaga.ShowHintPayload
	ShowGuideHintPayload                 = sharedsaga.ShowGuideHintPayload
	ShowIntroPayload                     = sharedsaga.ShowIntroPayload
	BlockPortalPayload                   = sharedsaga.BlockPortalPayload
	UnblockPortalPayload                 = sharedsaga.UnblockPortalPayload
	SpawnMonsterPayload                  = sharedsaga.SpawnMonsterPayload
	SpawnReactorDropsPayload             = sharedsaga.SpawnReactorDropsPayload
	ShowStoragePayload                   = sharedsaga.ShowStoragePayload
	DepositToStoragePayload              = sharedsaga.DepositToStoragePayload
	UpdateStorageMesosPayload            = sharedsaga.UpdateStorageMesosPayload
	TransferToStoragePayload             = sharedsaga.TransferToStoragePayload
	WithdrawFromStoragePayload           = sharedsaga.WithdrawFromStoragePayload
	TransferToCashShopPayload            = sharedsaga.TransferToCashShopPayload
	WithdrawFromCashShopPayload          = sharedsaga.WithdrawFromCashShopPayload
	ReleaseFromCharacterPayload          = sharedsaga.ReleaseFromCharacterPayload
	ReleaseFromStoragePayload            = sharedsaga.ReleaseFromStoragePayload
	RequestGuildNamePayload              = sharedsaga.RequestGuildNamePayload
	RequestGuildEmblemPayload            = sharedsaga.RequestGuildEmblemPayload
	RequestGuildDisbandPayload           = sharedsaga.RequestGuildDisbandPayload
	RequestGuildCapacityIncreasePayload  = sharedsaga.RequestGuildCapacityIncreasePayload
	CreateInvitePayload                  = sharedsaga.CreateInvitePayload
	CharacterCreatePayload               = sharedsaga.CharacterCreatePayload
	StartInstanceTransportPayload        = sharedsaga.StartInstanceTransportPayload
	SaveLocationPayload                  = sharedsaga.SaveLocationPayload
	WarpToSavedLocationPayload           = sharedsaga.WarpToSavedLocationPayload
	SelectGachaponRewardPayload          = sharedsaga.SelectGachaponRewardPayload
	EmitGachaponWinPayload               = sharedsaga.EmitGachaponWinPayload
	RegisterPartyQuestPayload            = sharedsaga.RegisterPartyQuestPayload
	WarpPartyQuestMembersToMapPayload    = sharedsaga.WarpPartyQuestMembersToMapPayload
	LeavePartyQuestPayload               = sharedsaga.LeavePartyQuestPayload
	EnterPartyQuestBonusPayload          = sharedsaga.EnterPartyQuestBonusPayload
	UpdatePqCustomDataPayload            = sharedsaga.UpdatePqCustomDataPayload
	HitReactorPayload                    = sharedsaga.HitReactorPayload
	BroadcastPqMessagePayload            = sharedsaga.BroadcastPqMessagePayload
	StageClearAttemptPqPayload           = sharedsaga.StageClearAttemptPqPayload
	FieldEffectWeatherPayload            = sharedsaga.FieldEffectWeatherPayload
	ExperienceDistributions              = sharedsaga.ExperienceDistributions
)

// ============================================================
// Local Saga model (immutable, private fields)
// ============================================================

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
		result:    s.steps[index].result,
	}

	return Saga{
		transactionId: s.transactionId,
		sagaType:      s.sagaType,
		initiatedBy:   s.initiatedBy,
		steps:         newSteps,
	}, nil
}

// WithStepResult returns a new Saga with the specified step's result updated
func (s Saga) WithStepResult(index int, result map[string]any) (Saga, error) {
	if index < 0 || index >= len(s.steps) {
		return Saga{}, fmt.Errorf("invalid step index: %d", index)
	}

	newSteps := make([]Step[any], len(s.steps))
	copy(newSteps, s.steps)

	newSteps[index] = Step[any]{
		stepId:    s.steps[index].stepId,
		status:    s.steps[index].status,
		action:    s.steps[index].action,
		payload:   s.steps[index].payload,
		createdAt: s.steps[index].createdAt,
		updatedAt: s.steps[index].updatedAt,
		result:    result,
	}

	return Saga{
		transactionId: s.transactionId,
		sagaType:      s.sagaType,
		initiatedBy:   s.initiatedBy,
		steps:         newSteps,
	}, nil
}

// WithStepPayload returns a new Saga with the specified step's payload updated
func (s Saga) WithStepPayload(index int, payload any) (Saga, error) {
	if index < 0 || index >= len(s.steps) {
		return Saga{}, fmt.Errorf("invalid step index: %d", index)
	}

	newSteps := make([]Step[any], len(s.steps))
	copy(newSteps, s.steps)

	newSteps[index] = Step[any]{
		stepId:    s.steps[index].stepId,
		status:    s.steps[index].status,
		action:    s.steps[index].action,
		payload:   payload,
		createdAt: s.steps[index].createdAt,
		updatedAt: s.steps[index].updatedAt,
		result:    s.steps[index].result,
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

// ============================================================
// Local Step model (immutable, private fields, has result)
// ============================================================

// Step represents a single step within a saga.
type Step[T any] struct {
	stepId    string
	status    Status
	action    Action
	payload   T
	createdAt time.Time
	updatedAt time.Time
	result    map[string]any
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

// Result returns the step result data (nil if unset)
func (s Step[T]) Result() map[string]any { return s.result }

// MarshalJSON implements json.Marshaler for Step
func (s Step[T]) MarshalJSON() ([]byte, error) {
	type alias struct {
		StepId    string         `json:"stepId"`
		Status    Status         `json:"status"`
		Action    Action         `json:"action"`
		Payload   T              `json:"payload"`
		CreatedAt time.Time      `json:"createdAt"`
		UpdatedAt time.Time      `json:"updatedAt"`
		Result    map[string]any `json:"result,omitempty"`
	}
	return json.Marshal(alias{
		StepId:    s.stepId,
		Status:    s.status,
		Action:    s.action,
		Payload:   s.payload,
		CreatedAt: s.createdAt,
		UpdatedAt: s.updatedAt,
		Result:    s.result,
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

// ============================================================
// Local payload types (divergent from shared library)
// ============================================================

// CreateSkillPayload represents the payload required to create a skill for a character.
// Local due to WorldId field not present in shared library.
type CreateSkillPayload struct {
	CharacterId uint32    `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id  `json:"worldId"`     // WorldId associated with the action
	SkillId     uint32    `json:"skillId"`     // SkillId to create
	Level       byte      `json:"level"`       // Skill level
	MasterLevel byte      `json:"masterLevel"` // Skill master level
	Expiration  time.Time `json:"expiration"`  // Skill expiration time
}

// UpdateSkillPayload represents the payload required to update a skill for a character.
// Local due to WorldId field not present in shared library.
type UpdateSkillPayload struct {
	CharacterId uint32    `json:"characterId"` // CharacterId associated with the action
	WorldId     world.Id  `json:"worldId"`     // WorldId associated with the action
	SkillId     uint32    `json:"skillId"`     // SkillId to update
	Level       byte      `json:"level"`       // New skill level
	MasterLevel byte      `json:"masterLevel"` // New skill master level
	Expiration  time.Time `json:"expiration"`  // New skill expiration time
}

// ApplyConsumableEffectPayload represents the payload required to apply consumable item effects to a character.
// Local due to character.Id and item.Id typed fields.
type ApplyConsumableEffectPayload struct {
	CharacterId character.Id `json:"characterId"` // CharacterId to apply item effects to
	WorldId     world.Id     `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id   `json:"channelId"`   // ChannelId associated with the action
	ItemId      item.Id      `json:"itemId"`      // Consumable item ID whose effects should be applied
}

// CancelConsumableEffectPayload represents the payload required to cancel consumable item effects on a character.
// Local due to character.Id and item.Id typed fields.
type CancelConsumableEffectPayload struct {
	CharacterId character.Id `json:"characterId"` // CharacterId to cancel item effects for
	WorldId     world.Id     `json:"worldId"`     // WorldId associated with the action
	ChannelId   channel.Id   `json:"channelId"`   // ChannelId associated with the action
	ItemId      item.Id      `json:"itemId"`      // Consumable item ID whose effects should be cancelled
}

// ValidateCharacterStatePayload represents the payload required to validate a character's state.
// Local due to validation.ConditionInput typed field.
type ValidateCharacterStatePayload struct {
	CharacterId uint32                      `json:"characterId"` // CharacterId associated with the action
	Conditions  []validation.ConditionInput `json:"conditions"`  // Conditions to validate
}

// AcceptToStoragePayload represents the payload for the accept_to_storage action (internal step).
// Local because it references the orchestrator-internal asset2.AssetData type.
type AcceptToStoragePayload struct {
	TransactionId uuid.UUID        `json:"transactionId"` // Saga transaction ID
	WorldId       world.Id         `json:"worldId"`       // World ID
	AccountId     uint32           `json:"accountId"`     // Account ID
	CharacterId   uint32           `json:"characterId"`   // Character initiating the transfer
	TemplateId    uint32           `json:"templateId"`    // Item template ID
	AssetData     asset2.AssetData `json:"assetData"`     // Flat asset data
}

// AcceptToCharacterPayload represents the payload for the accept_to_character action (internal step).
// Local because it references the orchestrator-internal asset2.AssetData type.
type AcceptToCharacterPayload struct {
	TransactionId uuid.UUID        `json:"transactionId"` // Saga transaction ID
	CharacterId   uint32           `json:"characterId"`   // Character ID
	InventoryType byte             `json:"inventoryType"` // Inventory type (equip, use, etc.)
	TemplateId    uint32           `json:"templateId"`    // Item template ID
	AssetData     asset2.AssetData `json:"assetData"`     // Flat asset data
}

// AcceptToCashShopPayload represents the payload for the accept_to_cash_shop action (internal step).
// Orchestrator-only payload type.
type AcceptToCashShopPayload struct {
	TransactionId   uuid.UUID `json:"transactionId"`   // Saga transaction ID
	CharacterId     uint32    `json:"characterId"`     // Character ID for session lookup
	AccountId       uint32    `json:"accountId"`       // Account ID
	CompartmentId   uuid.UUID `json:"compartmentId"`   // Cash shop compartment ID
	CompartmentType byte      `json:"compartmentType"` // Compartment type (1=Explorer, 2=Cygnus, 3=Legend)
	CashId          int64     `json:"cashId"`          // Preserved CashId from source item
	TemplateId      uint32    `json:"templateId"`      // Item template ID
	Quantity        uint32    `json:"quantity"`         // Quantity
	CommodityId     uint32    `json:"commodityId"`     // Commodity ID
	PurchasedBy     uint32    `json:"purchasedBy"`     // Who purchased the item
	Flag            uint16    `json:"flag"`             // Item flag
}

// ReleaseFromCashShopPayload represents the payload for the release_from_cash_shop action (internal step).
// Orchestrator-only payload type.
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

// ============================================================
// Step UnmarshalJSON
// ============================================================

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
		Result    map[string]any  `json:"result,omitempty"`
	}

	if err := json.Unmarshal(data, &actionOnly); err != nil {
		return err
	}

	s.stepId = actionOnly.StepId
	s.status = actionOnly.Status
	s.action = actionOnly.Action
	s.createdAt = actionOnly.CreatedAt
	s.updatedAt = actionOnly.UpdatedAt
	s.result = actionOnly.Result

	// Now handle the Payload field based on the Action type
	switch s.action {
	case AwardAsset:
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
	case FieldEffect:
		var payload FieldEffectPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case UiLock:
		var payload UiLockPayload
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
	case StartInstanceTransport:
		var payload StartInstanceTransportPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case CancelConsumableEffect:
		var payload CancelConsumableEffectPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case SaveLocation:
		var payload SaveLocationPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case WarpToSavedLocation:
		var payload WarpToSavedLocationPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case SelectGachaponReward:
		var payload SelectGachaponRewardPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case EmitGachaponWin:
		var payload EmitGachaponWinPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case RegisterPartyQuest:
		var payload RegisterPartyQuestPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case WarpPartyQuestMembersToMap:
		var payload WarpPartyQuestMembersToMapPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case LeavePartyQuest:
		var payload LeavePartyQuestPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case UpdatePqCustomData:
		var payload UpdatePqCustomDataPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case HitReactor:
		var payload HitReactorPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case BroadcastPqMessage:
		var payload BroadcastPqMessagePayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case StageClearAttemptPq:
		var payload StageClearAttemptPqPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case EnterPartyQuestBonus:
		var payload EnterPartyQuestBonusPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case FieldEffectWeather:
		var payload FieldEffectWeatherPayload
		if err := json.Unmarshal(actionOnly.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.action, err)
		}
		s.payload = any(payload).(T)
	case DeductExperience:
		var payload DeductExperiencePayload
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
