package handler

import (
	"testing"

	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
)

// TestMtsPageWindow pins the 16-item page window against the client's
// selector math (pageCount = ceil(total/16)): every page the selector can
// request must yield the matching slice, and the last page carries the
// remainder.
func TestMtsPageWindow(t *testing.T) {
	items := make([]fieldcb.MtsItem, 40) // ceil(40/16) = 3 pages: 16, 16, 8

	if got := len(mtsPageWindow(items, 0)); got != 16 {
		t.Fatalf("page 0 window = %d, want 16", got)
	}
	if got := len(mtsPageWindow(items, 1)); got != 16 {
		t.Fatalf("page 1 window = %d, want 16", got)
	}
	if got := len(mtsPageWindow(items, 2)); got != 8 {
		t.Fatalf("page 2 window = %d, want 8", got)
	}
	if got := len(mtsPageWindow(items, 3)); got != 0 {
		t.Fatalf("out-of-range page window = %d, want 0", got)
	}
	if got := len(mtsPageWindow(nil, 0)); got != 0 {
		t.Fatalf("empty set window = %d, want 0", got)
	}
	// Identity of the slice, not just its size: page 1 starts at item 16.
	items16 := mtsPageWindow(items, 1)
	if &items16[0] != &items[16] {
		t.Fatal("page 1 window does not start at item 16")
	}
}
