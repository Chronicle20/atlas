package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/clientbound/GuildInfo version=gms_v87 ida=0xacf7d3
// packet-audit:verify packet=guild/clientbound/GuildInfo version=gms_v83 ida=0xa37490
// packet-audit:verify packet=guild/clientbound/GuildInfo version=gms_v95 ida=0x4fb760
// packet-audit:verify packet=guild/clientbound/GuildInfo version=gms_v84 ida=0xa82e2b
func TestInfoInGuildRoundTrip(t *testing.T) {
	members := []GuildMemberInfo{
		{CharacterId: 1001, Name: "MemberOne", JobId: 100, Level: 50, Title: 0, Online: true, Signature: 12345, AllianceTitle: 1},
		{CharacterId: 1002, Name: "MemberTwo", JobId: 200, Level: 70, Title: 1, Online: false, Signature: 67890, AllianceTitle: 0},
	}
	titles := [5]string{"Master", "Jr. Master", "Member", "Newbie", "Recruit"}
	input := NewInfo(true, 500, "TestGuild", titles, members, 100, 5, 4, 3, 2, "Welcome!", 9999, 42)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := Info{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestInfoNotInGuildRoundTrip(t *testing.T) {
	input := NewInfo(false, 0, "", [5]string{}, nil, 0, 0, 0, 0, 0, "", 0, 0)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := Info{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
