package conversation

import (
	"atlas-npc-conversations/message"
	npcSender "atlas-npc-conversations/npc"
	"atlas-npc-conversations/saga"
	"atlas-npc-conversations/validation"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	// Start starts a conversation with an NPC
	Start(field field.Model, npcId uint32, characterId uint32, accountId uint32) error

	// StartQuest starts a quest conversation with the provided state machine
	// The caller is responsible for selecting the correct state machine (start or end) based on quest status
	StartQuest(field field.Model, questId uint32, npcId uint32, characterId uint32, stateMachine StateContainer) error

	// Continue continues a conversation with an NPC
	Continue(npcId uint32, characterId uint32, action byte, lastMessageType byte, selection int32) error

	// End ends a conversation
	End(characterId uint32) error
}

type ProcessorImpl struct {
	l                       logrus.FieldLogger
	ctx                     context.Context
	t                       tenant.Model
	db                      *gorm.DB
	evaluator               Evaluator
	executor                OperationExecutor
	npcConversationProvider NpcConversationProvider
}

// NewProcessorFactory is the factory function type for creating NpcConversationProvider
type NewProcessorFactory func(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) NpcConversationProvider

// npcConversationProviderFactory stores the factory function for creating NpcConversationProvider
// This is set by the npc package during initialization to break the import cycle
var npcConversationProviderFactory NewProcessorFactory

// SetNpcConversationProviderFactory sets the factory function for creating NpcConversationProvider
func SetNpcConversationProviderFactory(factory NewProcessorFactory) {
	npcConversationProviderFactory = factory
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	t := tenant.MustFromContext(ctx)
	evaluator := NewEvaluator(l, ctx, t)
	executor := NewOperationExecutor(l, ctx)

	var npcProvider NpcConversationProvider
	if npcConversationProviderFactory != nil {
		npcProvider = npcConversationProviderFactory(l, ctx, db)
	}

	return &ProcessorImpl{
		l:                       l,
		ctx:                     ctx,
		t:                       t,
		db:                      db,
		evaluator:               evaluator,
		executor:                executor,
		npcConversationProvider: npcProvider,
	}
}

func (p *ProcessorImpl) Start(field field.Model, npcId uint32, characterId uint32, accountId uint32) error {
	p.l.Debugf("Starting conversation with NPC [%d] with character [%d] in map [%d].", npcId, characterId, field.MapId())

	// Check if there's already a conversation in progress
	_, err := GetRegistry().GetPreviousContext(p.t, characterId)
	if err == nil {
		p.l.Debugf("Previous conversation for character [%d] exists, avoiding starting new conversation with NPC [%d].", characterId, npcId)
		return errors.New("another conversation exists")
	}

	// Get the conversation for this NPC
	if p.npcConversationProvider == nil {
		return errors.New("NPC conversation provider not initialized")
	}
	conversation, err := p.npcConversationProvider.ByNpcIdProvider(npcId)()
	if err != nil {
		p.l.WithError(err).Errorf("Failed to retrieve conversation for NPC [%d]", npcId)
		return err
	}

	// Get the start state
	startStateId := conversation.StartState()

	// Create a conversation context
	builder := NewConversationContextBuilder().
		SetField(field).
		SetCharacterId(characterId).
		SetNpcId(npcId).
		SetCurrentState(startStateId).
		SetConversation(conversation)

	// Add worldId and channelId to context for use in conditions
	if field.WorldId() > 0 {
		builder.AddContextValue("worldId", strconv.Itoa(int(field.WorldId())))
	}
	if field.ChannelId() > 0 {
		builder.AddContextValue("channelId", strconv.Itoa(int(field.ChannelId())))
	}

	// Add accountId to context
	if accountId > 0 {
		builder.AddContextValue("accountId", strconv.Itoa(int(accountId)))
		p.l.Debugf("Added accountId [%d] to context for character [%d]", accountId, characterId)
	}

	ctx := builder.Build()

	// Store the context
	GetRegistry().SetContext(p.t, ctx.CharacterId(), ctx)

	cont := true
	for cont {
		ctx, err = GetRegistry().GetPreviousContext(p.t, characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve conversation context for [%d].", characterId)
			return errors.New("conversation context not found")
		}

		cont, err = p.ProcessState(ctx)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to process state [%s] for character [%d] and NPC [%d]", startStateId, characterId, npcId)
			return err
		}
	}
	return nil
}

func (p *ProcessorImpl) StartQuest(f field.Model, questId uint32, npcId uint32, characterId uint32, stateMachine StateContainer) error {
	p.l.Debugf("Starting quest [%d] conversation with NPC [%d] for character [%d] in map [%d].", questId, npcId, characterId, f.MapId())

	// Check if there's already a conversation in progress
	_, err := GetRegistry().GetPreviousContext(p.t, characterId)
	if err == nil {
		p.l.Debugf("Previous conversation for character [%d] exists, avoiding starting quest [%d] conversation.", characterId, questId)
		return errors.New("another conversation exists")
	}

	// Get the start state from the provided state machine
	startStateId := stateMachine.StartState()

	// Create a conversation context for quest
	builder := NewConversationContextBuilder().
		SetField(f).
		SetCharacterId(characterId).
		SetNpcId(npcId).
		SetCurrentState(startStateId).
		SetConversation(stateMachine).
		SetConversationType(QuestConversationType).
		SetSourceId(questId)

	// Add questId to context for use in operations
	builder.AddContextValue("questId", strconv.FormatUint(uint64(questId), 10))

	// Add worldId and channelId to context for use in conditions
	if f.WorldId() > 0 {
		builder.AddContextValue("worldId", strconv.Itoa(int(f.WorldId())))
	}
	if f.ChannelId() > 0 {
		builder.AddContextValue("channelId", strconv.Itoa(int(f.ChannelId())))
	}

	ctx := builder.Build()

	// Store the context
	GetRegistry().SetContext(p.t, ctx.CharacterId(), ctx)

	cont := true
	for cont {
		ctx, err = GetRegistry().GetPreviousContext(p.t, characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve conversation context for [%d].", characterId)
			return errors.New("conversation context not found")
		}

		cont, err = p.ProcessState(ctx)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to process state [%s] for character [%d] and quest [%d]", startStateId, characterId, questId)
			return err
		}
	}
	return nil
}

