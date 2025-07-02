package conversation

import (
	"atlas-npc-conversations/message"
	"atlas-npc-conversations/npc"
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"time"
)

type Processor interface {
	// Start starts a conversation with an NPC
	Start(field field.Model, npcId uint32, characterId uint32) error

	// Continue continues a conversation with an NPC
	Continue(npcId uint32, characterId uint32, action byte, lastMessageType byte, selection int32) error

	// ContinueViaEvent continues a conversation via an event
	ContinueViaEvent(characterId uint32, action byte, referenceId int32) error

	// End ends a conversation
	End(characterId uint32) error

	// Create creates a new conversation
	Create(model Model) (Model, error)

	// Update updates an existing conversation
	Update(id uuid.UUID, model Model) (Model, error)

	// Delete deletes a conversation
	Delete(id uuid.UUID) error

	// ByIdProvider returns a provider for retrieving a conversation by ID
	ByIdProvider(id uuid.UUID) model.Provider[Model]

	// ByNpcIdProvider returns a provider for retrieving a conversation by NPC ID
	ByNpcIdProvider(npcId uint32) model.Provider[Model]

	// AllProvider returns a provider for retrieving all conversations
	AllProvider() model.Provider[[]Model]
}

type ProcessorImpl struct {
	l         logrus.FieldLogger
	ctx       context.Context
	t         tenant.Model
	db        *gorm.DB
	evaluator Evaluator
	executor  OperationExecutor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	t := tenant.MustFromContext(ctx)
	evaluator := NewEvaluator(l, ctx, t)
	executor := NewOperationExecutor(l, ctx)

	return &ProcessorImpl{
		l:         l,
		ctx:       ctx,
		t:         t,
		db:        db,
		evaluator: evaluator,
		executor:  executor,
	}
}

// ByIdProvider returns a provider for retrieving a conversation by ID
func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	return model.Map[Entity, Model](Make)(GetByIdProvider(p.t.Id())(id)(p.db))
}

// ByNpcIdProvider returns a provider for retrieving a conversation by NPC ID
func (p *ProcessorImpl) ByNpcIdProvider(npcId uint32) model.Provider[Model] {
	return model.Map[Entity, Model](Make)(GetByNpcIdProvider(p.t.Id())(npcId)(p.db))
}

// AllProvider returns a provider for retrieving all conversations
func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(GetAllProvider(p.t.Id())(p.db))(model.ParallelMap())
}

// Create creates a new conversation
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	p.l.Debugf("Creating conversation for NPC [%d]", m.NpcId())

	// Convert model to entity
	entity, err := ToEntity(m, p.t.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Failed to convert model to entity")
		return Model{}, err
	}

	// Auto-generate UUID if nil
	if entity.ID == uuid.Nil {
		entity.ID = uuid.New()
	}

	// Save to database
	result := p.db.Create(&entity)
	if result.Error != nil {
		p.l.WithError(result.Error).Errorf("Failed to create conversation")
		return Model{}, result.Error
	}

	// Convert back to model
	return Make(entity)
}

// Update updates an existing conversation
func (p *ProcessorImpl) Update(id uuid.UUID, m Model) (Model, error) {
	p.l.Debugf("Updating conversation [%s]", id)

	// Check if conversation exists
	var existingEntity Entity
	result := p.db.Where("tenant_id = ? AND id = ?", p.t.Id(), id).First(&existingEntity)
	if result.Error != nil {
		p.l.WithError(result.Error).Errorf("Failed to find conversation [%s]", id)
		return Model{}, result.Error
	}

	// Convert model to entity
	entity, err := ToEntity(m, p.t.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Failed to convert model to entity")
		return Model{}, err
	}

	// Ensure ID is preserved
	entity.ID = id

	// Update in database
	result = p.db.Model(&Entity{}).Where("tenant_id = ? AND id = ?", p.t.Id(), id).Updates(map[string]interface{}{
		"npc_id":     entity.NpcID,
		"data":       entity.Data,
		"updated_at": time.Now(),
	})
	if result.Error != nil {
		p.l.WithError(result.Error).Errorf("Failed to update conversation [%s]", id)
		return Model{}, result.Error
	}

	// Retrieve updated entity
	result = p.db.Where("tenant_id = ? AND id = ?", p.t.Id(), id).First(&entity)
	if result.Error != nil {
		p.l.WithError(result.Error).Errorf("Failed to retrieve updated conversation [%s]", id)
		return Model{}, result.Error
	}

	// Convert back to model
	return Make(entity)
}

