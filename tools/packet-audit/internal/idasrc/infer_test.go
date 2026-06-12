package idasrc

import (
	"fmt"
	"testing"
)

func mkFields(calls ...FieldCall) Fields { return Fields{Calls: calls} }
func r(op Primitive, guard string) FieldCall { return FieldCall{Op: op, Guard: guard} }

func TestInferDispatchClearMatch(t *testing.T) {
	base := mkFields(
		r(Decode1, ""),            // discriminator
		r(Decode4, "switch == 8"), // case 8: one int
		r(Decode4, "switch == 9"), // case 9: int + str
		r(DecodeStr, "switch == 9"),
		r(Decode2, "switch == 0xA"), // case 10: one short
	)
	hand := []FieldCall{r(Decode1, ""), r(Decode4, ""), r(DecodeStr, "")} // looks like case 9 (+ pre-branch)
	disp, conf, _ := InferDispatch(base, hand)
	if len(disp) != 1 || disp[0].Case != 9 {
		t.Fatalf("inferred %+v, want [{case:9}]", disp)
	}
	if conf < 0.6 {
		t.Errorf("confidence %.2f too low for a clear match", conf)
	}
}

func TestInferDispatchAmbiguous(t *testing.T) {
	base := mkFields(
		r(Decode1, ""),
		r(Decode4, "switch == 8"), // case 8: single int
		r(Decode4, "switch == 9"), // case 9: identical single int
	)
	hand := []FieldCall{r(Decode1, ""), r(Decode4, "")}
	_, conf, cands := InferDispatch(base, hand)
	if conf > 0.6 {
		t.Errorf("two identical-shape cases should be ambiguous, conf=%.2f", conf)
	}
	if len(cands) < 2 {
		t.Errorf("ambiguous result should surface candidate cases, got %v", cands)
	}
}

func TestInferDispatchUnresolvedWildcard(t *testing.T) {
	// An Unresolved live read matches any hand read (the undecompilable GW_Friend bulk).
	base := mkFields(
		r(Decode1, ""),
		r(Decode4, "switch == 9"),
		r(Unresolved, "switch == 9"), // stands for several hand-traced fields
	)
	hand := []FieldCall{r(Decode1, ""), r(Decode4, ""), r(Decode4, ""), r(DecodeStr, "")}
	disp, _, _ := InferDispatch(base, hand)
	if len(disp) != 1 || disp[0].Case != 9 {
		t.Fatalf("inferred %+v, want case 9 (Unresolved should still best-match)", disp)
	}
}

func TestInferDispatchUnresolvedAbsorbsRun(t *testing.T) {
	base := mkFields(
		r(Decode1, ""),            // discriminator (pre-branch)
		r(Decode1, "switch == 8"), // case 8: short, different shape
		r(Decode4, "switch == 9"), // case 9: D4, Str, Unresolved(GW_Friend bulk), trailing D1
		r(DecodeStr, "switch == 9"),
		r(Unresolved, "switch == 9"),
		r(Decode1, "switch == 9"),
	)
	// hand #Invite: GW_Friend expanded into several fields between name and the trailing flag.
	hand := []FieldCall{
		r(Decode1, ""), r(Decode4, ""), r(DecodeStr, ""),
		r(Decode4, ""), r(DecodeBuf, ""), r(Decode1, ""), r(Decode4, ""), // absorbed by the Unresolved
		r(Decode1, ""), // trailing flag re-anchors
	}
	disp, conf, _ := InferDispatch(base, hand)
	if len(disp) != 1 || disp[0].Case != 9 {
		t.Fatalf("inferred %+v, want case 9 (Unresolved must absorb the GW_Friend run)", disp)
	}
	if conf < 0.6 {
		t.Errorf("confidence %.2f too low — case 9 should decisively win once Unresolved absorbs the run", conf)
	}
}

// A trailing Unresolved (last live read) absorbs ALL remaining hand fields.
func TestInferDispatchTrailingUnresolvedAbsorbsRest(t *testing.T) {
	base := mkFields(
		r(Decode1, ""),
		r(Decode2, "switch == 1"),                              // case 1: short
		r(Decode4, "switch == 2"), r(Unresolved, "switch == 2"), // case 2: D4 then undecompilable rest
	)
	hand := []FieldCall{r(Decode1, ""), r(Decode4, ""), r(Decode4, ""), r(DecodeStr, ""), r(Decode1, "")}
	disp, _, _ := InferDispatch(base, hand)
	if len(disp) != 1 || disp[0].Case != 2 {
		t.Fatalf("inferred %+v, want case 2 (trailing Unresolved absorbs the rest)", disp)
	}
}

