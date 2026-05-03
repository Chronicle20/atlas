package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
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

// AnnounceSkillSpecial broadcasts the SKILL_SPECIAL CharacterEffect to the
// caster's own session. Used by passive procs (e.g., MP Eater) to play the
// skill's "special" visual without re-broadcasting a full skill-use cast.
func AnnounceSkillSpecial(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(skillId uint32) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(skillId uint32) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(skillId uint32) model2.Operator[session.Model] {
			return func(skillId uint32) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectWriter)(charpkt.CharacterSkillSpecialEffectBody(skillId))
			}
		}
	}
}

// AnnounceForeignSkillSpecial is the same broadcast targeted at other sessions
// on the caster's map.
func AnnounceForeignSkillSpecial(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, skillId uint32) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, skillId uint32) model2.Operator[session.Model] {
			return func(characterId uint32, skillId uint32) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(charcb.CharacterEffectForeignWriter)(charpkt.CharacterSkillSpecialEffectForeignBody(characterId, skillId))
			}
		}
	}
}
