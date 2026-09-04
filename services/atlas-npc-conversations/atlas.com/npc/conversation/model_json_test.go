package conversation

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

// TestRPSActionState_JSONRoundTrip verifies that a StateModel built with an
// rpsAction state survives a MarshalJSON/UnmarshalJSON round trip, mirroring
// the gachaponAction envelope wiring (npcId, entryCostMeso, failureState).
func TestRPSActionState_JSONRoundTrip(t *testing.T) {
	rpsAction, err := NewRPSActionBuilder().
		SetNpcId(9000019).
		SetEntryCostMeso(1000).
		SetFailureState("noMeso").
		Build()
	if err != nil {
		t.Fatalf("build rpsAction: %v", err)
	}

	state, err := NewStateBuilder().
		SetId("playRPS").
		SetRPSAction(rpsAction).
		Build()
	if err != nil {
		t.Fatalf("build state: %v", err)
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}

	// Confirm the envelope carries the "rpsAction" key (not silently dropped).
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, ok := raw["rpsAction"]; !ok {
		t.Fatalf("marshaled state envelope missing \"rpsAction\" key: %s", data)
	}

	var roundTripped StateModel
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}

	if roundTripped.Type() != RPSActionType {
		t.Errorf("Type() = %q, want %q", roundTripped.Type(), RPSActionType)
	}
	if roundTripped.Id() != "playRPS" {
		t.Errorf("Id() = %q, want %q", roundTripped.Id(), "playRPS")
	}

	got := roundTripped.RPSAction()
	if got == nil {
		t.Fatalf("RPSAction() = nil after round trip")
	}
	if got.NpcId() != 9000019 {
		t.Errorf("NpcId() = %d, want %d", got.NpcId(), 9000019)
	}
	if got.EntryCostMeso() != 1000 {
		t.Errorf("EntryCostMeso() = %d, want %d", got.EntryCostMeso(), 1000)
	}
	if got.FailureState() != "noMeso" {
		t.Errorf("FailureState() = %q, want %q", got.FailureState(), "noMeso")
	}
}

// originTransactionId must survive the Redis round-trip. The registry stores
// ConversationContext as JSON; a field missing from the marshal pair is
// silently dropped, and the redelivery guard in StartItem would never fire.
func TestConversationContextOriginTransactionIdSurvivesJSON(t *testing.T) {
	txn := uuid.New()
	in := NewConversationContextBuilder().
		SetCharacterId(42).
		SetNpcId(9010000).
		SetCurrentState("intro").
		SetConversationType(ItemConversationType).
		SetSourceId(2430013).
		SetOriginTransactionId(txn).
		Build()

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out ConversationContext
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.OriginTransactionId() == nil {
		t.Fatal("originTransactionId lost in JSON round-trip")
	}
	if *out.OriginTransactionId() != txn {
		t.Errorf("originTransactionId: got %s want %s", *out.OriginTransactionId(), txn)
	}
	if out.ConversationType() != ItemConversationType {
		t.Errorf("conversationType: got %q want %q", out.ConversationType(), ItemConversationType)
	}
	if out.SourceId() != 2430013 {
		t.Errorf("sourceId: got %d want 2430013", out.SourceId())
	}
}

const askTextBaseJSON = `"text":"The door reacts to the entry pass inserted. #bPassword#k!","defaultText":"","minLength":1,"maxLength":32,"contextKey":"answer","nextState":"wrong-password"`

func unmarshalAskText(t *testing.T, data string) AskTextModel {
	t.Helper()
	var m AskTextModel
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return m
}

