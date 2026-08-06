package handler

import (
	"atlas-channel/character/buff"
	"atlas-channel/character/statreset"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// CancelDebuffHandleFunc handles the client's CANCEL_DEBUFF nudge
// (CWvsContext::CheckTemporaryStatDuration): "one of my temporary stats looks
// expired — please re-evaluate me."
//
// The packet carries no payload, so the server cannot and must not cancel by
// name. It emits a per-character EXPIRE command; atlas-buffs owns the decision
// about what has genuinely lapsed and answers with the existing EXPIRED events,
// which already flow back through the buff consumer to the existing
// CharacterBuffCancel writer. A sweep that finds nothing lapsed emits nothing
// (FR-2.9 / NFR-2.1).
//
// Throttle FIRST, then emit: the amplification NFR-2 bounds is the Kafka
// message, so the guard must sit before the produce. Dropped nudges log at
// Debug — the unhandled-op line this replaces produced ~1,500 Info lines in
// under four minutes on one wedged client (NFR-4). (task-190 FR-2.5/2.10)
func CancelDebuffHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := charsb.CancelDebuff{}
		p.Decode(l, ctx)(r, readerOptions)

		t := tenant.MustFromContext(ctx)
		if !statreset.GetRegistry().Allow(t, s.CharacterId(), time.Now()) {
			l.Debugf("Throttled CANCEL_DEBUFF for character [%d].", s.CharacterId())
			return
		}

		if err := buff.NewProcessor(l, ctx).Expire(s.Field(), s.CharacterId()); err != nil {
			l.WithError(err).Errorf("Unable to request buff expiry sweep for character [%d].", s.CharacterId())
		}
	}
}
