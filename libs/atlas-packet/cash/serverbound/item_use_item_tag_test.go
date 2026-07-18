package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseItemTagRoundTrip(t *testing.T) {
	for _, first := range []bool{true, false} {
		for _, v := range pt.Variants {
			t.Run(v.Name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				input := ItemUseItemTag{slot: -1, updateTime: 1000, updateTimeFirst: first}
				output := *NewItemUseItemTag(first)
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

// v83 golden bytes: short slot (-1 = FF FF) + trailing int updateTime (1000 = E8 03 00 00)
func TestItemUseItemTagV83Bytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	m := ItemUseItemTag{slot: -1, updateTime: 1000, updateTimeFirst: false}
	got := m.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{0xFF, 0xFF, 0xE8, 0x03, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X, want % X", got, want)
	}
}
