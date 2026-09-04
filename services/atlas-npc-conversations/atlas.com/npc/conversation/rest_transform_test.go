package conversation

import (
	"reflect"
	"testing"
)

// TestTransformRoundTrip verifies that every Transform<X>/Extract<X> pair in
// this package round-trips a domain value through its REST wire form without
// loss. It covers all 18 pairs declared in rest.go, including the four added
// by this task (TransformChoice, TransformOperation, TransformOutcome,
// TransformOption).
func TestTransformRoundTrip(t *testing.T) {
	t.Run("State", func(t *testing.T) {
		choice1, err := NewChoiceBuilder().SetText("Yes").SetNextState("stateA").SetContext(map[string]string{"k1": "v1"}).Build()
		if err != nil {
			t.Fatalf("build choice1: %v", err)
		}
		choice2, err := NewChoiceBuilder().SetText("No").SetNextState("stateB").SetContext(map[string]string{"k2": "v2"}).Build()
		if err != nil {
			t.Fatalf("build choice2: %v", err)
		}
		dialogue, err := NewDialogueBuilder().
			SetDialogueType(SendOk).
			SetText("Hello there").
			SetSpeaker("CHARACTER").
			SetEndChat(false).
			SetSecondaryNpcId(12345).
			AddChoice(choice1).
			AddChoice(choice2).
			Build()
		if err != nil {
			t.Fatalf("build dialogue: %v", err)
		}
		state, err := NewStateBuilder().SetId("state1").SetDialogue(dialogue).Build()
		if err != nil {
			t.Fatalf("build state: %v", err)
		}

		rest, err := TransformState(state)
		if err != nil {
			t.Fatalf("TransformState: %v", err)
		}
		got, err := ExtractState(rest)
		if err != nil {
			t.Fatalf("ExtractState: %v", err)
		}
		if !reflect.DeepEqual(got, state) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", got, state)
		}
	})

	t.Run("Dialogue", func(t *testing.T) {
		choice1, err := NewChoiceBuilder().SetText("Yes").SetNextState("stateA").SetContext(map[string]string{"k1": "v1"}).Build()
		if err != nil {
			t.Fatalf("build choice1: %v", err)
		}
		choice2, err := NewChoiceBuilder().SetText("No").SetNextState("stateB").SetContext(map[string]string{"k2": "v2"}).Build()
		if err != nil {
			t.Fatalf("build choice2: %v", err)
		}
		dialogue, err := NewDialogueBuilder().
			SetDialogueType(SendOk).
			SetText("Hello there").
			SetSpeaker("CHARACTER").
			SetEndChat(false).
			SetSecondaryNpcId(12345).
			AddChoice(choice1).
			AddChoice(choice2).
			Build()
		if err != nil {
			t.Fatalf("build dialogue: %v", err)
		}

		rest, err := TransformDialogue(*dialogue)
		if err != nil {
			t.Fatalf("TransformDialogue: %v", err)
		}
		got, err := ExtractDialogue(rest)
		if err != nil {
			t.Fatalf("ExtractDialogue: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractDialogue returned nil")
		}
		if !reflect.DeepEqual(*got, *dialogue) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *dialogue)
		}
	})

	t.Run("Choice", func(t *testing.T) {
		choice, err := NewChoiceBuilder().SetText("Yes").SetNextState("stateA").SetContext(map[string]string{"k1": "v1"}).Build()
		if err != nil {
			t.Fatalf("build choice: %v", err)
		}

		rest, err := TransformChoice(choice)
		if err != nil {
			t.Fatalf("TransformChoice: %v", err)
		}
		got, err := ExtractChoice(rest)
		if err != nil {
			t.Fatalf("ExtractChoice: %v", err)
		}
		if !reflect.DeepEqual(got, choice) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", got, choice)
		}
	})

	t.Run("GenericAction", func(t *testing.T) {
		op, err := NewOperationBuilder().SetType("giveItem").SetParams(map[string]string{"itemId": "1234"}).Build()
		if err != nil {
			t.Fatalf("build operation: %v", err)
		}
		cond, err := NewConditionBuilder().
			SetType("item").
			SetOperator("gte").
			SetValue("5").
			SetReferenceId("1000").
			SetStep("step1").
			SetIncludeEquipped(true).
			Build()
		if err != nil {
			t.Fatalf("build condition: %v", err)
		}
		outcome, err := NewOutcomeBuilder().AddCondition(cond).SetNextState("next1").Build()
		if err != nil {
			t.Fatalf("build outcome: %v", err)
		}
		ga, err := NewGenericActionBuilder().AddOperation(op).AddOutcome(outcome).Build()
		if err != nil {
			t.Fatalf("build genericAction: %v", err)
		}

		rest, err := TransformGenericAction(*ga)
		if err != nil {
			t.Fatalf("TransformGenericAction: %v", err)
		}
		got, err := ExtractGenericAction(rest)
		if err != nil {
			t.Fatalf("ExtractGenericAction: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractGenericAction returned nil")
		}
		if !reflect.DeepEqual(*got, *ga) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *ga)
		}
	})

	t.Run("Operation", func(t *testing.T) {
		op, err := NewOperationBuilder().SetType("giveItem").SetParams(map[string]string{"itemId": "1234"}).Build()
		if err != nil {
			t.Fatalf("build operation: %v", err)
		}

		rest, err := TransformOperation(op)
		if err != nil {
			t.Fatalf("TransformOperation: %v", err)
		}
		got, err := ExtractOperation(rest)
		if err != nil {
			t.Fatalf("ExtractOperation: %v", err)
		}
		if !reflect.DeepEqual(got, op) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", got, op)
		}
	})

	t.Run("Outcome", func(t *testing.T) {
		cond, err := NewConditionBuilder().
			SetType("item").
			SetOperator("gte").
			SetValue("5").
			SetReferenceId("1000").
			SetStep("step1").
			SetIncludeEquipped(true).
			Build()
		if err != nil {
			t.Fatalf("build condition: %v", err)
		}
		outcome, err := NewOutcomeBuilder().AddCondition(cond).SetNextState("next1").Build()
		if err != nil {
			t.Fatalf("build outcome: %v", err)
		}

		rest, err := TransformOutcome(outcome)
		if err != nil {
			t.Fatalf("TransformOutcome: %v", err)
		}
		got, err := ExtractOutcome(rest)
		if err != nil {
			t.Fatalf("ExtractOutcome: %v", err)
		}
		if !reflect.DeepEqual(got, outcome) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", got, outcome)
		}
	})

	t.Run("CraftAction", func(t *testing.T) {
		craft, err := NewCraftActionBuilder().
			SetItemId("sword").
			SetMaterials([]uint32{1, 2}).
			SetQuantities([]uint32{3, 4}).
			SetMesoCost(500).
			SetStimulatorId(99).
			SetStimulatorFailChance(0.25).
			SetSuccessState("succ").
			SetFailureState("fail").
			SetMissingMaterialsState("missing").
			Build()
		if err != nil {
			t.Fatalf("build craftAction: %v", err)
		}

		rest, err := TransformCraftAction(*craft)
		if err != nil {
			t.Fatalf("TransformCraftAction: %v", err)
		}
		got, err := ExtractCraftAction(rest)
		if err != nil {
			t.Fatalf("ExtractCraftAction: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractCraftAction returned nil")
		}
		if !reflect.DeepEqual(*got, *craft) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *craft)
		}
	})

	t.Run("TransportAction", func(t *testing.T) {
		ta, err := NewTransportActionBuilder().
			SetRouteName("route1").
			SetFailureState("fail").
			SetCapacityFullState("full").
			SetAlreadyInTransitState("transit").
			SetRouteNotFoundState("notfound").
			SetServiceErrorState("svcerr").
			Build()
		if err != nil {
			t.Fatalf("build transportAction: %v", err)
		}

		rest := TransformTransportAction(*ta)
		got, err := ExtractTransportAction(rest)
		if err != nil {
			t.Fatalf("ExtractTransportAction: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractTransportAction returned nil")
		}
		if !reflect.DeepEqual(*got, *ta) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *ta)
		}
	})

	t.Run("GachaponAction", func(t *testing.T) {
		gachapon, err := NewGachaponActionBuilder().
			SetGachaponId("henesys").
			SetTicketItemId(555).
			SetFailureState("fail").
			Build()
		if err != nil {
			t.Fatalf("build gachaponAction: %v", err)
		}

		rest := TransformGachaponAction(*gachapon)
		got, err := ExtractGachaponAction(rest)
		if err != nil {
			t.Fatalf("ExtractGachaponAction: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractGachaponAction returned nil")
		}
		if !reflect.DeepEqual(*got, *gachapon) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *gachapon)
		}
	})

	t.Run("RPSAction", func(t *testing.T) {
		rps, err := NewRPSActionBuilder().
			SetNpcId(9000019).
			SetEntryCostMeso(1000).
			SetFailureState("noMeso").
			Build()
		if err != nil {
			t.Fatalf("build rpsAction: %v", err)
		}

		rest := TransformRPSAction(*rps)
		got, err := ExtractRPSAction(rest)
		if err != nil {
			t.Fatalf("ExtractRPSAction: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractRPSAction returned nil")
		}
		if !reflect.DeepEqual(*got, *rps) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *rps)
		}
	})

	t.Run("PartyQuestAction", func(t *testing.T) {
		pqa, err := NewPartyQuestActionBuilder().
			SetQuestId("pq1").
			SetFailureState("fail").
			SetNotInPartyState("noParty").
			SetNotLeaderState("noLeader").
			Build()
		if err != nil {
			t.Fatalf("build partyQuestAction: %v", err)
		}

		rest := TransformPartyQuestAction(*pqa)
		got, err := ExtractPartyQuestAction(rest)
		if err != nil {
			t.Fatalf("ExtractPartyQuestAction: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractPartyQuestAction returned nil")
		}
		if !reflect.DeepEqual(*got, *pqa) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *pqa)
		}
	})

	t.Run("PartyQuestBonusAction", func(t *testing.T) {
		pqba, err := NewPartyQuestBonusActionBuilder().SetFailureState("fail").Build()
		if err != nil {
			t.Fatalf("build partyQuestBonusAction: %v", err)
		}

		rest := TransformPartyQuestBonusAction(*pqba)
		got, err := ExtractPartyQuestBonusAction(rest)
		if err != nil {
			t.Fatalf("ExtractPartyQuestBonusAction: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractPartyQuestBonusAction returned nil")
		}
		if !reflect.DeepEqual(*got, *pqba) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *pqba)
		}
	})

	t.Run("ListSelection", func(t *testing.T) {
		choice, err := NewChoiceBuilder().SetText("Opt1").SetNextState("s1").SetContext(map[string]string{"k": "v"}).Build()
		if err != nil {
			t.Fatalf("build choice: %v", err)
		}
		ls, err := NewListSelectionBuilder().SetTitle("Choose").AddChoice(choice).Build()
		if err != nil {
			t.Fatalf("build listSelection: %v", err)
		}

		rest, err := TransformListSelection(*ls)
		if err != nil {
			t.Fatalf("TransformListSelection: %v", err)
		}
		got, err := ExtractListSelection(rest)
		if err != nil {
			t.Fatalf("ExtractListSelection: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractListSelection returned nil")
		}
		if !reflect.DeepEqual(*got, *ls) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *ls)
		}
	})

	t.Run("AskNumber", func(t *testing.T) {
		an, err := NewAskNumberBuilder().
			SetText("How many?").
			SetDefaultValue(5).
			SetMinValue(1).
			SetMaxValue(10).
			SetContextKey("qty").
			SetNextState("next").
			Build()
		if err != nil {
			t.Fatalf("build askNumber: %v", err)
		}

		rest := TransformAskNumber(*an)
		got, err := ExtractAskNumber(rest)
		if err != nil {
			t.Fatalf("ExtractAskNumber: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractAskNumber returned nil")
		}
		if !reflect.DeepEqual(*got, *an) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *an)
		}
	})

	t.Run("AskText", func(t *testing.T) {
		match1, err := NewAskTextMatchBuilder().
			SetValue("Open Sesame").
			SetNextState("open").
			Build()
		if err != nil {
			t.Fatalf("build match1: %v", err)
		}
		match2, err := NewAskTextMatchBuilder().
			SetValueFromContext("{context.magatiaPassword}").
			SetNextState("open-magatia").
			Build()
		if err != nil {
			t.Fatalf("build match2: %v", err)
		}

		at, err := NewAskTextBuilder().
			SetText("The door reacts to the entry pass inserted. #bPassword#k!").
			SetDefaultText("").
			SetMinLength(1).
			SetMaxLength(32).
			SetContextKey("answer").
			SetNextState("wrong-password").
			AddMatch(*match1).
			AddMatch(*match2).
			Build()
		if err != nil {
			t.Fatalf("build askText: %v", err)
		}

		rest := TransformAskText(*at)
		got, err := ExtractAskText(rest)
		if err != nil {
			t.Fatalf("ExtractAskText: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractAskText returned nil")
		}
		if !reflect.DeepEqual(*got, *at) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *at)
		}
		if len(got.Matches()) != 2 {
			t.Fatalf("expected 2 matches, got %d", len(got.Matches()))
		}
		if got.Matches()[0].Value() != "Open Sesame" || got.Matches()[0].NextState() != "open" {
			t.Errorf("match[0] mismatch: %#v", got.Matches()[0])
		}
		if got.Matches()[1].ValueFromContext() != "{context.magatiaPassword}" || got.Matches()[1].NextState() != "open-magatia" {
			t.Errorf("match[1] mismatch: %#v", got.Matches()[1])
		}
	})

	t.Run("AskTextNoMatches", func(t *testing.T) {
		at, err := NewAskTextBuilder().
			SetText("The door reacts to the entry pass inserted. #bPassword#k!").
			SetDefaultText("").
			SetMinLength(1).
			SetMaxLength(32).
			SetContextKey("answer").
			SetNextState("wrong-password").
			Build()
		if err != nil {
			t.Fatalf("build askText: %v", err)
		}

		rest := TransformAskText(*at)
		got, err := ExtractAskText(rest)
		if err != nil {
			t.Fatalf("ExtractAskText: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractAskText returned nil")
		}
		if !reflect.DeepEqual(*got, *at) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *at)
		}
		if (got.Matches() == nil) != (at.Matches() == nil) {
			t.Errorf("nil-ness of Matches changed across round trip: got nil=%v, want nil=%v", got.Matches() == nil, at.Matches() == nil)
		}
	})

	t.Run("AskStyle", func(t *testing.T) {
		as, err := NewAskStyleBuilder().
			SetText("Pick style").
			SetStyles([]uint32{1, 2, 3}).
			SetContextKey("style").
			SetNextState("next").
			Build()
		if err != nil {
			t.Fatalf("build askStyle: %v", err)
		}

		rest := TransformAskStyle(*as)
		got, err := ExtractAskStyle(rest)
		if err != nil {
			t.Fatalf("ExtractAskStyle: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractAskStyle returned nil")
		}
		if !reflect.DeepEqual(*got, *as) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *as)
		}
	})

	t.Run("AskSlideMenu", func(t *testing.T) {
		choice, err := NewChoiceBuilder().SetText("A").SetNextState("s2").SetContext(map[string]string{"m": "n"}).Build()
		if err != nil {
			t.Fatalf("build choice: %v", err)
		}
		asm, err := NewAskSlideMenuBuilder().
			SetTitle("Menu").
			SetMenuType(2).
			SetContextKey("sel").
			AddChoice(choice).
			Build()
		if err != nil {
			t.Fatalf("build askSlideMenu: %v", err)
		}

		rest := TransformAskSlideMenu(*asm)
		got, err := ExtractAskSlideMenu(rest)
		if err != nil {
			t.Fatalf("ExtractAskSlideMenu: %v", err)
		}
		if got == nil {
			t.Fatalf("ExtractAskSlideMenu returned nil")
		}
		if !reflect.DeepEqual(*got, *asm) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", *got, *asm)
		}
	})

	t.Run("OptionSet", func(t *testing.T) {
		opt, err := NewOptionBuilder().
			SetId(1).
			SetName("Sword").
			SetMaterials([]uint32{10, 20}).
			SetQuantities([]uint32{1, 2}).
			SetMeso(100).
			Build()
		if err != nil {
			t.Fatalf("build option: %v", err)
		}
		optSet, err := NewOptionSetBuilder().SetId("set1").AddOption(opt).Build()
		if err != nil {
			t.Fatalf("build optionSet: %v", err)
		}

		rest, err := TransformOptionSet(optSet)
		if err != nil {
			t.Fatalf("TransformOptionSet: %v", err)
		}
		got, err := ExtractOptionSet(rest)
		if err != nil {
			t.Fatalf("ExtractOptionSet: %v", err)
		}
		if !reflect.DeepEqual(got, optSet) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", got, optSet)
		}
	})

	t.Run("Option", func(t *testing.T) {
		opt, err := NewOptionBuilder().
			SetId(2).
			SetName("Shield").
			SetMaterials([]uint32{30, 40}).
			SetQuantities([]uint32{3, 4}).
			SetMeso(200).
			Build()
		if err != nil {
			t.Fatalf("build option: %v", err)
		}

		rest, err := TransformOption(opt)
		if err != nil {
			t.Fatalf("TransformOption: %v", err)
		}
		got, err := ExtractOption(rest)
		if err != nil {
			t.Fatalf("ExtractOption: %v", err)
		}
		if !reflect.DeepEqual(got, opt) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", got, opt)
		}
	})
}
