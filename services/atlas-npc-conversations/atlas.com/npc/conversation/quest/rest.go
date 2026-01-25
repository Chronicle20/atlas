package quest

import (
	"atlas-npc-conversations/conversation"
	"fmt"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

const (
	Resource = "quest-conversations"
)

// RestModel represents the REST model for quest conversations
type RestModel struct {
	Id                uuid.UUID            `json:"-"`                          // Conversation ID
	QuestId           uint32               `json:"questId"`                    // Quest ID
	NpcId             uint32               `json:"npcId,omitempty"`            // NPC ID (metadata)
	QuestName         string               `json:"questName,omitempty"`        // Quest name (metadata)
	StartStateMachine RestStateMachineModel `json:"startStateMachine"`          // State machine for quest acceptance
	EndStateMachine   *RestStateMachineModel `json:"endStateMachine,omitempty"` // State machine for quest completion (optional)
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return Resource
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return r.Id.String()
}

// SetID sets the resource ID
func (r *RestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid conversation ID: %w", err)
	}
	r.Id = id
	return nil
}

// GetReferences returns the resource references
func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

// GetReferencedIDs returns the referenced IDs
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

// GetReferencedStructs returns the referenced structs
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

// SetToOneReferenceID sets a to-one reference ID
func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many reference IDs
func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

// SetReferencedStructs sets referenced structs
func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	return nil
}

// RestStateMachineModel represents the REST model for a state machine
type RestStateMachineModel struct {
	StartState string           `json:"startState"` // Start state ID
	States     []RestStateModel `json:"states"`     // Conversation states
}

// RestStateModel represents the REST model for conversation states
type RestStateModel struct {
	Id            string                  `json:"id"`                      // State ID
	StateType     string                  `json:"type"`                    // State type
	Dialogue      *RestDialogueModel      `json:"dialogue,omitempty"`      // Dialogue model
	GenericAction *RestGenericActionModel `json:"genericAction,omitempty"` // Generic action model
	CraftAction   *RestCraftActionModel   `json:"craftAction,omitempty"`   // Craft action model
	ListSelection *RestListSelectionModel `json:"listSelection,omitempty"` // List selection model
	AskNumber     *RestAskNumberModel     `json:"askNumber,omitempty"`     // Ask number model
	AskStyle      *RestAskStyleModel      `json:"askStyle,omitempty"`      // Ask style model
}

// RestDialogueModel represents the REST model for dialogue states
type RestDialogueModel struct {
	DialogueType   string            `json:"dialogueType"`
	Text           string            `json:"text"`
	Speaker        string            `json:"speaker,omitempty"`         // Speaker type: "NPC" or "CHARACTER"
	EndChat        *bool             `json:"endChat,omitempty"`         // Whether to show end chat button (defaults to true)
	SecondaryNpcId uint32            `json:"secondaryNpcId,omitempty"`  // Optional secondary NPC template ID
	Choices        []RestChoiceModel `json:"choices,omitempty"`
}

// RestChoiceModel represents the REST model for dialogue choices
type RestChoiceModel struct {
	Text      string            `json:"text"`
	NextState string            `json:"nextState"`
	Context   map[string]string `json:"context,omitempty"`
}

// RestGenericActionModel represents the REST model for generic action states
type RestGenericActionModel struct {
	Operations []RestOperationModel `json:"operations,omitempty"`
	Outcomes   []RestOutcomeModel   `json:"outcomes,omitempty"`
}

// RestOperationModel represents the REST model for operations
type RestOperationModel struct {
	OperationType string            `json:"type"`
	Params        map[string]string `json:"params"`
}

// RestConditionModel represents the REST model for conditions
type RestConditionModel struct {
	Type            string `json:"type"`
	Operator        string `json:"operator"`
	Value           string `json:"value"`
	ReferenceId     string `json:"referenceId,omitempty"`
	Step            string `json:"step,omitempty"`
	IncludeEquipped bool   `json:"includeEquipped,omitempty"`
}

// RestOutcomeModel represents the REST model for outcomes
type RestOutcomeModel struct {
	Conditions []RestConditionModel `json:"conditions"`
	NextState  string               `json:"nextState,omitempty"`
}

// RestCraftActionModel represents the REST model for craft action states
type RestCraftActionModel struct {
	ItemId                string   `json:"itemId"`
	Materials             []uint32 `json:"materials"`
	Quantities            []uint32 `json:"quantities"`
	MesoCost              uint32   `json:"mesoCost"`
	StimulatorId          uint32   `json:"stimulatorId,omitempty"`
	StimulatorFailChance  float64  `json:"stimulatorFailChance,omitempty"`
	SuccessState          string   `json:"successState"`
	FailureState          string   `json:"failureState"`
	MissingMaterialsState string   `json:"missingMaterialsState"`
}

