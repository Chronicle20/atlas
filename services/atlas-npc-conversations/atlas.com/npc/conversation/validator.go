package conversation

import (
	"fmt"
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	StateId   string
	Field     string
	ErrorType string
	Message   string
}

// ValidationResult holds the results of conversation validation
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// Validator validates conversation structures
type Validator struct{}

// NewValidator creates a new conversation validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateNpc validates an NPC conversation model
func (v *Validator) ValidateNpc(m NpcConversation) ValidationResult {
	result := ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	// Validate basic requirements
	if m.NpcId() == 0 {
		result.addError("", "npcId", "required", "NPC ID is required")
	}

	if m.StartState() == "" {
		result.addError("", "startState", "required", "Start state is required")
	}

	if len(m.States()) == 0 {
		result.addError("", "states", "required", "At least one state is required")
		return result
	}

	// Build state ID map for reference checking
	stateIds := make(map[string]bool)
	for _, state := range m.States() {
		if state.Id() == "" {
			result.addError("", "id", "required", "State ID is required")
		} else {
			if stateIds[state.Id()] {
				result.addError(state.Id(), "id", "duplicate", fmt.Sprintf("Duplicate state ID: %s", state.Id()))
			}
			stateIds[state.Id()] = true
		}
	}

	// Validate start state exists
	if m.StartState() != "" && !stateIds[m.StartState()] {
		result.addError("", "startState", "invalid_reference", fmt.Sprintf("Start state '%s' does not exist", m.StartState()))
	}

	// Validate each state
	for _, state := range m.States() {
		v.validateState(state, stateIds, &result)
	}

	// Detect unreachable states
	reachable := v.findReachableStates(m)
	for stateId := range stateIds {
		if !reachable[stateId] && stateId != m.StartState() {
			result.addError(stateId, "reachability", "unreachable", fmt.Sprintf("State '%s' is unreachable", stateId))
		}
	}

	// Detect circular references (infinite loops without exit)
	v.detectCircularReferences(m, &result)

	return result
}

// validateState validates a single state
func (v *Validator) validateState(state StateModel, stateIds map[string]bool, result *ValidationResult) {
	switch state.Type() {
	case DialogueStateType:
		v.validateDialogue(state.Id(), state.Dialogue(), stateIds, result)
	case GenericActionType:
		v.validateGenericAction(state.Id(), state.GenericAction(), stateIds, result)
	case CraftActionType:
		v.validateCraftAction(state.Id(), state.CraftAction(), stateIds, result)
	case TransportActionType:
		v.validateTransportAction(state.Id(), state.TransportAction(), stateIds, result)
	case GachaponActionType:
		v.validateGachaponAction(state.Id(), state.GachaponAction(), stateIds, result)
	case PartyQuestActionType:
		v.validatePartyQuestAction(state.Id(), state.PartyQuestAction(), stateIds, result)
	case ListSelectionType:
		v.validateListSelection(state.Id(), state.ListSelection(), stateIds, result)
	case AskNumberType:
		v.validateAskNumber(state.Id(), state.AskNumber(), stateIds, result)
	case AskStyleType:
		v.validateAskStyle(state.Id(), state.AskStyle(), stateIds, result)
	case AskSlideMenuType:
		v.validateAskSlideMenu(state.Id(), state.AskSlideMenu(), stateIds, result)
	default:
		result.addError(state.Id(), "type", "invalid", fmt.Sprintf("Invalid state type: %s", state.Type()))
	}
}

