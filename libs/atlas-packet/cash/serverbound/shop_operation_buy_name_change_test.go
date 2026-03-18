package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestShopOperationBuyNameChangeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ShopOperationBuyNameChange{serialNumber: 12345, oldName: "OldPlayer", newName: "NewPlayer"}
			output := ShopOperationBuyNameChange{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
			if output.OldName() != input.OldName() {
				t.Errorf("oldName: got %v, want %v", output.OldName(), input.OldName())
			}
			if output.NewName() != input.NewName() {
				t.Errorf("newName: got %v, want %v", output.NewName(), input.NewName())
			}
		})
	}
}
