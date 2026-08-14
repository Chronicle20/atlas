package handler

import (
	"atlas-channel/mist"
	"atlas-channel/party"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// shieldedBySmoke reports whether any of the protection mists covering the
// character's position belongs to the character or to one of their online
// party members -- the exact conjunction the client evaluates in
// CAffectedAreaPool::IsSmokeAreaByPoint (v95 @0x434f40): nType == 2, started,
// same phase, owner in the party array or the local character, and the point
// inside the rect. The first four are decided when the protection is
// registered; this is the ownership half.
//
// partyIds is a thunk so the party REST call is made only when a covering
// mist is owned by someone else -- on an ordinary hit there is nothing
// covering the character and no lookup happens at all.
func shieldedBySmoke(covering []mist.Protection, characterId uint32, partyIds func() []uint32) bool {
	if len(covering) == 0 {
		return false
	}
	for _, p := range covering {
		if p.OwnerId() == characterId {
			return true
		}
	}
	ids := partyIds()
	if len(ids) == 0 {
		return false
	}
	inParty := make(map[uint32]bool, len(ids))
	for _, id := range ids {
		inParty[id] = true
	}
	for _, p := range covering {
		if inParty[p.OwnerId()] {
			return true
		}
	}
	return false
}

// newSmokeCheck builds the production inProtectiveMist dependency: the
// channel-local protection registry plus a lazily-resolved online party.
//
// Party membership is resolved at HIT time, not snapshot at cast time,
// because the client resolves it at hit time too (the array passed to
// IsSmokeAreaByPoint is filled immediately before the call by
// CWvsContext::GetOnlinePartyMemberID, v95 @0x93455c). A server snapshot
// would visibly disagree with the player's own screen.
func newSmokeCheck(l logrus.FieldLogger, ctx context.Context, t tenant.Model) func(f field.Model, characterId uint32, x, y int16) bool {
	return func(f field.Model, characterId uint32, x, y int16) bool {
		covering := mist.GetProtectionRegistry().Covering(t, f, x, y, time.Now())
		return shieldedBySmoke(covering, characterId, func() []uint32 {
			p, err := party.NewProcessor(l, ctx).GetByMemberId(characterId)
			if err != nil {
				// No party is the ordinary case for a solo player, not an
				// error: the caster's own mist has already been checked.
				l.WithError(err).Debugf("Smoke check: no party for character [%d].", characterId)
				return nil
			}
			ids := make([]uint32, 0, len(p.Members()))
			for _, m := range p.Members() {
				if !m.Online() {
					continue
				}
				ids = append(ids, m.Id())
			}
			return ids
		})
	}
}