func (p *ProcessorImpl) Continue(npcId uint32, characterId uint32, action byte, lastMessageType byte, selection int32) error {
	// Get the previous context
	ctx, err := GetRegistry().GetPreviousContext(p.t, characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve conversation context for [%d].", characterId)
		return errors.New("conversation context not found")
	}

	p.l.Debugf("Continuing conversation with NPC [%d] with character [%d] in map [%d].", ctx.NpcId(), characterId, ctx.Field().MapId())
	p.l.Debugf("Calling continue with: action [%d], lastMessageType [%d], selection [%d].", action, lastMessageType, selection)

	// Get the current state
	currentStateId := ctx.CurrentState()
	conversation := ctx.Conversation()

	// Find the current state in the conversation
	state, err := conversation.FindState(currentStateId)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to find state [%s] for character [%d]", currentStateId, characterId)
		return err
	}

	// Process the player's selection based on the state type
	var nextStateId string
	var choiceContext map[string]string

	switch state.Type() {
	case DialogueStateType:
		// For dialogue states, the action is the index of the choice
		dialogue := state.Dialogue()
		if dialogue == nil {
			return errors.New("dialogue is nil")
		}

		choice, _ := dialogue.ChoiceFromAction(action)
		nextStateId = choice.NextState()

		// Store the choice context for later use
		choiceContext = choice.Context()
	case ListSelectionType:
		// For list selection states, the selection is the index of the option
		listSelection := state.ListSelection()
		if listSelection == nil {
			return errors.New("listSelection is nil")
		}

		choice, _ := listSelection.ChoiceFromSelection(action, selection)
		nextStateId = choice.NextState()

		// Store the choice context for later use
		choiceContext = choice.Context()

	case AskNumberType:
		// For ask number states, the selection contains the number entered by the player
		askNumber := state.AskNumber()
		if askNumber == nil {
			return errors.New("askNumber is nil")
		}

		// Validate the selection against min/max values
		if selection < 0 {
			p.l.Errorf("Invalid number input [%d] for character [%d]: negative value", selection, characterId)
			return fmt.Errorf("invalid number input: negative value")
		}

		numberValue := uint32(selection)
		if numberValue < askNumber.MinValue() {
			p.l.Errorf("Invalid number input [%d] for character [%d]: below minimum [%d]", numberValue, characterId, askNumber.MinValue())
			return fmt.Errorf("number below minimum value")
		}
		if numberValue > askNumber.MaxValue() {
			p.l.Errorf("Invalid number input [%d] for character [%d]: above maximum [%d]", numberValue, characterId, askNumber.MaxValue())
			return fmt.Errorf("number above maximum value")
		}

		// Store the number in the context using the configured context key
		choiceContext = make(map[string]string)
		choiceContext[askNumber.ContextKey()] = fmt.Sprintf("%d", numberValue)

		// Calculate derived values if needed (e.g., totalCost = quantity * price)
		if price, exists := ctx.Context()["price"]; exists {
			if priceValue, err := strconv.ParseUint(price, 10, 32); err == nil {
				totalCost := uint64(numberValue) * priceValue
				choiceContext["totalCost"] = fmt.Sprintf("%d", totalCost)
			}
		}

		// Get the next state from the askNumber model
		nextStateId = askNumber.NextState()

	case AskStyleType:
		// For ask style states, the selection contains the index of the selected style
		askStyle := state.AskStyle()
		if askStyle == nil {
			return errors.New("askStyle is nil")
		}

		if action == 0 {
			// action = 0 on Exit / Cancel
		} else if action == 1 {
			var styles []uint32
			if len(askStyle.Styles()) > 0 {
				for _, style := range askStyle.Styles() {
					styles = append(styles, style)
				}
			} else if len(askStyle.StylesContextKey()) > 0 {
				// Get the styles array from context
				stylesValue, exists := ctx.Context()[askStyle.StylesContextKey()]
				if !exists {
					return fmt.Errorf("failed to get styles from context key '%s': %w", askStyle.StylesContextKey(), err)
				}

				// Parse the styles array (should be a JSON array of uint32)
				styleStrs := strings.Split(stylesValue, ",")
				if len(styleStrs) == 0 {
					return errors.New("styles array is empty, cannot select random cosmetic")
				}
				for _, styleStr := range styleStrs {
					var styleId int
					styleId, err = strconv.Atoi(styleStr)
					if err != nil {
						return err
					}
					styles = append(styles, uint32(styleId))
				}
			}

			// Validate the selection is within bounds
			if selection < 0 || selection >= int32(len(styles)) {
				p.l.Errorf("Invalid style selection [%d] for character [%d]: out of bounds", selection, characterId)
				return fmt.Errorf("invalid style selection: out of bounds")
			}

			// Get the selected style ID
			selectedStyleId := styles[selection]

			// Store the selected style in the context using the configured context key
			choiceContext = make(map[string]string)
			choiceContext[askStyle.ContextKey()] = fmt.Sprintf("%d", selectedStyleId)

			// Get the next state from the askStyle model
			nextStateId = askStyle.NextState()
		}

	case AskSlideMenuType:
		// For ask slide menu states, the selection is the index of the option
		askSlideMenu := state.AskSlideMenu()
		if askSlideMenu == nil {
			return errors.New("askSlideMenu is nil")
		}

		choice, _ := askSlideMenu.ChoiceFromSelection(action, selection)
		nextStateId = choice.NextState()

		// Store the choice context for later use
		choiceContext = choice.Context()

	default:
		// For other state types, we shouldn't be here (they should have been processed already)
		return fmt.Errorf("unexpected state type for Continue: %s", state.Type())
	}

	// If there's a next state, process it
	if nextStateId == "" {
		// No next state, end the conversation
		GetRegistry().ClearContext(p.t, characterId)
		return nil
	}

	// Update the context with the next state
	builder := NewConversationContextBuilder().
		SetField(ctx.Field()).
		SetCharacterId(ctx.CharacterId()).
		SetNpcId(ctx.NpcId()).
		SetCurrentState(nextStateId).
		SetConversation(ctx.Conversation())

	// Preserve existing context and add new context from the choice
	existingContext := ctx.Context()
	for k, v := range existingContext {
		builder.AddContextValue(k, v)
	}

	// Add new context from the choice (will overwrite existing values with the same keys)
	for k, v := range choiceContext {
		builder.AddContextValue(k, v)
	}

	ctx = builder.Build()

	// Store the context
	GetRegistry().SetContext(p.t, ctx.CharacterId(), ctx)

	cont := true
	for cont {
		var err error
		ctx, err = GetRegistry().GetPreviousContext(p.t, characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve conversation context for [%d].", characterId)
			return errors.New("conversation context not found")
		}

		cont, err = p.ProcessState(ctx)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to process state [%s] for character [%d] and NPC [%d]", nextStateId, characterId, npcId)
			return err
		}
	}
	return nil
}

