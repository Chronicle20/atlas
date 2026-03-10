package field

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

func TestFieldEffectWeatherStart(t *testing.T) {
	input := NewFieldEffectWeatherStart(5010000, "It's raining!")
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestFieldEffectWeatherEnd(t *testing.T) {
	input := NewFieldEffectWeatherEnd(5010000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
