package env

import (
	"context"
	"testing"
)

func TestWithContextRoundTrips(t *testing.T) {
	ctx := WithContext(context.Background(), Id("pr-123"))
	got, err := FromContext(ctx)()
	if err != nil {
		t.Fatalf("FromContext: %v", err)
	}
	if got != Id("pr-123") {
		t.Fatalf("got %q, want \"pr-123\"", got)
	}
}

func TestFromContextWithNoEnvironmentIsTheLegacyValue(t *testing.T) {
	// FR-1.8: absence is not an error. A context with no environment is a
	// legacy operation and resolves to the empty id, which every registry
	// query answers with "the local deployment owns it".
	got, err := FromContext(context.Background())()
	if err != nil {
		t.Fatalf("FromContext on a bare context returned an error: %v", err)
	}
	if got != Id("") {
		t.Fatalf("got %q, want the empty id", got)
	}
}

func TestSelfReadsTheProcessEnvironment(t *testing.T) {
	t.Setenv("ATLAS_ENVIRONMENT", "pr-123")
	if got := Self(); got != Id("pr-123") {
		t.Fatalf("Self() = %q, want \"pr-123\"", got)
	}
}

func TestSelfWithNoVariableIsTheLegacyValue(t *testing.T) {
	t.Setenv("ATLAS_ENVIRONMENT", "")
	if got := Self(); got != Id("") {
		t.Fatalf("Self() = %q, want the empty id", got)
	}
}

func TestValid(t *testing.T) {
	for _, ok := range []Id{"main", "pr-123", "staging-2", ""} {
		if !Valid(ok) {
			t.Errorf("Valid(%q) = false, want true", ok)
		}
	}
	for _, bad := range []Id{"PR-123", "pr_123", "-pr", "pr-", "a", "x/y"} {
		if Valid(bad) {
			t.Errorf("Valid(%q) = true, want false", bad)
		}
	}
}
