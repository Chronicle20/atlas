package party

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestUpdateWRoundTrip(t *testing.T) {
	members := []PartyMember{
		{Id: 100, Name: "Player1", JobId: 111, Level: 50, ChannelId: 1, MapId: 100000},
		{Id: 200, Name: "Player2", JobId: 222, Level: 70, ChannelId: -2, MapId: 200000},
		{Id: 300, Name: "Player3", JobId: 333, Level: 90, ChannelId: 3, MapId: 300000},
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewUpdateW(13, 5000, members, 100)
			output := UpdateW{}
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
