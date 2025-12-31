package conversation

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

const (
	Resource = "conversations"
)

// RestModel represents the REST model for NPC conversations
type RestModel struct {
	Id         uuid.UUID        `json:"-"`          // Conversation ID
	NpcId      uint32           `json:"npcId"`      // NPC ID
	StartState string           `json:"startState"` // Start state ID
	States     []RestStateModel `json:"states"`     // Conversation states
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

// RestStateModel represents the REST model for conversation states
type RestStateModel struct {
	Id            string                  `json:"id"`                      // State ID
	StateType     string                  `json:"type"`                    // State type
	Dialogue      *RestDialogueModel      `json:"dialogue,omitempty"`      // Dialogue model (if type is dialogue)
	GenericAction *RestGenericActionModel `json:"genericAction,omitempty"` // Generic action model (if type is genericAction)
	CraftAction   *RestCraftActionModel   `json:"craftAction,omitempty"`   // Craft action model (if type is craftAction)
	ListSelection *RestListSelectionModel `json:"listSelection,omitempty"` // List selection model (if type is listSelection)
}

// GetID returns the resource ID
func (r RestStateModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID
func (r *RestStateModel) SetID(id string) error {
	r.Id = id
	return nil
}

// GetName returns the resource name
func (r RestStateModel) GetName() string {
	return "states"
}

// GetReferences returns the resource references
func (r RestStateModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

// GetReferencedIDs returns the referenced IDs
func (r RestStateModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

// GetReferencedStructs returns the referenced structs
func (r RestStateModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

// SetToOneReferenceID sets a to-one reference ID
func (r *RestStateModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many reference IDs
func (r *RestStateModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

// SetReferencedStructs sets referenced structs
func (r *RestStateModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	return nil
}

// RestDialogueModel represents the REST model for dialogue states
type RestDialogueModel struct {
	DialogueType string            `json:"dialogueType"`      // Dialogue type
	Text         string            `json:"text"`              // Dialogue text
	Choices      []RestChoiceModel `json:"choices,omitempty"` // Dialogue choices
}

// RestChoiceModel represents the REST model for dialogue choices
type RestChoiceModel struct {
	Text      string            `json:"text"`              // Choice text
	NextState string            `json:"nextState"`         // Next state ID
	Context   map[string]string `json:"context,omitempty"` // Context data
}

// RestGenericActionModel represents the REST model for generic action states
type RestGenericActionModel struct {
	Operations []RestOperationModel `json:"operations,omitempty"` // Operations
	Outcomes   []RestOutcomeModel   `json:"outcomes,omitempty"`   // Outcomes
}

// RestOperationModel represents the REST model for operations
type RestOperationModel struct {
	OperationType string            `json:"type"`   // Operation type
	Params        map[string]string `json:"params"` // Operation parameters
}

// RestConditionModel represents the REST model for conditions
type RestConditionModel struct {
	Type        string `json:"type"`                  // Condition type
	Operator    string `json:"operator"`              // Operator
	Value       string `json:"value"`                 // Value
	ReferenceId string `json:"referenceId,omitempty"` // Reference ID (for items, quests, etc.)
	Step        string `json:"step,omitempty"`        // Step (for quest progress)
}

// RestOutcomeModel represents the REST model for outcomes
type RestOutcomeModel struct {
	Conditions []RestConditionModel `json:"conditions"`          // Outcome conditions
	NextState  string               `json:"nextState,omitempty"` // Next state ID
}

// RestCraftActionModel represents the REST model for craft action states
type RestCraftActionModel struct {
	ItemId                string   `json:"itemId"`                         // Item ID
	Materials             []uint32 `json:"materials"`                      // Material item IDs
	Quantities            []uint32 `json:"quantities"`                     // Material quantities
	MesoCost              uint32   `json:"mesoCost"`                       // Meso cost
	StimulatorId          uint32   `json:"stimulatorId,omitempty"`         // Stimulator item ID
	StimulatorFailChance  float64  `json:"stimulatorFailChance,omitempty"` // Stimulator failure chance
	MissingMaterialsState string   `json:"missingMaterialsState"`          // Missing materials state ID
}

// RestListSelectionModel represents the REST model for list selection states
type RestListSelectionModel struct {
	Title   string            `json:"title"`             // List selection title
	Choices []RestChoiceModel `json:"choices,omitempty"` // Dialogue choices
}

// RestOptionSetModel represents the REST model for option sets
type RestOptionSetModel struct {
	Id      string            `json:"id"`      // Option set ID
	Options []RestOptionModel `json:"options"` // Options
}

// GetID returns the resource ID
func (r RestOptionSetModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID
func (r *RestOptionSetModel) SetID(id string) error {
	r.Id = id
	return nil
}

// GetName returns the resource name
func (r RestOptionSetModel) GetName() string {
	return "optionSets"
}

// GetReferences returns the resource references
func (r RestOptionSetModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

// GetReferencedIDs returns the referenced IDs
func (r RestOptionSetModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

// GetReferencedStructs returns the referenced structs
func (r RestOptionSetModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

// SetToOneReferenceID sets a to-one reference ID
func (r *RestOptionSetModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many reference IDs
func (r *RestOptionSetModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

// SetReferencedStructs sets referenced structs
func (r *RestOptionSetModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	return nil
}

// RestOptionModel represents the REST model for options
type RestOptionModel struct {
	Id         uint32   `json:"id"`                   // Option ID
	Name       string   `json:"name"`                 // Option name
	Materials  []uint32 `json:"materials,omitempty"`  // Material item IDs
	Quantities []uint32 `json:"quantities,omitempty"` // Material quantities
	Meso       uint32   `json:"meso"`                 // Meso cost
}

// Transform converts a domain model to a REST model
func Transform(m Model) (RestModel, error) {
	// Transform states
	restStates := make([]RestStateModel, 0, len(m.States()))
	for _, state := range m.States() {
		restState, err := TransformState(state)
		if err != nil {
			return RestModel{}, err
		}
		restStates = append(restStates, restState)
	}

	return RestModel{
		Id:         m.Id(),
		NpcId:      m.NpcId(),
		StartState: m.StartState(),
		States:     restStates,
	}, nil
}

// TransformState converts a StateModel to a RestStateModel
func TransformState(m StateModel) (RestStateModel, error) {
	restState := RestStateModel{
		Id:        m.Id(),
		StateType: string(m.Type()),
	}

	switch m.Type() {
	case DialogueStateType:
		dialogue := m.Dialogue()
		if dialogue != nil {
			restDialogue, err := TransformDialogue(*dialogue)
			if err != nil {
				return RestStateModel{}, err
			}
			restState.Dialogue = &restDialogue
		}
	case GenericActionType:
		genericAction := m.GenericAction()
		if genericAction != nil {
			restGenericAction, err := TransformGenericAction(*genericAction)
			if err != nil {
				return RestStateModel{}, err
			}
			restState.GenericAction = &restGenericAction
		}
	case CraftActionType:
		craftAction := m.CraftAction()
		if craftAction != nil {
			restCraftAction, err := TransformCraftAction(*craftAction)
			if err != nil {
				return RestStateModel{}, err
			}
			restState.CraftAction = &restCraftAction
		}
	case ListSelectionType:
		listSelection := m.ListSelection()
		if listSelection != nil {
			restListSelection, err := TransformListSelection(*listSelection)
			if err != nil {
				return RestStateModel{}, err
			}
			restState.ListSelection = &restListSelection
		}
	}

	return restState, nil
}

// TransformDialogue converts a DialogueModel to a RestDialogueModel
func TransformDialogue(m DialogueModel) (RestDialogueModel, error) {
	restChoices := make([]RestChoiceModel, 0, len(m.Choices()))
	for _, choice := range m.Choices() {
		restChoices = append(restChoices, RestChoiceModel{
			Text:      choice.Text(),
			NextState: choice.NextState(),
			Context:   choice.Context(),
		})
	}

	return RestDialogueModel{
		DialogueType: string(m.DialogueType()),
		Text:         m.Text(),
		Choices:      restChoices,
	}, nil
}

// TransformGenericAction converts a GenericActionModel to a RestGenericActionModel
func TransformGenericAction(m GenericActionModel) (RestGenericActionModel, error) {
	restOperations := make([]RestOperationModel, 0, len(m.Operations()))
	for _, operation := range m.Operations() {
		restOperations = append(restOperations, RestOperationModel{
			OperationType: operation.Type(),
			Params:        operation.Params(),
		})
	}

	restOutcomes := make([]RestOutcomeModel, 0, len(m.Outcomes()))
	for _, outcome := range m.Outcomes() {
		// Convert ConditionModel to RestConditionModel
		restConditions := make([]RestConditionModel, 0, len(outcome.Conditions()))
		for _, condition := range outcome.Conditions() {
			var referenceIdStr string
			if condition.ReferenceId() != 0 {
				referenceIdStr = strconv.FormatUint(uint64(condition.ReferenceId()), 10)
			}

			restConditions = append(restConditions, RestConditionModel{
				Type:        condition.Type(),
				Operator:    condition.Operator(),
				Value:       condition.Value(),
				ReferenceId: referenceIdStr,
				Step:        condition.Step(),
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
func TransformCraftAction(m CraftActionModel) (RestCraftActionModel, error) {
	return RestCraftActionModel{
		ItemId:                m.ItemId(),
		Materials:             m.Materials(),
		Quantities:            m.Quantities(),
		MesoCost:              m.MesoCost(),
		StimulatorId:          m.StimulatorId(),
		StimulatorFailChance:  m.StimulatorFailChance(),
		MissingMaterialsState: m.MissingMaterialsState(),
	}, nil
}

// TransformListSelection converts a ListSelectionModel to a RestListSelectionModel
func TransformListSelection(m ListSelectionModel) (RestListSelectionModel, error) {
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

// TransformOptionSet converts an OptionSetModel to a RestOptionSetModel
func TransformOptionSet(m OptionSetModel) (RestOptionSetModel, error) {
	restOptions := make([]RestOptionModel, 0, len(m.Options()))
	for _, option := range m.Options() {
		restOptions = append(restOptions, RestOptionModel{
			Id:         option.Id(),
			Name:       option.Name(),
			Materials:  option.Materials(),
			Quantities: option.Quantities(),
			Meso:       option.Meso(),
		})
	}

	return RestOptionSetModel{
		Id:      m.Id(),
		Options: restOptions,
	}, nil
}

// Extract converts a REST model to a domain model
func Extract(r RestModel) (Model, error) {
	// Validate required fields
	if r.NpcId == 0 {
		return Model{}, fmt.Errorf("npcId is required")
	}
	if r.StartState == "" {
		return Model{}, fmt.Errorf("startState is required")
	}
	if len(r.States) == 0 {
		return Model{}, fmt.Errorf("states are required")
	}

	// Create a new model using the builder
	builder := NewBuilder()

	// Set ID if provided, otherwise it will be auto-generated
	if r.Id != uuid.Nil {
		builder.SetId(r.Id)
	}

	builder.SetNpcId(r.NpcId).
		SetStartState(r.StartState)

	// Extract states
	for _, restState := range r.States {
		state, err := ExtractState(restState)
		if err != nil {
			return Model{}, err
		}
		builder.AddState(state)
	}

	return builder.Build()
}

// ExtractState converts a RestStateModel to a StateModel
func ExtractState(r RestStateModel) (StateModel, error) {
	stateBuilder := NewStateBuilder().SetId(r.Id)

	switch StateType(r.StateType) {
	case DialogueStateType:
		if r.Dialogue == nil {
			return StateModel{}, fmt.Errorf("dialogue is required for dialogue state")
		}
		dialogue, err := ExtractDialogue(*r.Dialogue)
		if err != nil {
			return StateModel{}, err
		}
		stateBuilder.SetDialogue(dialogue)
	case GenericActionType:
		if r.GenericAction == nil {
			return StateModel{}, fmt.Errorf("genericAction is required for genericAction state")
		}
		genericAction, err := ExtractGenericAction(*r.GenericAction)
		if err != nil {
			return StateModel{}, err
		}
		stateBuilder.SetGenericAction(genericAction)
	case CraftActionType:
		if r.CraftAction == nil {
			return StateModel{}, fmt.Errorf("craftAction is required for craftAction state")
		}
		craftAction, err := ExtractCraftAction(*r.CraftAction)
		if err != nil {
			return StateModel{}, err
		}
		stateBuilder.SetCraftAction(craftAction)
	case ListSelectionType:
		if r.ListSelection == nil {
			return StateModel{}, fmt.Errorf("listSelection is required for listSelection state")
		}
		listSelection, err := ExtractListSelection(*r.ListSelection)
		if err != nil {
			return StateModel{}, err
		}
		stateBuilder.SetListSelection(listSelection)
	default:
		return StateModel{}, fmt.Errorf("invalid state type: %s", r.StateType)
	}

	return stateBuilder.Build()
}

// ExtractDialogue converts a RestDialogueModel to a DialogueModel
func ExtractDialogue(r RestDialogueModel) (*DialogueModel, error) {
	dialogueBuilder := NewDialogueBuilder().
		SetDialogueType(DialogueType(r.DialogueType)).
		SetText(r.Text)

	for _, restChoice := range r.Choices {
		choice, err := ExtractChoice(restChoice)
		if err != nil {
			return nil, err
		}
		dialogueBuilder.AddChoice(choice)
	}

	return dialogueBuilder.Build()
}

// ExtractChoice converts a RestChoiceModel to a ChoiceModel
func ExtractChoice(r RestChoiceModel) (ChoiceModel, error) {
	builder := NewChoiceBuilder().
		SetText(r.Text).
		SetNextState(r.NextState)

	if r.Context != nil {
		builder.SetContext(r.Context)
	}

	return builder.Build()
}

// ExtractGenericAction converts a RestGenericActionModel to a GenericActionModel
func ExtractGenericAction(r RestGenericActionModel) (*GenericActionModel, error) {
	genericActionBuilder := NewGenericActionBuilder()

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
func ExtractOperation(r RestOperationModel) (OperationModel, error) {
	return NewOperationBuilder().
		SetType(r.OperationType).
		SetParams(r.Params).
		Build()
}

// ExtractOutcome converts a RestOutcomeModel to an OutcomeModel
func ExtractOutcome(r RestOutcomeModel) (OutcomeModel, error) {
	outcomeBuilder := NewOutcomeBuilder()

	for _, c := range r.Conditions {
		condition, err := NewConditionBuilder().
			SetType(c.Type).
			SetOperator(c.Operator).
			SetValue(c.Value).
			SetReferenceId(c.ReferenceId).
			SetStep(c.Step).
			Build()

		if err != nil {
			return OutcomeModel{}, err
		}

		outcomeBuilder.AddCondition(condition)
	}

	if r.NextState != "" {
		outcomeBuilder.SetNextState(r.NextState)
	}

	return outcomeBuilder.Build()
}

// ExtractCraftAction converts a RestCraftActionModel to a CraftActionModel
func ExtractCraftAction(r RestCraftActionModel) (*CraftActionModel, error) {
	craftActionBuilder := NewCraftActionBuilder().
		SetItemId(r.ItemId).
		SetMaterials(r.Materials).
		SetQuantities(r.Quantities).
		SetMesoCost(r.MesoCost).
		SetStimulatorId(r.StimulatorId).
		SetStimulatorFailChance(r.StimulatorFailChance).
		SetMissingMaterialsState(r.MissingMaterialsState)

	return craftActionBuilder.Build()
}

// ExtractListSelection converts a RestListSelectionModel to a ListSelectionModel
func ExtractListSelection(r RestListSelectionModel) (*ListSelectionModel, error) {
	b := NewListSelectionBuilder().
		SetTitle(r.Title)

	for _, restChoice := range r.Choices {
		choice, err := ExtractChoice(restChoice)
		if err != nil {
			return nil, err
		}
		b.AddChoice(choice)
	}

	return b.Build()
}

// ExtractOptionSet converts a RestOptionSetModel to an OptionSetModel
func ExtractOptionSet(r RestOptionSetModel) (OptionSetModel, error) {
	optionSetBuilder := NewOptionSetBuilder().SetId(r.Id)

	for _, restOption := range r.Options {
		option, err := ExtractOption(restOption)
		if err != nil {
			return OptionSetModel{}, err
		}
		optionSetBuilder.AddOption(option)
	}

	return optionSetBuilder.Build()
}

// ExtractOption converts a RestOptionModel to an OptionModel
func ExtractOption(r RestOptionModel) (OptionModel, error) {
	optionBuilder := NewOptionBuilder().
		SetId(r.Id).
		SetName(r.Name).
		SetMeso(r.Meso)

	if len(r.Materials) > 0 {
		optionBuilder.SetMaterials(r.Materials)
	}
	if len(r.Quantities) > 0 {
		optionBuilder.SetQuantities(r.Quantities)
	}

	return optionBuilder.Build()
}

// ConversationStartRequest represents a request to start a conversation
type ConversationStartRequest struct {
	CharacterId uint32 `json:"characterId"` // Character ID
	NpcId       uint32 `json:"npcId"`       // NPC ID
	MapId       uint32 `json:"mapId"`       // Map ID
}

// ConversationContinueRequest represents a request to continue a conversation
type ConversationContinueRequest struct {
	CharacterId     uint32 `json:"characterId"`     // Character ID
	NpcId           uint32 `json:"npcId"`           // NPC ID
	Action          byte   `json:"action"`          // Action type
	LastMessageType byte   `json:"lastMessageType"` // Last message type
	Selection       int32  `json:"selection"`       // Selection index
}

// ConversationEventRequest represents a request to continue a conversation via an event
type ConversationEventRequest struct {
	CharacterId uint32 `json:"characterId"` // Character ID
	Action      byte   `json:"action"`      // Action type
	ReferenceId int32  `json:"referenceId"` // Reference ID
}

// ConversationEndRequest represents a request to end a conversation
type ConversationEndRequest struct {
	CharacterId uint32 `json:"characterId"` // Character ID
}
