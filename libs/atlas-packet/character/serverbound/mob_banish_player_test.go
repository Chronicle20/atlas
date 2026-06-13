package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// MOB_BANISH_PLAYER. v83/v84 senders are INLINED into CUserLocal::Update — no
// standalone function exists to pin, so no verify marker / evidence for those two
// versions (see structures/RESUME-STATE.md). The v87/v95/jms standalone
// CUserLocal::SendBanMapByMobRequest is a one-Encode4 wrapper; the v83/v84 inlined
// send is byte-identical, so the single codec covers all five versions.
//
// behavior/evidence: v83+v84 blocked (inlined sender, see RESUME-STATE.md)
//
// packet-audit:verify packet=character/serverbound/CharacterMobBanishPlayer version=gms_v87 ida=0x9df571
// packet-audit:verify packet=character/serverbound/CharacterMobBanishPlayer version=gms_v95 ida=0x908d50
// packet-audit:verify packet=character/serverbound/CharacterMobBanishPlayer version=jms_v185 ida=0xa28621
func TestMobBanishPlayer(t *testing.T) {
	input := MobBanishPlayer{mobTemplateId: 0x008B0B01}

	// Golden bytes (v83 baseline; identical to v87/v95). The inlined v83
	// CUserLocal::Update send and the v87 CUserLocal::SendBanMapByMobRequest
	// @0x9df571 both emit a single Encode4(dwMobTemplateID).
	got := input.Encode(nil, pt.CreateContext("GMS", 83, 1))(nil)
	want := []byte{
		0x01, 0x0B, 0x8B, 0x00, // mobTemplateId uint32 LE = 0x008B0B01 (Encode4 @0x9df571)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("MobBanishPlayer layout mismatch\n got % x\nwant % x", got, want)
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			pt.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
