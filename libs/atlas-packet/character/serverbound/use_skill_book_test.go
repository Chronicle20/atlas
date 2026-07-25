package serverbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestUseSkillBookRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := UseSkillBook{updateTime: 12345, slot: 2, itemId: 2290000}
			output := UseSkillBook{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}

// Golden bytes, v83: updateTime(4 LE) + slot(2 LE) + itemId(4 LE).
// 12345 = 0x3039; 2 = 0x0002; 2290000 = 0x22F150.
func TestUseSkillBookGoldenBytesV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	l, _ := testlog.NewNullLogger()
	got := UseSkillBook{updateTime: 12345, slot: 2, itemId: 2290000}.Encode(l, ctx)(nil)
	want := []byte{0x39, 0x30, 0x00, 0x00, 0x02, 0x00, 0x50, 0xF1, 0x22, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("golden bytes: got % X, want % X", got, want)
	}
}
