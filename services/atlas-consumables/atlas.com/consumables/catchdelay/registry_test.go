package catchdelay

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestAllow_WindowRejectsThenAdmits — the first attempt is admitted and arms the
// window, a second inside the window is rejected, and once the TTL lapses the
// item is usable again. This is what makes BRIDLE_MOB_CATCH_FAIL reason 1 (the
// item's delayMsg) reachable at all.
func TestAllow_WindowRejectsThenAdmits(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	InitRegistry(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}))

	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	r := GetRegistry()

	ok, err := r.Allow(ctx, 42, 2270008, 3*time.Second)
	if err != nil || !ok {
		t.Fatalf("first attempt: ok=%t err=%v, want admitted", ok, err)
	}

	ok, err = r.Allow(ctx, 42, 2270008, 3*time.Second)
	if err != nil || ok {
		t.Fatalf("second attempt inside the window: ok=%t err=%v, want rejected", ok, err)
	}

	// A different item is a different window.
	ok, err = r.Allow(ctx, 42, 2270002, 3*time.Second)
	if err != nil || !ok {
		t.Fatalf("a different item: ok=%t err=%v, want admitted", ok, err)
	}

	mr.FastForward(4 * time.Second)
	ok, err = r.Allow(ctx, 42, 2270008, 3*time.Second)
	if err != nil || !ok {
		t.Fatalf("after the window lapsed: ok=%t err=%v, want admitted", ok, err)
	}
}

// TestAllow_ZeroDelayAlwaysAdmits — most catch items carry no useDelay.
func TestAllow_ZeroDelayAlwaysAdmits(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	InitRegistry(goredis.NewClient(&goredis.Options{Addr: mr.Addr()}))

	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)

	for i := 0; i < 3; i++ {
		ok, err := GetRegistry().Allow(ctx, 42, 2270000, 0)
		if err != nil || !ok {
			t.Fatalf("attempt %d: ok=%t err=%v, want admitted", i, ok, err)
		}
	}
}
