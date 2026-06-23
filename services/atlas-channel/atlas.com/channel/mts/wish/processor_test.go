package wish

import (
	"fmt"
	"strings"
	"testing"
)

// TestExtract asserts the JSON:API RestModel maps onto the channel-side Model,
// including the worldId/serial fields the CANCEL_WISH resolution relies on (the
// serial is what VIEW_WISH renders as nITCSN and the client echoes back).
func TestExtract(t *testing.T) {
	rm := RestModel{Id: "abc-123", WorldId: 2, Serial: 4242, CharacterId: 9001, ItemId: 1302000}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Id() != "abc-123" || m.CharacterId() != 9001 || m.ItemId() != 1302000 {
		t.Errorf("Extract mismatch: id=%q char=%d item=%d", m.Id(), m.CharacterId(), m.ItemId())
	}
	if m.WorldId() != 2 {
		t.Errorf("Extract worldId = %d, want 2", m.WorldId())
	}
	if m.Serial() != 4242 {
		t.Errorf("Extract serial = %d, want 4242", m.Serial())
	}
}

// TestFindByItem asserts the serial->wish-entry resolution the DELETE_ZZIM /
// CANCEL_WISH arms rely on: given the character's wish list and the resolved item
// template, find the matching wish entry (whose Id is the wish UUID the
// REMOVE_WISH command needs).
func TestFindByItem(t *testing.T) {
	ms := []Model{
		{id: "w1", characterId: 9001, itemId: 1302000},
		{id: "w2", characterId: 9001, itemId: 2000000},
	}

	got, ok := findByItem(ms, 2000000)
	if !ok {
		t.Fatalf("findByItem(2000000): expected a match")
	}
	if got.Id() != "w2" {
		t.Errorf("findByItem(2000000): want w2, got %q", got.Id())
	}

	if _, ok := findByItem(ms, 9999999); ok {
		t.Errorf("findByItem(9999999): expected no match")
	}

	if _, ok := findByItem(nil, 1302000); ok {
		t.Errorf("findByItem(nil): expected no match")
	}
}

// TestFindBySerial asserts the CANCEL_WISH resolution: given the character's
// wishlist and the serial the client echoed back (the nITCSN VIEW_WISH wrote),
// find the wish entry whose Serial matches — that entry's Id is the wish UUID the
// REMOVE_WISH command needs. This is the round-trip the H5 fix establishes.
func TestFindBySerial(t *testing.T) {
	ms := []Model{
		{id: "w1", worldId: 0, serial: 11, characterId: 9001, itemId: 1302000},
		{id: "w2", worldId: 0, serial: 22, characterId: 9001, itemId: 2000000},
	}

	got, ok := findBySerial(ms, 22)
	if !ok {
		t.Fatalf("findBySerial(22): expected a match")
	}
	if got.Id() != "w2" {
		t.Errorf("findBySerial(22): want w2, got %q", got.Id())
	}

	if _, ok := findBySerial(ms, 99); ok {
		t.Errorf("findBySerial(99): expected no match for an unknown serial")
	}

	// Serial 0 is the stale/pre-fix sentinel and must never resolve to a wish,
	// even if (defensively) a row carried serial 0.
	zero := []Model{{id: "z", serial: 0, characterId: 9001, itemId: 1}}
	if _, ok := findBySerial(zero, 0); ok {
		t.Errorf("findBySerial(0): expected no match (0 is the no-serial sentinel)")
	}

	if _, ok := findBySerial(nil, 11); ok {
		t.Errorf("findBySerial(nil): expected no match")
	}
}

// TestResourcePath asserts the wishlist resource template formats to the
// atlas-mts per-character wishlist path.
func TestResourcePath(t *testing.T) {
	got := fmt.Sprintf(Resource, uint32(9001))
	if !strings.Contains(got, "characters/9001/mts/wishlist") {
		t.Errorf("resource path %q missing characters/9001/mts/wishlist", got)
	}
}