func TestInferDispatchJointResolvesConflict(t *testing.T) {
	base := mkFields(
		r(Decode1, ""),
		r(Decode1, "switch == 8"),                               // case 8 (Update-like): short
		r(Decode4, "switch == 9"), r(DecodeStr, "switch == 9"),
		r(Unresolved, "switch == 9"), r(Decode1, "switch == 9"), // case 9 (Invite-like): w/ Unresolved
	)
	entries := []EntryShape{
		{FName: "Foo#Invite", Hand: []FieldCall{r(Decode1, ""), r(Decode4, ""), r(DecodeStr, ""), r(Decode4, ""), r(Decode1, ""), r(Decode1, "")}},
		{FName: "Foo#Update", Hand: []FieldCall{r(Decode1, ""), r(Decode1, "")}},
	}
	res := InferDispatchJoint(base, entries)
	by := map[string]Assignment{}
	for _, a := range res {
		by[a.FName] = a
	}
	if len(by["Foo#Invite"].Dispatch) != 1 || by["Foo#Invite"].Dispatch[0].Case != 9 {
		t.Errorf("Invite -> %+v, want case 9", by["Foo#Invite"].Dispatch)
	}
	if len(by["Foo#Update"].Dispatch) != 1 || by["Foo#Update"].Dispatch[0].Case != 8 {
		t.Errorf("Update -> %+v, want case 8", by["Foo#Update"].Dispatch)
	}
	// one-to-one: distinct cases.
	if by["Foo#Invite"].Dispatch[0].Case == by["Foo#Update"].Dispatch[0].Case {
		t.Error("conflict: two entries assigned the same case")
	}
}

func TestInferDispatchJointDeterministic(t *testing.T) {
	base := mkFields(r(Decode1, ""), r(Decode4, "switch == 1"), r(Decode2, "switch == 2"))
	entries := []EntryShape{
		{FName: "A", Hand: []FieldCall{r(Decode1, ""), r(Decode4, "")}},
		{FName: "B", Hand: []FieldCall{r(Decode1, ""), r(Decode2, "")}},
	}
	first := InferDispatchJoint(base, entries)
	for i := 0; i < 5; i++ {
		if got := InferDispatchJoint(base, entries); fmt.Sprintf("%+v", got) != fmt.Sprintf("%+v", first) {
			t.Fatalf("non-deterministic: %+v vs %+v", got, first)
		}
	}
}

func TestInferDispatchJointConfidenceJointAware(t *testing.T) {
	// case 8 = single int (matches Update); case 9 = int+str+int (distinctively matches Invite).
	base := mkFields(
		r(Decode1, ""),
		r(Decode4, "switch == 8"),
		r(Decode4, "switch == 9"), r(DecodeStr, "switch == 9"), r(Decode4, "switch == 9"),
	)
	entries := []EntryShape{
		{FName: "X#Invite", Hand: []FieldCall{r(Decode1, ""), r(Decode4, ""), r(DecodeStr, ""), r(Decode4, "")}},
		{FName: "X#Update", Hand: []FieldCall{r(Decode1, ""), r(Decode4, "")}},
	}
	by := map[string]Assignment{}
	for _, a := range InferDispatchJoint(base, entries) {
		by[a.FName] = a
	}
	if by["X#Invite"].Dispatch[0].Case != 9 || by["X#Invite"].Confidence < 0.6 {
		t.Errorf("Invite -> %+v conf %.2f, want case 9 HIGH confidence (joint-resolved, distinctive)",
			by["X#Invite"].Dispatch, by["X#Invite"].Confidence)
	}
}

func TestInferDispatchJointShortEntryStaysAmbiguous(t *testing.T) {
	// Two cases with IDENTICAL single-read shapes; two 1-read entries. After the
	// one-to-one assignment, each entry still had an equally-good FREE alternative
	// before assignment -> low confidence (we can't tell which entry maps to which).
	base := mkFields(
		r(Decode1, ""),
		r(Decode4, "switch == 1"),
		r(Decode4, "switch == 2"),
	)
	entries := []EntryShape{
		{FName: "A", Hand: []FieldCall{r(Decode1, ""), r(Decode4, "")}},
		{FName: "B", Hand: []FieldCall{r(Decode1, ""), r(Decode4, "")}},
	}
	for _, a := range InferDispatchJoint(base, entries) {
		if a.Confidence >= 0.6 {
			t.Errorf("%s conf %.2f too high — identical-shape cases must stay ambiguous", a.FName, a.Confidence)
		}
	}
}

func TestInferDispatchJoint_VerbatimArm(t *testing.T) {
	// base: discriminator read, then a non-equality arm "v5 < 5" reading Decode2,
	// and an equality arm "v5 == 9" reading Decode4.
	base := Fields{Calls: []FieldCall{
		{Op: Decode1, Guard: ""},
		{Op: Decode2, Guard: "v5 < 5"},
		{Op: Decode4, Guard: "v5 == 9"},
	}}
	entries := []EntryShape{
		{FName: "Foo#Small", Hand: []FieldCall{{Op: Decode1}, {Op: Decode2}}}, // -> v5 < 5
		{FName: "Foo#Nine", Hand: []FieldCall{{Op: Decode1}, {Op: Decode4}}},  // -> v5 == 9
	}
	got := map[string]Selector{}
	for _, a := range InferDispatchJoint(base, entries) {
		if len(a.Dispatch) == 1 {
			got[a.FName] = a.Dispatch[0]
		}
	}
	if got["Foo#Small"].Guard != "v5 < 5" {
		t.Errorf("#Small dispatch = %+v, want Guard \"v5 < 5\"", got["Foo#Small"])
	}
	if got["Foo#Nine"].Case != 9 || got["Foo#Nine"].Discriminator != "v5" {
		t.Errorf("#Nine dispatch = %+v, want v5==9", got["Foo#Nine"])
	}
}
