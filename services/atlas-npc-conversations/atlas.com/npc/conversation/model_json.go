package conversation

import (
	"encoding/json"
	"errors"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/google/uuid"
)

// storedConversation is a lightweight StateContainer used after deserialization from Redis.
// It preserves the startState and states without requiring npc or quest package imports.
type storedConversation struct {
	startState string
	states     []StateModel
}

func (s storedConversation) StartState() string            { return s.startState }
func (s storedConversation) States() []StateModel          { return s.states }
func (s storedConversation) FindState(stateId string) (StateModel, error) {
	for _, state := range s.states {
		if state.Id() == stateId {
			return state, nil
		}
	}
	return StateModel{}, errors.New("state not found")
}

// conversationDataJSON is the JSON representation of a StateContainer.
type conversationDataJSON struct {
	StartState string       `json:"startState"`
	States     []StateModel `json:"states"`
}

// --- ChoiceModel ---

func (c ChoiceModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Text      string            `json:"text"`
		NextState string            `json:"nextState"`
		Context   map[string]string `json:"context,omitempty"`
	}{c.text, c.nextState, c.context})
}

func (c *ChoiceModel) UnmarshalJSON(data []byte) error {
	var aux struct {
		Text      string            `json:"text"`
		NextState string            `json:"nextState"`
		Context   map[string]string `json:"context,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	c.text = aux.Text
	c.nextState = aux.NextState
	c.context = aux.Context
	return nil
}

// --- DialogueModel ---

func (d DialogueModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		DialogueType   DialogueType  `json:"dialogueType"`
		Text           string        `json:"text"`
		Speaker        string        `json:"speaker,omitempty"`
		EndChat        bool          `json:"endChat"`
		SecondaryNpcId uint32        `json:"secondaryNpcId,omitempty"`
		Choices        []ChoiceModel `json:"choices"`
	}{d.dialogueType, d.text, d.speaker, d.endChat, d.secondaryNpcId, d.choices})
}

func (d *DialogueModel) UnmarshalJSON(data []byte) error {
	var aux struct {
		DialogueType   DialogueType  `json:"dialogueType"`
		Text           string        `json:"text"`
		Speaker        string        `json:"speaker,omitempty"`
		EndChat        bool          `json:"endChat"`
		SecondaryNpcId uint32        `json:"secondaryNpcId,omitempty"`
		Choices        []ChoiceModel `json:"choices"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	d.dialogueType = aux.DialogueType
	d.text = aux.Text
	d.speaker = aux.Speaker
	d.endChat = aux.EndChat
	d.secondaryNpcId = aux.SecondaryNpcId
	d.choices = aux.Choices
	return nil
}

// --- OperationModel ---

func (o OperationModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		OperationType string            `json:"operationType"`
		Params        map[string]string `json:"params,omitempty"`
	}{o.operationType, o.params})
}

func (o *OperationModel) UnmarshalJSON(data []byte) error {
	var aux struct {
		OperationType string            `json:"operationType"`
		Params        map[string]string `json:"params,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	o.operationType = aux.OperationType
	o.params = aux.Params
	return nil
}

// --- ConditionModel ---

func (c ConditionModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ConditionType   string `json:"conditionType"`
		Operator        string `json:"operator"`
		Value           string `json:"value"`
		ReferenceId     string `json:"referenceId,omitempty"`
		Step            string `json:"step,omitempty"`
		WorldId         string `json:"worldId,omitempty"`
		ChannelId       string `json:"channelId,omitempty"`
		IncludeEquipped bool   `json:"includeEquipped,omitempty"`
	}{c.conditionType, c.operator, c.value, c.referenceId, c.step, c.worldId, c.channelId, c.includeEquipped})
}

func (c *ConditionModel) UnmarshalJSON(data []byte) error {
	var aux struct {
		ConditionType   string `json:"conditionType"`
		Operator        string `json:"operator"`
		Value           string `json:"value"`
		ReferenceId     string `json:"referenceId,omitempty"`
		Step            string `json:"step,omitempty"`
		WorldId         string `json:"worldId,omitempty"`
		ChannelId       string `json:"channelId,omitempty"`
		IncludeEquipped bool   `json:"includeEquipped,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	c.conditionType = aux.ConditionType
	c.operator = aux.Operator
	c.value = aux.Value
	c.referenceId = aux.ReferenceId
	c.step = aux.Step
	c.worldId = aux.WorldId
	c.channelId = aux.ChannelId
	c.includeEquipped = aux.IncludeEquipped
	return nil
}

// --- OutcomeModel ---

func (o OutcomeModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Conditions []ConditionModel `json:"conditions"`
		NextState  string           `json:"nextState"`
	}{o.conditions, o.nextState})
}

