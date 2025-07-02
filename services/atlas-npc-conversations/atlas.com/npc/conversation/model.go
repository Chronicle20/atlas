package conversation

import (
	"errors"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/google/uuid"
	"time"
)

// Model represents a conversation tree for an NPC
type Model struct {
	id         uuid.UUID
	npcId      uint32
	startState string
	states     []StateModel
	createdAt  time.Time
	updatedAt  time.Time
}

// GetId returns the conversation ID
func (m Model) Id() uuid.UUID {
	return m.id
}

// GetNpcId returns the NPC ID
func (m Model) NpcId() uint32 {
	return m.npcId
}

// GetStartState returns the starting state ID
func (m Model) StartState() string {
	return m.startState
}

// GetStates returns the conversation states
func (m Model) States() []StateModel {
	return m.states
}

// GetCreatedAt returns the creation timestamp
func (m Model) CreatedAt() time.Time {
	return m.createdAt
}

// GetUpdatedAt returns the last update timestamp
func (m Model) UpdatedAt() time.Time {
	return m.updatedAt
}

// FindState finds a state by ID
func (m Model) FindState(stateId string) (StateModel, error) {
	for _, state := range m.states {
		if state.Id() == stateId {
			return state, nil
		}
	}
	return StateModel{}, errors.New("state not found")
}

// Builder is a builder for Model
type Builder struct {
	id         uuid.UUID
	npcId      uint32
	startState string
	states     []StateModel
	createdAt  time.Time
	updatedAt  time.Time
}

// NewBuilder creates a new Builder
func NewBuilder() *Builder {
	return &Builder{
		id:        uuid.Nil,
		states:    make([]StateModel, 0),
		createdAt: time.Now(),
		updatedAt: time.Now(),
	}
}

// SetId sets the conversation ID
func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// SetNpcId sets the NPC ID
func (b *Builder) SetNpcId(npcId uint32) *Builder {
	b.npcId = npcId
	return b
}

// SetStartState sets the starting state ID
func (b *Builder) SetStartState(startState string) *Builder {
	b.startState = startState
	return b
}

// SetStates sets the conversation states
func (b *Builder) SetStates(states []StateModel) *Builder {
	b.states = states
	return b
}

// AddState adds a conversation state
func (b *Builder) AddState(state StateModel) *Builder {
	b.states = append(b.states, state)
	return b
}

// SetCreatedAt sets the creation timestamp
func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

// SetUpdatedAt sets the last update timestamp
func (b *Builder) SetUpdatedAt(updatedAt time.Time) *Builder {
	b.updatedAt = updatedAt
	return b
}

// Build builds the Model
func (b *Builder) Build() (Model, error) {
	if b.npcId == 0 {
		return Model{}, errors.New("npcId is required")
	}
	if b.startState == "" {
		return Model{}, errors.New("startState is required")
	}
	if len(b.states) == 0 {
		return Model{}, errors.New("at least one state is required")
	}

	return Model{
		id:         b.id,
		npcId:      b.npcId,
		startState: b.startState,
		states:     b.states,
		createdAt:  b.createdAt,
		updatedAt:  b.updatedAt,
	}, nil
}

// StateType represents the type of a conversation state
type StateType string

const (
	DialogueStateType StateType = "dialogue"
	GenericActionType StateType = "genericAction"
	CraftActionType   StateType = "craftAction"
	ListSelectionType StateType = "listSelection"
)

// StateModel represents a state in a conversation
type StateModel struct {
	id            string
	stateType     StateType
	dialogue      *DialogueModel
	genericAction *GenericActionModel
	craftAction   *CraftActionModel
	listSelection *ListSelectionModel
}

// Id returns the state ID
func (s StateModel) Id() string {
	return s.id
}

// Type returns the state type
func (s StateModel) Type() StateType {
	return s.stateType
}

// Dialogue returns the dialogue model (if type is dialogue)
func (s StateModel) Dialogue() *DialogueModel {
	return s.dialogue
}

// GenericAction returns the generic action model (if type is genericAction)
func (s StateModel) GenericAction() *GenericActionModel {
	return s.genericAction
}

// CraftAction returns the craft action model (if type is craftAction)
func (s StateModel) CraftAction() *CraftActionModel {
	return s.craftAction
}

