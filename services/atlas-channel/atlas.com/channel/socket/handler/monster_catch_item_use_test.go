package handler

import (
	"testing"

	monstersb "github.com/Chronicle20/atlas/libs/atlas-packet/monster/serverbound"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// TestMonsterCatchItemUseDecode proves the handler's decode step recovers every
// field from the wire. The handler itself performs no validation — item checks
// live in atlas-consumables and monster checks in atlas-monsters (design §4.6).
func TestMonsterCatchItemUseDecode(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	encoded := monstersb.NewUseCatchItem(0x11223344, 7, 2270008, 0x07654321).Encode(nil, ctx)(nil)

	req := request.Request(encoded)
	reader := request.NewRequestReader(&req, 0)

	var p monstersb.UseCatchItem
	p.Decode(nil, ctx)(&reader, nil)

	if p.Slot() != 7 || p.ItemId() != 2270008 || p.MonsterUniqueId() != 0x07654321 {
		t.Fatalf("decoded %s", p.String())
	}
	if p.Operation() != monstersb.UseCatchItemHandle {
		t.Fatalf("Operation() = %q, want %q", p.Operation(), monstersb.UseCatchItemHandle)
	}
}
