package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte layout (IDA v83 CUIItemUpgrade::Update 0x82ae28 / v95 0x7bef50):
//   Encode4(m_nReturnResult) + Encode4(m_nResult) = 8 bytes. No version gate.
// m_nReturnResult echoes the open-arm mode byte; m_nResult echoes the
// server-chosen round-trip token.
// packet-audit:verify packet=field/serverbound/FieldItemUpgradeUpdate version=gms_v83 ida=0x82ae28
// packet-audit:verify packet=field/serverbound/FieldItemUpgradeUpdate version=gms_v95 ida=0x7bef50
func TestItemUpgradeUpdateByteOutput(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUpgradeUpdate{returnResult: 0, result: 0x0001FFFB}
			got := input.Encode(nil, ctx)(nil)
			if len(got) != 8 {
				t.Errorf("byte count: got %d, want 8", len(got))
			}
		})
	}
}

func TestItemUpgradeUpdateRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUpgradeUpdate{returnResult: 0, result: 0x0001FFFB}
			output := ItemUpgradeUpdate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ReturnResult() != input.ReturnResult() {
				t.Errorf("returnResult: got %d, want %d", output.ReturnResult(), input.ReturnResult())
			}
			if output.Result() != input.Result() {
				t.Errorf("result: got %d, want %d", output.Result(), input.Result())
			}
		})
	}
}
