package handler

import (
	"atlas-channel/character"
	skill2 "atlas-channel/character/skill"
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// CUserLocal::DoActiveSkill_Prepare (serverbound)
const CharacterSkillPrepareHandle = "CharacterSkillPrepareHandle"

// shouldBroadcastKeydown reports whether a keydown-skill prepare/cancel packet
// should be relayed to other sessions in the map.
//
// Conditions (D4):
//   - The character owns the skill (present in their skill book at level > 0).
//   - The skill resolves (via ctx's tenant version set) to a keydown-skill
//     Identity per IsKeyDownSkillIdentity. skillId is a raw wire id, and
//     keydown membership is version-scoped (task-187): wire 5101004 is the
//     keydown Brawler Corkscrew Blow at v0.62+ but the non-keydown SuperGM
//     Hide at v0.48.
func shouldBroadcastKeydown(ctx context.Context, skills []skill2.Model, skillId uint32) bool {
	t := tenant.MustFromContext(ctx)
	set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
	id, ok := set.Skill.Resolve(skill.Id(skillId))
	if !ok || !skill.IsKeyDownSkillIdentity(id) {
		return false
	}
	for _, sm := range skills {
		if sm.Id() == skill.Id(skillId) && sm.Level() > 0 {
			return true
		}
	}
	return false
}

// CharacterSkillPrepareHandleFunc handles the serverbound DoActiveSkill_Prepare
// packet. On validation pass it relays a foreign prepare packet to other map
// sessions. On miss it logs at debug level and returns (D3: no session destroy).
func CharacterSkillPrepareHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		info := packetmodel.NewSkillPrepareInfo()
		info.Decode(l, ctx)(r, readerOptions)

		cp := character.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.SkillModelDecorator)(s.CharacterId())
		if err != nil {
			l.Debugf("Character [%d] skill prepare [%d]: character not found, skipping broadcast.", s.CharacterId(), info.SkillId())
			return
		}

		if !shouldBroadcastKeydown(ctx, c.Skills(), info.SkillId()) {
			l.Debugf("Character [%d] skill prepare [%d]: not a keydown skill or not owned, skipping broadcast.", s.CharacterId(), info.SkillId())
			return
		}

		_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), AnnounceForeignSkillPrepare(l)(ctx)(wp)(s.CharacterId(), *info))
	}
}
