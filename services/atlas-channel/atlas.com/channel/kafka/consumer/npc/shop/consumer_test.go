package shop

import (
	"atlas-channel/remotemerchant"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

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
