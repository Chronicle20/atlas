package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseTargetSlotRoundTrip(t *testing.T) {
	// Negative slots are the equipped positions, which is the ONLY shape the
	// sandglass arm ever sees: the client resolves the drop point with
	// CDraggableItem::ModifyEquipItem and negates it.
	for _, s := range []int16{-1, -8, 3} {
		for _, first := range []bool{true, false} {
			for _, v := range pt.Variants {
				t.Run(v.Name, func(t *testing.T) {
					ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
					input := ItemUseTargetSlot{slot: s, updateTime: 1000, updateTimeFirst: first}
					output := *NewItemUseTargetSlot(first)
					pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
					if output.Slot() != input.Slot() {
						t.Errorf("slot = %d, want %d", output.Slot(), input.Slot())
					}
					if !first && output.UpdateTime() != input.UpdateTime() {
						t.Errorf("updateTime = %d, want %d", output.UpdateTime(), input.UpdateTime())
					}
				})
			}
		}
	}
}

// v83 golden bytes: short slot (-1 = FF FF) + trailing int updateTime (1000 = E8 03 00 00)
func TestItemUseTargetSlotV83Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := ItemUseTargetSlot{slot: -1, updateTime: 1000, updateTimeFirst: false}
	got := m.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{0xFF, 0xFF, 0xE8, 0x03, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}

// An equipped weapon position (-11) encodes as F5 FF little-endian.
func TestItemUseTargetSlotEquippedSlotBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := ItemUseTargetSlot{slot: -11, updateTimeFirst: true}
	got := m.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{0xF5, 0xFF}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}