func (p *ProcessorImpl) ProcessState(ctx ConversationContext) (bool, error) {
	stateId := ctx.CurrentState()
	state, err := ctx.Conversation().FindState(stateId)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to find state [%s] for NPC [%d]", stateId, ctx.NpcId())
		return false, err
	}

	// Process the state
	nextStateId, err := p.processState(ctx, state)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to process state [%s] for character [%d] and NPC [%d]", stateId, ctx.CharacterId(), ctx.NpcId())
		return false, err
	}

	// If there's a next state, update the context and store it
	if nextStateId != "" {
		// If the next state is the same as the current state, the sub-processor
		// already updated the registry (e.g., saga-waiting states). Don't overwrite.
		if nextStateId == stateId {
			return false, nil
		}

		// Update the context with the next state
		builder := NewConversationContextBuilder().
			SetField(ctx.Field()).
			SetCharacterId(ctx.CharacterId()).
			SetNpcId(ctx.NpcId()).
			SetCurrentState(nextStateId).
			SetConversation(ctx.Conversation()).
			SetConversationType(ctx.ConversationType()).
			SetSourceId(ctx.SourceId())

		// Preserve existing context
		existingContext := ctx.Context()
		for k, v := range existingContext {
			builder.AddContextValue(k, v)
		}

		ctx = builder.Build()

		// Store the context
		GetRegistry().SetContext(p.t, ctx.CharacterId(), ctx)

		return state.stateType == GenericActionType, nil
	} else {
		// No next state, end the conversation
		GetRegistry().ClearContext(p.t, ctx.CharacterId())
		return false, nil
	}
}

// processState processes a conversation state and returns the next state ID
func (p *ProcessorImpl) processState(ctx ConversationContext, state StateModel) (string, error) {
	p.l.Debugf("Processing state [%s] for character [%d]", state.Id(), ctx.CharacterId())

	// Process the state based on its type
	switch state.Type() {
	case DialogueStateType:
		// Process dialogue state
		return p.processDialogueState(ctx, state)
	case GenericActionType:
		// Process generic action state
		return p.processGenericActionState(ctx, state)
	case CraftActionType:
		// Process craft action state
		return p.processCraftActionState(ctx, state)
	case TransportActionType:
		// Process transport action state
		return p.processTransportActionState(ctx, state)
	case PartyQuestActionType:
		// Process party quest action state
		return p.processPartyQuestActionState(ctx, state)
	case PartyQuestBonusActionType:
		// Process party quest bonus action state
		return p.processPartyQuestBonusActionState(ctx, state)
	case GachaponActionType:
		// Process gachapon action state
		return p.processGachaponActionState(ctx, state)
	case ListSelectionType:
		// Process list selection state
		return p.processListSelectionState(ctx, state)
	case AskNumberType:
		// Process ask number state
		return p.processAskNumberState(ctx, state)
	case AskStyleType:
		// Process ask style state
		return p.processAskStyleState(ctx, state)
	case AskSlideMenuType:
		// Process ask slide menu state
		return p.processAskSlideMenuState(ctx, state)
	default:
		return "", errors.New("unknown state type")
	}
}

