package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v79 DENY_PARTY_REQUEST (op 122) — built in CWvsContext::OnPartyResult
// @0x987583 case 4 (invite-received): COutPacket(122) @0x98772e, Encode1(mode)
// @0x98774c, EncodeStr(blockedName) @0x987765, EncodeStr(charName) @0x987784.
// atlas InviteReject decodes WriteByte(unk)+WriteAsciiString(from) (the leading
// flag + first name); same shape as the verified v83 cell.
// gms_v72: DENY_PARTY_REQUEST built in CWvsContext::OnPartyResult @0x934f3c case
// 4 (invite-received) — same Encode1(mode)+EncodeStr(name) shape as v79 (party
// mode table unshifted at the low arms; task-113 party-family agent). atlas
// InviteReject decodes WriteByte(unk)+WriteAsciiString(from).
// packet-audit:verify packet=party/serverbound/PartyInviteReject version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/serverbound/PartyInviteReject version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/serverbound/PartyInviteReject version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/serverbound/PartyInviteReject version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/serverbound/PartyInviteReject version=gms_v95 ida=0xa10ab0
// packet-audit:verify packet=party/serverbound/PartyInviteReject version=jms_v185 ida=0xb297e7
// packet-audit:verify packet=party/serverbound/PartyInviteReject version=gms_v84 ida=0xa89cf3
func TestInviteRejectRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := InviteReject{unk: 1, from: "SomePartyLeader"}
			output := InviteReject{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Unk() != input.Unk() {
				t.Errorf("unk: got %v, want %v", output.Unk(), input.Unk())
			}
			if output.From() != input.From() {
				t.Errorf("from: got %v, want %v", output.From(), input.From())
			}
		})
	}
}
