package writer

import (
	"atlas-login/character"
	socketmodel "atlas-login/socket/model"
	"context"

	"github.com/Chronicle20/atlas-constants/world"
	charpkt "github.com/Chronicle20/atlas-packet/character"
	packetmodel "github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


func CharacterListBody(characters []character.Model, worldId world.Id, status int, pic string, availableCharacterSlots int16, characterSlots int16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			entries := make([]packetmodel.CharacterListEntry, len(characters))
			for i, c := range characters {
				entries[i] = toCharacterListEntry(c)
			}
			return charpkt.NewCharacterList(byte(status), entries, pic != "", uint32(characterSlots)).Encode(l, ctx)(options)
		}
	}
}

func toCharacterListEntry(c character.Model) packetmodel.CharacterListEntry {
	var petIds [3]uint64
	for _, p := range c.Pets() {
		petIds[p.Slot()] = p.CashId()
	}

	stats := packetmodel.NewCharacterStatistics(
		c.Id(), c.Name(), c.Gender(), c.SkinColor(), c.Face(), c.Hair(),
		petIds, c.Level(), uint16(c.JobId()),
		c.Strength(), c.Dexterity(), c.Intelligence(), c.Luck(),
		c.Hp(), c.MaxHp(), c.Mp(), c.MaxMp(),
		c.Ap(), c.HasSPTable(), c.RemainingSp(),
		c.Experience(), c.Fame(), c.GachaponExperience(),
		uint32(c.MapId()), c.SpawnPoint(),
	)

	avatar := socketmodel.NewFromCharacter(c, false)

	return packetmodel.NewCharacterListEntry(stats, avatar, c.Gm(), c.Rank(), c.RankMove(), c.JobRank(), c.JobRankMove())
}
