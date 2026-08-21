package handler

import (
	"atlas-channel/character"
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-packet/monster/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var monsterBombCharacterFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (character.Model, error) {
	cp := character.NewProcessor(l, ctx)
	return cp.GetById()(characterId)
}

var monsterBombSelfDestructFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32, characterId uint32) error {
	return monster.NewProcessor(l, ctx).SelfDestruct(f, monsterId, characterId)
}

// MonsterBombHandleFunc handles CMob::TryFirstSelfDestruction: the controller
// reports that a self-destructing mob's first-attack body rect intersected the
// local user. The decoded id is the mob OBJECT id (v95 @0x640ee0 encodes
// GetMobID(this)), the same identifier MonsterDestroy carries as uniqueId —
// never a template id (task-253 design §2.3).
//
// The channel keeps only the two guards it can answer from state it already
// holds: the reporter must be alive, and the target must be in the reporter's
// field. Whether the target actually carries a selfDestruction block is
// atlas-monsters' call — it is the authority on monster lifecycle, already
// holds the data behind a cache, and must re-check anyway (design D8).
//
// There is no failure packet and no enableActions: TryFirstSelfDestruction is
// fire-and-forget with no client-side response state, so a rejection is a log
// line and nothing else. Duplicate reports from several clients in the field
// are expected and harmless — Registry.SelfDestruct makes the detonation
// exactly-once server-side.
func MonsterBombHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := serverbound.MonsterBomb{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		c, err := monsterBombCharacterFunc(l, ctx, s.CharacterId())
		if err != nil {
			l.WithError(err).Debugf("MONSTER_BOMB: unable to resolve character [%d]; dropping report for monster [%d].", s.CharacterId(), p.MobId())
			return
		}
		if c.Hp() == 0 {
			l.Debugf("MONSTER_BOMB: character [%d] reported monster [%d] while dead; dropping.", s.CharacterId(), p.MobId())
			return
		}

		entry, ok := monster.GetLiveMirror().Lookup(tenant.MustFromContext(ctx), p.MobId())
		if !ok {
			l.Debugf("MONSTER_BOMB: monster [%d] is not in the live mirror; dropping report from character [%d].", p.MobId(), s.CharacterId())
			return
		}
		if entry.Field.Id() != s.Field().Id() {
			l.Debugf("MONSTER_BOMB: monster [%d] is not in the reporter's field; dropping report from character [%d].", p.MobId(), s.CharacterId())
			return
		}

		if err := monsterBombSelfDestructFunc(l, ctx, s.Field(), p.MobId(), s.CharacterId()); err != nil {
			l.WithError(err).Errorf("MONSTER_BOMB: unable to request self-destruct of monster [%d].", p.MobId())
		}
	}
}
