package serverbound

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// TestFoodDecode pins the v83 wire format that the taming-mob food handler
// (opcode 0x4D, SendTamingMobFoodItemUseRequest) consumes:
// ts(4), slot(2), itemId(4) -- all little-endian.
func TestFoodDecode(t *testing.T) {
	// ts = 100 (0x00000064), slot = 3 (0x0003), itemId = 2000000 (0x001E8480)
	raw := []byte{
		0x64, 0x00, 0x00, 0x00, // ts
		0x03, 0x00, // slot
		0x80, 0x84, 0x1E, 0x00, // itemId
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := Food{}
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if p.UpdateTime() != 100 {
		t.Errorf("ts: got %d, want 100", p.UpdateTime())
	}
	if p.Slot() != 3 {
		t.Errorf("slot: got %d, want 3", p.Slot())
	}
	if p.ItemId() != 2000000 {
		t.Errorf("itemId: got %d, want 2000000", p.ItemId())
	}
	if p.Operation() != MountFoodHandle {
		t.Errorf("operation: got %q, want %q", p.Operation(), MountFoodHandle)
	}
}
