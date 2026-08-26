package handler

import (
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	monstersb "github.com/Chronicle20/atlas/libs/atlas-packet/monster/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// autoAggroMirrorLookupFn resolves the live-mirror entry for a mob. Injected so
// the handler's admission logic is testable without a populated singleton.
var autoAggroMirrorLookupFn = func(t tenant.Model, uniqueId uint32) (monster.LiveEntry, bool) {
	return monster.GetLiveMirror().Lookup(t, uniqueId)
}

// autoAggroEmitFn forwards an admitted claim as SET_AGGRO.
var autoAggroEmitFn = func(l logrus.FieldLogger, ctx context.Context, f field.Model, monsterId uint32, characterId uint32) error {
	return monster.NewProcessor(l, ctx).SetAggro(f, monsterId, characterId)
}

// AutoAggroHandleFunc decodes AUTO_AGGRO (CMob::ApplyControl) and forwards it as
// SET_AGGRO. Every check here is cheap and local — the session's character and
// field, the client's own proximity score, the live mirror, and the rate gate.
// None of them is the authority: atlas-monsters owns the monster registry and
// re-applies every gate (FR-3.3). A rejected claim is a debug-logged drop with
// no response packet; AUTO_AGGRO has no client-visible failure path (FR-3.4).
func AutoAggroHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		var p monstersb.AutoAggro
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		if s.CharacterId() == 0 {
			l.Debugf("Dropping AUTO_AGGRO for mob [%d]: no character on session.", p.MobId())
			return
		}

		if p.Distance() > monster.AutoAggroProximityThreshold {
			l.Debugf("Dropping AUTO_AGGRO for mob [%d], character [%d]: distance [%d] exceeds proximity threshold.", p.MobId(), s.CharacterId(), p.Distance())
			return
		}

		e, ok := autoAggroMirrorLookupFn(tenant.MustFromContext(ctx), p.MobId())
		if !ok {
			l.Debugf("Dropping AUTO_AGGRO for mob [%d], character [%d]: mob absent from live mirror.", p.MobId(), s.CharacterId())
			return
		}

		if e.Field.Id() != s.Field().Id() {
			l.Debugf("Dropping AUTO_AGGRO for mob [%d], character [%d]: mob is in another field.", p.MobId(), s.CharacterId())
			return
		}

		if !monster.GetAutoAggroGate().Admit(tenant.MustFromContext(ctx), s.CharacterId(), p.MobId(), e.ControllerHasAggro, time.Now()) {
			l.Debugf("Dropping AUTO_AGGRO for mob [%d], character [%d]: rate gate closed.", p.MobId(), s.CharacterId())
			return
		}

		_ = autoAggroEmitFn(l, ctx, s.Field(), p.MobId(), s.CharacterId())
	}
}
