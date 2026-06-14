package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=buddy/clientbound/BuddyChannelChange version=gms_v83 ida=0xa3f2e8
// packet-audit:verify packet=buddy/clientbound/BuddyChannelChange version=gms_v87 ida=0xad7ae5
// packet-audit:verify packet=buddy/clientbound/BuddyChannelChange version=gms_v95 ida=0xa12630
// packet-audit:verify packet=buddy/clientbound/BuddyChannelChange version=jms_v185 ida=0xb2a873
// packet-audit:verify packet=buddy/clientbound/BuddyChannelChange version=gms_v84 ida=0xa8ada2
func TestBuddyChannelChangeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewBuddyChannelChange(14, 1000, 3)
			output := ChannelChange{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.ChannelId() != input.ChannelId() {
				t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
			}
		})
	}
}
