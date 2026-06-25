package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/serverbound/HealOverTime version=gms_v83 ida=0xa1e997
// packet-audit:verify packet=character/serverbound/HealOverTime version=gms_v87 ida=0xab5ca8
// packet-audit:verify packet=character/serverbound/HealOverTime version=gms_v95 ida=0x9f2a00
// packet-audit:verify packet=character/serverbound/HealOverTime version=gms_v84 ida=0xa69c4d
// packet-audit:verify packet=character/serverbound/HealOverTime version=jms_v185 ida=0xb054d6
//
// jms HEAL_OVER_TIME (opcode 0x54) is sent by CWvsContext::SendStatChangeRequestByItemOption@0xb054d6
// (misleading symbol; the opcode is the ground truth — called from CWvsContext::TryRecovery
// auto-recovery). Wire = updateTime(4)+val(4)+hp(2)+mp(2)+option(1)+extra(4); jms appends a
// trailing client validation dword (dword_CDA4F8) the GMS v83/v87/v95 senders do NOT. The
// codec encodes the option byte for (GMS<=95)||JMS and the trailing dword for JMS only.
func TestHealOverTimeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := HealOverTime{updateTime: 100, val: 200, hp: 50, mp: 30, unknown: 1, extra: 0xCAFEBABE}
			output := HealOverTime{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.HP() != input.HP() {
				t.Errorf("hp: got %v, want %v", output.HP(), input.HP())
			}
			if output.MP() != input.MP() {
				t.Errorf("mp: got %v, want %v", output.MP(), input.MP())
			}
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			// jms appends the validation dword (CWvsContext::SendStatChangeRequestByItemOption@0xb054d6,
			// opcode 0x54); GMS does not. Assert the round-trip preserves it only where it is on the wire.
			if v.Region == "JMS" {
				if output.Extra() != input.Extra() {
					t.Errorf("extra (jms trailing dword): got %#x, want %#x", output.Extra(), input.Extra())
				}
			} else if output.Extra() != 0 {
				t.Errorf("extra: GMS must not read a trailing dword, got %#x", output.Extra())
			}
		})
	}
}
