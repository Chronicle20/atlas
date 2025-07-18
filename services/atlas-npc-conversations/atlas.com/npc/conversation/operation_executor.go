package conversation

import (
	"atlas-npc-conversations/saga"
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

// OperationExecutor is the interface for executing operations in conversations
type OperationExecutor interface {
	// ExecuteOperation executes a single operation for a character
	ExecuteOperation(field field.Model, characterId uint32, operation OperationModel) error

	// ExecuteOperations executes multiple operations for a character
	ExecuteOperations(field field.Model, characterId uint32, operations []OperationModel) error
}

// OperationExecutorImpl is the implementation of the OperationExecutor interface
type OperationExecutorImpl struct {
	l     logrus.FieldLogger
	ctx   context.Context
	t     tenant.Model
	sagaP saga.Processor
}

// NewOperationExecutor creates a new operation executor
func NewOperationExecutor(l logrus.FieldLogger, ctx context.Context) OperationExecutor {
	t := tenant.MustFromContext(ctx)
	return &OperationExecutorImpl{
		l:     l,
		ctx:   ctx,
		t:     t,
		sagaP: saga.NewProcessor(l, ctx),
	}
}

// evaluateContextValue evaluates a context value, handling references to the conversation context
func (e *OperationExecutorImpl) evaluateContextValue(characterId uint32, paramName string, value string) (string, error) {
	// Check if the value is a context reference
	if strings.HasPrefix(value, "context.") {
		// Get the conversation context
		ctx, err := GetRegistry().GetPreviousContext(e.t, characterId)
		if err != nil {
			e.l.WithError(err).Errorf("Failed to get conversation context for character [%d]", characterId)
			return "", err
		}

		// Extract the context key
		contextKey := strings.TrimPrefix(value, "context.")

		// Look up the value in the context map
		contextValue, exists := ctx.Context()[contextKey]
		if !exists {
			e.l.Errorf("Context key [%s] not found in conversation context", contextKey)
			return "", fmt.Errorf("context key [%s] not found", contextKey)
		}

		return contextValue, nil
	}

	// If it's not a context reference, return the value as is
	return value, nil
}

// evaluateContextValueAsInt evaluates a context value as an integer
func (e *OperationExecutorImpl) evaluateContextValueAsInt(characterId uint32, paramName string, value string) (int, error) {
	// First evaluate the context value as a string
	strValue, err := e.evaluateContextValue(characterId, paramName, value)
	if err != nil {
		return 0, err
	}

	// Convert the string value to an integer
	intValue, err := strconv.Atoi(strValue)
	if err != nil {
		e.l.WithError(err).Errorf("Failed to convert value [%s] to integer for parameter [%s]", strValue, paramName)
		return 0, fmt.Errorf("value [%s] for parameter [%s] is not a valid integer", strValue, paramName)
	}

	return intValue, nil
}

// ExecuteOperation executes a single operation for a character
func (e *OperationExecutorImpl) ExecuteOperation(field field.Model, characterId uint32, operation OperationModel) error {
	e.l.Debugf("Executing operation [%s] for character [%d]", operation.Type(), characterId)

	// Check if this is a local operation or needs to be sent to the saga orchestrator
	if isLocalOperationType(operation.Type()) {
		return e.executeLocalOperation(field, characterId, operation)
	}

	// Create a saga for the operation
	s, err := e.createSagaForOperation(field, characterId, operation)
	if err != nil {
		e.l.WithError(err).Errorf("Failed to create saga for operation [%s]", operation.Type())
		return err
	}

	// Execute the saga
	return e.sagaP.Create(s)
}

// ExecuteOperations executes multiple operations for a character
func (e *OperationExecutorImpl) ExecuteOperations(field field.Model, characterId uint32, operations []OperationModel) error {
	e.l.Debugf("Executing %d operations for character [%d]", len(operations), characterId)

	// Group operations by type (local vs. remote)
	localOperations := make([]OperationModel, 0)
	remoteOperations := make([]OperationModel, 0)

	for _, operation := range operations {
		if isLocalOperationType(operation.Type()) {
			localOperations = append(localOperations, operation)
		} else {
			remoteOperations = append(remoteOperations, operation)
		}
	}

	// Execute local operations
	for _, operation := range localOperations {
		err := e.executeLocalOperation(field, characterId, operation)
		if err != nil {
			return err
		}
	}

	// If there are no remote operations, we're done
	if len(remoteOperations) == 0 {
		return nil
	}

	// Create a saga for the remote operations
	s, err := e.createSagaForOperations(field, characterId, remoteOperations)
	if err != nil {
		e.l.WithError(err).Errorf("Failed to create saga for remote operations")
		return err
	}

	// Execute the saga
	return e.sagaP.Create(s)
}

// isLocalOperationType checks if an operation can be executed locally
func isLocalOperationType(operationType string) bool {
	// Local operations start with "local:" prefix
	return strings.HasPrefix(operationType, "local:")
}

// executeLocalOperation executes a local operation
func (e *OperationExecutorImpl) executeLocalOperation(field field.Model, characterId uint32, operation OperationModel) error {
	// Remove the "local:" prefix
	operationType := strings.TrimPrefix(operation.Type(), "local:")

	// Execute the operation based on its type
	switch operationType {
	case "log":
		// Format: local:log
		// Context: message (string)
		messageValue, exists := operation.Params()["message"]
		if !exists {
			return errors.New("missing message parameter for log operation")
		}

		// Evaluate the message value
		message, err := e.evaluateContextValue(characterId, "message", messageValue)
		if err != nil {
			return err
		}

		e.l.Infof("NPC Log for character [%d]: %s", characterId, message)
		return nil

	case "debug":
		// Format: local:debug
		// Context: message (string)
		messageValue, exists := operation.Params()["message"]
		if !exists {
			return errors.New("missing message parameter for debug operation")
		}

		// Evaluate the message value
		message, err := e.evaluateContextValue(characterId, "message", messageValue)
		if err != nil {
			return err
		}

		e.l.Debugf("NPC Debug for character [%d]: %s", characterId, message)
		return nil

	default:
		return fmt.Errorf("unknown local operation type: %s", operationType)
	}
}

// createSagaForOperation creates a saga for a single operation
func (e *OperationExecutorImpl) createSagaForOperation(field field.Model, characterId uint32, operation OperationModel) (saga.Saga, error) {
	// Create a new saga builder
	builder := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy(fmt.Sprintf("npc-conversation-%s", operation.Type()))

	// Add a step for the operation
	stepId, status, action, payload, err := e.createStepForOperation(field, characterId, operation)
	if err != nil {
		return saga.Saga{}, err
	}
	builder.AddStep(stepId, status, action, payload)

	// Build the saga
	return builder.Build(), nil
}

// createSagaForOperations creates a saga for multiple operations
func (e *OperationExecutorImpl) createSagaForOperations(field field.Model, characterId uint32, operations []OperationModel) (saga.Saga, error) {
	// Create a new saga builder
	builder := saga.NewBuilder().
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("npc-conversation-batch")

	// Add steps for each operation
	for _, operation := range operations {
		stepId, status, action, payload, err := e.createStepForOperation(field, characterId, operation)
		if err != nil {
			return saga.Saga{}, err
		}
		builder.AddStep(stepId, status, action, payload)
	}

	// Build the saga
	return builder.Build(), nil
}

// createStepForOperation creates a saga step for an operation
func (e *OperationExecutorImpl) createStepForOperation(f field.Model, characterId uint32, operation OperationModel) (string, saga.Status, saga.Action, any, error) {
	// Generate a step ID
	stepId := fmt.Sprintf("%s-%d", operation.Type(), characterId)

	// Create a step based on the operation type
	switch operation.Type() {
	case "award_item":
		// Format: award_item
		// Context: itemId (uint32), quantity (uint32)
		itemIdValue, exists := operation.Params()["itemId"]
		if !exists {
			return "", "", "", nil, errors.New("missing itemId parameter for award_item operation")
		}

		// Evaluate the itemId value
		itemIdInt, err := e.evaluateContextValueAsInt(characterId, "itemId", itemIdValue)
		if err != nil {
			return "", "", "", nil, err
		}

		quantityValue, exists := operation.Params()["quantity"]
		if !exists {
			return "", "", "", nil, errors.New("missing quantity parameter for award_item operation")
		}

		// Evaluate the quantity value
		quantityInt, err := e.evaluateContextValueAsInt(characterId, "quantity", quantityValue)
		if err != nil {
			return "", "", "", nil, err
		}

		payload := saga.AwardItemActionPayload{
			CharacterId: characterId,
			Item: saga.ItemPayload{
				TemplateId: uint32(itemIdInt),
				Quantity:   uint32(quantityInt),
			},
		}

		return stepId, saga.Pending, saga.AwardInventory, payload, nil

	case "award_mesos":
		// Format: award_mesos
		// Context: amount (int32), actorId (uint32), actorType (string)
		amountValue, exists := operation.Params()["amount"]
		if !exists {
			return "", "", "", nil, errors.New("missing amount parameter for award_mesos operation")
		}

		// Evaluate the amount value
		amountInt, err := e.evaluateContextValueAsInt(characterId, "amount", amountValue)
		if err != nil {
			return "", "", "", nil, err
		}

		// Actor ID is optional
		var actorIdInt int = 0
		actorIdValue, exists := operation.Params()["actorId"]
		if exists {
			actorIdInt, err = e.evaluateContextValueAsInt(characterId, "actorId", actorIdValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		// Actor type is optional with default "NPC"
		actorType := "NPC"
		actorTypeValue, exists := operation.Params()["actorType"]
		if exists {
			actorType, err = e.evaluateContextValue(characterId, "actorType", actorTypeValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		payload := saga.AwardMesosPayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			ActorId:     uint32(actorIdInt),
			ActorType:   actorType,
			Amount:      int32(amountInt),
		}

		return stepId, saga.Pending, saga.AwardMesos, payload, nil

	case "award_exp":
		// Format: award_exp
		// Context: amount (uint32), type (string), attr1 (uint32)
		amountValue, exists := operation.Params()["amount"]
		if !exists {
			return "", "", "", nil, errors.New("missing amount parameter for award_exp operation")
		}

		// Evaluate the amount value
		amountInt, err := e.evaluateContextValueAsInt(characterId, "amount", amountValue)
		if err != nil {
			return "", "", "", nil, err
		}

		// Type is optional with default "WHITE"
		expType := "WHITE"
		expTypeValue, exists := operation.Params()["type"]
		if exists {
			expType, err = e.evaluateContextValue(characterId, "type", expTypeValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		// Attr1 is optional with default 0
		var attr1Int int = 0
		attr1Value, exists := operation.Params()["attr1"]
		if exists {
			attr1Int, err = e.evaluateContextValueAsInt(characterId, "attr1", attr1Value)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		payload := saga.AwardExperiencePayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			Distributions: []saga.ExperienceDistributions{
				{
					ExperienceType: expType,
					Amount:         uint32(amountInt),
					Attr1:          uint32(attr1Int),
				},
			},
		}

		return stepId, saga.Pending, saga.AwardExperience, payload, nil

	case "award_level":
		// Format: award_level
		// Context: amount (byte)
		amountValue, exists := operation.Params()["amount"]
		if !exists {
			return "", "", "", nil, errors.New("missing amount parameter for award_level operation")
		}

		// Evaluate the amount value
		amountInt, err := e.evaluateContextValueAsInt(characterId, "amount", amountValue)
		if err != nil {
			return "", "", "", nil, err
		}

		payload := saga.AwardLevelPayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			Amount:      byte(amountInt),
		}

		return stepId, saga.Pending, saga.AwardLevel, payload, nil

	case "warp_to_map":
		// Format: warp_to_map
		// Context: mapId (uint32), portalId (uint32)
		var mapIdInt int = 0
		mapIdValue, exists := operation.Params()["mapId"]
		if exists {
			var err error
			mapIdInt, err = e.evaluateContextValueAsInt(characterId, "mapId", mapIdValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		var portalIdInt int = 0
		portalIdValue, exists := operation.Params()["portalId"]
		if exists {
			var err error
			portalIdInt, err = e.evaluateContextValueAsInt(characterId, "portalId", portalIdValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		payload := saga.WarpToPortalPayload{
			CharacterId: characterId,
			FieldId:     field.NewBuilder(f.WorldId(), f.ChannelId(), _map.Id(mapIdInt)).Build().Id(),
			PortalId:    uint32(portalIdInt),
		}

		return stepId, saga.Pending, saga.WarpToPortal, payload, nil

	case "warp_to_random_portal":
		// Format: warp_to_random_portal
		// Context: mapId (uint32)
		var mapIdInt int = 0
		mapIdValue, exists := operation.Params()["mapId"]
		if exists {
			var err error
			mapIdInt, err = e.evaluateContextValueAsInt(characterId, "mapId", mapIdValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		payload := saga.WarpToRandomPortalPayload{
			CharacterId: characterId,
			FieldId:     field.NewBuilder(f.WorldId(), f.ChannelId(), _map.Id(mapIdInt)).Build().Id(),
		}

		return stepId, saga.Pending, saga.WarpToRandomPortal, payload, nil

	case "change_job":
		// Format: change_job
		// Context: jobId (uint16)
		jobIdValue, exists := operation.Params()["jobId"]
		if !exists {
			return "", "", "", nil, errors.New("missing jobId parameter for change_job operation")
		}

		// Evaluate the jobId value
		jobIdInt, err := e.evaluateContextValueAsInt(characterId, "jobId", jobIdValue)
		if err != nil {
			return "", "", "", nil, err
		}

		payload := saga.ChangeJobPayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			JobId:       job.Id(uint16(jobIdInt)),
		}

		return stepId, saga.Pending, saga.ChangeJob, payload, nil

	case "create_skill":
		// Format: create_skill
		// Context: skillId (uint32), level (byte), masterLevel (byte), expiration (time.Time)
		skillIdValue, exists := operation.Params()["skillId"]
		if !exists {
			return "", "", "", nil, errors.New("missing skillId parameter for create_skill operation")
		}

		// Evaluate the skillId value
		skillIdInt, err := e.evaluateContextValueAsInt(characterId, "skillId", skillIdValue)
		if err != nil {
			return "", "", "", nil, err
		}

		// Level is optional with default 1
		var levelInt int = 1
		levelValue, exists := operation.Params()["level"]
		if exists {
			levelInt, err = e.evaluateContextValueAsInt(characterId, "level", levelValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		// Master level is optional with default 1
		var masterLevelInt int = 1
		masterLevelValue, exists := operation.Params()["masterLevel"]
		if exists {
			masterLevelInt, err = e.evaluateContextValueAsInt(characterId, "masterLevel", masterLevelValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		payload := saga.CreateSkillPayload{
			CharacterId: characterId,
			SkillId:     uint32(skillIdInt),
			Level:       byte(levelInt),
			MasterLevel: byte(masterLevelInt),
			Expiration:  time.Now().Add(365 * 24 * time.Hour), // Default to 1 year from now
		}

		return stepId, saga.Pending, saga.CreateSkill, payload, nil

	case "update_skill":
		// Format: update_skill
		// Context: skillId (uint32), level (byte), masterLevel (byte), expiration (time.Time)
		skillIdValue, exists := operation.Params()["skillId"]
		if !exists {
			return "", "", "", nil, errors.New("missing skillId parameter for update_skill operation")
		}

		// Evaluate the skillId value
		skillIdInt, err := e.evaluateContextValueAsInt(characterId, "skillId", skillIdValue)
		if err != nil {
			return "", "", "", nil, err
		}

		// Level is optional with default 1
		var levelInt int = 1
		levelValue, exists := operation.Params()["level"]
		if exists {
			levelInt, err = e.evaluateContextValueAsInt(characterId, "level", levelValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		// Master level is optional with default 1
		var masterLevelInt int = 1
		masterLevelValue, exists := operation.Params()["masterLevel"]
		if exists {
			masterLevelInt, err = e.evaluateContextValueAsInt(characterId, "masterLevel", masterLevelValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		payload := saga.UpdateSkillPayload{
			CharacterId: characterId,
			SkillId:     uint32(skillIdInt),
			Level:       byte(levelInt),
			MasterLevel: byte(masterLevelInt),
			Expiration:  time.Now().Add(365 * 24 * time.Hour), // Default to 1 year from now
		}

		return stepId, saga.Pending, saga.UpdateSkill, payload, nil

	case "destroy_item":
		// Format: destroy_item
		// Context: itemId (uint32), quantity (uint32)
		itemIdValue, exists := operation.Params()["itemId"]
		if !exists {
			return "", "", "", nil, errors.New("missing itemId parameter for destroy_item operation")
		}

		// Evaluate the itemId value
		itemIdInt, err := e.evaluateContextValueAsInt(characterId, "itemId", itemIdValue)
		if err != nil {
			return "", "", "", nil, err
		}

		quantityValue, exists := operation.Params()["quantity"]
		if !exists {
			return "", "", "", nil, errors.New("missing quantity parameter for destroy_item operation")
		}

		// Evaluate the quantity value
		quantityInt, err := e.evaluateContextValueAsInt(characterId, "quantity", quantityValue)
		if err != nil {
			return "", "", "", nil, err
		}

		payload := saga.DestroyAssetPayload{
			CharacterId: characterId,
			TemplateId:  uint32(itemIdInt),
			Quantity:    uint32(quantityInt),
		}

		return stepId, saga.Pending, saga.DestroyAsset, payload, nil

	default:
		return "", "", "", nil, fmt.Errorf("unknown operation type: %s", operation.Type())
	}
}
