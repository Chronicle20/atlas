package wish

import (
	"fmt"
	"strings"
	"testing"
)

// TestExtract asserts the JSON:API RestModel maps onto the channel-side Model
// (id/characterId/itemId are the only fields the zzim/wish arms consume).
func TestExtract(t *testing.T) {
	rm := RestModel{Id: "abc-123", CharacterId: 9001, ItemId: 1302000}
	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if m.Id() != "abc-123" || m.CharacterId() != 9001 || m.ItemId() != 1302000 {
		t.Errorf("Extract mismatch: id=%q char=%d item=%d", m.Id(), m.CharacterId(), m.ItemId())
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

// TestResourcePath asserts the wishlist resource template formats to the
// atlas-mts per-character wishlist path.
func TestResourcePath(t *testing.T) {
	got := fmt.Sprintf(Resource, uint32(9001))
	if !strings.Contains(got, "characters/9001/mts/wishlist") {
		t.Errorf("resource path %q missing characters/9001/mts/wishlist", got)
	}
}