// processDialogueState processes a dialogue state
func (p *ProcessorImpl) processDialogueState(ctx ConversationContext, state StateModel) (string, error) {
	dialogue := state.Dialogue()
	if dialogue == nil {
		return "", errors.New("dialogue is nil")
	}

	// Replace context placeholders in the dialogue text
	processedText, err := ReplaceContextPlaceholders(dialogue.Text(), ctx.Context())
	if err != nil {
		p.l.WithError(err).Warnf("Failed to replace context placeholders in dialogue text for state [%s]. Using original text.", state.Id())
		// Use original text if replacement fails
		processedText = dialogue.Text()
	}

	// Build configurators for speaker settings
	var configs []npcSender.TalkConfigurator
	if dialogue.Speaker() != "" {
		configs = append(configs, npcSender.WithSpeaker(dialogue.Speaker()))
	}
	configs = append(configs, npcSender.WithEndChat(dialogue.EndChat()))
	if dialogue.SecondaryNpcId() != 0 {
		configs = append(configs, npcSender.WithSecondaryNpcId(dialogue.SecondaryNpcId()))
	}

	// Send the dialogue to the client
	npcProcessor := npcSender.NewProcessor(p.l, p.ctx)
	if dialogue.dialogueType == SendNext {
		npcProcessor.SendNext(ctx.Field().Channel(), ctx.CharacterId(), ctx.NpcId())(processedText, configs...)
	} else if dialogue.dialogueType == SendNextPrev {
		npcProcessor.SendNextPrevious(ctx.Field().Channel(), ctx.CharacterId(), ctx.NpcId())(processedText, configs...)
	} else if dialogue.dialogueType == SendPrev {
		npcProcessor.SendPrevious(ctx.Field().Channel(), ctx.CharacterId(), ctx.NpcId())(processedText, configs...)
	} else if dialogue.dialogueType == SendOk {
		npcProcessor.SendOk(ctx.Field().Channel(), ctx.CharacterId(), ctx.NpcId())(processedText, configs...)
	} else if dialogue.dialogueType == SendYesNo {
		npcProcessor.SendYesNo(ctx.Field().Channel(), ctx.CharacterId(), ctx.NpcId())(processedText, configs...)
	} else if dialogue.dialogueType == SendAcceptDecline {
		npcProcessor.SendAcceptDecline(ctx.Field().Channel(), ctx.CharacterId(), ctx.NpcId())(processedText, configs...)
	} else {
		p.l.Warnf("Unhandled dialog type [%s].", dialogue.dialogueType)
	}

	// If the dialogue has choices, wait for the player's selection
	if len(dialogue.Choices()) > 0 {
		// Return the current state ID to indicate that we're waiting for input
		return state.Id(), nil
	}

	// Otherwise, return the next state ID (for dialogues without choices)
	// For now, just return an empty string to end the conversation
	return "", nil
}

// processGenericActionState processes a generic action state
func (p *ProcessorImpl) processGenericActionState(ctx ConversationContext, state StateModel) (string, error) {
	genericAction := state.GenericAction()
	if genericAction == nil {
		return "", errors.New("genericAction is nil")
	}

	// Error recovery wrapper to ensure conversation cleanup on failures
	defer func() {
		if r := recover(); r != nil {
			p.l.Errorf("Panic recovered in processGenericActionState for character [%d]: %v", ctx.CharacterId(), r)
			GetRegistry().ClearContext(p.t, ctx.CharacterId())
		}
	}()

	// Execute operations with error recovery
	for _, operation := range genericAction.Operations() {
		err := p.executor.ExecuteOperation(ctx.Field(), ctx.CharacterId(), operation)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to execute operation [%s] for character [%d]. Cleaning up conversation context.", operation.Type(), ctx.CharacterId())
			// Clean up conversation context before returning error
			GetRegistry().ClearContext(p.t, ctx.CharacterId())
			return "", err
		}
	}

	// Evaluate outcomes with error recovery
	for _, outcome := range genericAction.Outcomes() {
		if len(outcome.Conditions()) == 0 {
			return outcome.NextState(), nil
		}

		// Evaluate the condition
		// TODO
		passed, err := p.evaluator.EvaluateCondition(ctx.CharacterId(), outcome.Conditions()[0])
		if err != nil {
			p.l.WithError(err).Errorf("Failed to evaluate condition [%+v] for character [%d]. Cleaning up conversation context.", outcome.Conditions()[0], ctx.CharacterId())
			// Clean up conversation context before returning error
			GetRegistry().ClearContext(p.t, ctx.CharacterId())
			return "", err
		}

		// If the condition passed, return the next state
		if passed {
			return outcome.NextState(), nil
		}
	}

	// If no outcome matched, return an empty string to end the conversation
	return "", nil
}

