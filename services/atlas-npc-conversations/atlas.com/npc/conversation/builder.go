package conversation

import (
	"errors"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// StateBuilder is a builder for StateModel
type StateBuilder struct {
	id                    string
	stateType             StateType
	dialogue              *DialogueModel
	genericAction         *GenericActionModel
	craftAction           *CraftActionModel
	transportAction       *TransportActionModel
	gachaponAction        *GachaponActionModel
	rpsAction             *RPSActionModel
	partyQuestAction      *PartyQuestActionModel
	partyQuestBonusAction *PartyQuestBonusActionModel
	listSelection         *ListSelectionModel
	askNumber             *AskNumberModel
	askText               *AskTextModel
	askStyle              *AskStyleModel
	askSlideMenu          *AskSlideMenuModel
}

// NewStateBuilder creates a new StateBuilder
func NewStateBuilder() *StateBuilder {
	return &StateBuilder{}
}

// SetId sets the state ID
func (b *StateBuilder) SetId(id string) *StateBuilder {
	b.id = id
	return b
}

// SetDialogue sets the dialogue model
func (b *StateBuilder) SetDialogue(dialogue *DialogueModel) *StateBuilder {
	b.stateType = DialogueStateType
	b.dialogue = dialogue
	b.genericAction = nil
	b.craftAction = nil
	b.transportAction = nil
	b.gachaponAction = nil
	b.rpsAction = nil
	b.partyQuestAction = nil
	b.partyQuestBonusAction = nil
	b.listSelection = nil
	b.askNumber = nil
	b.askText = nil
	b.askStyle = nil
	b.askSlideMenu = nil
	return b
}

// SetGenericAction sets the generic action model
func (b *StateBuilder) SetGenericAction(genericAction *GenericActionModel) *StateBuilder {
	b.stateType = GenericActionType
	b.dialogue = nil
	b.genericAction = genericAction
	b.craftAction = nil
	b.transportAction = nil
	b.gachaponAction = nil
	b.rpsAction = nil
	b.partyQuestAction = nil
	b.partyQuestBonusAction = nil
	b.listSelection = nil
	b.askNumber = nil
	b.askText = nil
	b.askStyle = nil
	b.askSlideMenu = nil
	return b
}

// SetCraftAction sets the craft action model
func (b *StateBuilder) SetCraftAction(craftAction *CraftActionModel) *StateBuilder {
	b.stateType = CraftActionType
	b.dialogue = nil
	b.genericAction = nil
	b.craftAction = craftAction
	b.transportAction = nil
	b.gachaponAction = nil
	b.rpsAction = nil
	b.partyQuestAction = nil
	b.partyQuestBonusAction = nil
	b.listSelection = nil
	b.askNumber = nil
	b.askText = nil
	b.askStyle = nil
	b.askSlideMenu = nil
	return b
}

// SetTransportAction sets the transport action model
func (b *StateBuilder) SetTransportAction(transportAction *TransportActionModel) *StateBuilder {
	b.stateType = TransportActionType
	b.dialogue = nil
	b.genericAction = nil
	b.craftAction = nil
	b.transportAction = transportAction
	b.gachaponAction = nil
	b.rpsAction = nil
	b.partyQuestAction = nil
	b.partyQuestBonusAction = nil
	b.listSelection = nil
	b.askNumber = nil
	b.askText = nil
	b.askStyle = nil
	b.askSlideMenu = nil
	return b
}

// SetGachaponAction sets the gachapon action model
func (b *StateBuilder) SetGachaponAction(gachaponAction *GachaponActionModel) *StateBuilder {
	b.stateType = GachaponActionType
	b.dialogue = nil
	b.genericAction = nil
	b.craftAction = nil
	b.transportAction = nil
	b.gachaponAction = gachaponAction
	b.rpsAction = nil
	b.partyQuestAction = nil
	b.partyQuestBonusAction = nil
	b.listSelection = nil
	b.askNumber = nil
	b.askText = nil
	b.askStyle = nil
	b.askSlideMenu = nil
	return b
}

// SetRPSAction sets the RPS action model
func (b *StateBuilder) SetRPSAction(rpsAction *RPSActionModel) *StateBuilder {
	b.stateType = RPSActionType
	b.dialogue = nil
	b.genericAction = nil
	b.craftAction = nil
	b.transportAction = nil
	b.gachaponAction = nil
	b.rpsAction = rpsAction
	b.partyQuestAction = nil
	b.partyQuestBonusAction = nil
	b.listSelection = nil
	b.askNumber = nil
	b.askText = nil
	b.askStyle = nil
	b.askSlideMenu = nil
	return b
}

// SetPartyQuestAction sets the party quest action model
func (b *StateBuilder) SetPartyQuestAction(partyQuestAction *PartyQuestActionModel) *StateBuilder {
	b.stateType = PartyQuestActionType
	b.dialogue = nil
	b.genericAction = nil
	b.craftAction = nil
	b.transportAction = nil
	b.gachaponAction = nil
	b.rpsAction = nil
	b.partyQuestAction = partyQuestAction
	b.partyQuestBonusAction = nil
	b.listSelection = nil
	b.askNumber = nil
	b.askText = nil
	b.askStyle = nil
	b.askSlideMenu = nil
	return b
}

// SetPartyQuestBonusAction sets the party quest bonus action model
func (b *StateBuilder) SetPartyQuestBonusAction(partyQuestBonusAction *PartyQuestBonusActionModel) *StateBuilder {
	b.stateType = PartyQuestBonusActionType
	b.dialogue = nil
	b.genericAction = nil
	b.craftAction = nil
	b.transportAction = nil
	b.gachaponAction = nil
	b.rpsAction = nil
	b.partyQuestAction = nil
	b.partyQuestBonusAction = partyQuestBonusAction
	b.listSelection = nil
	b.askNumber = nil
	b.askText = nil
	b.askStyle = nil
	b.askSlideMenu = nil
	return b
}

// SetListSelection sets the list selection model
func (b *StateBuilder) SetListSelection(listSelection *ListSelectionModel) *StateBuilder {
	b.stateType = ListSelectionType
	b.dialogue = nil
	b.genericAction = nil
	b.craftAction = nil
	b.transportAction = nil
	b.gachaponAction = nil
	b.rpsAction = nil
	b.partyQuestAction = nil
	b.listSelection = listSelection
	b.askNumber = nil
	b.askText = nil
	b.askStyle = nil
	b.askSlideMenu = nil
	return b
}

// SetAskNumber sets the ask number model
func (b *StateBuilder) SetAskNumber(askNumber *AskNumberModel) *StateBuilder {
	b.stateType = AskNumberType
	b.dialogue = nil
	b.genericAction = nil
	b.craftAction = nil
	b.transportAction = nil
	b.gachaponAction = nil
	b.rpsAction = nil
	b.partyQuestAction = nil
	b.partyQuestBonusAction = nil
	b.listSelection = nil
	b.askNumber = askNumber
	b.askText = nil
	b.askStyle = nil
	b.askSlideMenu = nil
	return b
}

// SetAskText sets the ask text model
func (b *StateBuilder) SetAskText(askText *AskTextModel) *StateBuilder {
	b.stateType = AskTextType
	b.dialogue = nil
	b.genericAction = nil
	b.craftAction = nil
	b.transportAction = nil
	b.gachaponAction = nil
	b.rpsAction = nil
	b.partyQuestAction = nil
	b.partyQuestBonusAction = nil
	b.listSelection = nil
	b.askNumber = nil
	b.askText = askText
	b.askStyle = nil
	b.askSlideMenu = nil
	return b
}

// SetAskStyle sets the ask style model
func (b *StateBuilder) SetAskStyle(askStyle *AskStyleModel) *StateBuilder {
	b.stateType = AskStyleType
	b.dialogue = nil
	b.genericAction = nil
	b.craftAction = nil
	b.transportAction = nil
	b.gachaponAction = nil
	b.rpsAction = nil
	b.partyQuestAction = nil
	b.partyQuestBonusAction = nil
	b.listSelection = nil
	b.askNumber = nil
	b.askText = nil
	b.askStyle = askStyle
	b.askSlideMenu = nil
	return b
}

// SetAskSlideMenu sets the ask slide menu model
func (b *StateBuilder) SetAskSlideMenu(askSlideMenu *AskSlideMenuModel) *StateBuilder {
	b.stateType = AskSlideMenuType
	b.dialogue = nil
	b.genericAction = nil
	b.craftAction = nil
	b.transportAction = nil
	b.gachaponAction = nil
	b.rpsAction = nil
	b.partyQuestAction = nil
	b.partyQuestBonusAction = nil
	b.listSelection = nil
	b.askNumber = nil
	b.askText = nil
	b.askStyle = nil
	b.askSlideMenu = askSlideMenu
	return b
}

// Build builds the StateModel
func (b *StateBuilder) Build() (StateModel, error) {
	if b.id == "" {
		return StateModel{}, errors.New("id is required")
	}

	switch b.stateType {
	case DialogueStateType:
		if b.dialogue == nil {
			return StateModel{}, errors.New("dialogue is required for dialogue state")
		}
	case GenericActionType:
		if b.genericAction == nil {
			return StateModel{}, errors.New("genericAction is required for genericAction state")
		}
	case CraftActionType:
		if b.craftAction == nil {
			return StateModel{}, errors.New("craftAction is required for craftAction state")
		}
	case TransportActionType:
		if b.transportAction == nil {
			return StateModel{}, errors.New("transportAction is required for transportAction state")
		}
	case GachaponActionType:
		if b.gachaponAction == nil {
			return StateModel{}, errors.New("gachaponAction is required for gachaponAction state")
		}
	case RPSActionType:
		if b.rpsAction == nil {
			return StateModel{}, errors.New("rpsAction is required for rpsAction state")
		}
	case PartyQuestActionType:
		if b.partyQuestAction == nil {
			return StateModel{}, errors.New("partyQuestAction is required for partyQuestAction state")
		}
	case PartyQuestBonusActionType:
		if b.partyQuestBonusAction == nil {
			return StateModel{}, errors.New("partyQuestBonusAction is required for partyQuestBonusAction state")
		}
	case ListSelectionType:
		if b.listSelection == nil {
			return StateModel{}, errors.New("listSelection is required for listSelection state")
		}
	case AskNumberType:
		if b.askNumber == nil {
			return StateModel{}, errors.New("askNumber is required for askNumber state")
		}
	case AskTextType:
		if b.askText == nil {
			return StateModel{}, errors.New("askText is required for askText state")
		}
	case AskStyleType:
		if b.askStyle == nil {
			return StateModel{}, errors.New("askStyle is required for askStyle state")
		}
	case AskSlideMenuType:
		if b.askSlideMenu == nil {
			return StateModel{}, errors.New("askSlideMenu is required for askSlideMenu state")
		}
	default:
		return StateModel{}, errors.New("invalid state type")
	}

	return StateModel{
		id:                    b.id,
		stateType:             b.stateType,
		dialogue:              b.dialogue,
		genericAction:         b.genericAction,
		craftAction:           b.craftAction,
		transportAction:       b.transportAction,
		gachaponAction:        b.gachaponAction,
		rpsAction:             b.rpsAction,
		partyQuestAction:      b.partyQuestAction,
		partyQuestBonusAction: b.partyQuestBonusAction,
		listSelection:         b.listSelection,
		askNumber:             b.askNumber,
		askText:               b.askText,
		askStyle:              b.askStyle,
		askSlideMenu:          b.askSlideMenu,
	}, nil
}

// DialogueBuilder is a builder for DialogueModel
type DialogueBuilder struct {
	dialogueType   DialogueType
	text           string
	speaker        string
	endChat        bool
	secondaryNpcId uint32
	choices        []ChoiceModel
}

// NewDialogueBuilder creates a new DialogueBuilder
func NewDialogueBuilder() *DialogueBuilder {
	return &DialogueBuilder{
		endChat: true, // Default to showing end chat button
		choices: make([]ChoiceModel, 0),
	}
}

// SetDialogueType sets the dialogue type
func (b *DialogueBuilder) SetDialogueType(dialogueType DialogueType) *DialogueBuilder {
	b.dialogueType = dialogueType
	return b
}

// SetText sets the dialogue text
func (b *DialogueBuilder) SetText(text string) *DialogueBuilder {
	b.text = text
	return b
}

// SetSpeaker sets the speaker type ("NPC" or "CHARACTER")
func (b *DialogueBuilder) SetSpeaker(speaker string) *DialogueBuilder {
	b.speaker = speaker
	return b
}

// SetEndChat sets whether to show the end chat button
func (b *DialogueBuilder) SetEndChat(endChat bool) *DialogueBuilder {
	b.endChat = endChat
	return b
}

// SetSecondaryNpcId sets the secondary NPC template ID
func (b *DialogueBuilder) SetSecondaryNpcId(npcId uint32) *DialogueBuilder {
	b.secondaryNpcId = npcId
	return b
}

// SetChoices sets the dialogue choices
func (b *DialogueBuilder) SetChoices(choices []ChoiceModel) *DialogueBuilder {
	b.choices = choices
	return b
}

// AddChoice adds a dialogue choice
func (b *DialogueBuilder) AddChoice(choice ChoiceModel) *DialogueBuilder {
	b.choices = append(b.choices, choice)
	return b
}

// Build builds the DialogueModel
func (b *DialogueBuilder) Build() (*DialogueModel, error) {
	if b.dialogueType == "" {
		return nil, errors.New("dialogueType is required")
	}
	if b.text == "" {
		return nil, errors.New("text is required")
	}

	// Validate choices based on dialogue type
	switch b.dialogueType {
	case SendOk:
		if len(b.choices) != 2 {
			return nil, errors.New("sendOk requires exactly 2 choices")
		}
	case SendNext:
		if len(b.choices) != 2 {
			return nil, errors.New("sendNext requires exactly 2 choices")
		}
	case SendNextPrev:
		if len(b.choices) != 3 {
			return nil, errors.New("sendNextPrev requires exactly 3 choices")
		}
	case SendPrev:
		if len(b.choices) != 3 {
			return nil, errors.New("sendPrev requires exactly 3 choices")
		}
	case SendYesNo:
		if len(b.choices) != 3 {
			return nil, errors.New("sendYesNo requires exactly 3 choices")
		}
	case SendAcceptDecline:
		if len(b.choices) != 3 {
			return nil, errors.New("sendAcceptDecline requires exactly 3 choices")
		}
	}

	return &DialogueModel{
		dialogueType:   b.dialogueType,
		text:           b.text,
		speaker:        b.speaker,
		endChat:        b.endChat,
		secondaryNpcId: b.secondaryNpcId,
		choices:        b.choices,
	}, nil
}

// ChoiceBuilder is a builder for ChoiceModel
type ChoiceBuilder struct {
	text      string
	nextState string
	context   map[string]string
}

// NewChoiceBuilder creates a new ChoiceBuilder
func NewChoiceBuilder() *ChoiceBuilder {
	return &ChoiceBuilder{
		context: make(map[string]string),
	}
}

// SetText sets the choice text
func (b *ChoiceBuilder) SetText(text string) *ChoiceBuilder {
	b.text = text
	return b
}

// SetNextState sets the next state ID
func (b *ChoiceBuilder) SetNextState(nextState string) *ChoiceBuilder {
	b.nextState = nextState
	return b
}

// SetContext sets the entire context map
func (b *ChoiceBuilder) SetContext(context map[string]string) *ChoiceBuilder {
	b.context = context
	return b
}

// AddContextValue adds a key-value pair to the context map
func (b *ChoiceBuilder) AddContextValue(key, value string) *ChoiceBuilder {
	b.context[key] = value
	return b
}

// Build builds the ChoiceModel
func (b *ChoiceBuilder) Build() (ChoiceModel, error) {
	if b.text == "" {
		return ChoiceModel{}, errors.New("text is required")
	}

	return ChoiceModel{
		text:      b.text,
		nextState: b.nextState,
		context:   b.context,
	}, nil
}

// GenericActionBuilder is a builder for GenericActionModel
type GenericActionBuilder struct {
	operations []OperationModel
	outcomes   []OutcomeModel
}

// NewGenericActionBuilder creates a new GenericActionBuilder
func NewGenericActionBuilder() *GenericActionBuilder {
	return &GenericActionBuilder{
		operations: make([]OperationModel, 0),
		outcomes:   make([]OutcomeModel, 0),
	}
}

// SetOperations sets the operations
func (b *GenericActionBuilder) SetOperations(operations []OperationModel) *GenericActionBuilder {
	b.operations = operations
	return b
}

// AddOperation adds an operation
func (b *GenericActionBuilder) AddOperation(operation OperationModel) *GenericActionBuilder {
	b.operations = append(b.operations, operation)
	return b
}

// SetOutcomes sets the outcomes
func (b *GenericActionBuilder) SetOutcomes(outcomes []OutcomeModel) *GenericActionBuilder {
	b.outcomes = outcomes
	return b
}

// AddOutcome adds an outcome
func (b *GenericActionBuilder) AddOutcome(outcome OutcomeModel) *GenericActionBuilder {
	b.outcomes = append(b.outcomes, outcome)
	return b
}

// Build builds the GenericActionModel
func (b *GenericActionBuilder) Build() (*GenericActionModel, error) {
	if len(b.operations) == 0 && len(b.outcomes) == 0 {
		return nil, errors.New("at least one operation or outcome is required")
	}

	return &GenericActionModel{
		operations: b.operations,
		outcomes:   b.outcomes,
	}, nil
}

// OperationBuilder is a builder for OperationModel
type OperationBuilder struct {
	operationType string
	params        map[string]string
}

// NewOperationBuilder creates a new OperationBuilder
func NewOperationBuilder() *OperationBuilder {
	return &OperationBuilder{
		params: make(map[string]string),
	}
}

// SetType sets the operation type
func (b *OperationBuilder) SetType(operationType string) *OperationBuilder {
	b.operationType = operationType
	return b
}

// SetParams sets the operation parameters
func (b *OperationBuilder) SetParams(params map[string]string) *OperationBuilder {
	b.params = params
	return b
}

// AddParamValue adds an operation parameter value
func (b *OperationBuilder) AddParamValue(key string, value string) *OperationBuilder {
	b.params[key] = value
	return b
}

// Build builds the OperationModel
func (b *OperationBuilder) Build() (OperationModel, error) {
	if b.operationType == "" {
		return OperationModel{}, errors.New("type is required")
	}

	return OperationModel{
		operationType: b.operationType,
		params:        b.params,
	}, nil
}

// ConditionBuilder is a builder for ConditionModel
type ConditionBuilder struct {
	conditionType   string
	operator        string
	value           string
	referenceId     string
	step            string
	worldId         string
	channelId       string
	includeEquipped bool
}

// NewConditionBuilder creates a new ConditionBuilder
func NewConditionBuilder() *ConditionBuilder {
	return &ConditionBuilder{}
}

// SetType sets the condition type
func (b *ConditionBuilder) SetType(condType string) *ConditionBuilder {
	b.conditionType = condType
	return b
}

// SetOperator sets the operator
func (b *ConditionBuilder) SetOperator(op string) *ConditionBuilder {
	b.operator = op
	return b
}

// SetValue sets the value
func (b *ConditionBuilder) SetValue(value string) *ConditionBuilder {
	b.value = value
	return b
}

// SetReferenceId sets the reference ID
func (b *ConditionBuilder) SetReferenceId(referenceId string) *ConditionBuilder {
	b.referenceId = referenceId
	return b
}

// SetStep sets the step
func (b *ConditionBuilder) SetStep(step string) *ConditionBuilder {
	b.step = step
	return b
}

// SetWorldId sets the worldId
func (b *ConditionBuilder) SetWorldId(worldId string) *ConditionBuilder {
	b.worldId = worldId
	return b
}

// SetChannelId sets the channelId
func (b *ConditionBuilder) SetChannelId(channelId string) *ConditionBuilder {
	b.channelId = channelId
	return b
}

// SetIncludeEquipped sets whether to include equipped items in item condition checks
func (b *ConditionBuilder) SetIncludeEquipped(includeEquipped bool) *ConditionBuilder {
	b.includeEquipped = includeEquipped
	return b
}

// Build builds the ConditionModel
func (b *ConditionBuilder) Build() (ConditionModel, error) {
	if b.conditionType == "" {
		return ConditionModel{}, errors.New("condition type is required")
	}
	if b.operator == "" {
		return ConditionModel{}, errors.New("operator is required")
	}
	if b.value == "" {
		return ConditionModel{}, errors.New("value is required")
	}

	return ConditionModel{
		conditionType:   b.conditionType,
		operator:        b.operator,
		value:           b.value,
		referenceId:     b.referenceId,
		step:            b.step,
		worldId:         b.worldId,
		channelId:       b.channelId,
		includeEquipped: b.includeEquipped,
	}, nil
}

// OutcomeBuilder is a builder for OutcomeModel
type OutcomeBuilder struct {
	conditions []ConditionModel
	nextState  string
}

// NewOutcomeBuilder creates a new OutcomeBuilder
func NewOutcomeBuilder() *OutcomeBuilder {
	return &OutcomeBuilder{
		conditions: make([]ConditionModel, 0),
	}
}

// AddCondition adds a outcome condition
func (b *OutcomeBuilder) AddCondition(condition ConditionModel) *OutcomeBuilder {
	b.conditions = append(b.conditions, condition)
	return b
}

// AddConditionFromInput adds a outcome condition from input parameters
func (b *OutcomeBuilder) AddConditionFromInput(condType string, operator string, value string) *OutcomeBuilder {
	condition, err := NewConditionBuilder().
		SetType(condType).
		SetOperator(operator).
		SetValue(value).
		Build()

	if err == nil {
		b.conditions = append(b.conditions, condition)
	}

	return b
}

// SetNextState sets the next state ID
func (b *OutcomeBuilder) SetNextState(nextState string) *OutcomeBuilder {
	b.nextState = nextState
	return b
}

// Build builds the OutcomeModel
// An empty nextState indicates a terminal state (end of conversation)
func (b *OutcomeBuilder) Build() (OutcomeModel, error) {
	return OutcomeModel{
		conditions: b.conditions,
		nextState:  b.nextState,
	}, nil
}

// CraftActionBuilder is a builder for CraftActionModel
type CraftActionBuilder struct {
	itemId                string
	materials             []uint32
	quantities            []uint32
	mesoCost              uint32
	stimulatorId          uint32
	stimulatorFailChance  float64
	successState          string
	failureState          string
	missingMaterialsState string
}

// NewCraftActionBuilder creates a new CraftActionBuilder
func NewCraftActionBuilder() *CraftActionBuilder {
	return &CraftActionBuilder{
		materials:  make([]uint32, 0),
		quantities: make([]uint32, 0),
	}
}

// SetItemId sets the item ID
func (b *CraftActionBuilder) SetItemId(itemId string) *CraftActionBuilder {
	b.itemId = itemId
	return b
}

// SetMaterials sets the material item IDs
func (b *CraftActionBuilder) SetMaterials(materials []uint32) *CraftActionBuilder {
	b.materials = materials
	return b
}

// AddMaterial adds a material item ID
func (b *CraftActionBuilder) AddMaterial(material uint32) *CraftActionBuilder {
	b.materials = append(b.materials, material)
	return b
}

// SetQuantities sets the material quantities
func (b *CraftActionBuilder) SetQuantities(quantities []uint32) *CraftActionBuilder {
	b.quantities = quantities
	return b
}

// AddQuantity adds a material quantity
func (b *CraftActionBuilder) AddQuantity(quantity uint32) *CraftActionBuilder {
	b.quantities = append(b.quantities, quantity)
	return b
}

// SetMesoCost sets the meso cost
func (b *CraftActionBuilder) SetMesoCost(mesoCost uint32) *CraftActionBuilder {
	b.mesoCost = mesoCost
	return b
}

// SetStimulatorId sets the stimulator item ID
func (b *CraftActionBuilder) SetStimulatorId(stimulatorId uint32) *CraftActionBuilder {
	b.stimulatorId = stimulatorId
	return b
}

// SetStimulatorFailChance sets the stimulator failure chance
func (b *CraftActionBuilder) SetStimulatorFailChance(stimulatorFailChance float64) *CraftActionBuilder {
	b.stimulatorFailChance = stimulatorFailChance
	return b
}

// SetSuccessState sets the success state ID
func (b *CraftActionBuilder) SetSuccessState(successState string) *CraftActionBuilder {
	b.successState = successState
	return b
}

// SetFailureState sets the failure state ID
func (b *CraftActionBuilder) SetFailureState(failureState string) *CraftActionBuilder {
	b.failureState = failureState
	return b
}

// SetMissingMaterialsState sets the missing materials state ID
func (b *CraftActionBuilder) SetMissingMaterialsState(missingMaterialsState string) *CraftActionBuilder {
	b.missingMaterialsState = missingMaterialsState
	return b
}

// Build builds the CraftActionModel
func (b *CraftActionBuilder) Build() (*CraftActionModel, error) {
	if b.itemId == "" {
		return nil, errors.New("itemId is required")
	}
	if len(b.materials) == 0 {
		return nil, errors.New("at least one material is required")
	}
	if len(b.quantities) != len(b.materials) {
		return nil, errors.New("quantities must match materials")
	}
	if b.successState == "" {
		return nil, errors.New("successState is required")
	}
	if b.failureState == "" {
		return nil, errors.New("failureState is required")
	}
	if b.missingMaterialsState == "" {
		return nil, errors.New("missingMaterialsState is required")
	}

	return &CraftActionModel{
		itemId:                b.itemId,
		materials:             b.materials,
		quantities:            b.quantities,
		mesoCost:              b.mesoCost,
		stimulatorId:          b.stimulatorId,
		stimulatorFailChance:  b.stimulatorFailChance,
		successState:          b.successState,
		failureState:          b.failureState,
		missingMaterialsState: b.missingMaterialsState,
	}, nil
}

// TransportActionBuilder is a builder for TransportActionModel
type TransportActionBuilder struct {
	routeName             string
	failureState          string
	capacityFullState     string
	alreadyInTransitState string
	routeNotFoundState    string
	serviceErrorState     string
}

// NewTransportActionBuilder creates a new TransportActionBuilder
func NewTransportActionBuilder() *TransportActionBuilder {
	return &TransportActionBuilder{}
}

// SetRouteName sets the transport route name
func (b *TransportActionBuilder) SetRouteName(routeName string) *TransportActionBuilder {
	b.routeName = routeName
	return b
}

// SetFailureState sets the general failure state ID
func (b *TransportActionBuilder) SetFailureState(failureState string) *TransportActionBuilder {
	b.failureState = failureState
	return b
}

// SetCapacityFullState sets the capacity full failure state ID
func (b *TransportActionBuilder) SetCapacityFullState(capacityFullState string) *TransportActionBuilder {
	b.capacityFullState = capacityFullState
	return b
}

// SetAlreadyInTransitState sets the already in transit failure state ID
func (b *TransportActionBuilder) SetAlreadyInTransitState(alreadyInTransitState string) *TransportActionBuilder {
	b.alreadyInTransitState = alreadyInTransitState
	return b
}

// SetRouteNotFoundState sets the route not found failure state ID
func (b *TransportActionBuilder) SetRouteNotFoundState(routeNotFoundState string) *TransportActionBuilder {
	b.routeNotFoundState = routeNotFoundState
	return b
}

// SetServiceErrorState sets the service error failure state ID
func (b *TransportActionBuilder) SetServiceErrorState(serviceErrorState string) *TransportActionBuilder {
	b.serviceErrorState = serviceErrorState
	return b
}

// Build builds the TransportActionModel
func (b *TransportActionBuilder) Build() (*TransportActionModel, error) {
	if b.routeName == "" {
		return nil, errors.New("routeName is required")
	}
	if b.failureState == "" {
		return nil, errors.New("failureState is required")
	}

	return &TransportActionModel{
		routeName:             b.routeName,
		failureState:          b.failureState,
		capacityFullState:     b.capacityFullState,
		alreadyInTransitState: b.alreadyInTransitState,
		routeNotFoundState:    b.routeNotFoundState,
		serviceErrorState:     b.serviceErrorState,
	}, nil
}

// GachaponActionBuilder builds GachaponActionModel
type GachaponActionBuilder struct {
	gachaponId   string
	ticketItemId uint32
	failureState string
}

func NewGachaponActionBuilder() *GachaponActionBuilder {
	return &GachaponActionBuilder{}
}

func (b *GachaponActionBuilder) SetGachaponId(gachaponId string) *GachaponActionBuilder {
	b.gachaponId = gachaponId
	return b
}

func (b *GachaponActionBuilder) SetTicketItemId(ticketItemId uint32) *GachaponActionBuilder {
	b.ticketItemId = ticketItemId
	return b
}

func (b *GachaponActionBuilder) SetFailureState(failureState string) *GachaponActionBuilder {
	b.failureState = failureState
	return b
}

func (b *GachaponActionBuilder) Build() (*GachaponActionModel, error) {
	if b.gachaponId == "" {
		return nil, errors.New("gachaponId is required")
	}
	if b.ticketItemId == 0 {
		return nil, errors.New("ticketItemId is required")
	}
	if b.failureState == "" {
		return nil, errors.New("failureState is required")
	}

	return &GachaponActionModel{
		gachaponId:   b.gachaponId,
		ticketItemId: b.ticketItemId,
		failureState: b.failureState,
	}, nil
}

// RPSActionBuilder builds RPSActionModel
type RPSActionBuilder struct {
	npcId         uint32
	entryCostMeso uint32
	failureState  string
}

func NewRPSActionBuilder() *RPSActionBuilder {
	return &RPSActionBuilder{}
}

func (b *RPSActionBuilder) SetNpcId(npcId uint32) *RPSActionBuilder {
	b.npcId = npcId
	return b
}

func (b *RPSActionBuilder) SetEntryCostMeso(entryCostMeso uint32) *RPSActionBuilder {
	b.entryCostMeso = entryCostMeso
	return b
}

func (b *RPSActionBuilder) SetFailureState(failureState string) *RPSActionBuilder {
	b.failureState = failureState
	return b
}

func (b *RPSActionBuilder) Build() (*RPSActionModel, error) {
	if b.npcId == 0 {
		return nil, errors.New("npcId is required")
	}
	if b.entryCostMeso == 0 {
		return nil, errors.New("entryCostMeso is required")
	}
	if b.failureState == "" {
		return nil, errors.New("failureState is required")
	}

	return &RPSActionModel{
		npcId:         b.npcId,
		entryCostMeso: b.entryCostMeso,
		failureState:  b.failureState,
	}, nil
}

// PartyQuestActionBuilder builds PartyQuestActionModel
type PartyQuestActionBuilder struct {
	questId         string
	failureState    string
	notInPartyState string
	notLeaderState  string
}

func NewPartyQuestActionBuilder() *PartyQuestActionBuilder {
	return &PartyQuestActionBuilder{}
}

func (b *PartyQuestActionBuilder) SetQuestId(questId string) *PartyQuestActionBuilder {
	b.questId = questId
	return b
}

func (b *PartyQuestActionBuilder) SetFailureState(failureState string) *PartyQuestActionBuilder {
	b.failureState = failureState
	return b
}

func (b *PartyQuestActionBuilder) SetNotInPartyState(notInPartyState string) *PartyQuestActionBuilder {
	b.notInPartyState = notInPartyState
	return b
}

func (b *PartyQuestActionBuilder) SetNotLeaderState(notLeaderState string) *PartyQuestActionBuilder {
	b.notLeaderState = notLeaderState
	return b
}

func (b *PartyQuestActionBuilder) Build() (*PartyQuestActionModel, error) {
	if b.questId == "" {
		return nil, errors.New("questId is required")
	}
	if b.failureState == "" {
		return nil, errors.New("failureState is required")
	}

	return &PartyQuestActionModel{
		questId:         b.questId,
		failureState:    b.failureState,
		notInPartyState: b.notInPartyState,
		notLeaderState:  b.notLeaderState,
	}, nil
}

// PartyQuestBonusActionBuilder builds PartyQuestBonusActionModel
type PartyQuestBonusActionBuilder struct {
	failureState string
}

func NewPartyQuestBonusActionBuilder() *PartyQuestBonusActionBuilder {
	return &PartyQuestBonusActionBuilder{}
}

func (b *PartyQuestBonusActionBuilder) SetFailureState(failureState string) *PartyQuestBonusActionBuilder {
	b.failureState = failureState
	return b
}

func (b *PartyQuestBonusActionBuilder) Build() (*PartyQuestBonusActionModel, error) {
	if b.failureState == "" {
		return nil, errors.New("failureState is required")
	}

	return &PartyQuestBonusActionModel{
		failureState: b.failureState,
	}, nil
}

// ListSelectionBuilder is a builder for ListSelectionModel
type ListSelectionBuilder struct {
	title   string
	choices []ChoiceModel
}

// NewListSelectionBuilder creates a new ListSelectionBuilder
func NewListSelectionBuilder() *ListSelectionBuilder {
	return &ListSelectionBuilder{choices: make([]ChoiceModel, 0)}
}

// SetTitle sets the list selection title
func (b *ListSelectionBuilder) SetTitle(title string) *ListSelectionBuilder {
	b.title = title
	return b
}

func (b *ListSelectionBuilder) AddChoice(choice ChoiceModel) *ListSelectionBuilder {
	b.choices = append(b.choices, choice)
	return b
}

// Build builds the ListSelectionModel
func (b *ListSelectionBuilder) Build() (*ListSelectionModel, error) {
	if b.title == "" {
		return nil, errors.New("title is required")
	}

	return &ListSelectionModel{
		title:   b.title,
		choices: b.choices,
	}, nil
}

// AskNumberBuilder is a builder for AskNumberModel
type AskNumberBuilder struct {
	text         string
	defaultValue uint32
	minValue     uint32
	maxValue     uint32
	contextKey   string
	nextState    string
}

// NewAskNumberBuilder creates a new AskNumberBuilder
func NewAskNumberBuilder() *AskNumberBuilder {
	return &AskNumberBuilder{
		contextKey: "quantity", // Default context key
	}
}

// SetText sets the ask number text
func (b *AskNumberBuilder) SetText(text string) *AskNumberBuilder {
	b.text = text
	return b
}

// SetDefaultValue sets the default value
func (b *AskNumberBuilder) SetDefaultValue(defaultValue uint32) *AskNumberBuilder {
	b.defaultValue = defaultValue
	return b
}

// SetMinValue sets the minimum value
func (b *AskNumberBuilder) SetMinValue(minValue uint32) *AskNumberBuilder {
	b.minValue = minValue
	return b
}

// SetMaxValue sets the maximum value
func (b *AskNumberBuilder) SetMaxValue(maxValue uint32) *AskNumberBuilder {
	b.maxValue = maxValue
	return b
}

// SetContextKey sets the context key
func (b *AskNumberBuilder) SetContextKey(contextKey string) *AskNumberBuilder {
	b.contextKey = contextKey
	return b
}

// SetNextState sets the next state ID
func (b *AskNumberBuilder) SetNextState(nextState string) *AskNumberBuilder {
	b.nextState = nextState
	return b
}

// Build builds the AskNumberModel
func (b *AskNumberBuilder) Build() (*AskNumberModel, error) {
	if b.text == "" {
		return nil, errors.New("text is required")
	}
	if b.minValue > b.defaultValue {
		return nil, errors.New("minValue must be less than or equal to defaultValue")
	}
	if b.defaultValue > b.maxValue {
		return nil, errors.New("defaultValue must be less than or equal to maxValue")
	}
	if b.maxValue == 0 {
		return nil, errors.New("maxValue must be greater than 0")
	}
	if b.contextKey == "" {
		b.contextKey = "quantity" // Ensure default
	}

	return &AskNumberModel{
		text:         b.text,
		defaultValue: b.defaultValue,
		minValue:     b.minValue,
		maxValue:     b.maxValue,
		contextKey:   b.contextKey,
		nextState:    b.nextState,
	}, nil
}

// AskTextBuilder is a builder for AskTextModel
type AskTextBuilder struct {
	text        string
	defaultText string
	minLength   uint16
	maxLength   uint16
	contextKey  string
	matches     []AskTextMatchModel
	nextState   string
}

// NewAskTextBuilder creates a new AskTextBuilder
func NewAskTextBuilder() *AskTextBuilder {
	return &AskTextBuilder{
		contextKey: "answer", // Default context key
		matches:    make([]AskTextMatchModel, 0),
	}
}

// SetText sets the ask text prompt
func (b *AskTextBuilder) SetText(text string) *AskTextBuilder {
	b.text = text
	return b
}

// SetDefaultText sets the default text
func (b *AskTextBuilder) SetDefaultText(defaultText string) *AskTextBuilder {
	b.defaultText = defaultText
	return b
}

// SetMinLength sets the minimum accepted text length
func (b *AskTextBuilder) SetMinLength(minLength uint16) *AskTextBuilder {
	b.minLength = minLength
	return b
}

// SetMaxLength sets the maximum accepted text length
func (b *AskTextBuilder) SetMaxLength(maxLength uint16) *AskTextBuilder {
	b.maxLength = maxLength
	return b
}

// SetContextKey sets the context key
func (b *AskTextBuilder) SetContextKey(contextKey string) *AskTextBuilder {
	b.contextKey = contextKey
	return b
}

// AddMatch adds a match to the ordered, first-match-wins branch table
func (b *AskTextBuilder) AddMatch(match AskTextMatchModel) *AskTextBuilder {
	b.matches = append(b.matches, match)
	return b
}

// SetNextState sets the next state ID
func (b *AskTextBuilder) SetNextState(nextState string) *AskTextBuilder {
	b.nextState = nextState
	return b
}

// Build builds the AskTextModel
func (b *AskTextBuilder) Build() (*AskTextModel, error) {
	if b.text == "" {
		return nil, errors.New("text is required")
	}
	if b.maxLength == 0 {
		return nil, errors.New("maxLength must be greater than 0")
	}
	if b.minLength > b.maxLength {
		return nil, errors.New("minLength must be less than or equal to maxLength")
	}
	if b.contextKey == "" {
		return nil, errors.New("contextKey is required")
	}
	if b.nextState == "" {
		return nil, errors.New("nextState is required")
	}

	return &AskTextModel{
		text:        b.text,
		defaultText: b.defaultText,
		minLength:   b.minLength,
		maxLength:   b.maxLength,
		contextKey:  b.contextKey,
		matches:     b.matches,
		nextState:   b.nextState,
	}, nil
}

// AskTextMatchBuilder is a builder for AskTextMatchModel
type AskTextMatchBuilder struct {
	value            string
	valueFromContext string
	nextState        string
}

// NewAskTextMatchBuilder creates a new AskTextMatchBuilder
func NewAskTextMatchBuilder() *AskTextMatchBuilder {
	return &AskTextMatchBuilder{}
}

// SetValue sets the literal value to match against
func (b *AskTextMatchBuilder) SetValue(value string) *AskTextMatchBuilder {
	b.value = value
	return b
}

// SetValueFromContext sets the context key whose value is matched against
func (b *AskTextMatchBuilder) SetValueFromContext(valueFromContext string) *AskTextMatchBuilder {
	b.valueFromContext = valueFromContext
	return b
}

// SetNextState sets the next state ID
func (b *AskTextMatchBuilder) SetNextState(nextState string) *AskTextMatchBuilder {
	b.nextState = nextState
	return b
}

// Build builds the AskTextMatchModel
func (b *AskTextMatchBuilder) Build() (*AskTextMatchModel, error) {
	if b.nextState == "" {
		return nil, errors.New("nextState is required")
	}
	if b.value == "" && b.valueFromContext == "" {
		return nil, errors.New("exactly one of value or valueFromContext is required")
	}
	if b.value != "" && b.valueFromContext != "" {
		return nil, errors.New("exactly one of value or valueFromContext is required")
	}

	return &AskTextMatchModel{
		value:            b.value,
		valueFromContext: b.valueFromContext,
		nextState:        b.nextState,
	}, nil
}

// AskStyleBuilder is a builder for AskStyleModel
type AskStyleBuilder struct {
	text             string
	styles           []uint32
	stylesContextKey string
	contextKey       string
	nextState        string
}

// NewAskStyleBuilder creates a new AskStyleBuilder
func NewAskStyleBuilder() *AskStyleBuilder {
	return &AskStyleBuilder{
		styles:     make([]uint32, 0),
		contextKey: "selectedStyle", // Default context key
	}
}

// SetText sets the ask style text
func (b *AskStyleBuilder) SetText(text string) *AskStyleBuilder {
	b.text = text
	return b
}

// SetStyles sets the available style IDs
func (b *AskStyleBuilder) SetStyles(styles []uint32) *AskStyleBuilder {
	b.styles = styles
	return b
}

// AddStyle adds a style ID
func (b *AskStyleBuilder) AddStyle(styleId uint32) *AskStyleBuilder {
	b.styles = append(b.styles, styleId)
	return b
}

// SetStylesContextKey sets the context key containing dynamically generated styles
func (b *AskStyleBuilder) SetStylesContextKey(key string) *AskStyleBuilder {
	b.stylesContextKey = key
	return b
}

// SetContextKey sets the context key
func (b *AskStyleBuilder) SetContextKey(contextKey string) *AskStyleBuilder {
	b.contextKey = contextKey
	return b
}

// SetNextState sets the next state ID
func (b *AskStyleBuilder) SetNextState(nextState string) *AskStyleBuilder {
	b.nextState = nextState
	return b
}

// Build builds the AskStyleModel
func (b *AskStyleBuilder) Build() (*AskStyleModel, error) {
	if b.text == "" {
		return nil, errors.New("text is required")
	}

	// Require either styles OR stylesContextKey (not both, not neither)
	hasStyles := len(b.styles) > 0
	hasStylesContextKey := b.stylesContextKey != ""

	if !hasStyles && !hasStylesContextKey {
		return nil, errors.New("either styles or stylesContextKey is required")
	}

	if b.nextState == "" {
		return nil, errors.New("nextState is required")
	}
	if b.contextKey == "" {
		b.contextKey = "selectedStyle" // Ensure default
	}

	return &AskStyleModel{
		text:             b.text,
		styles:           b.styles,
		stylesContextKey: b.stylesContextKey,
		contextKey:       b.contextKey,
		nextState:        b.nextState,
	}, nil
}

// AskSlideMenuBuilder is a builder for AskSlideMenuModel
type AskSlideMenuBuilder struct {
	title      string
	menuType   uint32
	contextKey string
	choices    []ChoiceModel
}

// NewAskSlideMenuBuilder creates a new AskSlideMenuBuilder
func NewAskSlideMenuBuilder() *AskSlideMenuBuilder {
	return &AskSlideMenuBuilder{
		choices:    make([]ChoiceModel, 0),
		contextKey: "selectedOption", // Default context key
	}
}

// SetTitle sets the slide menu title
func (b *AskSlideMenuBuilder) SetTitle(title string) *AskSlideMenuBuilder {
	b.title = title
	return b
}

// SetMenuType sets the menu type
func (b *AskSlideMenuBuilder) SetMenuType(menuType uint32) *AskSlideMenuBuilder {
	b.menuType = menuType
	return b
}

// SetContextKey sets the context key for storing the selection
func (b *AskSlideMenuBuilder) SetContextKey(contextKey string) *AskSlideMenuBuilder {
	b.contextKey = contextKey
	return b
}

// AddChoice adds a choice to the slide menu
func (b *AskSlideMenuBuilder) AddChoice(choice ChoiceModel) *AskSlideMenuBuilder {
	b.choices = append(b.choices, choice)
	return b
}

// Build builds the AskSlideMenuModel
func (b *AskSlideMenuBuilder) Build() (*AskSlideMenuModel, error) {
	if len(b.choices) == 0 {
		return nil, errors.New("at least one choice is required")
	}
	if b.contextKey == "" {
		b.contextKey = "selectedOption" // Ensure default
	}

	return &AskSlideMenuModel{
		title:      b.title,
		menuType:   b.menuType,
		contextKey: b.contextKey,
		choices:    b.choices,
	}, nil
}

// OptionSetBuilder is a builder for OptionSetModel
type OptionSetBuilder struct {
	id      string
	options []OptionModel
}

// NewOptionSetBuilder creates a new OptionSetBuilder
func NewOptionSetBuilder() *OptionSetBuilder {
	return &OptionSetBuilder{
		options: make([]OptionModel, 0),
	}
}

// SetId sets the option set ID
func (b *OptionSetBuilder) SetId(id string) *OptionSetBuilder {
	b.id = id
	return b
}

// SetOptions sets the options
func (b *OptionSetBuilder) SetOptions(options []OptionModel) *OptionSetBuilder {
	b.options = options
	return b
}

// AddOption adds an option
func (b *OptionSetBuilder) AddOption(option OptionModel) *OptionSetBuilder {
	b.options = append(b.options, option)
	return b
}

// Build builds the OptionSetModel
func (b *OptionSetBuilder) Build() (OptionSetModel, error) {
	if b.id == "" {
		return OptionSetModel{}, errors.New("id is required")
	}
	if len(b.options) == 0 {
		return OptionSetModel{}, errors.New("at least one option is required")
	}

	return OptionSetModel{
		id:      b.id,
		options: b.options,
	}, nil
}

// OptionBuilder is a builder for OptionModel
type OptionBuilder struct {
	id         uint32
	name       string
	materials  []uint32
	quantities []uint32
	meso       uint32
}

// NewOptionBuilder creates a new OptionBuilder
func NewOptionBuilder() *OptionBuilder {
	return &OptionBuilder{
		materials:  make([]uint32, 0),
		quantities: make([]uint32, 0),
	}
}

// SetId sets the option ID
func (b *OptionBuilder) SetId(id uint32) *OptionBuilder {
	b.id = id
	return b
}

// SetName sets the option name
func (b *OptionBuilder) SetName(name string) *OptionBuilder {
	b.name = name
	return b
}

// SetMaterials sets the material item IDs
func (b *OptionBuilder) SetMaterials(materials []uint32) *OptionBuilder {
	b.materials = materials
	return b
}

// AddMaterial adds a material item ID
func (b *OptionBuilder) AddMaterial(material uint32) *OptionBuilder {
	b.materials = append(b.materials, material)
	return b
}

// SetQuantities sets the material quantities
func (b *OptionBuilder) SetQuantities(quantities []uint32) *OptionBuilder {
	b.quantities = quantities
	return b
}

// AddQuantity adds a material quantity
func (b *OptionBuilder) AddQuantity(quantity uint32) *OptionBuilder {
	b.quantities = append(b.quantities, quantity)
	return b
}

// SetMeso sets the meso cost
func (b *OptionBuilder) SetMeso(meso uint32) *OptionBuilder {
	b.meso = meso
	return b
}

// Build builds the OptionModel
func (b *OptionBuilder) Build() (OptionModel, error) {
	if b.id == 0 {
		return OptionModel{}, errors.New("id is required")
	}
	if b.name == "" {
		return OptionModel{}, errors.New("name is required")
	}
	if len(b.materials) > 0 && len(b.quantities) != len(b.materials) {
		return OptionModel{}, errors.New("quantities must match materials")
	}

	return OptionModel{
		id:         b.id,
		name:       b.name,
		materials:  b.materials,
		quantities: b.quantities,
		meso:       b.meso,
	}, nil
}

// ConversationContextBuilder is a builder for ConversationContext
type ConversationContextBuilder struct {
	field               field.Model
	characterId         uint32
	npcId               uint32
	currentState        string
	conversation        StateContainer
	context             map[string]string
	pendingSagaId       *uuid.UUID
	conversationType    ConversationType
	sourceId            uint32
	originTransactionId *uuid.UUID
}

// NewConversationContextBuilder creates a new ConversationContextBuilder
func NewConversationContextBuilder() *ConversationContextBuilder {
	return &ConversationContextBuilder{
		context: make(map[string]string),
	}
}

// SetField sets the field
func (b *ConversationContextBuilder) SetField(field field.Model) *ConversationContextBuilder {
	b.field = field
	return b
}

// SetCharacterId sets the character ID
func (b *ConversationContextBuilder) SetCharacterId(characterId uint32) *ConversationContextBuilder {
	b.characterId = characterId
	return b
}

// SetNpcId sets the NPC ID
func (b *ConversationContextBuilder) SetNpcId(npcId uint32) *ConversationContextBuilder {
	b.npcId = npcId
	return b
}

// SetCurrentState sets the current state ID
func (b *ConversationContextBuilder) SetCurrentState(currentState string) *ConversationContextBuilder {
	b.currentState = currentState
	return b
}

// SetConversation sets the conversation state container (Model for NPC, StateMachine for Quest)
func (b *ConversationContextBuilder) SetConversation(conversation StateContainer) *ConversationContextBuilder {
	b.conversation = conversation
	return b
}

// SetContext sets the entire context map
func (b *ConversationContextBuilder) SetContext(context map[string]string) *ConversationContextBuilder {
	b.context = context
	return b
}

// AddContextValue adds a key-value pair to the context map
func (b *ConversationContextBuilder) AddContextValue(key, value string) *ConversationContextBuilder {
	b.context[key] = value
	return b
}

// SetPendingSagaId sets the pending saga ID
func (b *ConversationContextBuilder) SetPendingSagaId(sagaId uuid.UUID) *ConversationContextBuilder {
	b.pendingSagaId = &sagaId
	return b
}

// SetConversationType sets the conversation type
func (b *ConversationContextBuilder) SetConversationType(conversationType ConversationType) *ConversationContextBuilder {
	b.conversationType = conversationType
	return b
}

// SetSourceId sets the source ID (NpcId or QuestId)
func (b *ConversationContextBuilder) SetSourceId(sourceId uint32) *ConversationContextBuilder {
	b.sourceId = sourceId
	return b
}

// SetOriginTransactionId records the saga transaction that started this
// conversation.
func (b *ConversationContextBuilder) SetOriginTransactionId(id uuid.UUID) *ConversationContextBuilder {
	b.originTransactionId = &id
	return b
}

// Build builds the ConversationContext
func (b *ConversationContextBuilder) Build() ConversationContext {
	// Default to NPC conversation type if not set
	conversationType := b.conversationType
	if conversationType == "" {
		conversationType = NpcConversationType
	}

	// Default sourceId to npcId for backwards compatibility
	sourceId := b.sourceId
	if sourceId == 0 && b.npcId != 0 {
		sourceId = b.npcId
	}

	return ConversationContext{
		characterId:         b.characterId,
		npcId:               b.npcId,
		field:               b.field,
		currentState:        b.currentState,
		conversation:        b.conversation,
		context:             b.context,
		pendingSagaId:       b.pendingSagaId,
		conversationType:    conversationType,
		sourceId:            sourceId,
		originTransactionId: b.originTransactionId,
	}
}