func (o *OutcomeModel) UnmarshalJSON(data []byte) error {
	var aux struct {
		Conditions []ConditionModel `json:"conditions"`
		NextState  string           `json:"nextState"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	o.conditions = aux.Conditions
	o.nextState = aux.NextState
	return nil
}

// --- GenericActionModel ---

func (g GenericActionModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Operations []OperationModel `json:"operations"`
		Outcomes   []OutcomeModel   `json:"outcomes"`
	}{g.operations, g.outcomes})
}

func (g *GenericActionModel) UnmarshalJSON(data []byte) error {
	var aux struct {
		Operations []OperationModel `json:"operations"`
		Outcomes   []OutcomeModel   `json:"outcomes"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	g.operations = aux.Operations
	g.outcomes = aux.Outcomes
	return nil
}

// --- CraftActionModel ---

func (c CraftActionModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ItemId                string  `json:"itemId"`
		Materials             []uint32 `json:"materials"`
		Quantities            []uint32 `json:"quantities"`
		MesoCost              uint32  `json:"mesoCost,omitempty"`
		StimulatorId          uint32  `json:"stimulatorId,omitempty"`
		StimulatorFailChance  float64 `json:"stimulatorFailChance,omitempty"`
		SuccessState          string  `json:"successState"`
		FailureState          string  `json:"failureState"`
		MissingMaterialsState string  `json:"missingMaterialsState"`
	}{c.itemId, c.materials, c.quantities, c.mesoCost, c.stimulatorId, c.stimulatorFailChance, c.successState, c.failureState, c.missingMaterialsState})
}

func (c *CraftActionModel) UnmarshalJSON(data []byte) error {
	var aux struct {
		ItemId                string  `json:"itemId"`
		Materials             []uint32 `json:"materials"`
		Quantities            []uint32 `json:"quantities"`
		MesoCost              uint32  `json:"mesoCost,omitempty"`
		StimulatorId          uint32  `json:"stimulatorId,omitempty"`
		StimulatorFailChance  float64 `json:"stimulatorFailChance,omitempty"`
		SuccessState          string  `json:"successState"`
		FailureState          string  `json:"failureState"`
		MissingMaterialsState string  `json:"missingMaterialsState"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	c.itemId = aux.ItemId
	c.materials = aux.Materials
	c.quantities = aux.Quantities
	c.mesoCost = aux.MesoCost
	c.stimulatorId = aux.StimulatorId
	c.stimulatorFailChance = aux.StimulatorFailChance
	c.successState = aux.SuccessState
	c.failureState = aux.FailureState
	c.missingMaterialsState = aux.MissingMaterialsState
	return nil
}

// --- TransportActionModel ---

func (t TransportActionModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RouteName             string `json:"routeName"`
		FailureState          string `json:"failureState"`
		CapacityFullState     string `json:"capacityFullState,omitempty"`
		AlreadyInTransitState string `json:"alreadyInTransitState,omitempty"`
		RouteNotFoundState    string `json:"routeNotFoundState,omitempty"`
		ServiceErrorState     string `json:"serviceErrorState,omitempty"`
	}{t.routeName, t.failureState, t.capacityFullState, t.alreadyInTransitState, t.routeNotFoundState, t.serviceErrorState})
}

func (t *TransportActionModel) UnmarshalJSON(data []byte) error {
	var aux struct {
		RouteName             string `json:"routeName"`
		FailureState          string `json:"failureState"`
		CapacityFullState     string `json:"capacityFullState,omitempty"`
		AlreadyInTransitState string `json:"alreadyInTransitState,omitempty"`
		RouteNotFoundState    string `json:"routeNotFoundState,omitempty"`
		ServiceErrorState     string `json:"serviceErrorState,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	t.routeName = aux.RouteName
	t.failureState = aux.FailureState
	t.capacityFullState = aux.CapacityFullState
	t.alreadyInTransitState = aux.AlreadyInTransitState
	t.routeNotFoundState = aux.RouteNotFoundState
	t.serviceErrorState = aux.ServiceErrorState
	return nil
}

// --- GachaponActionModel ---

func (g GachaponActionModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		GachaponId   string `json:"gachaponId"`
		TicketItemId uint32 `json:"ticketItemId"`
		FailureState string `json:"failureState"`
	}{g.gachaponId, g.ticketItemId, g.failureState})
}

