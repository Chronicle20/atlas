package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestInviteRejectV61 pins the gms_v61 DENY_PARTY_REQUEST (op 113) wire.
//
// IDA-verified send site (GMS_v61.1_U_DEVM.exe, port 13338) — the client builds
// DENY_PARTY_REQUEST inside CWvsContext::OnPartyResult @0x857a8c case 4
// (invite-received, auto-decline branch): COutPacket(113) @0x857c29,
// Encode1(declineCode) @0x857c47, EncodeStr(inviterName) @0x857c60,
// EncodeStr(myName) @0x857c7f. atlas InviteReject models the leading flag +
// first name (WriteByte(unk)+WriteAsciiString(from)); the trailing myName is not
// modelled — same shape as the verified v72/v79/v83 cells.
//
// packet-audit:verify packet=party/serverbound/PartyInviteReject version=gms_v61 ida=0x857a8c
func TestInviteRejectV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	m := InviteReject{unk: 1, from: "SomePartyLeader"}
	want := []byte{
		0x01,       // Encode1 declineCode (@0x857c47)
		0x0F, 0x00, // WriteAsciiString length prefix (15)
		0x53, 0x6F, 0x6D, 0x65, 0x50, 0x61, 0x72, 0x74, 0x79, 0x4C, 0x65, 0x61, 0x64, 0x65, 0x72, // "SomePartyLeader" (@0x857c60)
	}
	if got := m.Encode(nil, ctx)(nil); !bytes.Equal(got, want) {
		t.Errorf("v61 InviteReject golden mismatch\n got: % x\nwant: % x", got, want)
	}
}

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