// RestListSelectionModel represents the REST model for list selection states
type RestListSelectionModel struct {
	Title   string            `json:"title"`
	Choices []RestChoiceModel `json:"choices,omitempty"`
}

// RestAskNumberModel represents the REST model for ask number states
type RestAskNumberModel struct {
	Text         string `json:"text"`
	DefaultValue uint32 `json:"default"`
	MinValue     uint32 `json:"min"`
	MaxValue     uint32 `json:"max"`
	ContextKey   string `json:"contextKey,omitempty"`
	NextState    string `json:"nextState,omitempty"`
}

// RestAskStyleModel represents the REST model for ask style states
type RestAskStyleModel struct {
	Text             string   `json:"text"`
	Styles           []uint32 `json:"styles,omitempty"`
	StylesContextKey string   `json:"stylesContextKey,omitempty"`
	ContextKey       string   `json:"contextKey,omitempty"`
	NextState        string   `json:"nextState,omitempty"`
}

// Transform converts a domain model to a REST model
func Transform(m Model) (RestModel, error) {
	startSM, err := TransformStateMachine(m.StartStateMachine())
	if err != nil {
		return RestModel{}, err
	}

	var endSM *RestStateMachineModel
	if m.HasEndStateMachine() {
		endSMValue, err := TransformStateMachine(*m.EndStateMachine())
		if err != nil {
			return RestModel{}, err
		}
		endSM = &endSMValue
	}

	return RestModel{
		Id:                m.Id(),
		QuestId:           m.QuestId(),
		NpcId:             m.NpcId(),
		QuestName:         m.QuestName(),
		StartStateMachine: startSM,
		EndStateMachine:   endSM,
	}, nil
}

// TransformStateMachine converts a StateMachine to RestStateMachineModel
func TransformStateMachine(sm StateMachine) (RestStateMachineModel, error) {
	restStates := make([]RestStateModel, 0, len(sm.States()))
	for _, state := range sm.States() {
		restState, err := TransformState(state)
		if err != nil {
			return RestStateMachineModel{}, err
		}
		restStates = append(restStates, restState)
	}

	return RestStateMachineModel{
		StartState: sm.StartState(),
		States:     restStates,
	}, nil
}

// TransformState converts a StateModel to a RestStateModel
func TransformState(m conversation.StateModel) (RestStateModel, error) {
	restState := RestStateModel{
		Id:        m.Id(),
		StateType: string(m.Type()),
	}

	switch m.Type() {
	case conversation.DialogueStateType:
		dialogue := m.Dialogue()
		if dialogue != nil {
			restDialogue, err := TransformDialogue(*dialogue)
			if err != nil {
				return RestStateModel{}, err
			}
			restState.Dialogue = &restDialogue
		}
	case conversation.GenericActionType:
		genericAction := m.GenericAction()
		if genericAction != nil {
			restGenericAction, err := TransformGenericAction(*genericAction)
			if err != nil {
				return RestStateModel{}, err
			}
			restState.GenericAction = &restGenericAction
		}
	case conversation.CraftActionType:
		craftAction := m.CraftAction()
		if craftAction != nil {
			restCraftAction, err := TransformCraftAction(*craftAction)
			if err != nil {
				return RestStateModel{}, err
			}
			restState.CraftAction = &restCraftAction
		}
	case conversation.ListSelectionType:
		listSelection := m.ListSelection()
		if listSelection != nil {
			restListSelection, err := TransformListSelection(*listSelection)
			if err != nil {
				return RestStateModel{}, err
			}
			restState.ListSelection = &restListSelection
		}
	case conversation.AskNumberType:
		askNumber := m.AskNumber()
		if askNumber != nil {
			restAskNumber := TransformAskNumber(*askNumber)
			restState.AskNumber = &restAskNumber
		}
	case conversation.AskStyleType:
		askStyle := m.AskStyle()
		if askStyle != nil {
			restAskStyle := TransformAskStyle(*askStyle)
			restState.AskStyle = &restAskStyle
		}
	}

	return restState, nil
}

