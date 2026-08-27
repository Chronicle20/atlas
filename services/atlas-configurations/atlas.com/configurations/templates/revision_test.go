package templates

import (
	"atlas-configurations/templates/socket"
	"atlas-configurations/templates/socket/handler"
	"testing"
)

// FR-2.2 / design §5: RestModel.Id carries json:"-", but Revision clears it
// explicitly rather than relying on the tag. This test is what keeps that
// stronger guarantee meaningful instead of tautological.
func TestRevisionIgnoresId(t *testing.T) {
	a := createTestRestModel("GMS", 83, 1)
	b := createTestRestModel("GMS", 83, 1)
	b.Id = "11111111-1111-1111-1111-111111111111"

	ra, err := Revision(a)
	if err != nil {
		t.Fatalf("Revision(a): %v", err)
	}
	rb, err := Revision(b)
	if err != nil {
		t.Fatalf("Revision(b): %v", err)
	}
	if ra != rb {
		t.Errorf("revision changed with Id set: %q != %q", ra, rb)
	}
}

// The revision must be blind to the nil-vs-empty distinction Normalize erases,
// because Make normalizes on read and Create normalizes on write. A revision
// that saw the difference would report drift on every template whose stored
// document omits a socket collection.
func TestRevisionNormalizesSocket(t *testing.T) {
	withNil := createTestRestModel("GMS", 83, 1)
	withNil.Socket = socket.RestModel{}

	withEmpty := createTestRestModel("GMS", 83, 1)
	withEmpty.Socket = socket.RestModel{
		Handlers: []handler.RestModel{},
	}

	rn, err := Revision(withNil)
	if err != nil {
		t.Fatalf("Revision(withNil): %v", err)
	}
	re, err := Revision(withEmpty)
	if err != nil {
		t.Fatalf("Revision(withEmpty): %v", err)
	}
	if rn != re {
		t.Errorf("nil and empty socket collections produced different revisions: %q != %q", rn, re)
	}
}

// A revision is a lowercase hex SHA-256: 64 characters, stable across calls.
func TestRevisionIsStableLowercaseHex(t *testing.T) {
	m := createTestRestModel("GMS", 83, 1)
	first, err := Revision(m)
	if err != nil {
		t.Fatalf("Revision: %v", err)
	}
	second, err := Revision(m)
	if err != nil {
		t.Fatalf("Revision (second call): %v", err)
	}
	if first != second {
		t.Errorf("Revision is not deterministic: %q != %q", first, second)
	}
	if len(first) != 64 {
		t.Errorf("revision length = %d, want 64: %q", len(first), first)
	}
	for _, c := range first {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Fatalf("revision is not lowercase hex: %q", first)
		}
	}
}

// Environment is server-owned (set from Entity.Environment by Make, never
// present in a shipped seed file) and must not enter the hash. Without this,
// any deployment with a non-empty ATLAS_ENVIRONMENT (atlas-main, pr-*) would
// see SeedDrift = true for every template, permanently.
func TestRevisionIgnoresEnvironment(t *testing.T) {
	base := createTestRestModel("GMS", 83, 1)

	for _, env := range []string{"", "main", "pr-123"} {
		m := base
		m.Environment = env
		r, err := Revision(m)
		if err != nil {
			t.Fatalf("Revision(environment=%q): %v", env, err)
		}
		want, err := Revision(base)
		if err != nil {
			t.Fatalf("Revision(base): %v", err)
		}
		if r != want {
			t.Errorf("Revision(environment=%q) = %q, want %q (same as base)", env, r, want)
		}
	}
}

// Two templates that differ in content must not collide.
func TestRevisionDiffersOnContentChange(t *testing.T) {
	a := createTestRestModel("GMS", 83, 1)
	b := createTestRestModel("GMS", 83, 1)
	b.UsesPin = !a.UsesPin

	ra, err := Revision(a)
	if err != nil {
		t.Fatalf("Revision(a): %v", err)
	}
	rb, err := Revision(b)
	if err != nil {
		t.Fatalf("Revision(b): %v", err)
	}
	if ra == rb {
		t.Errorf("differing models produced the same revision: %q", ra)
	}
}
