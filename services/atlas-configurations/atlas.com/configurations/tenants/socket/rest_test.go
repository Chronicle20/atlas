package socket

import (
	"atlas-configurations/tenants/socket/handler"
	"atlas-configurations/tenants/socket/writer"
	"encoding/json"
	"strings"
	"testing"
)

// An absent "unsupported" key must decode to a struct with two EMPTY (not nil)
// slices after Normalize, and must marshal back as real arrays. Both PRD
// acceptance criteria - "loads with both lists empty" and "carries an empty
// unsupported object" - come from this one invariant.
func TestNormalize_AbsentUnsupportedBecomesEmptyArrays(t *testing.T) {
	const in = `{"handlers":[],"writers":[]}`
	var rm RestModel
	if err := json.Unmarshal([]byte(in), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	rm = Normalize(rm)

	if rm.Unsupported.Handlers == nil {
		t.Error("Normalize left Unsupported.Handlers nil")
	}
	if rm.Unsupported.Writers == nil {
		t.Error("Normalize left Unsupported.Writers nil")
	}

	out, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, `"unsupported":{"handlers":[],"writers":[]}`) {
		t.Errorf("marshalled output missing empty unsupported arrays:\n%s", got)
	}
}

func TestNormalize_PreservesPopulatedUnsupported(t *testing.T) {
	rm := RestModel{
		Unsupported: UnsupportedRestModel{
			Handlers: []string{"GuestLoginHandle"},
			Writers:  []string{"MonsterCarnival"},
		},
	}
	rm = Normalize(rm)
	if len(rm.Unsupported.Handlers) != 1 || rm.Unsupported.Handlers[0] != "GuestLoginHandle" {
		t.Errorf("Normalize mangled Unsupported.Handlers: %+v", rm.Unsupported.Handlers)
	}
	if len(rm.Unsupported.Writers) != 1 || rm.Unsupported.Writers[0] != "MonsterCarnival" {
		t.Errorf("Normalize mangled Unsupported.Writers: %+v", rm.Unsupported.Writers)
	}
	if rm.Handlers == nil || rm.Writers == nil {
		t.Error("Normalize left Handlers/Writers nil")
	}
}

func TestRestModel_FNameRoundTrips(t *testing.T) {
	const in = `{"handlers":[{"opCode":"0x01","validator":"NoOpValidator","handler":"LoginHandle","fname":"CLogin::SendCheckPasswordPacket","services":["login"]}],"writers":[]}`
	var rm RestModel
	if err := json.Unmarshal([]byte(in), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := rm.Handlers[0].FName; got != "CLogin::SendCheckPasswordPacket" {
		t.Fatalf("FName = %q, want CLogin::SendCheckPasswordPacket", got)
	}
	out, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"fname":"CLogin::SendCheckPasswordPacket"`) {
		t.Errorf("fname dropped on marshal:\n%s", out)
	}
}

// fname is omitempty: an entry without one must not gain a "fname":"" key.
func TestRestModel_FNameOmittedWhenEmpty(t *testing.T) {
	rm := RestModel{
		Handlers: []handler.RestModel{{OpCode: "0x01", Validator: "NoOpValidator", Handler: "LoginHandle"}},
		Writers:  []writer.RestModel{{OpCode: "0x00", Writer: "AuthSuccess"}},
	}
	out, err := json.Marshal(Normalize(rm))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), `"fname"`) {
		t.Errorf("empty fname was emitted:\n%s", out)
	}
}

// Options is omitempty (design F7): an entry that supplied no options must not
// gain "options":null on round-trip, which would make the first save of any
// template a 200-line diff.
func TestRestModel_OptionsOmittedWhenAbsent(t *testing.T) {
	const in = `{"handlers":[{"opCode":"0x01","validator":"NoOpValidator","handler":"LoginHandle"}],"writers":[]}`
	var rm RestModel
	if err := json.Unmarshal([]byte(in), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(Normalize(rm))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), `"options"`) {
		t.Errorf("absent options round-tripped to a key:\n%s", out)
	}
}

// An explicitly-supplied empty options object is DIFFERENT from an absent one
// at the JSON level but identical semantically; it must survive as {} rather
// than being dropped, so the seed files stay byte-stable.
func TestRestModel_EmptyOptionsObjectSurvives(t *testing.T) {
	const in = `{"handlers":[],"writers":[{"opCode":"0xA5","writer":"MiniRoom","options":{},"services":["channel"]}]}`
	var rm RestModel
	if err := json.Unmarshal([]byte(in), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(Normalize(rm))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"options":{}`) {
		t.Errorf("explicit empty options object was dropped:\n%s", out)
	}
}
