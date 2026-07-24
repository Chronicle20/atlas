package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// SkillPrepareHandle is the handler/codec name for the serverbound
// DoActiveSkill_Prepare packet.
const SkillPrepareHandle = "SkillPrepare"

// SkillPrepare is the serverbound skill-prepare request the client sends from
// CUserLocal::DoActiveSkill_Prepare. It shares the one wire structure decoded by
// model.SkillPrepareInfo (skillId u32, level u8, action u16, actionSpeed u8 [,
// swallowMobId u32 on GMS v95+/JMS for skillId 33101005]).
//
// This thin wrapper exists so the registry op links to a distinct codec for the
// coverage matrix (one packet/evidence per op), exactly as the serverbound attack
// wrappers wrap the shared model.AttackInfo. The wrapper holds a SkillPrepareInfo
// and delegates Encode/Decode; the analyzer recurses into SkillPrepareInfo for the
// read order. The atlas-channel handler decodes the same model.SkillPrepareInfo
// directly — the wire structure is identical, so the wrapper verifies the same bytes.
// packet-audit:fname CUserLocal::DoActiveSkill_Prepare
type SkillPrepare struct {
	info model.SkillPrepareInfo
}

func NewSkillPrepare() SkillPrepare {
	return SkillPrepare{info: *model.NewSkillPrepareInfo()}
}

func (m SkillPrepare) Info() model.SkillPrepareInfo { return m.info }
func (m SkillPrepare) Operation() string            { return SkillPrepareHandle }

func (m SkillPrepare) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	return m.info.Encode(l, ctx)
}

func (m *SkillPrepare) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return m.info.Decode(l, ctx)
}