// ListSelection returns the list selection model (if type is listSelection)
func (s StateModel) ListSelection() *ListSelectionModel {
	return s.listSelection
}

// StateBuilder is a builder for StateModel
type StateBuilder struct {
	id            string
	stateType     StateType
	dialogue      *DialogueModel
	genericAction *GenericActionModel
	craftAction   *CraftActionModel
	listSelection *ListSelectionModel
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
	b.listSelection = nil
	return b
}

// SetGenericAction sets the generic action model
func (b *StateBuilder) SetGenericAction(genericAction *GenericActionModel) *StateBuilder {
	b.stateType = GenericActionType
	b.dialogue = nil
	b.genericAction = genericAction
	b.craftAction = nil
	b.listSelection = nil
	return b
}

// SetCraftAction sets the craft action model
func (b *StateBuilder) SetCraftAction(craftAction *CraftActionModel) *StateBuilder {
	b.stateType = CraftActionType
	b.dialogue = nil
	b.genericAction = nil
	b.craftAction = craftAction
	b.listSelection = nil
	return b
}

// SetListSelection sets the list selection model
func (b *StateBuilder) SetListSelection(listSelection *ListSelectionModel) *StateBuilder {
	b.stateType = ListSelectionType
	b.dialogue = nil
	b.genericAction = nil
	b.craftAction = nil
	b.listSelection = listSelection
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
	case ListSelectionType:
		if b.listSelection == nil {
			return StateModel{}, errors.New("listSelection is required for listSelection state")
		}
	default:
		return StateModel{}, errors.New("invalid state type")
	}

	return StateModel{
		id:            b.id,
		stateType:     b.stateType,
		dialogue:      b.dialogue,
		genericAction: b.genericAction,
		craftAction:   b.craftAction,
		listSelection: b.listSelection,
	}, nil
}

// DialogueType represents the type of dialogue
type DialogueType string

const (
	SendOk     DialogueType = "sendOk"
	SendYesNo  DialogueType = "sendYesNo"
	SendSimple DialogueType = "sendSimple"
	SendNext   DialogueType = "sendNext"
)

// DialogueModel represents a dialogue state
type DialogueModel struct {
	dialogueType DialogueType
	text         string
	choices      []ChoiceModel
}

// DialogueType returns the dialogue type
func (d DialogueModel) DialogueType() DialogueType {
	return d.dialogueType
}

// Text returns the dialogue text
func (d DialogueModel) Text() string {
	return d.text
}

// Choices returns the dialogue choices
func (d DialogueModel) Choices() []ChoiceModel {
	return d.choices
}

func (d DialogueModel) ChoiceFromAction(action byte) (ChoiceModel, bool) {
	choiceText := ""
	if d.dialogueType == SendNext {
		if action == 255 {
			choiceText = "Exit"
		} else {
			choiceText = "Next"
		}
	} else if d.dialogueType == SendYesNo {
		if action == 255 {
			choiceText = "Exit"
		} else if action == 0 {
			choiceText = "No"
		} else {
			choiceText = "Yes"
		}
	}

	for _, choice := range d.choices {
		if choice.Text() == choiceText {
			return choice, true
		}
	}
	return ChoiceModel{}, false
}

// DialogueBuilder is a builder for DialogueModel
type DialogueBuilder struct {
	dialogueType DialogueType
	text         string
	choices      []ChoiceModel
}

// NewDialogueBuilder creates a new DialogueBuilder
func NewDialogueBuilder() *DialogueBuilder {
	return &DialogueBuilder{
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
		if len(b.choices) != 1 {
			return nil, errors.New("sendOk requires exactly 1 choices")
		}
	case SendNext:
		if len(b.choices) != 2 {
			return nil, errors.New("sendNext requires exactly 2 choices")
		}
	case SendYesNo:
		if len(b.choices) != 3 {
			return nil, errors.New("sendYesNo requires exactly 3 choices")
		}
	case SendSimple:
		if len(b.choices) == 0 {
			return nil, errors.New("sendSimple requires at least 1 choice")
		}
	}

	return &DialogueModel{
		dialogueType: b.dialogueType,
		text:         b.text,
		choices:      b.choices,
	}, nil
}