// processCraftActionState processes a craft action state
func (p *ProcessorImpl) processCraftActionState(ctx ConversationContext, state StateModel) (string, error) {
	craftAction := state.CraftAction()
	if craftAction == nil {
		return "", errors.New("craftAction is nil")
	}

	// Get quantity multiplier from context (defaults to 1)
	quantityMultiplier := uint32(1)
	if quantityStr, exists := ctx.Context()["quantity"]; exists {
		if qty, err := strconv.ParseUint(quantityStr, 10, 32); err == nil {
			quantityMultiplier = uint32(qty)
		}
	}

	// Replace context placeholders in itemId
	itemIdStr, err := ReplaceContextPlaceholders(craftAction.ItemId(), ctx.Context())
	if err != nil {
		p.l.WithError(err).Errorf("Failed to replace context in itemId")
		return craftAction.FailureState(), nil
	}
	itemId, err := strconv.ParseUint(itemIdStr, 10, 32)
	if err != nil {
		p.l.WithError(err).Errorf("Invalid itemId: %s", itemIdStr)
		return craftAction.FailureState(), nil
	}

	// Calculate total materials and costs based on quantity multiplier
	totalMaterials := craftAction.Materials()
	totalQuantities := make([]uint32, len(craftAction.Quantities()))
	for i, qty := range craftAction.Quantities() {
		totalQuantities[i] = qty * quantityMultiplier
	}
	totalMesoCost := craftAction.MesoCost() * quantityMultiplier

	p.l.Debugf("Crafting item %d (quantity: %d) for character %d", itemId, quantityMultiplier, ctx.CharacterId())

	// Build saga to craft the item
	sagaId := uuid.New()
	sagaBuilder := saga.NewBuilder().
		SetTransactionId(sagaId).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy(fmt.Sprintf("NPC_%d", ctx.NpcId()))

	// Step 1: Validate character state (has mesos and materials)
	conditions := make([]validation.ConditionInput, 0)

	// Check meso requirement
	if totalMesoCost > 0 {
		conditions = append(conditions, validation.ConditionInput{
			Type:     "meso",
			Operator: ">=",
			Value:    int(totalMesoCost),
		})
	}

	// Check material requirements
	for i, materialId := range totalMaterials {
		conditions = append(conditions, validation.ConditionInput{
			Type:        "item",
			Operator:    ">=",
			Value:       int(totalQuantities[i]),
			ReferenceId: materialId,
		})
	}

	if len(conditions) > 0 {
		validatePayload := saga.ValidateCharacterStatePayload{
			CharacterId: ctx.CharacterId(),
			Conditions:  conditions,
		}
		sagaBuilder.AddStep("validate_resources", saga.Pending, saga.ValidateCharacterState, validatePayload)
	}

	// Step 2: Destroy materials
	for i, materialId := range totalMaterials {
		destroyPayload := saga.DestroyAssetPayload{
			CharacterId: ctx.CharacterId(),
			TemplateId:  materialId,
			Quantity:    totalQuantities[i],
		}
		sagaBuilder.AddStep(fmt.Sprintf("destroy_material_%d", materialId), saga.Pending, saga.DestroyAsset, destroyPayload)
	}

	// Step 3: Deduct mesos
	if totalMesoCost > 0 {
		mesoPayload := saga.AwardMesosPayload{
			CharacterId: ctx.CharacterId(),
			WorldId:     ctx.Field().WorldId(),
			ChannelId:   ctx.Field().ChannelId(),
			ActorId:     ctx.NpcId(),
			ActorType:   "NPC",
			Amount:      -int32(totalMesoCost),
		}
		sagaBuilder.AddStep("deduct_mesos", saga.Pending, saga.AwardMesos, mesoPayload)
	}

	// Step 4: Award crafted item
	craftPayload := saga.AwardItemActionPayload{
		CharacterId: ctx.CharacterId(),
		Item: saga.ItemPayload{
			TemplateId: uint32(itemId),
			Quantity:   quantityMultiplier,
		},
	}
	sagaBuilder.AddStep("award_crafted_item", saga.Pending, saga.AwardInventory, craftPayload)

	// Build and execute saga
	s := sagaBuilder.Build()

	// Send saga to orchestrator
	err = saga.NewProcessor(p.l, p.ctx).Create(s)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create crafting saga")
		return craftAction.FailureState(), nil
	}

	// Store saga ID and craft action details in context for later resumption
	ctx = ctx.SetPendingSagaId(sagaId)
	ctx.Context()["craftAction_successState"] = craftAction.SuccessState()
	ctx.Context()["craftAction_failureState"] = craftAction.FailureState()
	ctx.Context()["craftAction_missingMaterialsState"] = craftAction.MissingMaterialsState()

	// Update conversation context in registry
	GetRegistry().UpdateContext(p.t, ctx.CharacterId(), ctx)

	p.l.WithFields(logrus.Fields{
		"transaction_id": sagaId.String(),
		"character_id":   ctx.CharacterId(),
		"npc_id":         ctx.NpcId(),
	}).Debug("Saga created, conversation waiting for completion")

	// Return current state ID to keep conversation in "waiting" state
	// When saga completes, the saga status consumer will resume the conversation
	return state.Id(), nil
}

