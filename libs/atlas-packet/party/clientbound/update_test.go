package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/party"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestUpdateByteOutput verifies the byte output of Update across all tenant
// variants. IDA CWvsContext::OnPartyResult reads Decode4(partyId)+PARTYDATA.
//
// v83/JMS (GMS < 95 or JMS): mode(1)+partyId(4)+WritePartyData(298) = 303 bytes.
//   PARTYDATA = 298 bytes: portals use 4 ints each (no m_nSKillID); no PQ fields.
//   Confirmed: v83 OnPartyResult@0xa3e31c memset(3732,0,0x12A=298).
//   JMS v185 OnPartyResult@0xb297e7: qmemcpy(v120,...,0x12Au=298) — JMS is small.
//
// v95+ (GMS >= 95 only): mode(1)+partyId(4)+WritePartyData(378) = 383 bytes.
//   PARTYDATA = 378 bytes: portals add m_nSKillID (5th int); PQ fields (+56 bytes).
//   Confirmed: v95 IDA memset(3732,0,0x17A=378).
func TestUpdateByteOutput(t *testing.T) {
	members := []party.PartyMember{
		{Id: 100, Name: "Player1", JobId: 111, Level: 50, ChannelId: 1, MapId: 100000},
		{Id: 200, Name: "Player2", JobId: 222, Level: 70, ChannelId: -2, MapId: 200000},
		{Id: 300, Name: "Player3", JobId: 333, Level: 90, ChannelId: 3, MapId: 300000},
	}
	cases := []struct {
		variant   pt.TenantVariant
		wantBytes int
	}{
		{pt.Variants[0], 303}, // GMS v28  — v83 PARTYDATA (298 bytes)
		{pt.Variants[1], 303}, // GMS v83  — v83 PARTYDATA (298 bytes)
		{pt.Variants[2], 303}, // GMS v87  — v83 PARTYDATA (298 bytes)
		{pt.Variants[3], 383}, // GMS v95  — v95 PARTYDATA (378 bytes)
		{pt.Variants[4], 303}, // JMS v185 — small PARTYDATA (298 bytes); IDA @0xb297e7 qmemcpy 0x12A
	}
	for _, tc := range cases {
		t.Run(tc.variant.Name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.variant.Region, tc.variant.MajorVersion, tc.variant.MinorVersion)
			input := NewUpdate(13, 5000, members, 100)
			got := input.Encode(nil, ctx)(nil)
			if len(got) != tc.wantBytes {
				t.Errorf("byte count: got %d, want %d", len(got), tc.wantBytes)
			}
		})
	}
}

func TestUpdateRoundTrip(t *testing.T) {
	members := []party.PartyMember{
		{Id: 100, Name: "Player1", JobId: 111, Level: 50, ChannelId: 1, MapId: 100000},
		{Id: 200, Name: "Player2", JobId: 222, Level: 70, ChannelId: -2, MapId: 200000},
		{Id: 300, Name: "Player3", JobId: 333, Level: 90, ChannelId: 3, MapId: 300000},
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewUpdate(13, 5000, members, 100)
			output := Update{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.PartyId() != input.PartyId() {
				t.Errorf("partyId: got %v, want %v", output.PartyId(), input.PartyId())
			}
			if output.LeaderId() != input.LeaderId() {
				t.Errorf("leaderId: got %v, want %v", output.LeaderId(), input.LeaderId())
			}
			if len(output.Members()) != len(input.Members()) {
				t.Errorf("members length: got %v, want %v", len(output.Members()), len(input.Members()))
			}
		})
	}
}
