package consumergroup

import (
	"testing"
)

func TestResolve_envUnset_returnsDefault(t *testing.T) {
	t.Setenv("KAFKA_CONSUMER_GROUP", "")
	if got := Resolve("Character Service"); got != "Character Service" {
		t.Fatalf("Resolve = %q, want %q", got, "Character Service")
	}
}

func TestResolve_envSet_returnsEnvValue(t *testing.T) {
	t.Setenv("KAFKA_CONSUMER_GROUP", "Character Service [a3f7]")
	if got := Resolve("Character Service"); got != "Character Service [a3f7]" {
		t.Fatalf("Resolve = %q, want %q", got, "Character Service [a3f7]")
	}
}

func TestResolve_envEmptyAfterTrim_returnsDefault(t *testing.T) {
	t.Setenv("KAFKA_CONSUMER_GROUP", "   ")
	// design §5.4 decision: do NOT trim. Whitespace-only is a config bug,
	// but we keep verbatim to avoid silently masking it.
	if got := Resolve("Character Service"); got != "   " {
		t.Fatalf("Resolve = %q, want verbatim whitespace", got)
	}
}