func (g *GachaponActionModel) UnmarshalJSON(data []byte) error {
	var aux struct {
		GachaponId   string `json:"gachaponId"`
		TicketItemId uint32 `json:"ticketItemId"`
		FailureState string `json:"failureState"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	g.gachaponId = aux.GachaponId
	g.ticketItemId = aux.TicketItemId
	g.failureState = aux.FailureState
	return nil
}

// --- ListSelectionModel ---

func (l ListSelectionModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Title   string        `json:"title"`
		Choices []ChoiceModel `json:"choices"`
	}{l.title, l.choices})
}

func (l *ListSelectionModel) UnmarshalJSON(data []byte) error {
	var aux struct {
		Title   string        `json:"title"`
		Choices []ChoiceModel `json:"choices"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	l.title = aux.Title
	l.choices = aux.Choices
	return nil
}

// --- AskNumberModel ---

func (a AskNumberModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Text         string `json:"text"`
		DefaultValue uint32 `json:"defaultValue"`
		MinValue     uint32 `json:"minValue"`
		MaxValue     uint32 `json:"maxValue"`
		ContextKey   string `json:"contextKey"`
		NextState    string `json:"nextState"`
	}{a.text, a.defaultValue, a.minValue, a.maxValue, a.contextKey, a.nextState})
}

func (a *AskNumberModel) UnmarshalJSON(data []byte) error {
	var aux struct {
		Text         string `json:"text"`
		DefaultValue uint32 `json:"defaultValue"`
		MinValue     uint32 `json:"minValue"`
		MaxValue     uint32 `json:"maxValue"`
		ContextKey   string `json:"contextKey"`
		NextState    string `json:"nextState"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.text = aux.Text
	a.defaultValue = aux.DefaultValue
	a.minValue = aux.MinValue
	a.maxValue = aux.MaxValue
	a.contextKey = aux.ContextKey
	a.nextState = aux.NextState
	return nil
}

// --- AskStyleModel ---

func (a AskStyleModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Text             string   `json:"text"`
		Styles           []uint32 `json:"styles,omitempty"`
		StylesContextKey string   `json:"stylesContextKey,omitempty"`
		ContextKey       string   `json:"contextKey"`
		NextState        string   `json:"nextState"`
	}{a.text, a.styles, a.stylesContextKey, a.contextKey, a.nextState})
}

func (a *AskStyleModel) UnmarshalJSON(data []byte) error {
	var aux struct {
		Text             string   `json:"text"`
		Styles           []uint32 `json:"styles,omitempty"`
		StylesContextKey string   `json:"stylesContextKey,omitempty"`
		ContextKey       string   `json:"contextKey"`
		NextState        string   `json:"nextState"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.text = aux.Text
	a.styles = aux.Styles
	a.stylesContextKey = aux.StylesContextKey
	a.contextKey = aux.ContextKey
	a.nextState = aux.NextState
	return nil
}

// --- AskSlideMenuModel ---

func (a AskSlideMenuModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Title      string        `json:"title,omitempty"`
		MenuType   uint32        `json:"menuType"`
		ContextKey string        `json:"contextKey"`
		Choices    []ChoiceModel `json:"choices"`
	}{a.title, a.menuType, a.contextKey, a.choices})
}

