package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/serverbound/GuildSetEmblem version=gms_v95 ida=0x52d8c0
// packet-audit:verify packet=guild/serverbound/GuildSetEmblem version=jms_v185 ida=0x56e325
// packet-audit:verify packet=guild/serverbound/GuildSetEmblem version=gms_v87 ida=0x55874c
// packet-audit:verify packet=guild/serverbound/GuildSetEmblem version=gms_v83 ida=0x530f2f
func TestSetEmblemRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SetEmblem{logoBackground: 3, logoBackgroundColor: 5, logo: 7, logoColor: 9}
			output := SetEmblem{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.LogoBackground() != input.LogoBackground() {
				t.Errorf("logoBackground: got %v, want %v", output.LogoBackground(), input.LogoBackground())
			}
			if output.LogoBackgroundColor() != input.LogoBackgroundColor() {
				t.Errorf("logoBackgroundColor: got %v, want %v", output.LogoBackgroundColor(), input.LogoBackgroundColor())
			}
			if output.Logo() != input.Logo() {
				t.Errorf("logo: got %v, want %v", output.Logo(), input.Logo())
			}
			if output.LogoColor() != input.LogoColor() {
				t.Errorf("logoColor: got %v, want %v", output.LogoColor(), input.LogoColor())
			}
		})
	}
}