// validateDialogue validates a dialogue state
func (v *Validator) validateDialogue(stateId string, dialogue *DialogueModel, stateIds map[string]bool, result *ValidationResult) {
	if dialogue == nil {
		result.addError(stateId, "dialogue", "required", "Dialogue is required for dialogue state")
		return
	}

	if dialogue.Text() == "" {
		result.addError(stateId, "dialogue.text", "required", "Dialogue text is required")
	}

	if len(dialogue.Choices()) == 0 {
		result.addError(stateId, "dialogue.choices", "required", "At least one choice is required")
	}

	// Validate choice count based on dialogue type
	choiceCount := len(dialogue.Choices())
	switch dialogue.DialogueType() {
	case SendOk, SendNext, SendPrev:
		if choiceCount != 2 {
			result.addError(stateId, "dialogue.choices", "invalid_count", fmt.Sprintf("%s requires exactly 2 choices", dialogue.DialogueType()))
		}
	case SendYesNo, SendNextPrev:
		if choiceCount != 3 {
			result.addError(stateId, "dialogue.choices", "invalid_count", fmt.Sprintf("%s requires exactly 3 choices", dialogue.DialogueType()))
		}
	}

	// Validate each choice
	for i, choice := range dialogue.Choices() {
		if choice.Text() == "" {
			result.addError(stateId, fmt.Sprintf("dialogue.choices[%d].text", i), "required", "Choice text is required")
		}

		// Validate nextState reference (null/empty is valid for ending conversation)
		if choice.NextState() != "" && !stateIds[choice.NextState()] {
			result.addError(stateId, fmt.Sprintf("dialogue.choices[%d].nextState", i), "invalid_reference", fmt.Sprintf("Next state '%s' does not exist", choice.NextState()))
		}
	}
}

// validateGenericAction validates a generic action state
func (v *Validator) validateGenericAction(stateId string, action *GenericActionModel, stateIds map[string]bool, result *ValidationResult) {
	if action == nil {
		result.addError(stateId, "genericAction", "required", "Generic action is required for genericAction state")
		return
	}

	if len(action.Operations()) == 0 && len(action.Outcomes()) == 0 {
		result.addError(stateId, "genericAction", "required", "At least one operation or outcome is required")
	}

	// Validate operations
	for i, op := range action.Operations() {
		if op.Type() == "" {
			result.addError(stateId, fmt.Sprintf("genericAction.operations[%d].type", i), "required", "Operation type is required")
		}
		// Note: We could add more detailed operation parameter validation here
	}

	// Validate outcomes
	for i, outcome := range action.Outcomes() {
		if outcome.NextState() == "" {
			result.addError(stateId, fmt.Sprintf("genericAction.outcomes[%d].nextState", i), "required", "Next state is required for outcome")
		} else if !stateIds[outcome.NextState()] {
			result.addError(stateId, fmt.Sprintf("genericAction.outcomes[%d].nextState", i), "invalid_reference", fmt.Sprintf("Next state '%s' does not exist", outcome.NextState()))
		}

		// Validate conditions
		for j, cond := range outcome.Conditions() {
			if cond.Type() == "" {
				result.addError(stateId, fmt.Sprintf("genericAction.outcomes[%d].conditions[%d].type", i, j), "required", "Condition type is required")
			}
			if cond.Operator() == "" {
				result.addError(stateId, fmt.Sprintf("genericAction.outcomes[%d].conditions[%d].operator", i, j), "required", "Condition operator is required")
			}
			if cond.Value() == "" {
				result.addError(stateId, fmt.Sprintf("genericAction.outcomes[%d].conditions[%d].value", i, j), "required", "Condition value is required")
			}
		}
	}
}

// validateCraftAction validates a craft action state
func (v *Validator) validateCraftAction(stateId string, action *CraftActionModel, stateIds map[string]bool, result *ValidationResult) {
	if action == nil {
		result.addError(stateId, "craftAction", "required", "Craft action is required for craftAction state")
		return
	}

	if action.ItemId() == "" {
		result.addError(stateId, "craftAction.itemId", "required", "Item ID is required")
	}

	if len(action.Materials()) == 0 {
		result.addError(stateId, "craftAction.materials", "required", "At least one material is required")
	}

	if len(action.Materials()) != len(action.Quantities()) {
		result.addError(stateId, "craftAction.quantities", "invalid", "Quantities must match materials count")
	}

	// Validate state references
	if action.SuccessState() == "" {
		result.addError(stateId, "craftAction.successState", "required", "Success state is required")
	} else if !stateIds[action.SuccessState()] {
		result.addError(stateId, "craftAction.successState", "invalid_reference", fmt.Sprintf("Success state '%s' does not exist", action.SuccessState()))
	}

	if action.FailureState() == "" {
		result.addError(stateId, "craftAction.failureState", "required", "Failure state is required")
	} else if !stateIds[action.FailureState()] {
		result.addError(stateId, "craftAction.failureState", "invalid_reference", fmt.Sprintf("Failure state '%s' does not exist", action.FailureState()))
	}

	if action.MissingMaterialsState() == "" {
		result.addError(stateId, "craftAction.missingMaterialsState", "required", "Missing materials state is required")
	} else if !stateIds[action.MissingMaterialsState()] {
		result.addError(stateId, "craftAction.missingMaterialsState", "invalid_reference", fmt.Sprintf("Missing materials state '%s' does not exist", action.MissingMaterialsState()))
	}
}

