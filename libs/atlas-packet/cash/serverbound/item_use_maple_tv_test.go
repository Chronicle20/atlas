package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseMapleTVRoundTrip(t *testing.T) {
	// tvType drives the conditional prefix; cover every arm.
	cases := []struct {
		name   string
		tvType byte
		ear    bool
		recv   string
	}{
		{"tv0_normal", 0, false, "PartnerA"},     // byte pad + receiver
		{"tv1_star", 1, false, ""},               // no prefix, no receiver
		{"tv2_heart", 2, false, "PartnerB"},      // receiver only
		{"tv3_megassenger", 3, true, "PartnerC"}, // byte + ear + receiver
		{"tv4_star_m", 4, true, ""},              // ear, NO receiver
		{"tv5_heart_m", 5, false, "PartnerD"},    // ear + receiver
	}
	for _, v := range pt.Variants {
		for _, tc := range cases {
			t.Run(v.Name+"/"+tc.name, func(t *testing.T) {
				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				updateTimeFirst := v.Region == "GMS" && v.MajorVersion >= 95
				input := NewItemUseMapleTV(updateTimeFirst, tc.tvType)
				input.ear = tc.ear
				input.receiverName = tc.recv
				input.lines = [5]string{"l1", "l2", "l3", "l4", "l5"}
				if !updateTimeFirst {
					input.updateTime = 42
				}
				output := NewItemUseMapleTV(updateTimeFirst, tc.tvType)
				pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Ear() != input.Ear() {
					t.Errorf("ear: got %v, want %v", output.Ear(), input.Ear())
				}
				if output.ReceiverName() != input.ReceiverName() {
					t.Errorf("receiverName: got %q, want %q", output.ReceiverName(), input.ReceiverName())
				}
				for i := range input.lines {
					if output.Lines()[i] != input.Lines()[i] {
						t.Errorf("line %d: got %q, want %q", i, output.Lines()[i], input.Lines()[i])
					}
				}
				if output.UpdateTime() != input.UpdateTime() {
					t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
				}
			})
		}
	}
}