// processTransportActionState processes a transport action state
// Creates a saga to start an instance transport and waits for completion
func (p *ProcessorImpl) processTransportActionState(ctx ConversationContext, state StateModel) (string, error) {
	transportAction := state.TransportAction()
	if transportAction == nil {
		return "", errors.New("transportAction is nil")
	}

	p.l.WithFields(logrus.Fields{
		"route_name":   transportAction.RouteName(),
		"character_id": ctx.CharacterId(),
		"npc_id":       ctx.NpcId(),
	}).Debug("Processing transport action state")

	// Create a new saga ID
	sagaId := uuid.New()

	// Build the saga with a single start_instance_transport step
	sagaBuilder := saga.NewBuilder().
		SetTransactionId(sagaId).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy(fmt.Sprintf("NPC_%d_transport", ctx.NpcId()))

	// Add the transport step
	transportPayload := saga.StartInstanceTransportPayload{
		CharacterId: ctx.CharacterId(),
		WorldId:     ctx.Field().WorldId(),
		ChannelId:   ctx.Field().ChannelId(),
		RouteName:   transportAction.RouteName(),
	}
	sagaBuilder.AddStep("start_instance_transport", saga.Pending, saga.StartInstanceTransport, transportPayload)

	// Build and execute saga
	s := sagaBuilder.Build()

	// Send saga to orchestrator
	err := saga.NewProcessor(p.l, p.ctx).Create(s)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create transport saga")
		return transportAction.FailureState(), nil
	}

	// Store saga ID and transport action failure states in context for later resumption
	ctx = ctx.SetPendingSagaId(sagaId)
	ctx.Context()["transportAction_failureState"] = transportAction.FailureState()
	ctx.Context()["transportAction_capacityFullState"] = transportAction.CapacityFullState()
	ctx.Context()["transportAction_alreadyInTransitState"] = transportAction.AlreadyInTransitState()
	ctx.Context()["transportAction_routeNotFoundState"] = transportAction.RouteNotFoundState()
	ctx.Context()["transportAction_serviceErrorState"] = transportAction.ServiceErrorState()

	// Update conversation context in registry
	GetRegistry().UpdateContext(p.t, ctx.CharacterId(), ctx)

	p.l.WithFields(logrus.Fields{
		"transaction_id": sagaId.String(),
		"character_id":   ctx.CharacterId(),
		"npc_id":         ctx.NpcId(),
		"route_name":     transportAction.RouteName(),
	}).Debug("Transport saga created, conversation waiting for completion")

	// Return current state ID to keep conversation in "waiting" state
	// When saga completes/fails, the saga status consumer will resume the conversation
	// On success, the character will be warped and conversation ends naturally
	// On failure, we route to the appropriate error state
	return state.Id(), nil
}

// processPartyQuestActionState processes a party quest registration action state
// Creates a saga to register a party for a party quest and waits for completion
func (p *ProcessorImpl) processPartyQuestActionState(ctx ConversationContext, state StateModel) (string, error) {
	partyQuestAction := state.PartyQuestAction()
	if partyQuestAction == nil {
		return "", errors.New("partyQuestAction is nil")
	}

	p.l.WithFields(logrus.Fields{
		"quest_id":     partyQuestAction.QuestId(),
		"character_id": ctx.CharacterId(),
		"npc_id":       ctx.NpcId(),
	}).Debug("Processing party quest action state")

	// Create a new saga ID
	sagaId := uuid.New()

	// Build the saga with a single register_party_quest step
	sagaBuilder := saga.NewBuilder().
		SetTransactionId(sagaId).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy(fmt.Sprintf("NPC_%d_party_quest", ctx.NpcId()))

	// Add the register party quest step
	pqPayload := saga.RegisterPartyQuestPayload{
		CharacterId: ctx.CharacterId(),
		WorldId:     ctx.Field().WorldId(),
		ChannelId:   ctx.Field().ChannelId(),
		MapId:       ctx.Field().MapId(),
		QuestId:     partyQuestAction.QuestId(),
	}
	sagaBuilder.AddStep("register_party_quest", saga.Pending, saga.RegisterPartyQuest, pqPayload)

	// Build and execute saga
	s := sagaBuilder.Build()

	// Send saga to orchestrator
	err := saga.NewProcessor(p.l, p.ctx).Create(s)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create party quest saga")
		return partyQuestAction.FailureState(), nil
	}

	// Store saga ID and party quest action failure states in context for later resumption
	ctx = ctx.SetPendingSagaId(sagaId)
	ctx.Context()["partyQuestAction_failureState"] = partyQuestAction.FailureState()
	ctx.Context()["partyQuestAction_notInPartyState"] = partyQuestAction.NotInPartyState()
	ctx.Context()["partyQuestAction_notLeaderState"] = partyQuestAction.NotLeaderState()

	// Update conversation context in registry
	GetRegistry().UpdateContext(p.t, ctx.CharacterId(), ctx)

	p.l.WithFields(logrus.Fields{
		"transaction_id": sagaId.String(),
		"character_id":   ctx.CharacterId(),
		"npc_id":         ctx.NpcId(),
		"quest_id":       partyQuestAction.QuestId(),
	}).Debug("Party quest saga created, conversation waiting for completion")

	// Return current state ID to keep conversation in "waiting" state
	// When saga completes, party-quests service warps the party and conversation ends
	// When saga fails, we route to the appropriate error state
	return state.Id(), nil
}