// validateTransportAction validates a transport action state
func (v *Validator) validateTransportAction(stateId string, action *TransportActionModel, stateIds map[string]bool, result *ValidationResult) {
	if action == nil {
		result.addError(stateId, "transportAction", "required", "Transport action is required for transportAction state")
		return
	}

	if action.RouteName() == "" {
		result.addError(stateId, "transportAction.routeName", "required", "Route name is required")
	}

	// Validate failure state reference (required)
	if action.FailureState() == "" {
		result.addError(stateId, "transportAction.failureState", "required", "Failure state is required")
	} else if !stateIds[action.FailureState()] {
		result.addError(stateId, "transportAction.failureState", "invalid_reference", fmt.Sprintf("Failure state '%s' does not exist", action.FailureState()))
	}

	// Validate optional state references (only if specified)
	if action.CapacityFullState() != "" && !stateIds[action.CapacityFullState()] {
		result.addError(stateId, "transportAction.capacityFullState", "invalid_reference", fmt.Sprintf("Capacity full state '%s' does not exist", action.CapacityFullState()))
	}

	if action.AlreadyInTransitState() != "" && !stateIds[action.AlreadyInTransitState()] {
		result.addError(stateId, "transportAction.alreadyInTransitState", "invalid_reference", fmt.Sprintf("Already in transit state '%s' does not exist", action.AlreadyInTransitState()))
	}

	if action.RouteNotFoundState() != "" && !stateIds[action.RouteNotFoundState()] {
		result.addError(stateId, "transportAction.routeNotFoundState", "invalid_reference", fmt.Sprintf("Route not found state '%s' does not exist", action.RouteNotFoundState()))
	}

	if action.ServiceErrorState() != "" && !stateIds[action.ServiceErrorState()] {
		result.addError(stateId, "transportAction.serviceErrorState", "invalid_reference", fmt.Sprintf("Service error state '%s' does not exist", action.ServiceErrorState()))
	}
}

// validateGachaponAction validates a gachapon action state
func (v *Validator) validateGachaponAction(stateId string, action *GachaponActionModel, stateIds map[string]bool, result *ValidationResult) {
	if action == nil {
		result.addError(stateId, "gachaponAction", "required", "Gachapon action is required for gachaponAction state")
		return
	}

	if action.GachaponId() == "" {
		result.addError(stateId, "gachaponAction.gachaponId", "required", "Gachapon ID is required")
	}

	if action.TicketItemId() == 0 {
		result.addError(stateId, "gachaponAction.ticketItemId", "required", "Ticket item ID is required")
	}

	// Validate failure state reference (required)
	if action.FailureState() == "" {
		result.addError(stateId, "gachaponAction.failureState", "required", "Failure state is required")
	} else if !stateIds[action.FailureState()] {
		result.addError(stateId, "gachaponAction.failureState", "invalid_reference", fmt.Sprintf("Failure state '%s' does not exist", action.FailureState()))
	}
}

