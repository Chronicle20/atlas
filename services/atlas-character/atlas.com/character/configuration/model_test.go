package configuration

import (
	"testing"
	"time"
)

func TestExtractFoldsAbsentExpiryToTheDefault(t *testing.T) {
	if got := Extract(RestModel{}).PendingExpiry(); got != 168*time.Hour {
		t.Fatalf("PendingExpiry = %v, want 168h", got)
	}
}

func TestExtractHonoursAnOperatorOverride(t *testing.T) {
	if got := Extract(RestModel{PendingExpiryHours: 24}).PendingExpiry(); got != 24*time.Hour {
		t.Fatalf("PendingExpiry = %v, want 24h", got)
	}
}

// TestExtractFoldsAZeroExpiryToTheDefault pins the failure mode this package
// must never produce: a seeded-but-zero pendingExpiryHours (an operator typo,
// or a row created before the field existed) must NOT fold into a 0h expiry,
// which would expire every pending change the instant it is created. It must
// fold to the same 168h default an absent config gets.
func TestExtractFoldsAZeroExpiryToTheDefault(t *testing.T) {
	got := Extract(RestModel{Id: "imprint-configs", PendingExpiryHours: 0}).PendingExpiry()
	if got != DefaultPendingExpiry {
		t.Fatalf("PendingExpiry = %v, want the default %v", got, DefaultPendingExpiry)
	}
	if got == 0 {
		t.Fatal("PendingExpiry folded to 0h — every pending change would expire instantly")
	}
}
