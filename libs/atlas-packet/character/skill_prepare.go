package character

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

// CharacterSkillPrepareForeignBody returns a packet.Encode that serializes a
// clientbound remote skill-prepare relay packet. Wire-spec §3: charId u32,
// skillId u32, level u8, action u16, actionSpeed u8.
// The opcode is NOT encoded here — it is config-resolved by the caller.
func CharacterSkillPrepareForeignBody(characterId uint32, info model.SkillPrepareInfo) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		m := clientbound.NewSkillPrepareForeign(
			characterId,
			info.SkillId(),
			info.Level(),
			info.Action(),
			info.ActionSpeed(),
		)
		return m.Encode(l, ctx)
	}
}

// CharacterSkillCancelForeignBody returns a packet.Encode that serializes a
// clientbound remote skill-cancel relay packet. Wire-spec §4: charId u32, skillId u32.
// The opcode is NOT encoded here — it is config-resolved by the caller.
func CharacterSkillCancelForeignBody(characterId uint32, skillId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		m := clientbound.NewSkillCancelForeign(characterId, skillId)
		return m.Encode(l, ctx)
	}
}
