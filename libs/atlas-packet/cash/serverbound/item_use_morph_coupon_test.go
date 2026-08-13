package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestItemUseMorphCouponUpdateTimeFirstRoundTrip pins FR-1.2's leading-updateTime
// half: when the common ItemUse header already carried updateTime (GMS >= v87,
// JMS), the case-41 sub-body reads NOTHING. Zero bytes on the wire.
func TestItemUseMorphCouponUpdateTimeFirstRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseMorphCoupon{updateTimeFirst: true}
			output := *NewItemUseMorphCoupon(true)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != 0 {
				t.Errorf("updateTime: got %v, want 0 (nothing may be consumed when updateTimeFirst)", output.UpdateTime())
			}
		})
	}
}

// TestItemUseMorphCouponNoUpdateTimeFirstRoundTrip pins FR-1.2's trailing half:
// on GMS <= v84 the case-40 sub-body consumes exactly one trailing int32.
func TestItemUseMorphCouponNoUpdateTimeFirstRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseMorphCoupon{updateTime: 600000, updateTimeFirst: false}
			output := *NewItemUseMorphCoupon(false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}

// TestItemUseMorphCouponEncodedLength pins the byte count directly, so a future
// field added to the struct cannot pass the round-trip by symmetry alone.
func TestItemUseMorphCouponEncodedLength(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	if got := len(pt.Encode(t, ctx, ItemUseMorphCoupon{updateTime: 42, updateTimeFirst: false}.Encode, nil)); got != 4 {
		t.Errorf("trailing-updateTime encoded length = %d, want 4", got)
	}
	if got := len(pt.Encode(t, ctx, ItemUseMorphCoupon{updateTime: 42, updateTimeFirst: true}.Encode, nil)); got != 0 {
		t.Errorf("leading-updateTime encoded length = %d, want 0", got)
	}
}
