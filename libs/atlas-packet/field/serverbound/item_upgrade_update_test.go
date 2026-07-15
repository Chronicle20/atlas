package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Byte layout (IDA v83 CUIItemUpgrade::Update 0x82ae28 / v95 0x7bef50 / v84
// CUIItemUpgrade::Update sub_8562D1 0x8562d1):
//   Encode4(m_nReturnResult) + Encode4(m_nResult) = 8 bytes. No version gate.
// m_nReturnResult echoes the open-arm mode byte; m_nResult echoes the
// server-chosen round-trip token.
// NOTE (task-129): v84 COutPacket ctor opcode = 267/0x10B (verified live at
// sub_8562D1) — NOT 0x104. The template's ItemUpgradeUpdateHandle 0x104 was a
// mis-derived seed value (0x104 is a CITC sender) and is corrected to 0x10B.
// NOTE (task-129 gms_v79 extension): v79 COutPacket ctor opcode = 250/0xFA
// (verified live at CUIItemUpgrade::Update 0x7998da, port 13340) — body
// Encode4(m_nReturnResult=this[32]) + Encode4(m_nResult=this[33]); same 8-byte
// shape, no version gate. v83 0x104=260 (d-10).
// packet-audit:verify packet=field/serverbound/FieldItemUpgradeUpdate version=gms_v79 ida=0x7998da
// packet-audit:verify packet=field/serverbound/FieldItemUpgradeUpdate version=gms_v83 ida=0x82ae28
// packet-audit:verify packet=field/serverbound/FieldItemUpgradeUpdate version=gms_v84 ida=0x8562d1
// packet-audit:verify packet=field/serverbound/FieldItemUpgradeUpdate version=gms_v87 ida=0x88eea2
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