// processPartyQuestBonusActionState processes a party quest bonus entry action state
// Creates a saga to enter the bonus stage of a party quest and waits for completion
func (p *ProcessorImpl) processPartyQuestBonusActionState(ctx ConversationContext, state StateModel) (string, error) {
	bonusAction := state.PartyQuestBonusAction()
	if bonusAction == nil {
		return "", errors.New("partyQuestBonusAction is nil")
	}

	p.l.WithFields(logrus.Fields{
		"character_id": ctx.CharacterId(),
		"npc_id":       ctx.NpcId(),
	}).Debug("Processing party quest bonus action state")

	// Create a new saga ID
	sagaId := uuid.New()

	// Build the saga with a single enter_party_quest_bonus step
	sagaBuilder := saga.NewBuilder().
		SetTransactionId(sagaId).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy(fmt.Sprintf("NPC_%d_pq_bonus", ctx.NpcId()))

	// Add the enter party quest bonus step
	bonusPayload := saga.EnterPartyQuestBonusPayload{
		CharacterId: ctx.CharacterId(),
		WorldId:     ctx.Field().WorldId(),
	}
	sagaBuilder.AddStep("enter_party_quest_bonus", saga.Pending, saga.EnterPartyQuestBonus, bonusPayload)

	// Build and execute saga
	s := sagaBuilder.Build()

	// Send saga to orchestrator
	err := saga.NewProcessor(p.l, p.ctx).Create(s)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create party quest bonus saga")
		return bonusAction.FailureState(), nil
	}

	// Store saga ID and bonus action failure state in context for later resumption
	ctx = ctx.SetPendingSagaId(sagaId)
	ctx.Context()["partyQuestBonusAction_failureState"] = bonusAction.FailureState()

	// Update conversation context in registry
	GetRegistry().UpdateContext(p.t, ctx.CharacterId(), ctx)

	p.l.WithFields(logrus.Fields{
		"transaction_id": sagaId.String(),
		"character_id":   ctx.CharacterId(),
		"npc_id":         ctx.NpcId(),
	}).Debug("Party quest bonus saga created, conversation waiting for completion")

	// Return current state ID to keep conversation in "waiting" state
	// When saga completes, party-quests service warps the party to bonus and conversation ends
	// When saga fails, we route to the failure state
	return state.Id(), nil
}

// processGachaponActionState processes a gachapon action state
// Creates a saga to destroy the ticket and select a gachapon reward
func (p *ProcessorImpl) processGachaponActionState(ctx ConversationContext, state StateModel) (string, error) {
	gachaponAction := state.GachaponAction()
	if gachaponAction == nil {
		return "", errors.New("gachaponAction is nil")
	}

	p.l.WithFields(logrus.Fields{
		"gachapon_id":    gachaponAction.GachaponId(),
		"ticket_item_id": gachaponAction.TicketItemId(),
		"character_id":   ctx.CharacterId(),
		"npc_id":         ctx.NpcId(),
	}).Debug("Processing gachapon action state")

	sagaId := uuid.New()

	sagaBuilder := saga.NewBuilder().
		SetTransactionId(sagaId).
		SetSagaType(saga.GachaponTransaction).
		SetInitiatedBy(fmt.Sprintf("NPC_%d_gachapon", ctx.NpcId()))

	// Step 1: Destroy the gachapon ticket
	destroyPayload := saga.DestroyAssetPayload{
		CharacterId: ctx.CharacterId(),
		TemplateId:  gachaponAction.TicketItemId(),
		Quantity:    1,
	}
	sagaBuilder.AddStep("destroy_ticket", saga.Pending, saga.DestroyAsset, destroyPayload)

	// Step 2: Select gachapon reward (this dynamically injects AwardAsset + EmitGachaponWin)
	selectPayload := saga.SelectGachaponRewardPayload{
		CharacterId: ctx.CharacterId(),
		WorldId:     ctx.Field().WorldId(),
		GachaponId:  gachaponAction.GachaponId(),
	}
	sagaBuilder.AddStep("select_gachapon_reward", saga.Pending, saga.SelectGachaponReward, selectPayload)

	s := sagaBuilder.Build()

	err := saga.NewProcessor(p.l, p.ctx).Create(s)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create gachapon saga")
		return gachaponAction.FailureState(), nil
	}

	ctx = ctx.SetPendingSagaId(sagaId)
	ctx.Context()["gachaponAction_failureState"] = gachaponAction.FailureState()

	GetRegistry().UpdateContext(p.t, ctx.CharacterId(), ctx)

	p.l.WithFields(logrus.Fields{
		"transaction_id": sagaId.String(),
		"character_id":   ctx.CharacterId(),
		"npc_id":         ctx.NpcId(),
		"gachapon_id":    gachaponAction.GachaponId(),
	}).Debug("Gachapon saga created, conversation waiting for completion")

	return state.Id(), nil
}

// processListSelectionState processes a list selection state
func (p *ProcessorImpl) processListSelectionState(ctx ConversationContext, state StateModel) (string, error) {
	listSelection := state.ListSelection()
	if listSelection == nil {
		return "", errors.New("listSelection is nil")
	}

	// Replace context placeholders in the title
	processedTitle, err := ReplaceContextPlaceholders(listSelection.Title(), ctx.Context())
	if err != nil {
		p.l.WithError(err).Warnf("Failed to replace context placeholders in list selection title for state [%s]. Using original title.", state.Id())
		processedTitle = listSelection.Title()
	}

	mb := message.NewBuilder().AddText(processedTitle).NewLine()
	for i, choice := range listSelection.Choices() {
		if choice.NextState() == "" || choice.Text() == "Exit" {
			continue
		}

		// Replace context placeholders in choice text
		processedChoiceText, err := ReplaceContextPlaceholders(choice.Text(), ctx.Context())
		if err != nil {
			p.l.WithError(err).Warnf("Failed to replace context placeholders in choice text for state [%s]. Using original text.", state.Id())
			processedChoiceText = choice.Text()
		}

		mb.OpenItem(i).BlueText().AddText(processedChoiceText).CloseItem().NewLine()
	}

	npcSender.NewProcessor(p.l, p.ctx).SendSimple(ctx.Field().Channel(), ctx.CharacterId(), ctx.NpcId())(mb.String())
	return state.Id(), nil
}

