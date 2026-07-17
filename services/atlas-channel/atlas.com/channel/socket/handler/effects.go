package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

// AnnounceSkillUse is the self-facing CharacterEffect broadcast for a
// successful skill cast. Lifted out of character_skill_use.go so
// per-skill handler subpackages can reuse it without importing
// socket/handler back-channel internals.
func AnnounceSkillUse(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
			return func(skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectWriter)(charpkt.CharacterSkillUseEffectBody(skillId, characterLevel, skillLevel, false, false, false))
			}
		}
	}
}

// AnnounceForeignSkillUse is the same broadcast targeted at other
// sessions on the caster's map.
func AnnounceForeignSkillUse(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
			return func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectForeignWriter)(charpkt.CharacterSkillUseEffectForeignBody(characterId, skillId, characterLevel, skillLevel, false, false, false))
			}
		}
	}
}

// AnnounceBerserkEffect is the self-facing CharacterEffect broadcast carrying
// the Dark Knight Berserk aura flag. Identical to AnnounceSkillUse except the
// darkForceEffect bool is threaded through: the packet encoder writes it as a
// trailing byte only for skill.DarkKnightBerserkId (effect_body.go derives
// that gate from the skill id).
func AnnounceBerserkEffect(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
			return func(skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectWriter)(charpkt.CharacterSkillUseEffectBody(skillId, characterLevel, skillLevel, active, false, false))
			}
		}
	}
}

// AnnounceForeignBerserkEffect is the same broadcast targeted at other
// sessions on the Dark Knight's map.
func AnnounceForeignBerserkEffect(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
			return func(characterId uint32, skillId uint32, characterLevel byte, skillLevel byte, active bool) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectForeignWriter)(charpkt.CharacterSkillUseEffectForeignBody(characterId, skillId, characterLevel, skillLevel, active, false, false))
			}
		}
	}
}

// AnnounceForeignSkillPrepare broadcasts a keydown-skill prepare packet to all
// other sessions on the caster's map. Foreign-only: the caster renders its own aura.
func AnnounceForeignSkillPrepare(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, info packetmodel.SkillPrepareInfo) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, info packetmodel.SkillPrepareInfo) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, info packetmodel.SkillPrepareInfo) model2.Operator[session.Model] {
			return func(characterId uint32, info packetmodel.SkillPrepareInfo) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterSkillPrepareForeignWriter)(charpkt.CharacterSkillPrepareForeignBody(characterId, info))
			}
		}
	}
}

// AnnounceForeignSkillCancel broadcasts a keydown-skill cancel packet to all
// other sessions on the caster's map. Foreign-only: the caster renders its own aura.
func AnnounceForeignSkillCancel(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, skillId uint32) model2.Operator[session.Model] {
			return func(characterId uint32, skillId uint32) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterSkillCancelForeignWriter)(charpkt.CharacterSkillCancelForeignBody(characterId, skillId))
			}
		}
	}
}