// TransformDialogue converts a DialogueModel to a RestDialogueModel
func TransformDialogue(m conversation.DialogueModel) (RestDialogueModel, error) {
	restChoices := make([]RestChoiceModel, 0, len(m.Choices()))
	for _, choice := range m.Choices() {
		restChoices = append(restChoices, RestChoiceModel{
			Text:      choice.Text(),
			NextState: choice.NextState(),
			Context:   choice.Context(),
		})
	}

	endChat := m.EndChat()
	return RestDialogueModel{
		DialogueType:   string(m.DialogueType()),
		Text:           m.Text(),
		Speaker:        m.Speaker(),
		EndChat:        &endChat,
		SecondaryNpcId: m.SecondaryNpcId(),
		Choices:        restChoices,
	}, nil
}

// TransformGenericAction converts a GenericActionModel to a RestGenericActionModel
func TransformGenericAction(m conversation.GenericActionModel) (RestGenericActionModel, error) {
	restOperations := make([]RestOperationModel, 0, len(m.Operations()))
	for _, operation := range m.Operations() {
		restOperations = append(restOperations, RestOperationModel{
			OperationType: operation.Type(),
			Params:        operation.Params(),
		})
	}

	restOutcomes := make([]RestOutcomeModel, 0, len(m.Outcomes()))
	for _, outcome := range m.Outcomes() {
		restConditions := make([]RestConditionModel, 0, len(outcome.Conditions()))
		for _, condition := range outcome.Conditions() {
			var referenceIdStr string
			if condition.ReferenceId() != 0 {
				referenceIdStr = strconv.FormatUint(uint64(condition.ReferenceId()), 10)
			}

			restConditions = append(restConditions, RestConditionModel{
				Type:            condition.Type(),
				Operator:        condition.Operator(),
				Value:           condition.Value(),
				ReferenceId:     referenceIdStr,
				Step:            condition.Step(),
				IncludeEquipped: condition.IncludeEquipped(),
			})
		}

		restOutcomes = append(restOutcomes, RestOutcomeModel{
			Conditions: restConditions,
			NextState:  outcome.NextState(),
		})
	}

	return RestGenericActionModel{
		Operations: restOperations,
		Outcomes:   restOutcomes,
	}, nil
}

// TransformCraftAction converts a CraftActionModel to a RestCraftActionModel
func TransformCraftAction(m conversation.CraftActionModel) (RestCraftActionModel, error) {
	return RestCraftActionModel{
		ItemId:                m.ItemId(),
		Materials:             m.Materials(),
		Quantities:            m.Quantities(),
		MesoCost:              m.MesoCost(),
		StimulatorId:          m.StimulatorId(),
		StimulatorFailChance:  m.StimulatorFailChance(),
		SuccessState:          m.SuccessState(),
		FailureState:          m.FailureState(),
		MissingMaterialsState: m.MissingMaterialsState(),
	}, nil
}

// TransformListSelection converts a ListSelectionModel to a RestListSelectionModel
func TransformListSelection(m conversation.ListSelectionModel) (RestListSelectionModel, error) {
	restChoices := make([]RestChoiceModel, 0, len(m.Choices()))
	for _, choice := range m.Choices() {
		restChoices = append(restChoices, RestChoiceModel{
			Text:      choice.Text(),
			NextState: choice.NextState(),
			Context:   choice.Context(),
		})
	}

	return RestListSelectionModel{
		Title:   m.Title(),
		Choices: restChoices,
	}, nil
}

// TransformAskNumber converts an AskNumberModel to a RestAskNumberModel
func TransformAskNumber(m conversation.AskNumberModel) RestAskNumberModel {
	return RestAskNumberModel{
		Text:         m.Text(),
		DefaultValue: m.DefaultValue(),
		MinValue:     m.MinValue(),
		MaxValue:     m.MaxValue(),
		ContextKey:   m.ContextKey(),
		NextState:    m.NextState(),
	}
}

// TransformAskStyle converts an AskStyleModel to a RestAskStyleModel
func TransformAskStyle(m conversation.AskStyleModel) RestAskStyleModel {
	return RestAskStyleModel{
		Text:             m.Text(),
		Styles:           m.Styles(),
		StylesContextKey: m.StylesContextKey(),
		ContextKey:       m.ContextKey(),
		NextState:        m.NextState(),
	}
}

// Extract converts a REST model to a domain model
func Extract(r RestModel) (Model, error) {
	if r.QuestId == 0 {
		return Model{}, fmt.Errorf("questId is required")
	}

	startSM, err := ExtractStateMachine(r.StartStateMachine)
	if err != nil {
		return Model{}, fmt.Errorf("startStateMachine: %w", err)
	}

	var endSM *StateMachine
	if r.EndStateMachine != nil {
		endSMValue, err := ExtractStateMachine(*r.EndStateMachine)
		if err != nil {
			return Model{}, fmt.Errorf("endStateMachine: %w", err)
		}
		endSM = &endSMValue
	}

	builder := NewBuilder()
	if r.Id != uuid.Nil {
		builder.SetId(r.Id)
	}

	builder.SetQuestId(r.QuestId).
		SetNpcId(r.NpcId).
		SetQuestName(r.QuestName).
		SetStartStateMachine(startSM).
		SetEndStateMachine(endSM)

	return builder.Build()
}

