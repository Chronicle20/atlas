package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/party"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestJoinByteOutput verifies the byte output of Join across all tenant variants.
// Wire layout: mode(1)+partyId(4)+targetName(2+len)+WritePartyData(?).
// targetName="Player2" → 2+7=9 bytes. Total with PARTYDATA:
//   v83/JMS: 1+4+9+298 = 312 bytes (JMS uses small PARTYDATA; IDA @0xb297e7 qmemcpy 0x12A)
//   v95 (GMS only): 1+4+9+378 = 392 bytes
func TestJoinByteOutput(t *testing.T) {
	members := []party.PartyMember{
		{Id: 100, Name: "Player1", JobId: 111, Level: 50, ChannelId: 1, MapId: 100000},
		{Id: 200, Name: "Player2", JobId: 222, Level: 70, ChannelId: 2, MapId: 200000},
	}
	cases := []struct {
		variant   pt.TenantVariant
		wantBytes int
	}{
		{pt.Variants[0], 312}, // GMS v28  — v83 PARTYDATA
		{pt.Variants[1], 312}, // GMS v83  — v83 PARTYDATA
		{pt.Variants[2], 312}, // GMS v87  — v83 PARTYDATA
		{pt.Variants[3], 392}, // GMS v95  — v95 PARTYDATA
		{pt.Variants[4], 312}, // JMS v185 — small PARTYDATA (298 bytes); IDA @0xb297e7 qmemcpy 0x12A
	}
	for _, tc := range cases {
		t.Run(tc.variant.Name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.variant.Region, tc.variant.MajorVersion, tc.variant.MinorVersion)
			input := NewJoin(12, 5000, "Player2", members, 100)
			got := input.Encode(nil, ctx)(nil)
			if len(got) != tc.wantBytes {
				t.Errorf("byte count: got %d, want %d", len(got), tc.wantBytes)
			}
		})
	}
}

func TestJoinRoundTrip(t *testing.T) {
	members := []party.PartyMember{
		{Id: 100, Name: "Player1", JobId: 111, Level: 50, ChannelId: 1, MapId: 100000},
		{Id: 200, Name: "Player2", JobId: 222, Level: 70, ChannelId: 2, MapId: 200000},
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewJoin(12, 5000, "Player2", members, 100)
			output := Join{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.PartyId() != input.PartyId() {
				t.Errorf("partyId: got %v, want %v", output.PartyId(), input.PartyId())
			}
			if output.TargetName() != input.TargetName() {
				t.Errorf("targetName: got %v, want %v", output.TargetName(), input.TargetName())
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
