package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestShopOperationSimple(t *testing.T) {
	input := NewShopOperationSimple(0)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ShopOperationSimple{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationGenericError(t *testing.T) {
	input := NewShopOperationGenericError(11)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ShopOperationGenericError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationGenericErrorWithReason(t *testing.T) {
	input := NewShopOperationGenericErrorWithReason(12, "test error")
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ShopOperationGenericError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestShopOperationLevelRequirement(t *testing.T) {
	input := NewShopOperationLevelRequirement(9, 200)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ShopOperationLevelRequirement{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
