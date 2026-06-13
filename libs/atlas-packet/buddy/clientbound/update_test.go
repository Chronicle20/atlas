package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=buddy/clientbound/BuddyUpdate version=gms_v83 ida=0xa3f2e8
// packet-audit:verify packet=buddy/clientbound/BuddyUpdate version=gms_v87 ida=0xad7ae5
// packet-audit:verify packet=buddy/clientbound/BuddyUpdate version=gms_v95 ida=0xa12630
func TestBuddyUpdateRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewBuddyUpdate(8, 1000, "TestPlayer", "Default Group", 1, false)
			output := Update{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.CharacterName() != input.CharacterName() {
				t.Errorf("characterName: got %v, want %v", output.CharacterName(), input.CharacterName())
			}
			if output.Group() != input.Group() {
				t.Errorf("group: got %v, want %v", output.Group(), input.Group())
			}
			if output.ChannelId() != input.ChannelId() {
				t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
			}
			if output.InShop() != input.InShop() {
				t.Errorf("inShop: got %v, want %v", output.InShop(), input.InShop())
			}
		})
	}
}