// ExtractStateMachine converts a RestStateMachineModel to a StateMachine
func ExtractStateMachine(r RestStateMachineModel) (StateMachine, error) {
	smBuilder := NewStateMachineBuilder().SetStartState(r.StartState)

	for _, restState := range r.States {
		state, err := ExtractState(restState)
		if err != nil {
			return StateMachine{}, err
		}
		smBuilder.AddState(state)
	}

	return smBuilder.Build()
}

// ExtractState converts a RestStateModel to a StateModel
func ExtractState(r RestStateModel) (conversation.StateModel, error) {
	stateBuilder := conversation.NewStateBuilder().SetId(r.Id)

	switch conversation.StateType(r.StateType) {
	case conversation.DialogueStateType:
		if r.Dialogue == nil {
			return conversation.StateModel{}, fmt.Errorf("dialogue is required for dialogue state")
		}
		dialogue, err := ExtractDialogue(*r.Dialogue)
		if err != nil {
			return conversation.StateModel{}, err
		}
		stateBuilder.SetDialogue(dialogue)
	case conversation.GenericActionType:
		if r.GenericAction == nil {
			return conversation.StateModel{}, fmt.Errorf("genericAction is required for genericAction state")
		}
		genericAction, err := ExtractGenericAction(*r.GenericAction)
		if err != nil {
			return conversation.StateModel{}, err
		}
		stateBuilder.SetGenericAction(genericAction)
	case conversation.CraftActionType:
		if r.CraftAction == nil {
			return conversation.StateModel{}, fmt.Errorf("craftAction is required for craftAction state")
		}
		craftAction, err := ExtractCraftAction(*r.CraftAction)
		if err != nil {
			return conversation.StateModel{}, err
		}
		stateBuilder.SetCraftAction(craftAction)
	case conversation.ListSelectionType:
		if r.ListSelection == nil {
			return conversation.StateModel{}, fmt.Errorf("listSelection is required for listSelection state")
		}
		listSelection, err := ExtractListSelection(*r.ListSelection)
		if err != nil {
			return conversation.StateModel{}, err
		}
		stateBuilder.SetListSelection(listSelection)
	case conversation.AskNumberType:
		if r.AskNumber == nil {
			return conversation.StateModel{}, fmt.Errorf("askNumber is required for askNumber state")
		}
		askNumber, err := ExtractAskNumber(*r.AskNumber)
		if err != nil {
			return conversation.StateModel{}, err
		}
		stateBuilder.SetAskNumber(askNumber)
	case conversation.AskStyleType:
		if r.AskStyle == nil {
			return conversation.StateModel{}, fmt.Errorf("askStyle is required for askStyle state")
		}
		askStyle, err := ExtractAskStyle(*r.AskStyle)
		if err != nil {
			return conversation.StateModel{}, err
		}
		stateBuilder.SetAskStyle(askStyle)
	default:
		return conversation.StateModel{}, fmt.Errorf("invalid state type: %s", r.StateType)
	}

	return stateBuilder.Build()
}

// ExtractDialogue converts a RestDialogueModel to a DialogueModel
func ExtractDialogue(r RestDialogueModel) (*conversation.DialogueModel, error) {
	dialogueBuilder := conversation.NewDialogueBuilder().
		SetDialogueType(conversation.DialogueType(r.DialogueType)).
		SetText(r.Text).
		SetSpeaker(normalizeSpeaker(r.Speaker)).
		SetSecondaryNpcId(r.SecondaryNpcId)

	// Handle endChat - defaults to true if not specified
	if r.EndChat != nil {
		dialogueBuilder.SetEndChat(*r.EndChat)
	} else {
		dialogueBuilder.SetEndChat(true)
	}

	for _, restChoice := range r.Choices {
		choice, err := ExtractChoice(restChoice)
		if err != nil {
			return nil, err
		}
		dialogueBuilder.AddChoice(choice)
	}

	return dialogueBuilder.Build()
}

// normalizeSpeaker returns the speaker value, defaulting to "NPC" if empty
func normalizeSpeaker(speaker string) string {
	if speaker == "" {
		return "NPC"
	}
	return speaker
}

