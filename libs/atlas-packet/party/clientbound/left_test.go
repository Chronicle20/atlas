package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/party"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestLeftByteOutput verifies the byte output of Left across all tenant variants.
// Wire layout: mode(1)+partyId(4)+targetId(4)+const1(1)+forced(1)+targetName(2+len)+WritePartyData(?).
// targetName="Player1" → 2+7=9. Fixed prefix: 1+4+4+1+1+9 = 20. Total:
//   v83/JMS: 20+298 = 318 bytes (JMS uses small PARTYDATA; IDA @0xb297e7 qmemcpy 0x12A)
//   v95 (GMS only): 20+378 = 398 bytes
func TestLeftByteOutput(t *testing.T) {
	members := []party.PartyMember{
		{Id: 100, Name: "Player1", JobId: 111, Level: 50, ChannelId: 1, MapId: 100000},
		{Id: 200, Name: "Player2", JobId: 222, Level: 70, ChannelId: -2, MapId: 200000},
	}
	cases := []struct {
		variant   pt.TenantVariant
		wantBytes int
	}{
		{pt.Variants[0], 318}, // GMS v28  — v83 PARTYDATA
		{pt.Variants[1], 318}, // GMS v83  — v83 PARTYDATA
		{pt.Variants[2], 318}, // GMS v87  — v83 PARTYDATA
		{pt.Variants[3], 398}, // GMS v95  — v95 PARTYDATA
		{pt.Variants[4], 318}, // JMS v185 — small PARTYDATA (298 bytes); IDA @0xb297e7 qmemcpy 0x12A
	}
	for _, tc := range cases {
		t.Run(tc.variant.Name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.variant.Region, tc.variant.MajorVersion, tc.variant.MinorVersion)
			input := NewLeft(10, 5000, 100, "Player1", false, members, 200)
			got := input.Encode(nil, ctx)(nil)
			if len(got) != tc.wantBytes {
				t.Errorf("byte count: got %d, want %d", len(got), tc.wantBytes)
			}
		})
	}
}

func TestLeftRoundTrip(t *testing.T) {
	members := []party.PartyMember{
		{Id: 100, Name: "Player1", JobId: 111, Level: 50, ChannelId: 1, MapId: 100000},
		{Id: 200, Name: "Player2", JobId: 222, Level: 70, ChannelId: -2, MapId: 200000},
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewLeft(10, 5000, 100, "Player1", false, members, 200)
			output := Left{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.PartyId() != input.PartyId() {
				t.Errorf("partyId: got %v, want %v", output.PartyId(), input.PartyId())
			}
			if output.TargetId() != input.TargetId() {
				t.Errorf("targetId: got %v, want %v", output.TargetId(), input.TargetId())
			}
			if output.TargetName() != input.TargetName() {
				t.Errorf("targetName: got %v, want %v", output.TargetName(), input.TargetName())
			}
			if output.Forced() != input.Forced() {
				t.Errorf("forced: got %v, want %v", output.Forced(), input.Forced())
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

func TestLeftForcedRoundTrip(t *testing.T) {
	members := []party.PartyMember{
		{Id: 100, Name: "Player1", JobId: 111, Level: 50, ChannelId: 1, MapId: 100000},
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewLeft(10, 5000, 100, "Player1", true, members, 100)
			output := Left{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Forced() != true {
				t.Errorf("forced: got %v, want true", output.Forced())
			}
		})
	}
}
