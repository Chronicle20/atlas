package writer

import (
	"atlas-login/character"
	"atlas-login/maps/location"
	socketmodel "atlas-login/socket/model"
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


func CharacterListBody(characters []character.Model, worldId world.Id, status int, pic string, availableCharacterSlots int16, characterSlots int16) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			entries := make([]packetmodel.CharacterListEntry, len(characters))
			for i, c := range characters {
				entries[i] = toCharacterListEntry(l, ctx, c, false)
			}
			return charpkt.NewCharacterList(byte(status), entries, pic != "", uint32(characterSlots)).Encode(l, ctx)(options)
		}
	}
}

func toCharacterListEntry(l logrus.FieldLogger, ctx context.Context, c character.Model, viewAll bool) packetmodel.CharacterListEntry {
	var petIds [3]uint64
	for _, p := range c.Pets() {
		petIds[p.Slot()] = p.CashId()
	}

	mapId := _map.Id(0)
	f, err := location.GetField(l, ctx, c.Id())
	if err != nil {
		l.WithError(err).Warnf("character_list: atlas-maps location unreachable for [%d]; rendering map=0.", c.Id())
	} else {
		mapId = f.MapId()
	}

	stats := packetmodel.NewCharacterStatistics(
		c.Id(), c.Name(), c.Gender(), c.SkinColor(), c.Face(), c.Hair(),
		petIds, c.Level(), uint16(c.JobId()),
		c.Strength(), c.Dexterity(), c.Intelligence(), c.Luck(),
		c.Hp(), c.MaxHp(), c.Mp(), c.MaxMp(),
		c.Ap(), c.HasSPTable(), c.RemainingSp(),
		c.Experience(), c.Fame(), c.GachaponExperience(),
		uint32(mapId), c.SpawnPoint(),
	)

	avatar := socketmodel.NewFromCharacter(c, false)

	return packetmodel.NewCharacterListEntry(stats, avatar, viewAll, c.Gm(), c.Rank(), c.RankMove(), c.JobRank(), c.JobRankMove())
}
