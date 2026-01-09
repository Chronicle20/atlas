package conversation

import (
	"atlas-npc-conversations/cosmetic"
	npcMap "atlas-npc-conversations/map"
	"atlas-npc-conversations/pet"
	"atlas-npc-conversations/saga"
	"atlas-npc-conversations/validation"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
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
	l           logrus.FieldLogger
	ctx         context.Context
	t           tenant.Model
	sagaP       saga.Processor
	petP        pet.Processor
	cosmeticP   cosmetic.Processor
	mapP        npcMap.Processor
	validationP validation.Processor
}

// NewOperationExecutor creates a new operation executor
func NewOperationExecutor(l logrus.FieldLogger, ctx context.Context) OperationExecutor {
	t := tenant.MustFromContext(ctx)
	appearanceProvider := cosmetic.NewRestAppearanceProvider(l, ctx)
	return &OperationExecutorImpl{
		l:           l,
		ctx:         ctx,
		t:           t,
		sagaP:       saga.NewProcessor(l, ctx),
		petP:        pet.NewProcessor(l, ctx),
		cosmeticP:   cosmetic.NewProcessor(l, ctx, appearanceProvider),
		mapP:        npcMap.NewProcessor(l, ctx),
		validationP: validation.NewProcessor(l, ctx),
	}
}

// inventoryCheckerImpl implements cosmetic.InventoryChecker using the validation processor
type inventoryCheckerImpl struct {
	l           logrus.FieldLogger
	validationP validation.Processor
}

// HasItem checks if a character has at least one of the specified item
func (c *inventoryCheckerImpl) HasItem(characterId uint32, itemId uint32) (bool, error) {
	condition := validation.ConditionInput{
		Type:        "item",
		Operator:    ">=",
		Value:       1,
		ReferenceId: itemId,
	}

	result, err := c.validationP.ValidateCharacterState(characterId, []validation.ConditionInput{condition})
	if err != nil {
		c.l.WithError(err).Errorf("Failed to check item %d for character %d", itemId, characterId)
		return false, err
	}

	return result.Passed(), nil
}

// evaluateContextValue evaluates a context value, handling references to the conversation context
// Supports both "{context.xxx}" and "context.xxx" formats
func (e *OperationExecutorImpl) evaluateContextValue(characterId uint32, paramName string, value string) (string, error) {
	// Get the conversation context
	ctx, err := GetRegistry().GetPreviousContext(e.t, characterId)
	if err != nil {
		e.l.WithError(err).Errorf("Failed to get conversation context for character [%d]", characterId)
		return "", err
	}

	// Use the new ExtractContextValue function which supports both formats
	extractedValue, isContextRef, err := ExtractContextValue(value, ctx.Context())
	if err != nil {
		e.l.WithError(err).Errorf("Failed to extract context value for parameter [%s]", paramName)
		return "", err
	}

	if isContextRef {
		e.l.Debugf("Resolved context reference [%s] to [%s] for character [%d]", value, extractedValue, characterId)
	}

	return extractedValue, nil
}

// getContextValue retrieves a value from the conversation context by key
func (e *OperationExecutorImpl) getContextValue(characterId uint32, key string) (string, error) {
	ctx, err := GetRegistry().GetPreviousContext(e.t, characterId)
	if err != nil {
		return "", fmt.Errorf("failed to get conversation context for character [%d]: %w", characterId, err)
	}

	value, exists := ctx.Context()[key]
	if !exists {
		return "", fmt.Errorf("context key '%s' not found for character [%d]", key, characterId)
	}

	return value, nil
}

// setContextValue stores a value in the conversation context
func (e *OperationExecutorImpl) setContextValue(characterId uint32, key string, value string) error {
	ctx, err := GetRegistry().GetPreviousContext(e.t, characterId)
	if err != nil {
		return fmt.Errorf("failed to get conversation context for character [%d]: %w", characterId, err)
	}

	// Update the context map
	contextMap := ctx.Context()
	if contextMap == nil {
		return fmt.Errorf("context map is nil for character [%d]", characterId)
	}
	contextMap[key] = value

	// Save the updated context back to the registry
	GetRegistry().UpdateContext(e.t, characterId, ctx)

	return nil
}

