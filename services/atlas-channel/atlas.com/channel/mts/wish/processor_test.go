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

// TestWorldResourcePath asserts the cross-character want-ad resource template
// formats to the atlas-mts per-world wishlist path.
func TestWorldResourcePath(t *testing.T) {
	got := fmt.Sprintf(WorldResource, byte(2))
	if !strings.Contains(got, "worlds/2/mts/wishlist") {
		t.Errorf("world resource path %q missing worlds/2/mts/wishlist", got)
	}
}

// TestToMtsItemWithSeller asserts the cross-character Wanted-tab variant carries
// the want-ad's serial as nITCSN, the wish price, and the owner name in the
// seller (sGameID) column — while ToMtsItem (the viewer's own entries) leaves the
// seller column empty.
func TestToMtsItemWithSeller(t *testing.T) {
	wm, err := Extract(RestModel{Id: "w1", WorldId: 0, Serial: 7777, CharacterId: 9001, ItemId: 1302000, Price: 1500})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	withSeller := ToMtsItemWithSeller(wm, "Aria")
	if withSeller.GameId() != "Aria" {
		t.Errorf("seller column (sGameID): want Aria, got %q", withSeller.GameId())
	}
	if withSeller.ItcSn() != 7777 {
		t.Errorf("itcSn: want 7777 (the wish serial), got %d", withSeller.ItcSn())
	}
	if withSeller.Price() != 1500 {
		t.Errorf("price: want 1500 (the wish price), got %d", withSeller.Price())
	}
	if withSeller.Item().TemplateId() != 1302000 {
		t.Errorf("item template: want 1302000, got %d", withSeller.Item().TemplateId())
	}

	// The viewer's own entries (ToMtsItem) leave the seller column empty.
	if own := ToMtsItem(wm); own.GameId() != "" {
		t.Errorf("ToMtsItem seller column: want empty, got %q", own.GameId())
	}
}

// TestToMtsItemWithSellerEmptyName asserts a blank owner name still produces a
// valid item (a name-lookup failure must blank the seller column, not drop the
// want-ad).
func TestToMtsItemWithSellerEmptyName(t *testing.T) {
	wm, err := Extract(RestModel{Id: "w1", WorldId: 0, Serial: 7777, CharacterId: 9001, ItemId: 1302000})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	it := ToMtsItemWithSeller(wm, "")
	if it.GameId() != "" {
		t.Errorf("seller column: want empty, got %q", it.GameId())
	}
	if it.ItcSn() != 7777 {
		t.Errorf("itcSn: want 7777, got %d", it.ItcSn())
	}
}