// processAskNumberState processes an ask number state
func (p *ProcessorImpl) processAskNumberState(ctx ConversationContext, state StateModel) (string, error) {
	askNumber := state.AskNumber()
	if askNumber == nil {
		return "", errors.New("askNumber is nil")
	}

	// Replace context placeholders in the ask number text
	processedText, err := ReplaceContextPlaceholders(askNumber.Text(), ctx.Context())
	if err != nil {
		p.l.WithError(err).Warnf("Failed to replace context placeholders in ask number text for state [%s]. Using original text.", state.Id())
		processedText = askNumber.Text()
	}

	// Send the ask number request to the client
	err = npcSender.NewProcessor(p.l, p.ctx).SendNumber(ctx.Field().Channel(), ctx.CharacterId(), ctx.NpcId(), processedText, askNumber.DefaultValue(), askNumber.MinValue(), askNumber.MaxValue())

	if err != nil {
		p.l.WithError(err).Errorf("Failed to send number request for state [%s] to character [%d]", state.Id(), ctx.CharacterId())
		return "", err
	}

	// Return the current state ID to indicate that we're waiting for input
	return state.Id(), nil
}

// processAskStyleState processes an ask style state
func (p *ProcessorImpl) processAskStyleState(ctx ConversationContext, state StateModel) (string, error) {
	askStyle := state.AskStyle()
	if askStyle == nil {
		return "", errors.New("askStyle is nil")
	}

	// Resolve styles - support both static and dynamic
	styles := askStyle.Styles()

	// If no static styles, try to load from context
	if len(styles) == 0 && askStyle.StylesContextKey() != "" {
		contextKey := askStyle.StylesContextKey()

		// Get styles from context
		stylesStr, exists := ctx.Context()[contextKey]
		if !exists {
			p.l.Errorf("StylesContextKey [%s] not found in context for character [%d] in state [%s]",
				contextKey, ctx.CharacterId(), state.Id())
			return "", fmt.Errorf("styles not found in context: %s", contextKey)
		}

		// Decode styles
		var err error
		styles, err = decodeUint32Array(stylesStr)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to decode styles from context key [%s] for state [%s]", contextKey, state.Id())
			return "", fmt.Errorf("invalid styles in context: %w", err)
		}

		p.l.Debugf("Loaded %d styles from context key [%s] for character [%d] in state [%s]",
			len(styles), contextKey, ctx.CharacterId(), state.Id())
	}

	// Validate we have styles
	if len(styles) == 0 {
		p.l.Errorf("No styles available for state [%s] (neither static nor from context)", state.Id())
		return "", errors.New("no styles available (neither static nor from context)")
	}

	// Replace context placeholders in the ask style text
	processedText, err := ReplaceContextPlaceholders(askStyle.Text(), ctx.Context())
	if err != nil {
		p.l.WithError(err).Warnf("Failed to replace context placeholders in ask style text for state [%s]. Using original text.", state.Id())
		processedText = askStyle.Text()
	}

	// Send the ask style request to the client
	err = npcSender.NewProcessor(p.l, p.ctx).SendStyle(ctx.Field().Channel(), ctx.CharacterId(), ctx.NpcId(), processedText, styles)

	if err != nil {
		p.l.WithError(err).Errorf("Failed to send style request for state [%s] to character [%d]", state.Id(), ctx.CharacterId())
		return "", err
	}

	// Return the current state ID to indicate that we're waiting for input
	return state.Id(), nil
}

// processAskSlideMenuState processes an ask slide menu state
func (p *ProcessorImpl) processAskSlideMenuState(ctx ConversationContext, state StateModel) (string, error) {
	askSlideMenu := state.AskSlideMenu()
	if askSlideMenu == nil {
		return "", errors.New("askSlideMenu is nil")
	}

	// Replace context placeholders in the title
	processedTitle, err := ReplaceContextPlaceholders(askSlideMenu.Title(), ctx.Context())
	if err != nil {
		p.l.WithError(err).Warnf("Failed to replace context placeholders in slide menu title for state [%s]. Using original title.", state.Id())
		processedTitle = askSlideMenu.Title()
	}

	// Build the message with choices
	mb := message.NewBuilder().AddText(processedTitle)
	for i, choice := range askSlideMenu.Choices() {
		if choice.NextState() == "" || choice.Text() == "Exit" {
			continue
		}

		// Replace context placeholders in choice text
		processedChoiceText, err := ReplaceContextPlaceholders(choice.Text(), ctx.Context())
		if err != nil {
			p.l.WithError(err).Warnf("Failed to replace context placeholders in choice text for state [%s]. Using original text.", state.Id())
			processedChoiceText = choice.Text()
		}

		mb.DimensionalMirrorOption(i, processedChoiceText)
	}

	// Send the slide menu request to the client
	err = npcSender.NewProcessor(p.l, p.ctx).SendSlideMenu(ctx.Field().Channel(), ctx.CharacterId(), ctx.NpcId(), mb.String(), askSlideMenu.MenuType())

	if err != nil {
		p.l.WithError(err).Errorf("Failed to send slide menu request for state [%s] to character [%d]", state.Id(), ctx.CharacterId())
		return "", err
	}

	// Return the current state ID to indicate that we're waiting for input
	return state.Id(), nil
}

func (p *ProcessorImpl) End(characterId uint32) error {
	p.l.Debugf("Ending conversation with character [%d].", characterId)
	GetRegistry().ClearContext(p.t, characterId)
	return nil
}