func (a *AskSlideMenuModel) UnmarshalJSON(data []byte) error {
	var aux struct {
		Title      string        `json:"title,omitempty"`
		MenuType   uint32        `json:"menuType"`
		ContextKey string        `json:"contextKey"`
		Choices    []ChoiceModel `json:"choices"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.title = aux.Title
	a.menuType = aux.MenuType
	a.contextKey = aux.ContextKey
	a.choices = aux.Choices
	return nil
}

// --- StateModel ---

func (s StateModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id              string               `json:"id"`
		StateType       StateType            `json:"stateType"`
		Dialogue        *DialogueModel       `json:"dialogue,omitempty"`
		GenericAction   *GenericActionModel   `json:"genericAction,omitempty"`
		CraftAction     *CraftActionModel     `json:"craftAction,omitempty"`
		TransportAction *TransportActionModel `json:"transportAction,omitempty"`
		GachaponAction  *GachaponActionModel  `json:"gachaponAction,omitempty"`
		ListSelection   *ListSelectionModel   `json:"listSelection,omitempty"`
		AskNumber       *AskNumberModel       `json:"askNumber,omitempty"`
		AskStyle        *AskStyleModel        `json:"askStyle,omitempty"`
		AskSlideMenu    *AskSlideMenuModel    `json:"askSlideMenu,omitempty"`
	}{s.id, s.stateType, s.dialogue, s.genericAction, s.craftAction, s.transportAction, s.gachaponAction, s.listSelection, s.askNumber, s.askStyle, s.askSlideMenu})
}

func (s *StateModel) UnmarshalJSON(data []byte) error {
	var aux struct {
		Id              string               `json:"id"`
		StateType       StateType            `json:"stateType"`
		Dialogue        *DialogueModel       `json:"dialogue,omitempty"`
		GenericAction   *GenericActionModel   `json:"genericAction,omitempty"`
		CraftAction     *CraftActionModel     `json:"craftAction,omitempty"`
		TransportAction *TransportActionModel `json:"transportAction,omitempty"`
		GachaponAction  *GachaponActionModel  `json:"gachaponAction,omitempty"`
		ListSelection   *ListSelectionModel   `json:"listSelection,omitempty"`
		AskNumber       *AskNumberModel       `json:"askNumber,omitempty"`
		AskStyle        *AskStyleModel        `json:"askStyle,omitempty"`
		AskSlideMenu    *AskSlideMenuModel    `json:"askSlideMenu,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	s.id = aux.Id
	s.stateType = aux.StateType
	s.dialogue = aux.Dialogue
	s.genericAction = aux.GenericAction
	s.craftAction = aux.CraftAction
	s.transportAction = aux.TransportAction
	s.gachaponAction = aux.GachaponAction
	s.listSelection = aux.ListSelection
	s.askNumber = aux.AskNumber
	s.askStyle = aux.AskStyle
	s.askSlideMenu = aux.AskSlideMenu
	return nil
}

// --- ConversationContext ---

func (c ConversationContext) MarshalJSON() ([]byte, error) {
	var convData *conversationDataJSON
	if c.conversation != nil {
		cd := conversationDataJSON{StartState: c.conversation.StartState()}
		if hs, ok := c.conversation.(interface{ States() []StateModel }); ok {
			cd.States = hs.States()
		}
		convData = &cd
	}

	return json.Marshal(struct {
		Field            field.Model           `json:"field"`
		CharacterId      uint32                `json:"characterId"`
		NpcId            uint32                `json:"npcId"`
		CurrentState     string                `json:"currentState"`
		Conversation     *conversationDataJSON `json:"conversation,omitempty"`
		Context          map[string]string     `json:"context,omitempty"`
		PendingSagaId    *uuid.UUID            `json:"pendingSagaId,omitempty"`
		ConversationType ConversationType      `json:"conversationType"`
		SourceId         uint32                `json:"sourceId"`
	}{c.field, c.characterId, c.npcId, c.currentState, convData, c.context, c.pendingSagaId, c.conversationType, c.sourceId})
}

func (c *ConversationContext) UnmarshalJSON(data []byte) error {
	var aux struct {
		Field            field.Model           `json:"field"`
		CharacterId      uint32                `json:"characterId"`
		NpcId            uint32                `json:"npcId"`
		CurrentState     string                `json:"currentState"`
		Conversation     *conversationDataJSON `json:"conversation,omitempty"`
		Context          map[string]string     `json:"context,omitempty"`
		PendingSagaId    *uuid.UUID            `json:"pendingSagaId,omitempty"`
		ConversationType ConversationType      `json:"conversationType"`
		SourceId         uint32                `json:"sourceId"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	c.field = aux.Field
	c.characterId = aux.CharacterId
	c.npcId = aux.NpcId
	c.currentState = aux.CurrentState
	c.context = aux.Context
	c.pendingSagaId = aux.PendingSagaId
	c.conversationType = aux.ConversationType
	c.sourceId = aux.SourceId
	if aux.Conversation != nil {
		c.conversation = storedConversation{
			startState: aux.Conversation.StartState,
			states:     aux.Conversation.States,
		}
	}
	return nil
}
