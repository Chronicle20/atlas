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

func TestResolve_envWhitespaceOnly_returnsVerbatim(t *testing.T) {
	t.Setenv("KAFKA_CONSUMER_GROUP", "   ")
	// design §5.4 decision: do NOT trim. Whitespace-only is a config bug,
	// but we keep verbatim to avoid silently masking it.
	if got := Resolve("Character Service"); got != "   " {
		t.Fatalf("Resolve = %q, want verbatim whitespace", got)
	}
}

func TestResolve_envWithFormat_substitutes(t *testing.T) {
	// PR-env case: patch generator emits "Channel Service - %s [a1b2]";
	// caller passes the per-channel id as varargs.
	t.Setenv("KAFKA_CONSUMER_GROUP", "Channel Service - %s [a1b2]")
	got := Resolve("Channel Service - %s", "ch-7")
	want := "Channel Service - ch-7 [a1b2]"
	if got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

func TestResolve_defaultWithFormat_substitutes(t *testing.T) {
	// Production case: env unset, default carries the %s.
	t.Setenv("KAFKA_CONSUMER_GROUP", "")
	got := Resolve("Channel Service - %s", "ch-7")
	want := "Channel Service - ch-7"
	if got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

func TestResolve_zeroArgs_doesNotFormat(t *testing.T) {
	// Existing zero-args callers (atlas-account etc.) must keep working
	// even if some future env value happens to contain "%s".
	t.Setenv("KAFKA_CONSUMER_GROUP", "%s literal")
	if got := Resolve("Account Service"); got != "%s literal" {
		t.Fatalf("Resolve = %q, want %q (no formatting when zero args)", got, "%s literal")
	}
}
