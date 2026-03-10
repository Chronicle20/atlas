package character

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestCharacterSpawnEncode(t *testing.T) {
	avatar := model.Avatar{}
	cts := &model.CharacterTemporaryStat{}
	guild := GuildEmblem{Name: "TestGuild"}
	input := NewCharacterSpawn(12345, 50, "TestChar", guild, cts, 100, avatar, nil, true, 100, 200, 6)
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