// Delete deletes a conversation
func (p *ProcessorImpl) Delete(id uuid.UUID) error {
	p.l.Debugf("Deleting conversation [%s]", id)

	// Delete from database
	result := p.db.Where("tenant_id = ? AND id = ?", p.t.Id(), id).Delete(&Entity{})
	if result.Error != nil {
		p.l.WithError(result.Error).Errorf("Failed to delete conversation [%s]", id)
		return result.Error
	}

	return nil
}

func (p *ProcessorImpl) Start(field field.Model, npcId uint32, characterId uint32) error {
	p.l.Debugf("Starting conversation with NPC [%d] with character [%d] in map [%d].", npcId, characterId, field.MapId())

	// Check if there's already a conversation in progress
	_, err := GetRegistry().GetPreviousContext(p.t, characterId)
	if err == nil {
		p.l.Debugf("Previous conversation for character [%d] exists, avoiding starting new conversation with NPC [%d].", characterId, npcId)
		return errors.New("another conversation exists")
	}

	// Get the conversation for this NPC
	conversation, err := p.ByNpcIdProvider(npcId)()
	if err != nil {
		p.l.WithError(err).Errorf("Failed to retrieve conversation for NPC [%d]", npcId)
		return err
	}

	// Get the start state
	startStateId := conversation.StartState()

	// Create a conversation context
	ctx, err := NewConversationContextBuilder().
		SetField(field).
		SetCharacterId(characterId).
		SetNpcId(npcId).
		SetCurrentState(startStateId).
		SetConversation(conversation).
		Build()
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create conversation context for character [%d] and NPC [%d]", characterId, npcId)
		return err
	}

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

	ctx, err = builder.Build()
	if err != nil {
		p.l.WithError(err).Errorf("Failed to update conversation context for character [%d] and NPC [%d]", ctx.CharacterId(), ctx.NpcId())
		return err
	}

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
			p.l.WithError(err).Errorf("Failed to process state [%s] for character [%d] and NPC [%d]", nextStateId, characterId, npcId)
			return err
		}
	}
	return nil
}

