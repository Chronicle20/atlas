package conversation

import (
	"errors"
	"strconv"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// StateContainer is an interface for state machine containers
// Both NPC conversations (npc.Model) and quest state machines (quest.StateMachine) implement this
type StateContainer interface {
	StartState() string
	FindState(stateId string) (StateModel, error)
}

// NpcConversation extends StateContainer with NPC-specific fields
// This is implemented by npc.Model
type NpcConversation interface {
	StateContainer
	NpcId() uint32
	States() []StateModel
}

// NpcConversationProvider is an interface for NPC conversation data access
// This is implemented by npc.Processor to break the import cycle
type NpcConversationProvider interface {
	ByNpcIdProvider(npcId uint32) func() (NpcConversation, error)
}

// StateType represents the type of a conversation state
type StateType string

const (
	DialogueStateType         StateType = "dialogue"
	GenericActionType         StateType = "genericAction"
	CraftActionType           StateType = "craftAction"
	TransportActionType       StateType = "transportAction"
	GachaponActionType        StateType = "gachaponAction"
	RPSActionType             StateType = "rpsAction"
	PartyQuestActionType      StateType = "partyQuestAction"
	PartyQuestBonusActionType StateType = "partyQuestBonusAction"
	ListSelectionType         StateType = "listSelection"
	AskNumberType             StateType = "askNumber"
	AskStyleType              StateType = "askStyle"
	AskSlideMenuType          StateType = "askSlideMenu"
)

// StateModel represents a state in a conversation
type StateModel struct {
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
	askStyle              *AskStyleModel
	askSlideMenu          *AskSlideMenuModel
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

// TransportAction returns the transport action model (if type is transportAction)
func (s StateModel) TransportAction() *TransportActionModel {
	return s.transportAction
}

// GachaponAction returns the gachapon action model (if type is gachaponAction)
func (s StateModel) GachaponAction() *GachaponActionModel {
	return s.gachaponAction
}

// RPSAction returns the RPS action model (if type is rpsAction)
func (s StateModel) RPSAction() *RPSActionModel {
	return s.rpsAction
}

// PartyQuestAction returns the party quest action model (if type is partyQuestAction)
func (s StateModel) PartyQuestAction() *PartyQuestActionModel {
	return s.partyQuestAction
}

// PartyQuestBonusAction returns the party quest bonus action model (if type is partyQuestBonusAction)
func (s StateModel) PartyQuestBonusAction() *PartyQuestBonusActionModel {
	return s.partyQuestBonusAction
}

// ListSelection returns the list selection model (if type is listSelection)
func (s StateModel) ListSelection() *ListSelectionModel {
	return s.listSelection
}

// AskNumber returns the ask number model (if type is askNumber)
func (s StateModel) AskNumber() *AskNumberModel {
	return s.askNumber
}

// AskStyle returns the ask style model (if type is askStyle)
func (s StateModel) AskStyle() *AskStyleModel {
	return s.askStyle
}

// AskSlideMenu returns the ask slide menu model (if type is askSlideMenu)
func (s StateModel) AskSlideMenu() *AskSlideMenuModel {
	return s.askSlideMenu
}

// DialogueType represents the type of dialogue
type DialogueType string

const (
	SendOk            DialogueType = "sendOk"
	SendYesNo         DialogueType = "sendYesNo"
	SendAcceptDecline DialogueType = "sendAcceptDecline"
	SendNext          DialogueType = "sendNext"
	SendNextPrev      DialogueType = "sendNextPrev"
	SendPrev          DialogueType = "sendPrev"
)

// DialogueModel represents a dialogue state
type DialogueModel struct {
	dialogueType   DialogueType
	text           string
	speaker        string // Speaker type: "NPC" or "CHARACTER"
	endChat        bool   // Whether to show the end chat button (default: true)
	secondaryNpcId uint32 // Optional secondary NPC template ID for dual-NPC dialogues
	choices        []ChoiceModel
}

// DialogueType returns the dialogue type
func (d DialogueModel) DialogueType() DialogueType {
	return d.dialogueType
}

// Text returns the dialogue text
func (d DialogueModel) Text() string {
	return d.text
}

// Speaker returns the speaker type ("NPC" or "CHARACTER")
func (d DialogueModel) Speaker() string {
	return d.speaker
}

// EndChat returns whether to show the end chat button
func (d DialogueModel) EndChat() bool {
	return d.endChat
}

// SecondaryNpcId returns the secondary NPC template ID (0 means none)
func (d DialogueModel) SecondaryNpcId() uint32 {
	return d.secondaryNpcId
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
	} else if d.dialogueType == SendNextPrev {
		if action == 255 {
			choiceText = "Exit"
		} else if action == 0 {
			choiceText = "Previous"
		} else {
			choiceText = "Next"
		}
	} else if d.dialogueType == SendPrev {
		if action == 255 {
			choiceText = "Exit"
		} else if action == 0 {
			choiceText = "Previous"
		} else {
			choiceText = "Ok"
		}
	} else if d.dialogueType == SendOk {
		if action == 255 {
			choiceText = "Exit"
		} else {
			choiceText = "Ok"
		}
	} else if d.dialogueType == SendYesNo {
		if action == 255 {
			choiceText = "Exit"
		} else if action == 0 {
			choiceText = "No"
		} else {
			choiceText = "Yes"
		}
	} else if d.dialogueType == SendAcceptDecline {
		if action == 255 {
			choiceText = "Exit"
		} else if action == 0 {
			choiceText = "Decline"
		} else {
			choiceText = "Accept"
		}
	}

	for _, choice := range d.choices {
		if choice.Text() == choiceText {
			return choice, true
		}
	}
	return ChoiceModel{}, false
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

// ConditionModel represents a condition in the conversation domain
type ConditionModel struct {
	conditionType   string
	operator        string
	value           string
	referenceId     string // String from JSON, will be converted to uint32 when needed
	step            string
	worldId         string // String from JSON, will be resolved from context for mapCapacity
	channelId       string // String from JSON, will be resolved from context for mapCapacity
	includeEquipped bool   // For item conditions: also check equipped items
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

// ReferenceId returns the reference ID as uint32
// Note: This method does NOT support context references. Use ReferenceIdRaw() for that.
func (c ConditionModel) ReferenceId() uint32 {
	if c.referenceId == "" {
		return 0
	}
	// Convert string to uint32
	id, err := strconv.ParseUint(c.referenceId, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(id)
}

// ReferenceIdRaw returns the raw reference ID string (may contain context reference like {context.itemId})
func (c ConditionModel) ReferenceIdRaw() string {
	return c.referenceId
}

// Step returns the step for quest progress
func (c ConditionModel) Step() string {
	return c.step
}

// WorldId returns the worldId (as string, may contain context reference)
func (c ConditionModel) WorldId() string {
	return c.worldId
}

// ChannelId returns the channelId (as string, may contain context reference)
func (c ConditionModel) ChannelId() string {
	return c.channelId
}

// IncludeEquipped returns whether to include equipped items in item condition checks
func (c ConditionModel) IncludeEquipped() bool {
	return c.includeEquipped
}

// OutcomeModel represents an outcome in a generic action
type OutcomeModel struct {
	conditions []ConditionModel
	nextState  string
}

// Conditions returns the outcome condition
func (o OutcomeModel) Conditions() []ConditionModel {
	return o.conditions
}

// NextState returns the next state ID
func (o OutcomeModel) NextState() string {
	return o.nextState
}

// CraftActionModel represents a craft action state
type CraftActionModel struct {
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

// ItemId returns the item ID
func (c CraftActionModel) ItemId() string {
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

// NewCraftActionModelDirect constructs a CraftActionModel without any
// validation. Intended exclusively for testing scenarios that need to exercise
// processor-level defensive checks (e.g. materials/quantities mismatch) that
// cannot be produced through the validated CraftActionBuilder.
func NewCraftActionModelDirect(itemId string, materials, quantities []uint32, mesoCost uint32, stimulatorId uint32, stimulatorFailChance float64, successState, failureState, missingMaterialsState string) *CraftActionModel {
	return &CraftActionModel{
		itemId:                itemId,
		materials:             materials,
		quantities:            quantities,
		mesoCost:              mesoCost,
		stimulatorId:          stimulatorId,
		stimulatorFailChance:  stimulatorFailChance,
		successState:          successState,
		failureState:          failureState,
		missingMaterialsState: missingMaterialsState,
	}
}

// TransportActionModel represents a transport action state
// Used for instance-based transports that go through saga-orchestrator
type TransportActionModel struct {
	routeName             string // Route name to resolve to UUID at runtime
	failureState          string // General failure state (fallback)
	capacityFullState     string // State when transport is at capacity
	alreadyInTransitState string // State when character is already in a transport
	routeNotFoundState    string // State when route doesn't exist
	serviceErrorState     string // State when transport service fails
}

// RouteName returns the transport route name
func (t TransportActionModel) RouteName() string {
	return t.routeName
}

// FailureState returns the general failure state ID
func (t TransportActionModel) FailureState() string {
	return t.failureState
}

// CapacityFullState returns the capacity full failure state ID
func (t TransportActionModel) CapacityFullState() string {
	return t.capacityFullState
}

// AlreadyInTransitState returns the already in transit failure state ID
func (t TransportActionModel) AlreadyInTransitState() string {
	return t.alreadyInTransitState
}

// RouteNotFoundState returns the route not found failure state ID
func (t TransportActionModel) RouteNotFoundState() string {
	return t.routeNotFoundState
}

// ServiceErrorState returns the service error failure state ID
func (t TransportActionModel) ServiceErrorState() string {
	return t.serviceErrorState
}

// GachaponActionModel represents a gachapon action state
type GachaponActionModel struct {
	gachaponId   string // Gachapon machine ID (e.g., "henesys")
	ticketItemId uint32 // Ticket item ID to consume
	failureState string // General failure state
}

func (g GachaponActionModel) GachaponId() string {
	return g.gachaponId
}

func (g GachaponActionModel) TicketItemId() uint32 {
	return g.ticketItemId
}

func (g GachaponActionModel) FailureState() string {
	return g.failureState
}

// RPSActionModel represents an RPS (Rock Paper Scissors) entry action state
type RPSActionModel struct {
	npcId         uint32 // Entry NPC template ID (e.g., 9000019)
	entryCostMeso uint32 // Meso cost deducted to enter the game
	failureState  string // General failure state (e.g., not enough meso)
}

func (r RPSActionModel) NpcId() uint32 {
	return r.npcId
}

func (r RPSActionModel) EntryCostMeso() uint32 {
	return r.entryCostMeso
}

func (r RPSActionModel) FailureState() string {
	return r.failureState
}

// PartyQuestActionModel represents a party quest registration action state
type PartyQuestActionModel struct {
	questId         string // Party quest definition ID (e.g., "henesys_pq")
	failureState    string // General failure state (fallback)
	notInPartyState string // State when character has no party
	notLeaderState  string // State when character isn't party leader
}

func (p PartyQuestActionModel) QuestId() string {
	return p.questId
}

func (p PartyQuestActionModel) FailureState() string {
	return p.failureState
}

func (p PartyQuestActionModel) NotInPartyState() string {
	return p.notInPartyState
}

func (p PartyQuestActionModel) NotLeaderState() string {
	return p.notLeaderState
}

// PartyQuestBonusActionModel represents a party quest bonus entry action state
type PartyQuestBonusActionModel struct {
	failureState string // Failure state (fallback)
}

func (p PartyQuestBonusActionModel) FailureState() string {
	return p.failureState
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

// AskNumberModel represents an ask number state
type AskNumberModel struct {
	text         string
	defaultValue uint32
	minValue     uint32
	maxValue     uint32
	contextKey   string
	nextState    string
}

// Text returns the ask number text
func (a AskNumberModel) Text() string {
	return a.text
}

// DefaultValue returns the default value
func (a AskNumberModel) DefaultValue() uint32 {
	return a.defaultValue
}

// MinValue returns the minimum value
func (a AskNumberModel) MinValue() uint32 {
	return a.minValue
}

// MaxValue returns the maximum value
func (a AskNumberModel) MaxValue() uint32 {
	return a.maxValue
}

// ContextKey returns the context key
func (a AskNumberModel) ContextKey() string {
	return a.contextKey
}

// NextState returns the next state ID
func (a AskNumberModel) NextState() string {
	return a.nextState
}

// AskStyleModel represents an ask style state
type AskStyleModel struct {
	text             string
	styles           []uint32
	stylesContextKey string
	contextKey       string
	nextState        string
}

// Text returns the ask style text
func (a AskStyleModel) Text() string {
	return a.text
}

// Styles returns the available style IDs
func (a AskStyleModel) Styles() []uint32 {
	return a.styles
}

// StylesContextKey returns the context key containing dynamically generated styles
func (a AskStyleModel) StylesContextKey() string {
	return a.stylesContextKey
}

// ContextKey returns the context key
func (a AskStyleModel) ContextKey() string {
	return a.contextKey
}

// NextState returns the next state ID
func (a AskStyleModel) NextState() string {
	return a.nextState
}

// AskSlideMenuModel represents an ask slide menu state
type AskSlideMenuModel struct {
	title      string
	menuType   uint32
	contextKey string
	choices    []ChoiceModel
}

// Title returns the slide menu title
func (a AskSlideMenuModel) Title() string {
	return a.title
}

// MenuType returns the menu type
func (a AskSlideMenuModel) MenuType() uint32 {
	return a.menuType
}

// ContextKey returns the context key for storing the selection
func (a AskSlideMenuModel) ContextKey() string {
	return a.contextKey
}

// Choices returns the available choices
func (a AskSlideMenuModel) Choices() []ChoiceModel {
	return a.choices
}

// ChoiceFromSelection returns the choice for a given action and selection
func (a AskSlideMenuModel) ChoiceFromSelection(action byte, selection int32) (ChoiceModel, error) {
	if action == 0 {
		for _, choice := range a.choices {
			if choice.Text() == "Exit" {
				return choice, nil
			}
		}
		return ChoiceModel{}, errors.New("invalid selection")
	}

	if selection < 0 || selection >= int32(len(a.choices)) {
		return ChoiceModel{}, errors.New("invalid selection")
	}
	return a.choices[selection], nil
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

// ConversationType represents the type of conversation (NPC or Quest)
type ConversationType string

const (
	NpcConversationType   ConversationType = "npc"
	QuestConversationType ConversationType = "quest"
	// ItemConversationType is a scripted item's own dialogue (the 243xxxx
	// family). SourceId carries the item id; NpcId carries only the avatar the
	// dialogue renders with.
	ItemConversationType ConversationType = "item"
)

// ConversationContext represents the current state of a conversation
type ConversationContext struct {
	field            field.Model
	characterId      uint32
	npcId            uint32
	currentState     string
	conversation     StateContainer
	context          map[string]string
	pendingSagaId    *uuid.UUID
	conversationType ConversationType
	sourceId         uint32
	// originTransactionId is the saga transaction that started this
	// conversation, when one did. It exists so a redelivered start command can
	// be recognised as its own: Kafka is at-least-once, and re-emitting
	// START_ERROR for a conversation this very transaction already opened would
	// fail a saga that had already succeeded.
	//
	// Deliberately NOT folded into the registry's sagaIndex, which is keyed by
	// sagas a conversation INITIATED — the opposite direction.
	originTransactionId *uuid.UUID
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

// Conversation returns the conversation state container (Model for NPC, StateMachine for Quest)
func (c ConversationContext) Conversation() StateContainer {
	return c.conversation
}

// Context returns the context map
func (c ConversationContext) Context() map[string]string {
	return c.context
}

// PendingSagaId returns the pending saga ID if one exists
func (c ConversationContext) PendingSagaId() *uuid.UUID {
	return c.pendingSagaId
}

// SetPendingSagaId sets the pending saga ID
func (c ConversationContext) SetPendingSagaId(sagaId uuid.UUID) ConversationContext {
	c.pendingSagaId = &sagaId
	return c
}

// ClearPendingSaga clears the pending saga ID
func (c ConversationContext) ClearPendingSaga() ConversationContext {
	c.pendingSagaId = nil
	return c
}

// SetCurrentState sets the current state
func (c ConversationContext) SetCurrentState(state string) ConversationContext {
	c.currentState = state
	return c
}

// ConversationType returns the conversation type (NPC or Quest)
func (c ConversationContext) ConversationType() ConversationType {
	return c.conversationType
}

// SourceId returns the source ID (NpcId or QuestId depending on Type)
func (c ConversationContext) SourceId() uint32 {
	return c.sourceId
}

// OriginTransactionId returns the saga transaction that started this
// conversation, or nil for a conversation not started by a saga.
func (c ConversationContext) OriginTransactionId() *uuid.UUID {
	return c.originTransactionId
}
