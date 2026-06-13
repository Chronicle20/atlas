package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=login/clientbound/ServerListRecommendations version=gms_v83 ida=0x5f8340
// packet-audit:verify packet=login/clientbound/ServerListRecommendations version=gms_v84 ida=0x60d2ba
// packet-audit:verify packet=login/clientbound/ServerListRecommendations version=gms_v87 ida=0x62fad6
// packet-audit:verify packet=login/clientbound/ServerListRecommendations version=gms_v95 ida=0x5d7280
// packet-audit:verify packet=login/clientbound/ServerListRecommendations version=jms_v185 ida=0x66e6f1
func TestServerListRecommendationsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ServerListRecommendations{
				recommendations: []model.WorldRecommendation{
					model.NewWorldRecommendation(0, "Most popular"),
					model.NewWorldRecommendation(1, "New world"),
				},
			}
			output := ServerListRecommendations{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if len(output.Recommendations()) != len(input.Recommendations()) {
				t.Errorf("recommendations length: got %v, want %v", len(output.Recommendations()), len(input.Recommendations()))
			}
			for i, r := range output.Recommendations() {
				if r.WorldId() != input.Recommendations()[i].WorldId() {
					t.Errorf("recommendation[%d].worldId: got %v, want %v", i, r.WorldId(), input.Recommendations()[i].WorldId())
				}
				if r.Reason() != input.Recommendations()[i].Reason() {
					t.Errorf("recommendation[%d].reason: got %v, want %v", i, r.Reason(), input.Recommendations()[i].Reason())
				}
			}
		})
	}
}
