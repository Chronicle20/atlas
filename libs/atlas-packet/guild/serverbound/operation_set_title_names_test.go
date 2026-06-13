package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=guild/serverbound/GuildSetTitleNames version=gms_v95 ida=0x0
// packet-audit:verify packet=guild/serverbound/GuildSetTitleNames version=jms_v185 ida=ABSENT
func TestSetTitleNamesRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SetTitleNames{titles: []string{"Master", "Jr. Master", "Member", "Rookie", "Intern"}}
			output := SetTitleNames{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Titles()) != len(input.Titles()) {
				t.Fatalf("titles length: got %v, want %v", len(output.Titles()), len(input.Titles()))
			}
			for i, title := range output.Titles() {
				if title != input.Titles()[i] {
					t.Errorf("titles[%d]: got %v, want %v", i, title, input.Titles()[i])
				}
			}
		})
	}
}
