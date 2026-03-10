package character

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestCharacterInfoEncode(t *testing.T) {
	pets := []InfoPet{
		{Slot: 0, TemplateId: 5000001, Name: "Kitty", Level: 10, Closeness: 100, Fullness: 50},
	}
	input := NewCharacterInfo(12345, 50, 100, 10, "TestGuild", pets, []uint32{50200004}, 1142007)
	l, _ := testlog.NewNullLogger()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			encoded := input.Encode(l, ctx)(nil)
			if len(encoded) == 0 {
				t.Error("expected non-empty encoded bytes")
			}
		})
	}
}