// validatePartyQuestAction validates a party quest action state
func (v *Validator) validatePartyQuestAction(stateId string, action *PartyQuestActionModel, stateIds map[string]bool, result *ValidationResult) {
	if action == nil {
		result.addError(stateId, "partyQuestAction", "required", "Party quest action is required for partyQuestAction state")
		return
	}

	if action.QuestId() == "" {
		result.addError(stateId, "partyQuestAction.questId", "required", "Quest ID is required")
	}

	// Validate failure state reference (required)
	if action.FailureState() == "" {
		result.addError(stateId, "partyQuestAction.failureState", "required", "Failure state is required")
	} else if !stateIds[action.FailureState()] {
		result.addError(stateId, "partyQuestAction.failureState", "invalid_reference", fmt.Sprintf("Failure state '%s' does not exist", action.FailureState()))
	}

	// Validate optional state references (only if specified)
	if action.NotInPartyState() != "" && !stateIds[action.NotInPartyState()] {
		result.addError(stateId, "partyQuestAction.notInPartyState", "invalid_reference", fmt.Sprintf("Not in party state '%s' does not exist", action.NotInPartyState()))
	}

	if action.NotLeaderState() != "" && !stateIds[action.NotLeaderState()] {
		result.addError(stateId, "partyQuestAction.notLeaderState", "invalid_reference", fmt.Sprintf("Not leader state '%s' does not exist", action.NotLeaderState()))
	}
}

// validateListSelection validates a list selection state
func (v *Validator) validateListSelection(stateId string, listSelection *ListSelectionModel, stateIds map[string]bool, result *ValidationResult) {
	if listSelection == nil {
		result.addError(stateId, "listSelection", "required", "List selection is required for listSelection state")
		return
	}

	if listSelection.Title() == "" {
		result.addError(stateId, "listSelection.title", "required", "List selection title is required")
	}

	if len(listSelection.Choices()) == 0 {
		result.addError(stateId, "listSelection.choices", "required", "At least one choice is required")
	}

	// Validate each choice
	for i, choice := range listSelection.Choices() {
		if choice.Text() == "" {
			result.addError(stateId, fmt.Sprintf("listSelection.choices[%d].text", i), "required", "Choice text is required")
		}

		// Validate nextState reference (null/empty is valid for ending conversation)
		if choice.NextState() != "" && !stateIds[choice.NextState()] {
			result.addError(stateId, fmt.Sprintf("listSelection.choices[%d].nextState", i), "invalid_reference", fmt.Sprintf("Next state '%s' does not exist", choice.NextState()))
		}
	}
}

// validateAskNumber validates an ask number state
func (v *Validator) validateAskNumber(stateId string, askNumber *AskNumberModel, stateIds map[string]bool, result *ValidationResult) {
	if askNumber == nil {
		result.addError(stateId, "askNumber", "required", "Ask number is required for askNumber state")
		return
	}

	if askNumber.Text() == "" {
		result.addError(stateId, "askNumber.text", "required", "Ask number text is required")
	}

	if askNumber.MaxValue() == 0 {
		result.addError(stateId, "askNumber.maxValue", "required", "Max value must be greater than 0")
	}

	if askNumber.MinValue() > askNumber.DefaultValue() {
		result.addError(stateId, "askNumber.defaultValue", "invalid", "Default value must be >= min value")
	}

	if askNumber.DefaultValue() > askNumber.MaxValue() {
		result.addError(stateId, "askNumber.defaultValue", "invalid", "Default value must be <= max value")
	}

	// Validate nextState reference
	if askNumber.NextState() != "" && !stateIds[askNumber.NextState()] {
		result.addError(stateId, "askNumber.nextState", "invalid_reference", fmt.Sprintf("Next state '%s' does not exist", askNumber.NextState()))
	}
}

// validateAskStyle validates an ask style state
func (v *Validator) validateAskStyle(stateId string, askStyle *AskStyleModel, stateIds map[string]bool, result *ValidationResult) {
	if askStyle == nil {
		result.addError(stateId, "askStyle", "required", "Ask style is required for askStyle state")
		return
	}

	if askStyle.Text() == "" {
		result.addError(stateId, "askStyle.text", "required", "Ask style text is required")
	}

	// Validate that either styles OR stylesContextKey is provided
	hasStyles := len(askStyle.Styles()) > 0
	hasStylesContextKey := askStyle.StylesContextKey() != ""

	if !hasStyles && !hasStylesContextKey {
		result.addError(stateId, "askStyle", "required", "Either styles or stylesContextKey is required")
	}

	// Validate nextState reference
	if askStyle.NextState() != "" && !stateIds[askStyle.NextState()] {
		result.addError(stateId, "askStyle.nextState", "invalid_reference", fmt.Sprintf("Next state '%s' does not exist", askStyle.NextState()))
	}
}