// ChoiceModel represents a choice in a dialogue
type ChoiceModel struct {
	text      string
	nextState string
	context   map[string]string
}

// Text returns the choice text
func (c ChoiceModel) Text() string {
	return c.text
}

// NextState returns the next state ID
func (c ChoiceModel) NextState() string {
	return c.nextState
}

// Context returns the context map
func (c ChoiceModel) Context() map[string]string {
	return c.context
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

// GenericActionModel represents a generic action state
type GenericActionModel struct {
	operations []OperationModel
	outcomes   []OutcomeModel
}

// Operations returns the operations
func (g GenericActionModel) Operations() []OperationModel {
	return g.operations
}

// Outcomes returns the outcomes
func (g GenericActionModel) Outcomes() []OutcomeModel {
	return g.outcomes
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

// OperationModel represents an operation in a generic action
type OperationModel struct {
	operationType string
	params        map[string]string
}

// Type returns the operation type
func (o OperationModel) Type() string {
	return o.operationType
}

// Params returns the operation parameters
func (o OperationModel) Params() map[string]string {
	return o.params
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

// ConditionModel represents a condition in the conversation domain
type ConditionModel struct {
	conditionType string
	operator      string
	value         string
}

// Type returns the condition type
func (c ConditionModel) Type() string {
	return c.conditionType
}

// Operator returns the operator
func (c ConditionModel) Operator() string {
	return c.operator
}

// Value returns the value
func (c ConditionModel) Value() string {
	return c.value
}

// ConditionBuilder is a builder for ConditionModel
type ConditionBuilder struct {
	conditionType string
	operator      string
	value         string
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
		conditionType: b.conditionType,
		operator:      b.operator,
		value:         b.value,
	}, nil
}

// OutcomeModel represents an outcome in a generic action
type OutcomeModel struct {
	conditions   []ConditionModel
	nextState    string
	successState string
	failureState string
}

// Conditions returns the outcome condition
func (o OutcomeModel) Conditions() []ConditionModel {
	return o.conditions
}

// NextState returns the next state ID
func (o OutcomeModel) NextState() string {
	return o.nextState
}

// SuccessState returns the success state ID
func (o OutcomeModel) SuccessState() string {
	return o.successState
}

// FailureState returns the failure state ID
func (o OutcomeModel) FailureState() string {
	return o.failureState
}

// OutcomeBuilder is a builder for OutcomeModel
type OutcomeBuilder struct {
	conditions   []ConditionModel
	nextState    string
	successState string
	failureState string
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

// SetSuccessState sets the success state ID
func (b *OutcomeBuilder) SetSuccessState(successState string) *OutcomeBuilder {
	b.successState = successState
	return b
}

// SetFailureState sets the failure state ID
func (b *OutcomeBuilder) SetFailureState(failureState string) *OutcomeBuilder {
	b.failureState = failureState
	return b
}

// Build builds the OutcomeModel
func (b *OutcomeBuilder) Build() (OutcomeModel, error) {
	if b.nextState == "" && b.successState == "" && b.failureState == "" {
		return OutcomeModel{}, errors.New("at least one of nextState, successState, or failureState is required")
	}

	return OutcomeModel{
		conditions:   b.conditions,
		nextState:    b.nextState,
		successState: b.successState,
		failureState: b.failureState,
	}, nil
}

// CraftActionModel represents a craft action state
type CraftActionModel struct {
	itemId                uint32
	materials             []uint32
	quantities            []uint32
	mesoCost              uint32
	stimulatorId          uint32
	stimulatorFailChance  float64
	successState          string
	failureState          string
	missingMaterialsState string
}

// ItemId returns the item ID
func (c CraftActionModel) ItemId() uint32 {
	return c.itemId
}

// Materials returns the material item IDs
func (c CraftActionModel) Materials() []uint32 {
	return c.materials
}

// Quantities returns the material quantities
func (c CraftActionModel) Quantities() []uint32 {
	return c.quantities
}

// MesoCost returns the meso cost
func (c CraftActionModel) MesoCost() uint32 {
	return c.mesoCost
}

// StimulatorId returns the stimulator item ID
func (c CraftActionModel) StimulatorId() uint32 {
	return c.stimulatorId
}

// StimulatorFailChance returns the stimulator failure chance
func (c CraftActionModel) StimulatorFailChance() float64 {
	return c.stimulatorFailChance
}

// SuccessState returns the success state ID
func (c CraftActionModel) SuccessState() string {
	return c.successState
}

// FailureState returns the failure state ID
func (c CraftActionModel) FailureState() string {
	return c.failureState
}

// MissingMaterialsState returns the missing materials state ID
func (c CraftActionModel) MissingMaterialsState() string {
	return c.missingMaterialsState
}

// CraftActionBuilder is a builder for CraftActionModel
type CraftActionBuilder struct {
	itemId                uint32
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
func (b *CraftActionBuilder) SetItemId(itemId uint32) *CraftActionBuilder {
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
	if b.itemId == 0 {
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

// ListSelectionModel represents a list selection state
type ListSelectionModel struct {
	title   string
	choices []ChoiceModel
}

// Title returns the list selection title
func (l ListSelectionModel) Title() string {
	return l.title
}

func (l ListSelectionModel) Choices() []ChoiceModel {
	return l.choices
}

func (l ListSelectionModel) ChoiceFromSelection(action byte, selection int32) (ChoiceModel, error) {
	if action == 0 {
		for _, choice := range l.choices {
			if choice.Text() == "Exit" {
				return choice, nil
			}
		}
		return ChoiceModel{}, errors.New("invalid selection")
	}

	if selection < 0 || selection >= int32(len(l.choices)) {
		return ChoiceModel{}, errors.New("invalid selection")
	}
	return l.choices[selection], nil
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

// OptionSetModel represents an option set
type OptionSetModel struct {
	id      string
	options []OptionModel
}

// Id returns the option set ID
func (o OptionSetModel) Id() string {
	return o.id
}

// Options returns the options
func (o OptionSetModel) Options() []OptionModel {
	return o.options
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

// OptionModel represents an option in an option set
type OptionModel struct {
	id         uint32
	name       string
	materials  []uint32
	quantities []uint32
	meso       uint32
}

// Id returns the option ID
func (o OptionModel) Id() uint32 {
	return o.id
}

// Name returns the option name
func (o OptionModel) Name() string {
	return o.name
}

// Materials returns the material item IDs
func (o OptionModel) Materials() []uint32 {
	return o.materials
}

// Quantities returns the material quantities
func (o OptionModel) Quantities() []uint32 {
	return o.quantities
}

// Meso returns the meso cost
func (o OptionModel) Meso() uint32 {
	return o.meso
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

// ConversationContext represents the current state of a conversation
type ConversationContext struct {
	field        field.Model
	characterId  uint32
	npcId        uint32
	currentState string
	conversation Model
	context      map[string]string
}

// Field returns the field
func (c ConversationContext) Field() field.Model {
	return c.field
}

// CharacterId returns the character ID
func (c ConversationContext) CharacterId() uint32 {
	return c.characterId
}

// NpcId returns the NPC ID
func (c ConversationContext) NpcId() uint32 {
	return c.npcId
}

// CurrentState returns the current state ID
func (c ConversationContext) CurrentState() string {
	return c.currentState
}

// Conversation returns the conversation model
func (c ConversationContext) Conversation() Model {
	return c.conversation
}

// Context returns the context map
func (c ConversationContext) Context() map[string]string {
	return c.context
}

// ConversationContextBuilder is a builder for ConversationContext
type ConversationContextBuilder struct {
	field        field.Model
	characterId  uint32
	npcId        uint32
	currentState string
	conversation Model
	context      map[string]string
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

// SetConversation sets the conversation model
func (b *ConversationContextBuilder) SetConversation(conversation Model) *ConversationContextBuilder {
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

// Build builds the ConversationContext
func (b *ConversationContextBuilder) Build() (ConversationContext, error) {
	if b.characterId == 0 {
		return ConversationContext{}, errors.New("characterId is required")
	}
	if b.npcId == 0 {
		return ConversationContext{}, errors.New("npcId is required")
	}
	if b.currentState == "" {
		return ConversationContext{}, errors.New("currentState is required")
	}

	return ConversationContext{
		characterId:  b.characterId,
		npcId:        b.npcId,
		field:        b.field,
		currentState: b.currentState,
		conversation: b.conversation,
		context:      b.context,
	}, nil
}
