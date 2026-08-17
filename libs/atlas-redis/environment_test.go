package redis

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
)

func TestEnvironmentEntityKey(t *testing.T) {
	got := environmentEntityKey("ingestrun", env.Id("pr-123"), "run-7")
	want := "atlas:ingestrun:pr-123:run-7"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEnvironmentEntityKeyLegacyEnvironmentIsUnprefixed(t *testing.T) {
	// An empty environment is the legacy value: the key must be
	// byte-identical to what namespacedKey produced before this type
	// existed, so main's existing Redis state stays addressable (NFR-7).
	got := environmentEntityKey("ingestrun", env.Id(""), "run-7")
	want := "atlas:ingestrun:run-7"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func newTestEnvironmentRegistry[V any](t *testing.T) *EnvironmentRegistry[string, V] {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewEnvironmentRegistry[string, V](client, "ingestrun", func(s string) string { return s })
}

// TestEnvironmentRegistryRoundTripDiscriminatesByEnvironment proves the whole
// point of the task: two different env.Ids for the same K never collide.
// A registry that ignored the environment argument would fail this test.
func TestEnvironmentRegistryRoundTripDiscriminatesByEnvironment(t *testing.T) {
	ctx := context.Background()
	reg := newTestEnvironmentRegistry[string](t)

	if err := reg.Put(ctx, env.Id("pr-123"), "run-7", "alpha"); err != nil {
		t.Fatalf("put pr-123: %v", err)
	}
	if err := reg.Put(ctx, env.Id("pr-456"), "run-7", "beta"); err != nil {
		t.Fatalf("put pr-456: %v", err)
	}

	got, err := reg.Get(ctx, env.Id("pr-123"), "run-7")
	if err != nil {
		t.Fatalf("get pr-123: %v", err)
	}
	if got != "alpha" {
		t.Fatalf("pr-123: got %q, want %q", got, "alpha")
	}

	got, err = reg.Get(ctx, env.Id("pr-456"), "run-7")
	if err != nil {
		t.Fatalf("get pr-456: %v", err)
	}
	if got != "beta" {
		t.Fatalf("pr-456: got %q, want %q", got, "beta")
	}

	if err := reg.Remove(ctx, env.Id("pr-123"), "run-7"); err != nil {
		t.Fatalf("remove pr-123: %v", err)
	}
	if _, err := reg.Get(ctx, env.Id("pr-123"), "run-7"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for removed pr-123 key, got %v", err)
	}
	got, err = reg.Get(ctx, env.Id("pr-456"), "run-7")
	if err != nil {
		t.Fatalf("get pr-456 after removing pr-123: %v", err)
	}
	if got != "beta" {
		t.Fatalf("pr-456 unaffected: got %q, want %q", got, "beta")
	}
}

// TestEnvironmentRegistryLegacyEmptyEnvironmentIsAddressable proves the empty
// environment id (the legacy value) still round-trips and does not collide
// with a named environment sharing the same key.
func TestEnvironmentRegistryLegacyEmptyEnvironmentIsAddressable(t *testing.T) {
	ctx := context.Background()
	reg := newTestEnvironmentRegistry[string](t)

	if err := reg.Put(ctx, env.Id(""), "run-7", "legacy"); err != nil {
		t.Fatalf("put legacy: %v", err)
	}
	if err := reg.Put(ctx, env.Id("pr-123"), "run-7", "pr"); err != nil {
		t.Fatalf("put pr-123: %v", err)
	}

	got, err := reg.Get(ctx, env.Id(""), "run-7")
	if err != nil {
		t.Fatalf("get legacy: %v", err)
	}
	if got != "legacy" {
		t.Fatalf("legacy: got %q, want %q", got, "legacy")
	}

	got, err = reg.Get(ctx, env.Id("pr-123"), "run-7")
	if err != nil {
		t.Fatalf("get pr-123: %v", err)
	}
	if got != "pr" {
		t.Fatalf("pr-123: got %q, want %q", got, "pr")
	}
}

func TestEnvironmentRegistryGetAllValuesScopesByEnvironment(t *testing.T) {
	ctx := context.Background()
	reg := newTestEnvironmentRegistry[string](t)

	if err := reg.Put(ctx, env.Id("pr-123"), "a", "one"); err != nil {
		t.Fatalf("put pr-123/a: %v", err)
	}
	if err := reg.Put(ctx, env.Id("pr-123"), "b", "two"); err != nil {
		t.Fatalf("put pr-123/b: %v", err)
	}
	if err := reg.Put(ctx, env.Id("pr-456"), "c", "three"); err != nil {
		t.Fatalf("put pr-456/c: %v", err)
	}

	values, err := reg.GetAllValues(ctx, env.Id("pr-123"))
	if err != nil {
		t.Fatalf("get all values pr-123: %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("pr-123: got %d values, want 2 (%v)", len(values), values)
	}
}