// ExtractChoice converts a RestChoiceModel to a ChoiceModel
func ExtractChoice(r RestChoiceModel) (conversation.ChoiceModel, error) {
	builder := conversation.NewChoiceBuilder().
		SetText(r.Text).
		SetNextState(r.NextState)

	if r.Context != nil {
		builder.SetContext(r.Context)
	}

	return builder.Build()
}

// ExtractGenericAction converts a RestGenericActionModel to a GenericActionModel
func ExtractGenericAction(r RestGenericActionModel) (*conversation.GenericActionModel, error) {
	genericActionBuilder := conversation.NewGenericActionBuilder()

	for _, restOperation := range r.Operations {
		operation, err := ExtractOperation(restOperation)
		if err != nil {
			return nil, err
		}
		genericActionBuilder.AddOperation(operation)
	}

	for _, restOutcome := range r.Outcomes {
		outcome, err := ExtractOutcome(restOutcome)
		if err != nil {
			return nil, err
		}
		genericActionBuilder.AddOutcome(outcome)
	}

	return genericActionBuilder.Build()
}

// ExtractOperation converts a RestOperationModel to an OperationModel
func ExtractOperation(r RestOperationModel) (conversation.OperationModel, error) {
	return conversation.NewOperationBuilder().
		SetType(r.OperationType).
		SetParams(r.Params).
		Build()
}

// ExtractOutcome converts a RestOutcomeModel to an OutcomeModel
func ExtractOutcome(r RestOutcomeModel) (conversation.OutcomeModel, error) {
	outcomeBuilder := conversation.NewOutcomeBuilder()

	for _, c := range r.Conditions {
		condition, err := conversation.NewConditionBuilder().
			SetType(c.Type).
			SetOperator(c.Operator).
			SetValue(c.Value).
			SetReferenceId(c.ReferenceId).
			SetStep(c.Step).
			SetIncludeEquipped(c.IncludeEquipped).
			Build()

		if err != nil {
			return conversation.OutcomeModel{}, err
		}

		outcomeBuilder.AddCondition(condition)
	}

	if r.NextState != "" {
		outcomeBuilder.SetNextState(r.NextState)
	}

	return outcomeBuilder.Build()
}

// ExtractCraftAction converts a RestCraftActionModel to a CraftActionModel
func ExtractCraftAction(r RestCraftActionModel) (*conversation.CraftActionModel, error) {
	craftActionBuilder := conversation.NewCraftActionBuilder().
		SetItemId(r.ItemId).
		SetMaterials(r.Materials).
		SetQuantities(r.Quantities).
		SetMesoCost(r.MesoCost).
		SetStimulatorId(r.StimulatorId).
		SetStimulatorFailChance(r.StimulatorFailChance).
		SetSuccessState(r.SuccessState).
		SetFailureState(r.FailureState).
		SetMissingMaterialsState(r.MissingMaterialsState)

	return craftActionBuilder.Build()
}

// ExtractListSelection converts a RestListSelectionModel to a ListSelectionModel
func ExtractListSelection(r RestListSelectionModel) (*conversation.ListSelectionModel, error) {
	b := conversation.NewListSelectionBuilder().SetTitle(r.Title)

	for _, restChoice := range r.Choices {
		choice, err := ExtractChoice(restChoice)
		if err != nil {
			return nil, err
		}
		b.AddChoice(choice)
	}

	return b.Build()
}

// ExtractAskNumber converts a RestAskNumberModel to an AskNumberModel
func ExtractAskNumber(r RestAskNumberModel) (*conversation.AskNumberModel, error) {
	b := conversation.NewAskNumberBuilder().
		SetText(r.Text).
		SetDefaultValue(r.DefaultValue).
		SetMinValue(r.MinValue).
		SetMaxValue(r.MaxValue).
		SetNextState(r.NextState)

	if r.ContextKey != "" {
		b.SetContextKey(r.ContextKey)
	}

	return b.Build()
}

// ExtractAskStyle converts a RestAskStyleModel to an AskStyleModel
func ExtractAskStyle(r RestAskStyleModel) (*conversation.AskStyleModel, error) {
	b := conversation.NewAskStyleBuilder().
		SetText(r.Text).
		SetNextState(r.NextState)

	if len(r.Styles) > 0 {
		b.SetStyles(r.Styles)
	}

	if r.StylesContextKey != "" {
		b.SetStylesContextKey(r.StylesContextKey)
	}

	if r.ContextKey != "" {
		b.SetContextKey(r.ContextKey)
	}

	return b.Build()
}
