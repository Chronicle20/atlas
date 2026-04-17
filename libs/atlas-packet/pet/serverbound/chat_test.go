package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestChatRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ChatRequest{petId: 12345, updateTime: 100, nType: 1, nAction: 2, msg: "meow"}
			output := ChatRequest{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.PetId() != input.PetId() {
				t.Errorf("petId: got %v, want %v", output.PetId(), input.PetId())
			}
			if output.NType() != input.NType() {
				t.Errorf("nType: got %v, want %v", output.NType(), input.NType())
			}
			if output.NAction() != input.NAction() {
				t.Errorf("nAction: got %v, want %v", output.NAction(), input.NAction())
			}
			if output.Msg() != input.Msg() {
				t.Errorf("msg: got %v, want %v", output.Msg(), input.Msg())
			}
		})
	}
}
