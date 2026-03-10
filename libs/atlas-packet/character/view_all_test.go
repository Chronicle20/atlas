package character

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestCharacterViewAllCountRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterViewAllCount{code: 3, worldCount: 5, unk: 0}
			output := CharacterViewAllCount{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
			if output.WorldCount() != input.WorldCount() {
				t.Errorf("worldCount: got %v, want %v", output.WorldCount(), input.WorldCount())
			}
		})
	}
}

func TestCharacterViewAllSearchFailedRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterViewAllSearchFailed{code: 4}
			output := CharacterViewAllSearchFailed{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
		})
	}
}

func TestCharacterViewAllErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterViewAllError{code: 5}
			output := CharacterViewAllError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
		})
	}
}