// TestAskTextJSONRoundTrip verifies the AskTextModel/AskTextMatchModel JSONB
// codec, including the ordered, first-match-wins semantics of matches.
func TestAskTextJSONRoundTrip(t *testing.T) {
	t.Run("no matches", func(t *testing.T) {
		original := unmarshalAskText(t, `{`+askTextBaseJSON+`}`)

		marshalled, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(marshalled, &raw); err != nil {
			t.Fatalf("unmarshal raw: %v", err)
		}
		if _, ok := raw["matches"]; ok {
			t.Errorf("expected matches to be omitted from marshalled JSON, got present")
		}

		roundTripped := unmarshalAskText(t, string(marshalled))
		if !reflect.DeepEqual(original, roundTripped) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", roundTripped, original)
		}
		if roundTripped.Matches() != nil {
			t.Errorf("expected Matches() to be nil, got %#v", roundTripped.Matches())
		}
	})

	t.Run("empty matches", func(t *testing.T) {
		original := unmarshalAskText(t, `{`+askTextBaseJSON+`,"matches":[]}`)

		marshalled, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(marshalled, &raw); err != nil {
			t.Fatalf("unmarshal raw: %v", err)
		}
		if _, ok := raw["matches"]; ok {
			t.Errorf("expected empty matches to be omitted from marshalled JSON (omitempty), got present")
		}

		roundTripped := unmarshalAskText(t, string(marshalled))

		// The chosen canonical representation for "no matches present on the
		// wire" is nil, not an empty-but-non-nil slice. json.Unmarshal of a
		// literal `[]` produces a non-nil empty slice, but since omitempty
		// drops it on marshal, the round trip settles on nil; compare
		// against a nil-matches variant of the original rather than the
		// freshly-parsed `[]` value so nil-vs-[] never registers as a
		// mismatch.
		want := original
		want.matches = nil
		if !reflect.DeepEqual(want, roundTripped) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", roundTripped, want)
		}
		if roundTripped.Matches() != nil {
			t.Errorf("expected Matches() to be nil, got %#v", roundTripped.Matches())
		}
	})

	t.Run("literal match", func(t *testing.T) {
		original := unmarshalAskText(t, `{`+askTextBaseJSON+`,"matches":[{"value":"Open Sesame","nextState":"open"}]}`)

		marshalled, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		roundTripped := unmarshalAskText(t, string(marshalled))

		if !reflect.DeepEqual(original, roundTripped) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", roundTripped, original)
		}
		if len(roundTripped.Matches()) != 1 {
			t.Fatalf("expected 1 match, got %d", len(roundTripped.Matches()))
		}
		got := roundTripped.Matches()[0]
		if got.Value() != "Open Sesame" || got.ValueFromContext() != "" || got.NextState() != "open" {
			t.Errorf("unexpected match: %#v", got)
		}
	})

	t.Run("context match", func(t *testing.T) {
		original := unmarshalAskText(t, `{`+askTextBaseJSON+`,"matches":[{"valueFromContext":"{context.magatiaPassword}","nextState":"open"}]}`)

		marshalled, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		roundTripped := unmarshalAskText(t, string(marshalled))

		if !reflect.DeepEqual(original, roundTripped) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", roundTripped, original)
		}
		if len(roundTripped.Matches()) != 1 {
			t.Fatalf("expected 1 match, got %d", len(roundTripped.Matches()))
		}
		got := roundTripped.Matches()[0]
		if got.Value() != "" || got.ValueFromContext() != "{context.magatiaPassword}" || got.NextState() != "open" {
			t.Errorf("unexpected match: %#v", got)
		}
	})

	t.Run("order preserved", func(t *testing.T) {
		original := unmarshalAskText(t, `{`+askTextBaseJSON+`,"matches":[{"value":"a","nextState":"sa"},{"value":"b","nextState":"sb"},{"value":"c","nextState":"sc"}]}`)

		marshalled, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		roundTripped := unmarshalAskText(t, string(marshalled))

		if !reflect.DeepEqual(original, roundTripped) {
			t.Errorf("round trip mismatch:\n got  = %#v\n want = %#v", roundTripped, original)
		}

		matches := roundTripped.Matches()
		if len(matches) != 3 {
			t.Fatalf("expected 3 matches, got %d", len(matches))
		}
		wantValues := []string{"a", "b", "c"}
		wantNextStates := []string{"sa", "sb", "sc"}
		for i := range wantValues {
			if matches[i].Value() != wantValues[i] || matches[i].NextState() != wantNextStates[i] {
				t.Errorf("match[%d] = %#v, want value=%q nextState=%q", i, matches[i], wantValues[i], wantNextStates[i])
			}
		}
	})
}

// TestStateModelJSONRoundTripAskText verifies the StateModel envelope carries
// askText for the AskTextType state, and that every other state model
// pointer stays nil after unmarshal.
func TestStateModelJSONRoundTripAskText(t *testing.T) {
	askText := unmarshalAskText(t, `{`+askTextBaseJSON+`,"matches":[{"value":"Open Sesame","nextState":"open"}]}`)

	original := StateModel{
		id:        "state-1",
		stateType: AskTextType,
		askText:   &askText,
	}

	marshalled, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(marshalled, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, ok := raw["askText"]; !ok {
		t.Errorf("expected askText to be present in marshalled envelope")
	}

	var roundTripped StateModel
	if err := json.Unmarshal(marshalled, &roundTripped); err != nil {
		t.Fatalf("unmarshal round trip: %v", err)
	}

	if roundTripped.stateType != AskTextType {
		t.Errorf("stateType = %q, want %q", roundTripped.stateType, AskTextType)
	}
	if roundTripped.askText == nil {
		t.Fatalf("askText is nil")
	}
	if !reflect.DeepEqual(*roundTripped.askText, askText) {
		t.Errorf("askText mismatch:\n got  = %#v\n want = %#v", *roundTripped.askText, askText)
	}

	if roundTripped.dialogue != nil {
		t.Errorf("dialogue should be nil, got %#v", roundTripped.dialogue)
	}
	if roundTripped.genericAction != nil {
		t.Errorf("genericAction should be nil, got %#v", roundTripped.genericAction)
	}
	if roundTripped.craftAction != nil {
		t.Errorf("craftAction should be nil, got %#v", roundTripped.craftAction)
	}
	if roundTripped.transportAction != nil {
		t.Errorf("transportAction should be nil, got %#v", roundTripped.transportAction)
	}
	if roundTripped.gachaponAction != nil {
		t.Errorf("gachaponAction should be nil, got %#v", roundTripped.gachaponAction)
	}
	if roundTripped.rpsAction != nil {
		t.Errorf("rpsAction should be nil, got %#v", roundTripped.rpsAction)
	}
	if roundTripped.partyQuestAction != nil {
		t.Errorf("partyQuestAction should be nil, got %#v", roundTripped.partyQuestAction)
	}
	if roundTripped.partyQuestBonusAction != nil {
		t.Errorf("partyQuestBonusAction should be nil, got %#v", roundTripped.partyQuestBonusAction)
	}
	if roundTripped.listSelection != nil {
		t.Errorf("listSelection should be nil, got %#v", roundTripped.listSelection)
	}
	if roundTripped.askNumber != nil {
		t.Errorf("askNumber should be nil, got %#v", roundTripped.askNumber)
	}
	if roundTripped.askStyle != nil {
		t.Errorf("askStyle should be nil, got %#v", roundTripped.askStyle)
	}
	if roundTripped.askSlideMenu != nil {
		t.Errorf("askSlideMenu should be nil, got %#v", roundTripped.askSlideMenu)
	}
}
