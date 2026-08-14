package shop

import (
	"atlas-channel/remotemerchant"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// fakeWriterProducer is a writer.Producer stub for tests that need to pass
// one through but never expect a writer to actually be looked up (the sweep
// tests below never let a tick fire, so no BodyFunc is ever resolved).
func fakeWriterProducer(string) (writer.BodyFunc, error) {
	return nil, errors.New("fakeWriterProducer: not implemented")
}

// waitForSweepGuard polls until the per-tenant dedup guard reaches the wanted
// state or the deadline passes, failing the test on timeout. Sweep-guard
// release happens asynchronously (inside the swept goroutine's own deferred
// cleanup after ctx cancellation), so tests must poll rather than assert
// immediately after cancel().
func waitForSweepGuard(t *testing.T, tenantId uuid.UUID, want bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		remoteMerchantSweepStartedMu.Lock()
		got := remoteMerchantSweepStarted[tenantId]
		remoteMerchantSweepStartedMu.Unlock()
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("sweep guard for tenant %s = %t, want %t (timed out)", tenantId, got, want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestStartRemoteMerchantSweep_RestartsAfterListenerKeyDrains reproduces the
// round-3 code review finding: InitHandlers runs once per (tenant, world,
// channel) listener key, each with its own cancellable ctx from
// listener.Registry.Add. Draining ONE such key (an ordinary channel
// scale-down, not tenant teardown — listener/registry.go Drain,
// configuration/projection/apply.go OpDrain) cancels only that key's ctx.
// If the key that won the per-tenant sweep-dedup claim is the one that
// drains, its sweep goroutine must release the claim so that a later
// InitHandlers call for the SAME tenant (this key restarting, or any of the
// tenant's other still-live keys) can start a replacement sweeper. Before
// this fix, remoteMerchantSweepStarted[t.Id()] stayed true forever once set,
// permanently disabling the sweep for the tenant after any single key's
// ordinary drain.
func TestStartRemoteMerchantSweep_RestartsAfterListenerKeyDrains(t *testing.T) {
	ten := mustTenant(t)

	// Clean slate: an earlier test/run may have left a claim for this
	// (randomly generated) tenant id, though in practice each mustTenant()
	// call mints a fresh uuid.
	remoteMerchantSweepStartedMu.Lock()
	delete(remoteMerchantSweepStarted, ten.Id())
	remoteMerchantSweepStartedMu.Unlock()

	tctx := tenant.WithContext(context.Background(), ten)

	// Listener key #1 (e.g. world 0 / channel 0) wins the claim.
	ctx1, cancel1 := context.WithCancel(tctx)
	startRemoteMerchantSweep(nullLogger(t), ctx1, fakeWriterProducer)

	remoteMerchantSweepStartedMu.Lock()
	started := remoteMerchantSweepStarted[ten.Id()]
	remoteMerchantSweepStartedMu.Unlock()
	if !started {
		t.Fatal("expected the first call to claim the sweep for this tenant")
	}

	// A second call for the SAME tenant, while key #1 is still live, must be
	// a no-op — this is the existing dedup guarantee, unchanged.
	ctx2, cancel2 := context.WithCancel(tctx)
	t.Cleanup(cancel2)
	startRemoteMerchantSweep(nullLogger(t), ctx2, fakeWriterProducer)

	// Key #1 drains — an ordinary channel scale-down, not tenant teardown.
	cancel1()

	// The claim must be released once key #1's goroutine actually exits.
	waitForSweepGuard(t, ten.Id(), false)

	// A later call for the same tenant (key #1 restarting after scale-up, or
	// one of the tenant's other still-live keys reconciling) must be able to
	// claim and start a fresh sweeper — this is exactly what was broken.
	ctx3, cancel3 := context.WithCancel(tctx)
	t.Cleanup(cancel3)
	startRemoteMerchantSweep(nullLogger(t), ctx3, fakeWriterProducer)

	remoteMerchantSweepStartedMu.Lock()
	restarted := remoteMerchantSweepStarted[ten.Id()]
	remoteMerchantSweepStartedMu.Unlock()
	if !restarted {
		t.Fatal("sweep did not restart for the same tenant after the winning listener key drained")
	}
}

func mustTenant(t *testing.T) tenant.Model {
	t.Helper()
	m, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return m
}

func nullLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

func TestUnlockPending_HitUnlocksAndClears(t *testing.T) {
	ten := mustTenant(t)
	remotemerchant.GetRegistry().Put(ten, 1234, remotemerchant.Entry{
		ItemId: item.Id(5450000), Slot: slot.Position(3), At: time.Now(),
	})
	t.Cleanup(func() { remotemerchant.GetRegistry().ClearCharacter(ten, 1234) })

	var unlocked int
	unlockPendingRemoteMerchant(nullLogger(t), ten, 1234, func() { unlocked++ })

	if unlocked != 1 {
		t.Errorf("unlock calls = %d, want 1", unlocked)
	}
	if _, ok := remotemerchant.GetRegistry().Take(ten, 1234); ok {
		t.Error("registry entry survived the unlock")
	}
}

// TestUnlockPending_MissDoesNotUnlock protects the ordinary NPC-talk path:
// v61/72/79/83/84/87/95 OPEN_NPC_SHOP cells are verified and must stay
// byte-identical, so no unconditional EnableActions may be added here.
func TestUnlockPending_MissDoesNotUnlock(t *testing.T) {
	ten := mustTenant(t)

	var unlocked int
	unlockPendingRemoteMerchant(nullLogger(t), ten, 999999, func() { unlocked++ })

	if unlocked != 0 {
		t.Errorf("unlock calls = %d, want 0", unlocked)
	}
}