// evaluateArithmeticExpression evaluates simple arithmetic expressions
// Supports: +, -, *, / operators
// Example: "10 * 5" -> 50, "100 / 2" -> 50
func evaluateArithmeticExpression(expr string) (int, error) {
	expr = strings.TrimSpace(expr)

	// Check for multiplication
	if strings.Contains(expr, "*") {
		parts := strings.Split(expr, "*")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid multiplication expression: %s", expr)
		}
		left, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, fmt.Errorf("invalid left operand: %s", parts[0])
		}
		right, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, fmt.Errorf("invalid right operand: %s", parts[1])
		}
		return left * right, nil
	}

	// Check for division
	if strings.Contains(expr, "/") {
		parts := strings.Split(expr, "/")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid division expression: %s", expr)
		}
		left, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, fmt.Errorf("invalid left operand: %s", parts[0])
		}
		right, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, fmt.Errorf("invalid right operand: %s", parts[1])
		}
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return left / right, nil
	}

	// Check for addition
	if strings.Contains(expr, "+") {
		parts := strings.Split(expr, "+")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid addition expression: %s", expr)
		}
		left, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, fmt.Errorf("invalid left operand: %s", parts[0])
		}
		right, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, fmt.Errorf("invalid right operand: %s", parts[1])
		}
		return left + right, nil
	}

	// Check for subtraction (but not negative numbers)
	if strings.Contains(expr, "-") && !strings.HasPrefix(expr, "-") {
		parts := strings.Split(expr, "-")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid subtraction expression: %s", expr)
		}
		left, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, fmt.Errorf("invalid left operand: %s", parts[0])
		}
		right, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, fmt.Errorf("invalid right operand: %s", parts[1])
		}
		return left - right, nil
	}

	// No operator found, try to parse as integer
	return strconv.Atoi(expr)
}

