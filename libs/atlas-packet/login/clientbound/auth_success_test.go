package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestAuthSuccessRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AuthSuccess{
				accountId: 1001,
				name:      "TestUser",
				gender:    1,
				usesPin:   true,
				pic:       "123456",
			}
			output := AuthSuccess{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.AccountId() != input.AccountId() {
				t.Errorf("accountId: got %v, want %v", output.AccountId(), input.AccountId())
			}
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Gender() != input.Gender() {
				t.Errorf("gender: got %v, want %v", output.Gender(), input.Gender())
			}
			if v.Region == "GMS" && v.MajorVersion > 12 {
				if output.UsesPin() != input.UsesPin() {
					t.Errorf("usesPin: got %v, want %v", output.UsesPin(), input.UsesPin())
				}
			}
		})
	}
}
