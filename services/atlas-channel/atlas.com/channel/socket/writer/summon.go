package writer

import (
	summoncb "github.com/Chronicle20/atlas/libs/atlas-packet/summon/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// SummonSpawnBody builds the SummonSpawn clientbound packet for a freshly
// created summon, broadcast to every session in the owner's map.
func SummonSpawnBody(ownerCharacterId uint32, summonId uint32, skillId uint32, level byte, x int16, y int16, stance byte, movementType byte, puppet bool, animated bool) packet.Encode {
	return summoncb.NewSummonSpawn(ownerCharacterId, summonId, skillId, level, x, y, stance, movementType, puppet, animated).Encode
}

// SummonRemoveBody builds the SummonRemove clientbound packet for a destroyed
// summon, broadcast to every session in the owner's map.
func SummonRemoveBody(ownerCharacterId uint32, summonId uint32, animated bool) packet.Encode {
	return summoncb.NewSummonRemove(ownerCharacterId, summonId, animated).Encode
}
