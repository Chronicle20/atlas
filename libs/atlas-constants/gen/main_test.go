package main

import (
	"strings"
	"testing"
)

func TestEmitIdentities_Golden(t *testing.T) {
	ids, err := LoadIdentities("identities.yaml")
	if err != nil {
		t.Fatal(err)
	}
	skillGo, jobGo := EmitIdentities(ids)
	// SuperGmHide is a skill identity with canonical token 9101004
	if !strings.Contains(skillGo, "SuperGmHide Identity = 9101004") {
		t.Fatalf("missing SuperGmHide constant:\n%s", skillGo)
	}
	// Pirate is a job identity with canonical token 500
	if !strings.Contains(jobGo, "Pirate Identity = 500") {
		t.Fatalf("missing Pirate constant:\n%s", jobGo)
	}
	// token uniqueness per domain is validated
	if err := ValidateIdentityTokens(ids); err != nil {
		t.Fatalf("token collision: %v", err)
	}
}
