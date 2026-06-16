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