// evaluateContextValueAsInt evaluates a context value as an integer
// Supports arithmetic expressions like "10 * {context.quantity}"
func (e *OperationExecutorImpl) evaluateContextValueAsInt(characterId uint32, paramName string, value string) (int, error) {
	// First evaluate the context value as a string (replaces {context.xxx} with actual values)
	strValue, err := e.evaluateContextValue(characterId, paramName, value)
	if err != nil {
		return 0, err
	}

	// Check if the result contains arithmetic operators
	if strings.ContainsAny(strValue, "+-*/") {
		// Evaluate arithmetic expression
		intValue, err := evaluateArithmeticExpression(strValue)
		if err != nil {
			e.l.WithError(err).Errorf("Failed to evaluate arithmetic expression [%s] for parameter [%s]", strValue, paramName)
			return 0, fmt.Errorf("arithmetic evaluation failed for [%s]: %w", strValue, err)
		}
		return intValue, nil
	}

	// No arithmetic, convert directly to integer
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

	// Execute the saga with enhanced error handling
	err = e.sagaP.Create(s)
	if err != nil {
		e.l.WithError(err).Errorf("Failed to create saga for operation [%s] - saga orchestrator communication failed", operation.Type())
		return fmt.Errorf("saga orchestrator communication failed: %w", err)
	}

	return nil
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

	// Execute the saga with enhanced error handling
	err = e.sagaP.Create(s)
	if err != nil {
		e.l.WithError(err).Errorf("Failed to create saga for remote operations - saga orchestrator communication failed")
		return fmt.Errorf("saga orchestrator communication failed for remote operations: %w", err)
	}

	return nil
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

	case "generate_hair_styles":
		// Format: local:generate_hair_styles
		// Params: baseStyles (string), genderFilter (string), preserveColor (string),
		//         validateExists (string), excludeEquipped (string), outputContextKey (string)
		styles, err := e.cosmeticP.GenerateHairStyles(characterId, operation.Params())
		if err != nil {
			e.l.WithError(err).Errorf("Failed to generate hair styles for character [%d]", characterId)
			return fmt.Errorf("failed to generate hair styles: %w", err)
		}

		// Store in context
		outputKey := operation.Params()["outputContextKey"]
		if outputKey == "" {
			outputKey = "generatedStyles"
		}

		err = e.storeStylesInContext(characterId, outputKey, styles)
		if err != nil {
			e.l.WithError(err).Errorf("Failed to store hair styles in context for character [%d]", characterId)
			return err
		}

		e.l.Infof("Generated and stored %d hair styles in context key [%s] for character [%d]",
			len(styles), outputKey, characterId)
		return nil

	case "generate_hair_colors":
		// Format: local:generate_hair_colors
		// Params: colors (string), outputContextKey (string)
		colors, err := e.cosmeticP.GenerateHairColors(characterId, operation.Params())
		if err != nil {
			e.l.WithError(err).Errorf("Failed to generate hair colors for character [%d]", characterId)
			return fmt.Errorf("failed to generate hair colors: %w", err)
		}

		// Store in context
		outputKey := operation.Params()["outputContextKey"]
		if outputKey == "" {
			outputKey = "generatedColors"
		}

		err = e.storeStylesInContext(characterId, outputKey, colors)
		if err != nil {
			e.l.WithError(err).Errorf("Failed to store hair colors in context for character [%d]", characterId)
			return err
		}

		e.l.Infof("Generated and stored %d hair colors in context key [%s] for character [%d]",
			len(colors), outputKey, characterId)
		return nil

	case "generate_face_styles":
		// Format: local:generate_face_styles
		// Params: baseStyles (string), genderFilter (string), validateExists (string),
		//         excludeEquipped (string), outputContextKey (string)
		faces, err := e.cosmeticP.GenerateFaceStyles(characterId, operation.Params())
		if err != nil {
			e.l.WithError(err).Errorf("Failed to generate face styles for character [%d]", characterId)
			return fmt.Errorf("failed to generate face styles: %w", err)
		}

		// Store in context
		outputKey := operation.Params()["outputContextKey"]
		if outputKey == "" {
			outputKey = "generatedFaces"
		}

		err = e.storeStylesInContext(characterId, outputKey, faces)
		if err != nil {
			e.l.WithError(err).Errorf("Failed to store face styles in context for character [%d]", characterId)
			return err
		}

		e.l.Infof("Generated and stored %d face styles in context key [%s] for character [%d]",
			len(faces), outputKey, characterId)
		return nil

	case "generate_face_colors":
		// Format: local:generate_face_colors
		// Params: colorOffsets (string, comma-separated offsets like "100,300,400,700"),
		//         validateExists (string), excludeEquipped (string), outputContextKey (string)
		// Used for cosmetic lens NPCs that change eye/face color
		colors, err := e.cosmeticP.GenerateFaceColors(characterId, operation.Params())
		if err != nil {
			e.l.WithError(err).Errorf("Failed to generate face colors for character [%d]", characterId)
			return fmt.Errorf("failed to generate face colors: %w", err)
		}

		// Store in context
		outputKey := operation.Params()["outputContextKey"]
		if outputKey == "" {
			outputKey = "generatedFaceColors"
		}

		err = e.storeStylesInContext(characterId, outputKey, colors)
		if err != nil {
			e.l.WithError(err).Errorf("Failed to store face colors in context for character [%d]", characterId)
			return err
		}

		e.l.Infof("Generated and stored %d face colors in context key [%s] for character [%d]",
			len(colors), outputKey, characterId)
		return nil

	case "select_random_cosmetic":
		// Format: local:select_random_cosmetic
		// Params: stylesContextKey (string), outputContextKey (string)
		stylesContextKey, exists := operation.Params()["stylesContextKey"]
		if !exists {
			return errors.New("missing stylesContextKey parameter for select_random_cosmetic operation")
		}

		outputContextKey, exists := operation.Params()["outputContextKey"]
		if !exists {
			return errors.New("missing outputContextKey parameter for select_random_cosmetic operation")
		}

		// Get the styles array from context
		stylesValue, err := e.getContextValue(characterId, stylesContextKey)
		if err != nil {
			return fmt.Errorf("failed to get styles from context key '%s': %w", stylesContextKey, err)
		}

		// Parse the styles array (should be a JSON array of uint32)
		styles := strings.Split(stylesValue, ",")
		if len(styles) == 0 {
			return errors.New("styles array is empty, cannot select random cosmetic")
		}

		// Randomly select one style
		selectedStyle := styles[rand.Intn(len(styles))]

		// Store the selected style in the output context key
		err = e.setContextValue(characterId, outputContextKey, fmt.Sprintf("%s", selectedStyle))
		if err != nil {
			return fmt.Errorf("failed to store selected style in context: %w", err)
		}

		e.l.Infof("Selected random cosmetic %d from %d options for character [%d], stored in context key [%s]",
			selectedStyle, len(styles), characterId, outputContextKey)
		return nil

	case "select_random_weighted":
		// Format: local:select_random_weighted
		// Params: items (comma-separated string of values), weights (comma-separated string of integers), outputContextKey (string)
		// Example: items="1040052,1040054,1040130", weights="10,10,15", outputContextKey="selectedItem"
		itemsValue, exists := operation.Params()["items"]
		if !exists {
			return errors.New("missing items parameter for select_random_weighted operation")
		}

		weightsValue, exists := operation.Params()["weights"]
		if !exists {
			return errors.New("missing weights parameter for select_random_weighted operation")
		}

		outputContextKey, exists := operation.Params()["outputContextKey"]
		if !exists {
			return errors.New("missing outputContextKey parameter for select_random_weighted operation")
		}

		// Parse items (comma-separated string)
		itemsStr := strings.TrimSpace(itemsValue)
		if itemsStr == "" {
			return errors.New("items parameter is empty")
		}
		itemList := strings.Split(itemsStr, ",")
		for i := range itemList {
			itemList[i] = strings.TrimSpace(itemList[i])
		}

		// Parse weights (comma-separated string of integers)
		weightsStr := strings.TrimSpace(weightsValue)
		if weightsStr == "" {
			return errors.New("weights parameter is empty")
		}
		weightStrs := strings.Split(weightsStr, ",")
		weights := make([]int, len(weightStrs))
		for i, weightStr := range weightStrs {
			weight, err := strconv.Atoi(strings.TrimSpace(weightStr))
			if err != nil {
				return fmt.Errorf("invalid weight value '%s': %w", weightStr, err)
			}
			if weight < 0 {
				return fmt.Errorf("weight value must be non-negative, got %d", weight)
			}
			weights[i] = weight
		}

		// Validate items and weights have the same length
		if len(itemList) != len(weights) {
			return fmt.Errorf("items and weights must have the same length (items: %d, weights: %d)", len(itemList), len(weights))
		}

		// Calculate total weight
		totalWeight := 0
		for _, weight := range weights {
			totalWeight += weight
		}

		if totalWeight == 0 {
			return errors.New("total weight is zero, cannot perform weighted random selection")
		}

		// Perform weighted random selection
		randomPick := rand.Intn(totalWeight)
		selectedItem := ""
		for i := range itemList {
			randomPick -= weights[i]
			if randomPick < 0 {
				selectedItem = itemList[i]
				break
			}
		}

		// Store the selected item in the output context key
		err := e.setContextValue(characterId, outputContextKey, selectedItem)
		if err != nil {
			return fmt.Errorf("failed to store selected item in context: %w", err)
		}

		e.l.Infof("Selected random weighted item '%s' from %d options (total weight: %d) for character [%d], stored in context key [%s]",
			selectedItem, len(itemList), totalWeight, characterId, outputContextKey)
		return nil

	case "calculate_lens_coupon":
		// Format: local:calculate_lens_coupon
		// Params: selectedFaceContextKey (string), outputContextKey (string)
		// Calculates the one-time lens item ID based on the selected face color
		// Formula: itemId = 5152100 + (selectedFace / 100) % 10
		// This maps face colors (0-7) to items 5152100-5152107
		selectedFaceContextKey, exists := operation.Params()["selectedFaceContextKey"]
		if !exists {
			return errors.New("missing selectedFaceContextKey parameter for calculate_lens_coupon operation")
		}

		outputContextKey, exists := operation.Params()["outputContextKey"]
		if !exists {
			return errors.New("missing outputContextKey parameter for calculate_lens_coupon operation")
		}

		// Get the selected face from context
		selectedFaceStr, err := e.getContextValue(characterId, selectedFaceContextKey)
		if err != nil {
			return fmt.Errorf("failed to get selected face from context key '%s': %w", selectedFaceContextKey, err)
		}

		// Parse the face ID
		selectedFace, err := strconv.ParseUint(selectedFaceStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid selected face value '%s': %w", selectedFaceStr, err)
		}

		// Calculate the color from face ID: (faceId / 100) % 10
		// Face IDs like 20000, 20100, 20200, etc. have color encoded in the hundreds place
		color := (selectedFace / 100) % 10

		// Calculate the lens item ID: 5152100 + color
		lensItemId := 5152100 + color

		// Store the calculated item ID in context
		err = e.setContextValue(characterId, outputContextKey, strconv.FormatUint(lensItemId, 10))
		if err != nil {
			return fmt.Errorf("failed to store lens item ID in context: %w", err)
		}

		e.l.Infof("Calculated lens coupon item ID %d for face %d (color %d) for character [%d], stored in context key [%s]",
			lensItemId, selectedFace, color, characterId, outputContextKey)
		return nil

	case "fetch_map_player_counts":
		// Format: local:fetch_map_player_counts
		// Params: mapIds (comma-separated string of map IDs)
		mapIdsValue, exists := operation.Params()["mapIds"]
		if !exists {
			return errors.New("missing mapIds parameter for fetch_map_player_counts operation")
		}

		// Evaluate the mapIds value (supports context references)
		mapIdsStr, err := e.evaluateContextValue(characterId, "mapIds", mapIdsValue)
		if err != nil {
			return err
		}

		// Parse comma-separated map IDs
		mapIdStrs := strings.Split(mapIdsStr, ",")
		mapIds := make([]uint32, 0, len(mapIdStrs))
		for _, mapIdStr := range mapIdStrs {
			trimmed := strings.TrimSpace(mapIdStr)
			if trimmed == "" {
				continue
			}
			mapId, err := strconv.ParseUint(trimmed, 10, 32)
			if err != nil {
				e.l.WithError(err).Errorf("Invalid map ID '%s' in mapIds parameter", trimmed)
				return fmt.Errorf("invalid map ID '%s': %w", trimmed, err)
			}
			mapIds = append(mapIds, uint32(mapId))
		}

		if len(mapIds) == 0 {
			return errors.New("no valid map IDs provided in mapIds parameter")
		}

		// Get world and channel from field
		worldId := byte(field.WorldId())
		channelId := byte(field.ChannelId())

		// Fetch player counts for all maps in parallel
		counts, err := e.mapP.GetPlayerCountsInMaps(worldId, channelId, mapIds)
		if err != nil {
			// Log warning but don't fail - graceful degradation
			e.l.WithError(err).Warnf("Failed to fetch player counts for maps, using 0 for all")
		}

		// Store each count in the context with key: playerCount_{mapId}
		for _, mapId := range mapIds {
			count := 0
			if counts != nil {
				count = counts[mapId]
			}
			key := fmt.Sprintf("playerCount_%d", mapId)
			err = e.setContextValue(characterId, key, strconv.Itoa(count))
			if err != nil {
				e.l.WithError(err).Errorf("Failed to store player count for map [%d] in context", mapId)
				return fmt.Errorf("failed to store player count in context: %w", err)
			}
		}

		e.l.Infof("Fetched and stored player counts for %d maps for character [%d]", len(mapIds), characterId)
		return nil

	case "generate_face_colors_for_onetime_lens":
		// Format: local:generate_face_colors_for_onetime_lens
		// Params: validateExists (string), excludeEquipped (string), outputContextKey (string)
		// Checks which one-time lens items (5152100-5152107) the character owns
		// and generates corresponding face colors for those items

		// Create inventory checker
		inventoryChecker := &inventoryCheckerImpl{
			l:           e.l,
			validationP: e.validationP,
		}

		// Call cosmetic processor with inventory checker
		colors, err := e.cosmeticP.GenerateFaceColorsForOnetimeLens(characterId, inventoryChecker, operation.Params())
		if err != nil {
			e.l.WithError(err).Errorf("Failed to generate face colors for one-time lens for character [%d]", characterId)
			return fmt.Errorf("failed to generate face colors for one-time lens: %w", err)
		}

		// Store in context
		outputKey := operation.Params()["outputContextKey"]
		if outputKey == "" {
			outputKey = "onetimeLensColors"
		}

		err = e.storeStylesInContext(characterId, outputKey, colors)
		if err != nil {
			e.l.WithError(err).Errorf("Failed to store one-time lens colors in context for character [%d]", characterId)
			return err
		}

		e.l.Infof("Generated and stored %d one-time lens face colors in context key [%s] for character [%d]",
			len(colors), outputKey, characterId)
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
		// Context: itemId (uint32), quantity (uint32), expiration (int64 milliseconds, optional)
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

		// Evaluate the optional expiration value (in milliseconds from now)
		var expiration time.Time
		if expirationValue, hasExpiration := operation.Params()["expiration"]; hasExpiration {
			expirationMs, err := e.evaluateContextValueAsInt(characterId, "expiration", expirationValue)
			if err != nil {
				return "", "", "", nil, err
			}
			if expirationMs > 0 {
				expiration = time.Now().Add(time.Duration(expirationMs) * time.Millisecond)
			}
		}

		payload := saga.AwardItemActionPayload{
			CharacterId: characterId,
			Item: saga.ItemPayload{
				TemplateId: uint32(itemIdInt),
				Quantity:   uint32(quantityInt),
				Expiration: expiration,
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
		// Context: mapId (uint32), portalId (uint32) OR portalName (string)
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

		var portalName string
		portalNameValue, exists := operation.Params()["portalName"]
		if exists {
			var err error
			portalName, err = e.evaluateContextValue(characterId, "portalName", portalNameValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		payload := saga.WarpToPortalPayload{
			CharacterId: characterId,
			FieldId:     field.NewBuilder(f.WorldId(), f.ChannelId(), _map.Id(mapIdInt)).Build().Id(),
			PortalId:    uint32(portalIdInt),
			PortalName:  portalName,
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

	case "increase_buddy_capacity":
		// Format: increase_buddy_capacity
		// Context: amount (byte)
		amountValue, exists := operation.Params()["amount"]
		if !exists {
			return "", "", "", nil, errors.New("missing amount parameter for increase_buddy_capacity operation")
		}

		// Evaluate the amount value
		amountInt, err := e.evaluateContextValueAsInt(characterId, "amount", amountValue)
		if err != nil {
			return "", "", "", nil, err
		}

		payload := saga.IncreaseBuddyCapacityPayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			Amount:      byte(amountInt),
		}

		return stepId, saga.Pending, saga.IncreaseBuddyCapacity, payload, nil

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
		// Params: itemId (uint32, required), quantity (uint32, optional, default 0), removeAll (bool, optional, default false)
		itemIdValue, exists := operation.Params()["itemId"]
		if !exists {
			return "", "", "", nil, errors.New("missing itemId parameter for destroy_item operation")
		}

		// Evaluate the itemId value
		itemIdInt, err := e.evaluateContextValueAsInt(characterId, "itemId", itemIdValue)
		if err != nil {
			return "", "", "", nil, err
		}

		// Quantity is optional, defaults to 0 (ignored when removeAll is true)
		quantityInt := 0
		if quantityValue, exists := operation.Params()["quantity"]; exists {
			quantityInt, err = e.evaluateContextValueAsInt(characterId, "quantity", quantityValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		// Check if removeAll parameter is present
		removeAll := false
		if removeAllValue, exists := operation.Params()["removeAll"]; exists {
			removeAllStr, err := e.evaluateContextValue(characterId, "removeAll", removeAllValue)
			if err != nil {
				return "", "", "", nil, err
			}
			removeAll = removeAllStr == "true"
		}

		payload := saga.DestroyAssetPayload{
			CharacterId: characterId,
			TemplateId:  uint32(itemIdInt),
			Quantity:    uint32(quantityInt),
			RemoveAll:   removeAll,
		}

		return stepId, saga.Pending, saga.DestroyAsset, payload, nil

	case "gain_closeness":
		// Format: gain_closeness
		// Supports either petId (uint32) or petIndex (int8) + characterId lookup
		// When petIndex is used, the pet at that slot for the character is resolved
		var petId uint32

		// Check if petId is provided directly
		if petIdValue, exists := operation.Params()["petId"]; exists {
			petIdInt, err := e.evaluateContextValueAsInt(characterId, "petId", petIdValue)
			if err != nil {
				return "", "", "", nil, err
			}
			petId = uint32(petIdInt)
		} else if petIndexValue, exists := operation.Params()["petIndex"]; exists {
			// petIndex is provided, need to resolve to petId
			petIndexInt, err := e.evaluateContextValueAsInt(characterId, "petIndex", petIndexValue)
			if err != nil {
				return "", "", "", nil, err
			}

			// Query pets for the character and find the one at the specified slot
			petIdResult, err := e.petP.GetPetIdBySlot(characterId, int8(petIndexInt))()
			if err != nil {
				return "", "", "", nil, fmt.Errorf("failed to resolve pet at slot %d for character %d: %w", petIndexInt, characterId, err)
			}
			petId = petIdResult
		} else {
			return "", "", "", nil, errors.New("missing petId or petIndex parameter for gain_closeness operation")
		}

		amountValue, exists := operation.Params()["amount"]
		if !exists {
			return "", "", "", nil, errors.New("missing amount parameter for gain_closeness operation")
		}

		// Evaluate the amount value
		amountInt, err := e.evaluateContextValueAsInt(characterId, "amount", amountValue)
		if err != nil {
			return "", "", "", nil, err
		}

		payload := saga.GainClosenessPayload{
			PetId:  petId,
			Amount: uint16(amountInt),
		}

		return stepId, saga.Pending, saga.GainCloseness, payload, nil

	case "change_hair":
		// Format: change_hair
		// Context: styleId (uint32)
		styleIdValue, exists := operation.Params()["styleId"]
		if !exists {
			return "", "", "", nil, errors.New("missing styleId parameter for change_hair operation")
		}

		// Evaluate the styleId value
		styleIdInt, err := e.evaluateContextValueAsInt(characterId, "styleId", styleIdValue)
		if err != nil {
			return "", "", "", nil, err
		}

		payload := saga.ChangeHairPayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			StyleId:     uint32(styleIdInt),
		}

		return stepId, saga.Pending, saga.ChangeHair, payload, nil

	case "change_face":
		// Format: change_face
		// Context: styleId (uint32)
		styleIdValue, exists := operation.Params()["styleId"]
		if !exists {
			return "", "", "", nil, errors.New("missing styleId parameter for change_face operation")
		}

		// Evaluate the styleId value
		styleIdInt, err := e.evaluateContextValueAsInt(characterId, "styleId", styleIdValue)
		if err != nil {
			return "", "", "", nil, err
		}

		payload := saga.ChangeFacePayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			StyleId:     uint32(styleIdInt),
		}

		return stepId, saga.Pending, saga.ChangeFace, payload, nil

	case "change_skin":
		// Format: change_skin
		// Context: styleId (byte)
		styleIdValue, exists := operation.Params()["styleId"]
		if !exists {
			return "", "", "", nil, errors.New("missing styleId parameter for change_skin operation")
		}

		// Evaluate the styleId value
		styleIdInt, err := e.evaluateContextValueAsInt(characterId, "styleId", styleIdValue)
		if err != nil {
			return "", "", "", nil, err
		}

		payload := saga.ChangeSkinPayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			StyleId:     byte(styleIdInt),
		}

		return stepId, saga.Pending, saga.ChangeSkin, payload, nil

	case "spawn_monster":
		// Format: spawn_monster
		// Params: monsterId (uint32), x (int16), y (int16), mapId (uint32, optional - defaults to character's current map),
		//         count (int, optional, default 1), team (int8, optional, default 0)
		// Note: Foothold is resolved by saga-orchestrator via atlas-data lookup
		monsterIdValue, exists := operation.Params()["monsterId"]
		if !exists {
			return "", "", "", nil, errors.New("missing monsterId parameter for spawn_monster operation")
		}

		monsterIdInt, err := e.evaluateContextValueAsInt(characterId, "monsterId", monsterIdValue)
		if err != nil {
			return "", "", "", nil, err
		}

		xValue, exists := operation.Params()["x"]
		if !exists {
			return "", "", "", nil, errors.New("missing x parameter for spawn_monster operation")
		}

		xInt, err := e.evaluateContextValueAsInt(characterId, "x", xValue)
		if err != nil {
			return "", "", "", nil, err
		}

		yValue, exists := operation.Params()["y"]
		if !exists {
			return "", "", "", nil, errors.New("missing y parameter for spawn_monster operation")
		}

		yInt, err := e.evaluateContextValueAsInt(characterId, "y", yValue)
		if err != nil {
			return "", "", "", nil, err
		}

		// MapId is optional, defaults to character's current map
		mapIdInt := int(f.MapId())
		if mapIdValue, exists := operation.Params()["mapId"]; exists {
			mapIdInt, err = e.evaluateContextValueAsInt(characterId, "mapId", mapIdValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		// Count is optional, defaults to 1
		countInt := 1
		if countValue, exists := operation.Params()["count"]; exists {
			countInt, err = e.evaluateContextValueAsInt(characterId, "count", countValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		// Team is optional, defaults to 0
		teamInt := 0
		if teamValue, exists := operation.Params()["team"]; exists {
			teamInt, err = e.evaluateContextValueAsInt(characterId, "team", teamValue)
			if err != nil {
				return "", "", "", nil, err
			}
		}

		payload := saga.SpawnMonsterPayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			MapId:       uint32(mapIdInt),
			MonsterId:   uint32(monsterIdInt),
			X:           int16(xInt),
			Y:           int16(yInt),
			Team:        int8(teamInt),
			Count:       countInt,
		}

		return stepId, saga.Pending, saga.SpawnMonster, payload, nil

	case "complete_quest":
		// Format: complete_quest
		// Params: questId (uint32), npcId (uint32, optional - defaults to conversation NPC),
		//         force (bool, optional - if true, skip requirement checks)
		var questIdInt int
		var err error
		if questIdValue, exists := operation.Params()["questId"]; exists {
			questIdInt, err = e.evaluateContextValueAsInt(characterId, "questId", questIdValue)
			if err != nil {
				return "", "", "", nil, err
			}
		} else {
			// Check context for questId (set by quest conversations)
			ctx, err := GetRegistry().GetPreviousContext(e.t, characterId)
			if err != nil {
				return "", "", "", nil, fmt.Errorf("failed to get conversation context for questId: %w", err)
			}
			if contextQuestId, exists := ctx.Context()["questId"]; exists {
				questIdInt, err = strconv.Atoi(contextQuestId)
				if err != nil {
					return "", "", "", nil, fmt.Errorf("invalid questId in context: %w", err)
				}
			} else {
				return "", "", "", nil, errors.New("missing questId parameter for complete_quest operation")
			}
		}

		// NpcId is optional - if not provided, get from conversation context
		var npcIdInt int
		if npcIdValue, exists := operation.Params()["npcId"]; exists {
			npcIdInt, err = e.evaluateContextValueAsInt(characterId, "npcId", npcIdValue)
			if err != nil {
				return "", "", "", nil, err
			}
		} else {
			// Get NPC ID from conversation context
			ctx, err := GetRegistry().GetPreviousContext(e.t, characterId)
			if err != nil {
				return "", "", "", nil, fmt.Errorf("failed to get conversation context for NPC ID: %w", err)
			}
			npcIdInt = int(ctx.NpcId())
		}

		// Force is optional - if true, skip requirement validation (forceCompleteQuest behavior)
		force := false
		if forceValue, exists := operation.Params()["force"]; exists {
			forceStr, err := e.evaluateContextValue(characterId, "force", forceValue)
			if err != nil {
				return "", "", "", nil, err
			}
			force = forceStr == "true"
		}

		payload := saga.CompleteQuestPayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			QuestId:     uint32(questIdInt),
			NpcId:       uint32(npcIdInt),
			Force:       force,
		}

		return stepId, saga.Pending, saga.CompleteQuest, payload, nil

	case "start_quest":
		// Format: start_quest
		// Params: questId (uint32, optional - defaults to context questId for quest conversations),
		//         npcId (uint32, optional - defaults to conversation NPC)
		var questIdInt int
		var err error
		if questIdValue, exists := operation.Params()["questId"]; exists {
			questIdInt, err = e.evaluateContextValueAsInt(characterId, "questId", questIdValue)
			if err != nil {
				return "", "", "", nil, err
			}
		} else {
			// Check context for questId (set by quest conversations)
			ctx, err := GetRegistry().GetPreviousContext(e.t, characterId)
			if err != nil {
				return "", "", "", nil, fmt.Errorf("failed to get conversation context for questId: %w", err)
			}
			if contextQuestId, exists := ctx.Context()["questId"]; exists {
				questIdInt, err = strconv.Atoi(contextQuestId)
				if err != nil {
					return "", "", "", nil, fmt.Errorf("invalid questId in context: %w", err)
				}
			} else {
				return "", "", "", nil, errors.New("missing questId parameter for start_quest operation")
			}
		}

		// NpcId is optional - if not provided, get from conversation context
		var npcIdInt int
		if npcIdValue, exists := operation.Params()["npcId"]; exists {
			npcIdInt, err = e.evaluateContextValueAsInt(characterId, "npcId", npcIdValue)
			if err != nil {
				return "", "", "", nil, err
			}
		} else {
			// Get NPC ID from conversation context
			ctx, err := GetRegistry().GetPreviousContext(e.t, characterId)
			if err != nil {
				return "", "", "", nil, fmt.Errorf("failed to get conversation context for NPC ID: %w", err)
			}
			npcIdInt = int(ctx.NpcId())
		}

		payload := saga.StartQuestPayload{
			CharacterId: characterId,
			QuestId:     uint32(questIdInt),
			NpcId:       uint32(npcIdInt),
		}

		return stepId, saga.Pending, saga.StartQuest, payload, nil

	case "apply_consumable_effect":
		// Format: apply_consumable_effect
		// Params: itemId (uint32)
		// Applies consumable item effects to a character without consuming from inventory
		// Used for NPC-initiated buffs (e.g., cm.useItem() in scripts)
		itemIdValue, exists := operation.Params()["itemId"]
		if !exists {
			return "", "", "", nil, errors.New("missing itemId parameter for apply_consumable_effect operation")
		}

		itemIdInt, err := e.evaluateContextValueAsInt(characterId, "itemId", itemIdValue)
		if err != nil {
			return "", "", "", nil, err
		}

		payload := saga.ApplyConsumableEffectPayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			ItemId:      uint32(itemIdInt),
		}

		return stepId, saga.Pending, saga.ApplyConsumableEffect, payload, nil

	case "send_message":
		// Format: send_message
		// Params: messageType (string: "NOTICE", "POP_UP", "PINK_TEXT", "BLUE_TEXT"), message (string)
		// Sends a system message to the character
		// Used for NPC-initiated messages (e.g., cm.playerMessage() in scripts)
		messageTypeValue, exists := operation.Params()["messageType"]
		if !exists {
			return "", "", "", nil, errors.New("missing messageType parameter for send_message operation")
		}

		messageType, err := e.evaluateContextValue(characterId, "messageType", messageTypeValue)
		if err != nil {
			return "", "", "", nil, err
		}

		messageValue, exists := operation.Params()["message"]
		if !exists {
			return "", "", "", nil, errors.New("missing message parameter for send_message operation")
		}

		message, err := e.evaluateContextValue(characterId, "message", messageValue)
		if err != nil {
			return "", "", "", nil, err
		}

		payload := saga.SendMessagePayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			MessageType: messageType,
			Message:     message,
		}

		return stepId, saga.Pending, saga.SendMessage, payload, nil

	case "award_fame":
		// Format: award_fame
		// Params: amount (int16)
		// Awards fame to a character (can be negative to remove fame)
		// Used for quest rewards (e.g., qm.gainFame() in scripts)
		amountValue, exists := operation.Params()["amount"]
		if !exists {
			return "", "", "", nil, errors.New("missing amount parameter for award_fame operation")
		}

		amountInt, err := e.evaluateContextValueAsInt(characterId, "amount", amountValue)
		if err != nil {
			return "", "", "", nil, err
		}

		payload := saga.AwardFamePayload{
			CharacterId: characterId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			Amount:      int16(amountInt),
		}

		return stepId, saga.Pending, saga.AwardFame, payload, nil

	case "open_storage":
		// Format: open_storage
		// Params: accountId (uint32, required)
		// Opens the storage UI for the character via the NPC they're talking to
		// Used for storage keeper NPCs (e.g., Fredrick in FM)
		accountIdValue, exists := operation.Params()["accountId"]
		if !exists {
			return "", "", "", nil, errors.New("missing accountId parameter for open_storage operation")
		}

		accountIdInt, err := e.evaluateContextValueAsInt(characterId, "accountId", accountIdValue)
		if err != nil {
			return "", "", "", nil, err
		}

		// Get NPC ID from conversation context
		ctx, err := GetRegistry().GetPreviousContext(e.t, characterId)
		if err != nil {
			return "", "", "", nil, fmt.Errorf("failed to get conversation context for NPC ID: %w", err)
		}
		npcId := ctx.NpcId()

		payload := saga.ShowStoragePayload{
			CharacterId: characterId,
			NpcId:       npcId,
			WorldId:     f.WorldId(),
			ChannelId:   f.ChannelId(),
			AccountId:   uint32(accountIdInt),
		}

		return stepId, saga.Pending, saga.ShowStorage, payload, nil

	default:
		return "", "", "", nil, fmt.Errorf("unknown operation type: %s", operation.Type())
	}
}

// storeStylesInContext stores a uint32 array of styles in the conversation context
func (e *OperationExecutorImpl) storeStylesInContext(characterId uint32, key string, styles []uint32) error {
	// Get current context
	ctx, err := GetRegistry().GetPreviousContext(e.t, characterId)
	if err != nil {
		e.l.WithError(err).Errorf("Failed to get conversation context for character [%d]", characterId)
		return fmt.Errorf("failed to get conversation context: %w", err)
	}

	// Encode styles as comma-separated string
	stylesStr := encodeUint32Array(styles)

	// Update context
	ctx.Context()[key] = stylesStr

	// Save context
	GetRegistry().UpdateContext(e.t, characterId, ctx)

	e.l.Debugf("Stored %d styles in context key [%s] for character [%d]: %s",
		len(styles), key, characterId, stylesStr)

	return nil
}

// encodeUint32Array encodes a uint32 array as a comma-separated string
func encodeUint32Array(arr []uint32) string {
	if len(arr) == 0 {
		return ""
	}

	strs := make([]string, len(arr))
	for i, val := range arr {
		strs[i] = strconv.FormatUint(uint64(val), 10)
	}
	return strings.Join(strs, ",")
}

// decodeUint32Array decodes a comma-separated string into a uint32 array
func decodeUint32Array(str string) ([]uint32, error) {
	if str == "" {
		return []uint32{}, nil
	}

	parts := strings.Split(str, ",")
	result := make([]uint32, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		val, err := strconv.ParseUint(trimmed, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid number '%s': %w", trimmed, err)
		}

		result = append(result, uint32(val))
	}

	return result, nil
}
