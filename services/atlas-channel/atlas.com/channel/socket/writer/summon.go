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

// SummonMoveBody builds the SummonMove clientbound packet for a moved summon,
// rebroadcasting the raw movement blob byte-faithfully to other sessions.
func SummonMoveBody(ownerCharacterId uint32, summonId uint32, startX int16, startY int16, rawMovement []byte) packet.Encode {
	return summoncb.NewSummonMove(ownerCharacterId, summonId, startX, startY, rawMovement).Encode
}

// SummonAttackBody builds the SummonAttack clientbound packet for a summon
// attack, broadcasting the clamped per-target damage to other sessions in the
// owner's map. targets carry {monsterOid, clamped damage} pairs.
func SummonAttackBody(ownerCharacterId uint32, summonId uint32, direction byte, targets []summoncb.SummonAttackTarget) packet.Encode {
	return summoncb.NewSummonAttack(ownerCharacterId, summonId, direction, targets).Encode
}

// SummonDamageBody builds the SummonDamage clientbound packet for a puppet
// summon that took monster damage, broadcast to other sessions in the owner's
// map so they render the floating damage number.
func SummonDamageBody(ownerCharacterId uint32, summonId uint32, damage int32, monsterIdFrom uint32) packet.Encode {
	return summoncb.NewSummonDamage(ownerCharacterId, summonId, uint32(damage), monsterIdFrom).Encode
}

// SummonSkillBody builds the SummonSkill clientbound packet for a Beholder aura
// skill pulse, broadcast map-wide (including the owner) so every client renders
// the heal/buff visual. summonSkillId is the summon's source skill id.
func SummonSkillBody(ownerCharacterId uint32, summonSkillId uint32, newStance byte) packet.Encode {
	return summoncb.NewSummonSkill(ownerCharacterId, summonSkillId, newStance).Encode
}
