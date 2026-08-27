package character_test

import (
	"atlas-character/character"
	"testing"
)

// TestBuilder_Build_MissingAccountId asserts the identity invariant Build()
// enforces: a Builder with no accountId set fails rather than silently
// producing a zero-value character.
func TestBuilder_Build_MissingAccountId(t *testing.T) {
	cfg := character.NewBuilderConfiguration(false, false, 24)
	_, err := character.NewBuilder(cfg, 0, 1, "TestChar", 0, 0, 30000, 20000).Build()
	if err == nil {
		t.Fatal("Build() error = nil, want error for missing accountId")
	}
}

// TestBuilder_Build_MissingName asserts the identity invariant Build()
// enforces: a Builder with an empty name fails rather than silently
// producing a zero-value character.
func TestBuilder_Build_MissingName(t *testing.T) {
	cfg := character.NewBuilderConfiguration(false, false, 24)
	_, err := character.NewBuilder(cfg, 100, 1, "", 0, 0, 30000, 20000).Build()
	if err == nil {
		t.Fatal("Build() error = nil, want error for missing name")
	}
}

// TestBuilder_Build_WithIdentity asserts Build() succeeds once the identity
// invariant is satisfied.
func TestBuilder_Build_WithIdentity(t *testing.T) {
	cfg := character.NewBuilderConfiguration(false, false, 24)
	m, err := character.NewBuilder(cfg, 100, 1, "TestChar", 0, 0, 30000, 20000).Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if m.AccountId() != 100 {
		t.Errorf("m.AccountId() = %d, want 100", m.AccountId())
	}
	if m.Name() != "TestChar" {
		t.Errorf("m.Name() = %s, want TestChar", m.Name())
	}
}
