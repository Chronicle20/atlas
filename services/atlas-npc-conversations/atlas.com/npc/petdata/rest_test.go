package petdata

import "testing"

func TestExtractPopulatesName(t *testing.T) {
	rm := RestModel{
		Id:          5000029,
		Name:        "Baby Dragon",
		ReqPetLevel: 15,
		ReqItemId:   5380000,
		Evolutions:  []EvolutionRestModel{{TemplateId: 5000030, Probability: 33}},
	}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Name() != "Baby Dragon" {
		t.Errorf("Name() = %q, want %q", m.Name(), "Baby Dragon")
	}
	if !m.IsEvolvable() {
		t.Errorf("IsEvolvable() = false, want true")
	}
}

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract for the fields Model actually stores. Evolutions is NOT a
// round-trippable field: Model.evolutions collapses the incoming
// []EvolutionRestModel down to a count (see model.go), so Transform has no
// TemplateId/Probability data to reconstruct entries from. This test asserts
// that honestly -- Transform emits a nil Evolutions rather than fabricating
// placeholder entries -- instead of asserting a full reflect.DeepEqual round
// trip that this field can never satisfy by design.
func TestTransformRoundTrip(t *testing.T) {
	rm := RestModel{
		Id:          5000029,
		Name:        "Baby Dragon",
		ReqPetLevel: 15,
		ReqItemId:   5380000,
		Evolutions:  []EvolutionRestModel{{TemplateId: 5000030, Probability: 33}},
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	rm2, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm2.Id != rm.Id {
		t.Errorf("Id mismatch. Expected %d, got %d", rm.Id, rm2.Id)
	}
	if rm2.Name != rm.Name {
		t.Errorf("Name mismatch. Expected %q, got %q", rm.Name, rm2.Name)
	}
	if rm2.ReqPetLevel != rm.ReqPetLevel {
		t.Errorf("ReqPetLevel mismatch. Expected %d, got %d", rm.ReqPetLevel, rm2.ReqPetLevel)
	}
	if rm2.ReqItemId != rm.ReqItemId {
		t.Errorf("ReqItemId mismatch. Expected %d, got %d", rm.ReqItemId, rm2.ReqItemId)
	}
	if rm2.Evolutions != nil {
		t.Errorf("expected Evolutions to be nil (Model has no evolutions collection to source from), got %+v", rm2.Evolutions)
	}

	m2, err := Extract(rm2)
	if err != nil {
		t.Fatalf("Extract (second pass) failed: %v", err)
	}

	if m2.Id() != m.Id() || m2.Name() != m.Name() || m2.ReqPetLevel() != m.ReqPetLevel() || m2.ReqItemId() != m.ReqItemId() {
		t.Errorf("round trip mismatch on non-evolutions fields. Expected %+v, got %+v", m, m2)
	}
	// evolutions (the count) is expected to drop to zero on the second pass:
	// Transform could not carry the original collection forward, so Extract
	// has nothing left to count.
	if m2.IsEvolvable() {
		t.Errorf("expected second-pass Model to report not evolvable once Evolutions data is lost, got IsEvolvable()=true")
	}
}