// validateAskSlideMenu validates an ask slide menu state
func (v *Validator) validateAskSlideMenu(stateId string, askSlideMenu *AskSlideMenuModel, stateIds map[string]bool, result *ValidationResult) {
	if askSlideMenu == nil {
		result.addError(stateId, "askSlideMenu", "required", "Ask slide menu is required for askSlideMenu state")
		return
	}

	// Title is optional for slide menus (e.g., dimensional mirror style)

	if len(askSlideMenu.Choices()) == 0 {
		result.addError(stateId, "askSlideMenu.choices", "required", "At least one choice is required")
	}

	// Validate each choice
	for i, choice := range askSlideMenu.Choices() {
		if choice.Text() == "" {
			result.addError(stateId, fmt.Sprintf("askSlideMenu.choices[%d].text", i), "required", "Choice text is required")
		}

		// Validate nextState reference (null/empty is valid for ending conversation)
		if choice.NextState() != "" && !stateIds[choice.NextState()] {
			result.addError(stateId, fmt.Sprintf("askSlideMenu.choices[%d].nextState", i), "invalid_reference", fmt.Sprintf("Next state '%s' does not exist", choice.NextState()))
		}
	}
}

// findReachableStates performs a graph traversal to find all reachable states
func (v *Validator) findReachableStates(m NpcConversation) map[string]bool {
	reachable := make(map[string]bool)
	visited := make(map[string]bool)

	var visit func(stateId string)
	visit = func(stateId string) {
		if visited[stateId] || stateId == "" {
			return
		}
		visited[stateId] = true
		reachable[stateId] = true

		// Find state
		state, err := m.FindState(stateId)
		if err != nil {
			return
		}

		// Visit all next states
		switch state.Type() {
		case DialogueStateType:
			if dialogue := state.Dialogue(); dialogue != nil {
				for _, choice := range dialogue.Choices() {
					visit(choice.NextState())
				}
			}
		case GenericActionType:
			if action := state.GenericAction(); action != nil {
				for _, outcome := range action.Outcomes() {
					visit(outcome.NextState())
				}
			}
		case CraftActionType:
			if action := state.CraftAction(); action != nil {
				visit(action.SuccessState())
				visit(action.FailureState())
				visit(action.MissingMaterialsState())
			}
		case TransportActionType:
			if action := state.TransportAction(); action != nil {
				// Transport actions only have failure states (success = player warped)
				visit(action.FailureState())
				visit(action.CapacityFullState())
				visit(action.AlreadyInTransitState())
				visit(action.RouteNotFoundState())
				visit(action.ServiceErrorState())
			}
		case GachaponActionType:
			if action := state.GachaponAction(); action != nil {
				visit(action.FailureState())
			}
		case PartyQuestActionType:
			if action := state.PartyQuestAction(); action != nil {
				visit(action.FailureState())
				visit(action.NotInPartyState())
				visit(action.NotLeaderState())
			}
		case ListSelectionType:
			if listSelection := state.ListSelection(); listSelection != nil {
				for _, choice := range listSelection.Choices() {
					visit(choice.NextState())
				}
			}
		case AskNumberType:
			if askNumber := state.AskNumber(); askNumber != nil {
				visit(askNumber.NextState())
			}
		case AskStyleType:
			if askStyle := state.AskStyle(); askStyle != nil {
				visit(askStyle.NextState())
			}
		case AskSlideMenuType:
			if askSlideMenu := state.AskSlideMenu(); askSlideMenu != nil {
				for _, choice := range askSlideMenu.Choices() {
					visit(choice.NextState())
				}
			}
		}
	}

	visit(m.StartState())
	return reachable
}

