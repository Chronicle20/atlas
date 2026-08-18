package main

import (
	"os"
	"strings"
	"testing"
)

// TestTickerDoesNotCloseOverACachedTenantList pins design C4: the tenant
// list must be resolved inside the tick, not before it. A tenant
// provisioned after this pod started must be picked up without a restart,
// because a baseline pod cannot be redeployed to serve an ephemeral
// environment (G7, NG6).
func TestTickerDoesNotCloseOverACachedTenantList(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "service.ForEachOwnedEnvironment") {
		t.Fatal("tick loop does not use service.ForEachOwnedEnvironment")
	}
	tickerAt := strings.Index(s, "time.NewTicker")
	getAllAt := strings.Index(s, "NewProcessor(l, rt.Context()).GetAll()")
	if getAllAt >= 0 && getAllAt < tickerAt {
		t.Fatal("tenant list is still loaded before the ticker and closed over (design C4)")
	}
}