func (p *ProcessorImpl) ContinueViaEvent(characterId uint32, action byte, referenceId int32) error {
	// Get the previous context
	cc, err := GetRegistry().GetPreviousContext(p.t, characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve conversation context for [%d].", characterId)
		return errors.New("conversation context not found")
	}

	p.l.Debugf("Continuing conversation via event for character [%d] with NPC [%d] in map [%d].", characterId, cc.NpcId(), cc.Field().MapId())
	p.l.Debugf("Event details: action [%d], referenceId [%d].", action, referenceId)

	// Get the current state
	currentStateId := cc.CurrentState()
	conversation := cc.Conversation()

	// Find the current state in the conversation - just to validate it exists
	_, err = conversation.FindState(currentStateId)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to find state [%s] for character [%d]", currentStateId, characterId)
		return err
	}

	// Process the event based on the action
	// For now, we'll just use the referenceId as a direct reference to the next state ID
	// In a more complex implementation, we might have a mapping of event references to state transitions

	// Find the next state using the referenceId
	var nextStateId string

	// This is a simplified approach - in a real implementation, you might have a more complex mapping
	// between event references and state transitions
	if referenceId >= 0 {
		// Use the referenceId as an index into the available states
		states := conversation.States()
		if int(referenceId) < len(states) {
			nextStateId = states[referenceId].Id()
		} else {
			return fmt.Errorf("invalid referenceId: %d (states: %d)", referenceId, len(states))
		}
	} else {
		// Negative referenceId could be used for special transitions
		// For now, just end the conversation
		nextStateId = ""
	}

	// If there's a next state, process it
	if nextStateId != "" {
		// Find the next state
		nextState, err := conversation.FindState(nextStateId)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to find next state [%s] for character [%d]", nextStateId, characterId)
			return err
		}

		// Process the next state
		finalStateId, err := p.processState(cc, nextState)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to process next state [%s] for character [%d]", nextStateId, characterId)
			return err
		}

		// If there's a final state, update the context and store it
		if finalStateId != "" {
			// Update the context with the final state
			builder := NewConversationContextBuilder().
				SetField(cc.Field()).
				SetCharacterId(characterId).
				SetNpcId(cc.NpcId()).
				SetCurrentState(finalStateId).
				SetConversation(conversation)

			// Preserve existing context
			existingContext := cc.Context()
			for k, v := range existingContext {
				builder.AddContextValue(k, v)
			}

			newCtx, err := builder.Build()
			if err != nil {
				p.l.WithError(err).Errorf("Failed to update conversation context for character [%d]", characterId)
				return err
			}

			// Store the updated context
			GetRegistry().SetContext(p.t, characterId, newCtx)
		} else {
			// No final state, end the conversation
			GetRegistry().ClearContext(p.t, characterId)
		}
	} else {
		// No next state, end the conversation
		GetRegistry().ClearContext(p.t, characterId)
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
		// Update the context with the next state
		builder := NewConversationContextBuilder().
			SetField(ctx.Field()).
			SetCharacterId(ctx.CharacterId()).
			SetNpcId(ctx.NpcId()).
			SetCurrentState(nextStateId).
			SetConversation(ctx.Conversation())

		// Preserve existing context
		existingContext := ctx.Context()
		for k, v := range existingContext {
			builder.AddContextValue(k, v)
		}

		ctx, err = builder.Build()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to update conversation context for character [%d] and NPC [%d]", ctx.CharacterId(), ctx.NpcId())
			return false, err
		}

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
	case ListSelectionType:
		// Process list selection state
		return p.processListSelectionState(ctx, state)
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

	// TODO: Send the dialogue to the client
	if dialogue.dialogueType == SendNext {
		npc.NewProcessor(p.l, p.ctx).SendNext(ctx.Field().WorldId(), ctx.Field().ChannelId(), ctx.CharacterId(), ctx.NpcId())(dialogue.Text())
	} else if dialogue.dialogueType == SendOk {
		npc.NewProcessor(p.l, p.ctx).SendOk(ctx.Field().WorldId(), ctx.Field().ChannelId(), ctx.CharacterId(), ctx.NpcId())(dialogue.Text())
	} else if dialogue.dialogueType == SendYesNo {
		npc.NewProcessor(p.l, p.ctx).SendYesNo(ctx.Field().WorldId(), ctx.Field().ChannelId(), ctx.CharacterId(), ctx.NpcId())(dialogue.Text())
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

	// Execute operations
	for _, operation := range genericAction.Operations() {
		err := p.executor.ExecuteOperation(ctx.Field(), ctx.CharacterId(), operation)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to execute operation [%s] for character [%d]", operation.Type(), ctx.CharacterId())
			return "", err
		}
	}

	// Evaluate outcomes
	for _, outcome := range genericAction.Outcomes() {
		if len(outcome.Conditions()) == 0 {
			return outcome.NextState(), nil
		}

		// Evaluate the condition
		// TODO
		passed, err := p.evaluator.EvaluateCondition(ctx.CharacterId(), outcome.Conditions()[0])
		if err != nil {
			p.l.WithError(err).Errorf("Failed to evaluate condition [%+v] for character [%d]", outcome.Conditions()[0], ctx.CharacterId())
			return "", err
		}

		// If the condition passed, return the appropriate next state
		if passed {
			if outcome.SuccessState() != "" {
				return outcome.SuccessState(), nil
			}
			return outcome.NextState(), nil
		} else if outcome.FailureState() != "" {
			return outcome.FailureState(), nil
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

	// TODO: Implement craft action processing
	// For now, just return the success state
	return craftAction.SuccessState(), nil
}

// processListSelectionState processes a list selection state
func (p *ProcessorImpl) processListSelectionState(ctx ConversationContext, state StateModel) (string, error) {
	listSelection := state.ListSelection()
	if listSelection == nil {
		return "", errors.New("listSelection is nil")
	}

	mb := message.NewBuilder().AddText(listSelection.Title()).NewLine()
	for i, choice := range listSelection.Choices() {
		if choice.NextState() == "" {
			continue
		}
		mb.OpenItem(i).BlueText().AddText(choice.Text()).CloseItem().NewLine()
	}

	npc.NewProcessor(p.l, p.ctx).SendSimple(ctx.Field().WorldId(), ctx.Field().ChannelId(), ctx.CharacterId(), ctx.NpcId())(mb.String())
	return state.Id(), nil
}

func (p *ProcessorImpl) End(characterId uint32) error {
	p.l.Debugf("Ending conversation with character [%d].", characterId)
	GetRegistry().ClearContext(p.t, characterId)
	return nil
}