// detectCircularReferences detects circular references (infinite loops)
func (v *Validator) detectCircularReferences(m NpcConversation, result *ValidationResult) {
	// Build adjacency list
	graph := make(map[string][]string)

	for _, state := range m.States() {
		nextStates := v.getNextStates(state)
		graph[state.Id()] = nextStates
	}

	// Check if there's a path from start that loops without exit
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []string{}

	var detectCycle func(stateId string) bool
	detectCycle = func(stateId string) bool {
		if stateId == "" {
			// Empty state means conversation ends - no cycle
			return false
		}

		visited[stateId] = true
		recStack[stateId] = true
		path = append(path, stateId)

		for _, nextState := range graph[stateId] {
			if nextState == "" {
				// This path has an exit
				continue
			}

			if !visited[nextState] {
				if detectCycle(nextState) {
					return true
				}
			} else if recStack[nextState] {
				// Found a cycle
				cycleStart := -1
				for i, s := range path {
					if s == nextState {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cyclePath := append(path[cycleStart:], nextState)
					// Check if this cycle has any exits
					if v.cycleHasNoExit(cyclePath, graph) {
						result.addError(nextState, "circular_reference", "infinite_loop", fmt.Sprintf("Infinite loop detected: %s", strings.Join(cyclePath, " -> ")))
					}
				}
				return true
			}
		}

		path = path[:len(path)-1]
		recStack[stateId] = false
		return false
	}

	detectCycle(m.StartState())
}

// cycleHasNoExit checks if a cycle has at least one exit point
func (v *Validator) cycleHasNoExit(cycle []string, graph map[string][]string) bool {
	cycleMap := make(map[string]bool)
	for _, stateId := range cycle {
		cycleMap[stateId] = true
	}

	// Check if any state in the cycle has an edge to outside the cycle or to empty (end)
	for _, stateId := range cycle {
		for _, nextState := range graph[stateId] {
			if nextState == "" || !cycleMap[nextState] {
				return false // Has an exit
			}
		}
	}

	return true // No exit found
}

// getNextStates returns all next states for a given state
func (v *Validator) getNextStates(state StateModel) []string {
	var nextStates []string

	switch state.Type() {
	case DialogueStateType:
		if dialogue := state.Dialogue(); dialogue != nil {
			for _, choice := range dialogue.Choices() {
				nextStates = append(nextStates, choice.NextState())
			}
		}
	case GenericActionType:
		if action := state.GenericAction(); action != nil {
			for _, outcome := range action.Outcomes() {
				nextStates = append(nextStates, outcome.NextState())
			}
		}
	case CraftActionType:
		if action := state.CraftAction(); action != nil {
			nextStates = append(nextStates, action.SuccessState(), action.FailureState(), action.MissingMaterialsState())
		}
	case TransportActionType:
		if action := state.TransportAction(); action != nil {
			// Transport actions only have failure states (success = player warped)
			nextStates = append(nextStates, action.FailureState(), action.CapacityFullState(), action.AlreadyInTransitState(), action.RouteNotFoundState(), action.ServiceErrorState())
		}
	case GachaponActionType:
		if action := state.GachaponAction(); action != nil {
			nextStates = append(nextStates, action.FailureState())
		}
	case PartyQuestActionType:
		if action := state.PartyQuestAction(); action != nil {
			nextStates = append(nextStates, action.FailureState(), action.NotInPartyState(), action.NotLeaderState())
		}
	case ListSelectionType:
		if listSelection := state.ListSelection(); listSelection != nil {
			for _, choice := range listSelection.Choices() {
				nextStates = append(nextStates, choice.NextState())
			}
		}
	case AskNumberType:
		if askNumber := state.AskNumber(); askNumber != nil {
			nextStates = append(nextStates, askNumber.NextState())
		}
	case AskStyleType:
		if askStyle := state.AskStyle(); askStyle != nil {
			nextStates = append(nextStates, askStyle.NextState())
		}
	case AskSlideMenuType:
		if askSlideMenu := state.AskSlideMenu(); askSlideMenu != nil {
			for _, choice := range askSlideMenu.Choices() {
				nextStates = append(nextStates, choice.NextState())
			}
		}
	}

	return nextStates
}

// addError adds a validation error to the result
func (r *ValidationResult) addError(stateId, field, errorType, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{
		StateId:   stateId,
		Field:     field,
		ErrorType: errorType,
		Message:   message,
	})
}
