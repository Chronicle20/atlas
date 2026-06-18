package party

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// TestPartyDataTownPortalRoundTrip verifies a member's Mystic Door town portal
// survives WritePartyData -> ReadPartyData. Before the fix the aTownPortal[6]
// array was hard-zeroed, so a party JOIN/UPDATE wiped the door and the v83
// client (which renders party-member town doors solely from this array) drew
// nothing. A doorless member must round-trip with HasDoor=false.
func TestPartyDataTownPortalRoundTrip(t *testing.T) {
	members := []PartyMember{
		{
			Id: 100, Name: "Caster", JobId: 111, Level: 50, ChannelId: 1, MapId: 240000000,
			HasDoor: true, DoorTownMapId: 240000000, DoorFieldMapId: 240010000, DoorX: 1234, DoorY: -567,
		},
		{Id: 200, Name: "Member", JobId: 222, Level: 70, ChannelId: 1, MapId: 240000000},
	}

	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			w := response.NewWriter(logrus.New())
			WritePartyData(ctx, w, members, 100)
			raw := w.Bytes()

			req := request.Request(raw)
			r := request.NewRequestReader(&req, 0)
			got, leaderId := ReadPartyData(ctx, &r)

			if leaderId != 100 {
				t.Fatalf("leaderId: got %d, want 100", leaderId)
			}
			if len(got) != 2 {
				t.Fatalf("members: got %d, want 2", len(got))
			}
			d := got[0]
			if !d.HasDoor || d.DoorTownMapId != 240000000 || d.DoorFieldMapId != 240010000 || d.DoorX != 1234 || d.DoorY != -567 {
				t.Fatalf("caster door portal not round-tripped: %+v", d)
			}
			if got[1].HasDoor {
				t.Fatalf("doorless member must round-trip HasDoor=false: %+v", got[1])
			}
		})
	}
}
