package handler

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	mountsb "github.com/Chronicle20/atlas/libs/atlas-packet/mount/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// TestMountFoodDecode pins the v83 wire format MountFoodHandleFunc consumes:
// ts(4), slot(2), itemId(4), all little-endian (opcode 0x4D).
func TestMountFoodDecode(t *testing.T) {
	raw := []byte{
		0x64, 0x00, 0x00, 0x00, // ts = 100
		0x07, 0x00, // slot = 7
		0x80, 0x84, 0x1E, 0x00, // itemId = 2000000
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := mountsb.Food{}
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if p.UpdateTime() != 100 {
		t.Errorf("ts: got %d, want 100", p.UpdateTime())
	}
	if p.Slot() != 7 {
		t.Errorf("slot: got %d, want 7", p.Slot())
	}
	if p.ItemId() != 2000000 {
		t.Errorf("itemId: got %d, want 2000000", p.ItemId())
	}
	if p.Operation() != mountsb.MountFoodHandle {
		t.Errorf("operation: got %q, want %q", p.Operation(), mountsb.MountFoodHandle)
	}
}

// TestMountFoodHandleFuncSymbol verifies the handler constructor returns a
// non-nil closure with the standard handler signature.
func TestMountFoodHandleFuncSymbol(t *testing.T) {
	got := MountFoodHandleFunc(logrus.New(), context.Background(), nil)
	if got == nil {
		t.Fatal("MountFoodHandleFunc returned nil closure")
	}
}
