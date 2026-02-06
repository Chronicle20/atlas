package conversation

import (
	"fmt"
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

// RestStateModel represents the REST model for conversation states
// This is shared between NPC conversations and quest state machines
type RestStateModel struct {
	Id              string                     `json:"id"`                        // State ID
	StateType       string                     `json:"type"`                      // State type
	Dialogue        *RestDialogueModel         `json:"dialogue,omitempty"`        // Dialogue model (if type is dialogue)
	GenericAction   *RestGenericActionModel    `json:"genericAction,omitempty"`   // Generic action model (if type is genericAction)
	CraftAction     *RestCraftActionModel      `json:"craftAction,omitempty"`     // Craft action model (if type is craftAction)
	TransportAction *RestTransportActionModel  `json:"transportAction,omitempty"` // Transport action model (if type is transportAction)
	ListSelection *RestListSelectionModel `json:"listSelection,omitempty"` // List selection model (if type is listSelection)
	AskNumber     *RestAskNumberModel    `json:"askNumber,omitempty"`     // Ask number model (if type is askNumber)
	AskStyle                   *RestAskStyleModel                   `json:"askStyle,omitempty"`                   // Ask style model (if type is askStyle)
	AskSlideMenu               *RestAskSlideMenuModel               `json:"askSlideMenu,omitempty"`               // Ask slide menu model (if type is askSlideMenu)
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
func (r *RestStateModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many reference IDs
func (r *RestStateModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// SetReferencedStructs sets referenced structs
func (r *RestStateModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

// RestDialogueModel represents the REST model for dialogue states
type RestDialogueModel struct {
	DialogueType   string            `json:"dialogueType"`             // Dialogue type
	Text           string            `json:"text"`                     // Dialogue text
	Speaker        string            `json:"speaker,omitempty"`        // Speaker type: "NPC" or "CHARACTER"
	EndChat        *bool             `json:"endChat,omitempty"`        // Whether to show end chat button (defaults to true)
	SecondaryNpcId uint32            `json:"secondaryNpcId,omitempty"` // Optional secondary NPC template ID
	Choices        []RestChoiceModel `json:"choices,omitempty"`        // Dialogue choices
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
	Type            string `json:"type"`                      // Condition type
	Operator        string `json:"operator"`                  // Operator
	Value           string `json:"value"`                     // Value
	ReferenceId     string `json:"referenceId,omitempty"`     // Reference ID (for items, quests, etc.)
	Step            string `json:"step,omitempty"`            // Step (for quest progress)
	IncludeEquipped bool   `json:"includeEquipped,omitempty"` // For item conditions: also check equipped items
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
	SuccessState          string   `json:"successState"`                   // Success state ID
	FailureState          string   `json:"failureState"`                   // Failure state ID
	MissingMaterialsState string   `json:"missingMaterialsState"`          // Missing materials state ID
}

// RestTransportActionModel represents the REST model for transport action states
// Used for instance-based transports that go through saga-orchestrator
type RestTransportActionModel struct {
	RouteName             string `json:"routeName"`                       // Transport route name
	FailureState          string `json:"failureState"`                    // General failure state ID
	CapacityFullState     string `json:"capacityFullState,omitempty"`     // State when transport is at capacity
	AlreadyInTransitState string `json:"alreadyInTransitState,omitempty"` // State when character is already in transit
	RouteNotFoundState    string `json:"routeNotFoundState,omitempty"`    // State when route doesn't exist
	ServiceErrorState     string `json:"serviceErrorState,omitempty"`     // State when transport service fails
}

// RestListSelectionModel represents the REST model for list selection states
type RestListSelectionModel struct {
	Title   string            `json:"title"`             // List selection title
	Choices []RestChoiceModel `json:"choices,omitempty"` // Dialogue choices
}

// RestAskNumberModel represents the REST model for ask number states
type RestAskNumberModel struct {
	Text         string `json:"text"`                 // Ask number text
	DefaultValue uint32 `json:"default"`              // Default value
	MinValue     uint32 `json:"min"`                  // Minimum value
	MaxValue     uint32 `json:"max"`                  // Maximum value
	ContextKey   string `json:"contextKey,omitempty"` // Context key (defaults to "quantity")
	NextState    string `json:"nextState,omitempty"`  // Next state ID
}

// RestAskStyleModel represents the REST model for ask style states
type RestAskStyleModel struct {
	Text             string   `json:"text"`                       // Ask style text
	Styles           []uint32 `json:"styles,omitempty"`           // Available style IDs (optional if stylesContextKey provided)
	StylesContextKey string   `json:"stylesContextKey,omitempty"` // Context key containing dynamic styles (optional if styles provided)
	ContextKey       string   `json:"contextKey,omitempty"`       // Context key (defaults to "selectedStyle")
	NextState        string   `json:"nextState,omitempty"`        // Next state ID
}

// RestAskSlideMenuModel represents the REST model for ask slide menu states
type RestAskSlideMenuModel struct {
	Title      string            `json:"title"`                 // Slide menu title
	MenuType   uint32            `json:"menuType"`              // Menu type (determines UI style)
	ContextKey string            `json:"contextKey,omitempty"`  // Context key (defaults to "selectedOption")
	Choices    []RestChoiceModel `json:"choices,omitempty"`     // Menu choices
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
func (r *RestOptionSetModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many reference IDs
func (r *RestOptionSetModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// SetReferencedStructs sets referenced structs
func (r *RestOptionSetModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
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
	case TransportActionType:
		transportAction := m.TransportAction()
		if transportAction != nil {
			restTransportAction := TransformTransportAction(*transportAction)
			restState.TransportAction = &restTransportAction
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
	case AskNumberType:
		askNumber := m.AskNumber()
		if askNumber != nil {
			restAskNumber := TransformAskNumber(*askNumber)
			restState.AskNumber = &restAskNumber
		}
	case AskStyleType:
		askStyle := m.AskStyle()
		if askStyle != nil {
			restAskStyle := TransformAskStyle(*askStyle)
			restState.AskStyle = &restAskStyle
		}
	case AskSlideMenuType:
		askSlideMenu := m.AskSlideMenu()
		if askSlideMenu != nil {
			restAskSlideMenu := TransformAskSlideMenu(*askSlideMenu)
			restState.AskSlideMenu = &restAskSlideMenu
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
func TransformCraftAction(m CraftActionModel) (RestCraftActionModel, error) {
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

// TransformTransportAction converts a TransportActionModel to a RestTransportActionModel
func TransformTransportAction(m TransportActionModel) RestTransportActionModel {
	return RestTransportActionModel{
		RouteName:             m.RouteName(),
		FailureState:          m.FailureState(),
		CapacityFullState:     m.CapacityFullState(),
		AlreadyInTransitState: m.AlreadyInTransitState(),
		RouteNotFoundState:    m.RouteNotFoundState(),
		ServiceErrorState:     m.ServiceErrorState(),
	}
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

// TransformAskNumber converts an AskNumberModel to a RestAskNumberModel
func TransformAskNumber(m AskNumberModel) RestAskNumberModel {
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
func TransformAskStyle(m AskStyleModel) RestAskStyleModel {
	return RestAskStyleModel{
		Text:             m.Text(),
		Styles:           m.Styles(),
		StylesContextKey: m.StylesContextKey(),
		ContextKey:       m.ContextKey(),
		NextState:        m.NextState(),
	}
}

// TransformAskSlideMenu converts an AskSlideMenuModel to a RestAskSlideMenuModel
func TransformAskSlideMenu(m AskSlideMenuModel) RestAskSlideMenuModel {
	restChoices := make([]RestChoiceModel, 0, len(m.Choices()))
	for _, choice := range m.Choices() {
		restChoices = append(restChoices, RestChoiceModel{
			Text:      choice.Text(),
			NextState: choice.NextState(),
			Context:   choice.Context(),
		})
	}

	return RestAskSlideMenuModel{
		Title:      m.Title(),
		MenuType:   m.MenuType(),
		ContextKey: m.ContextKey(),
		Choices:    restChoices,
	}
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
	case TransportActionType:
		if r.TransportAction == nil {
			return StateModel{}, fmt.Errorf("transportAction is required for transportAction state")
		}
		transportAction, err := ExtractTransportAction(*r.TransportAction)
		if err != nil {
			return StateModel{}, err
		}
		stateBuilder.SetTransportAction(transportAction)
	case ListSelectionType:
		if r.ListSelection == nil {
			return StateModel{}, fmt.Errorf("listSelection is required for listSelection state")
		}
		listSelection, err := ExtractListSelection(*r.ListSelection)
		if err != nil {
			return StateModel{}, err
		}
		stateBuilder.SetListSelection(listSelection)
	case AskNumberType:
		if r.AskNumber == nil {
			return StateModel{}, fmt.Errorf("askNumber is required for askNumber state")
		}
		askNumber, err := ExtractAskNumber(*r.AskNumber)
		if err != nil {
			return StateModel{}, err
		}
		stateBuilder.SetAskNumber(askNumber)
	case AskStyleType:
		if r.AskStyle == nil {
			return StateModel{}, fmt.Errorf("askStyle is required for askStyle state")
		}
		askStyle, err := ExtractAskStyle(*r.AskStyle)
		if err != nil {
			return StateModel{}, err
		}
		stateBuilder.SetAskStyle(askStyle)
	case AskSlideMenuType:
		if r.AskSlideMenu == nil {
			return StateModel{}, fmt.Errorf("askSlideMenu is required for askSlideMenu state")
		}
		askSlideMenu, err := ExtractAskSlideMenu(*r.AskSlideMenu)
		if err != nil {
			return StateModel{}, err
		}
		stateBuilder.SetAskSlideMenu(askSlideMenu)
	default:
		return StateModel{}, fmt.Errorf("invalid state type: %s", r.StateType)
	}

	return stateBuilder.Build()
}

// ExtractDialogue converts a RestDialogueModel to a DialogueModel
func ExtractDialogue(r RestDialogueModel) (*DialogueModel, error) {
	dialogueBuilder := NewDialogueBuilder().
		SetDialogueType(DialogueType(r.DialogueType)).
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
			SetIncludeEquipped(c.IncludeEquipped).
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
		SetSuccessState(r.SuccessState).
		SetFailureState(r.FailureState).
		SetMissingMaterialsState(r.MissingMaterialsState)

	return craftActionBuilder.Build()
}

// ExtractTransportAction converts a RestTransportActionModel to a TransportActionModel
func ExtractTransportAction(r RestTransportActionModel) (*TransportActionModel, error) {
	transportActionBuilder := NewTransportActionBuilder().
		SetRouteName(r.RouteName).
		SetFailureState(r.FailureState).
		SetCapacityFullState(r.CapacityFullState).
		SetAlreadyInTransitState(r.AlreadyInTransitState).
		SetRouteNotFoundState(r.RouteNotFoundState).
		SetServiceErrorState(r.ServiceErrorState)

	return transportActionBuilder.Build()
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

// ExtractAskNumber converts a RestAskNumberModel to an AskNumberModel
func ExtractAskNumber(r RestAskNumberModel) (*AskNumberModel, error) {
	b := NewAskNumberBuilder().
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
func ExtractAskStyle(r RestAskStyleModel) (*AskStyleModel, error) {
	b := NewAskStyleBuilder().
		SetText(r.Text).
		SetNextState(r.NextState)

	// Set styles if provided
	if len(r.Styles) > 0 {
		b.SetStyles(r.Styles)
	}

	// Set stylesContextKey if provided
	if r.StylesContextKey != "" {
		b.SetStylesContextKey(r.StylesContextKey)
	}

	if r.ContextKey != "" {
		b.SetContextKey(r.ContextKey)
	}

	return b.Build()
}

// ExtractAskSlideMenu converts a RestAskSlideMenuModel to an AskSlideMenuModel
func ExtractAskSlideMenu(r RestAskSlideMenuModel) (*AskSlideMenuModel, error) {
	b := NewAskSlideMenuBuilder().
		SetTitle(r.Title).
		SetMenuType(r.MenuType)

	if r.ContextKey != "" {
		b.SetContextKey(r.ContextKey)
	}

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
