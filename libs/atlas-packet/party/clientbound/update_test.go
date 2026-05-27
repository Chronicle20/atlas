package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/party"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestUpdateByteOutput verifies the byte output of Update across all tenant
// variants. IDA CWvsContext::OnPartyResult@0xa10ab0 mode=7/38 reads
// Decode4(partyId)+PARTYDATA::Decode(0x17A=378 bytes). Atlas Update writes
// mode(1)+partyId(4)+WritePartyData(378) = 383 bytes, version-independent.
// Fix: WritePartyData now emits aTownPortal[6].m_nSKillID (24 bytes) +
// aPQReward[6]+aPQRewardType[6]+dwPQRewardMobTemplateID+bPQReward (56 bytes).
func TestUpdateByteOutput(t *testing.T) {
	const wantBytes = 383 // mode(1) + partyId(4) + WritePartyData(378)
	members := []party.PartyMember{
		{Id: 100, Name: "Player1", JobId: 111, Level: 50, ChannelId: 1, MapId: 100000},
		{Id: 200, Name: "Player2", JobId: 222, Level: 70, ChannelId: -2, MapId: 200000},
		{Id: 300, Name: "Player3", JobId: 333, Level: 90, ChannelId: 3, MapId: 300000},
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewUpdate(13, 5000, members, 100)
			got := input.Encode(nil, ctx)(nil)
			if len(got) != wantBytes {
				t.Errorf("byte count: got %d, want %d", len(got), wantBytes)
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
