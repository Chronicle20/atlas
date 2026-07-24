package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte layout (IDA, design §2.1 step 3 — identical appends in v83
// CUIItemUpgrade::OnButtonClicked sub_82AED3 and v95 0x7c0ca0):
//
//	Encode4(itemTI) + Encode4(slotPosition) + Encode4(updateTime) = 12 bytes,
//
// appended AFTER the shared ItemUse prefix. No version gate.
func TestItemUseViciousHammerByteOutput(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseViciousHammer{itemTI: 1, slotPosition: -5, updateTime: 0xDEADBEEF}
			got := input.Encode(nil, ctx)(nil)
			if len(got) != 12 {
				t.Errorf("byte count: got %d, want 12", len(got))
			}
		})
	}
}

func TestItemUseViciousHammerRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseViciousHammer{itemTI: 1, slotPosition: -5, updateTime: 12345}
			output := ItemUseViciousHammer{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ItemTI() != input.ItemTI() {
				t.Errorf("itemTI: got %d, want %d", output.ItemTI(), input.ItemTI())
			}
			if output.SlotPosition() != input.SlotPosition() {
				t.Errorf("slotPosition: got %d, want %d", output.SlotPosition(), input.SlotPosition())
			}
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %d, want %d", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
