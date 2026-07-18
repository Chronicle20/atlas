package hpsync

import (
	"atlas-channel/character"
	"atlas-channel/party"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	partycb "github.com/Chronicle20/atlas/libs/atlas-packet/party/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// loadPartyCharacterFunc loads the party-decorated character whose HP gauges
// are being synced. Seam tests can replace.
var loadPartyCharacterFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (character.Model, error) {
	cp := character.NewProcessor(l, ctx)
	return cp.GetById(cp.PartyDecorator)(characterId)
}

// loadCharacterFunc loads an individual party member's character record. Seam
// tests can replace.
var loadCharacterFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (character.Model, error) {
	return character.NewProcessor(l, ctx).GetById()(characterId)
}

// announceMemberHPFunc sends subjectId's HP/MaxHp as a PartyMemberHP packet to
// the session of toCharacterId on channel ch, if that session is present on
// this channel (no-op otherwise). Seam tests can replace.
var announceMemberHPFunc = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, ch channel.Model, toCharacterId uint32, subjectId uint32, hp uint16, maxHp uint16) error {
	return session.NewProcessor(l, ctx).IfPresentByCharacterId(ch)(toCharacterId, session.Announce(l)(ctx)(wp)(partycb.PartyMemberHPWriter)(partycb.NewPartyMemberHP(subjectId, hp, maxHp).Encode))
}

// Sync pushes the bidirectional party-member HP gauges for characterId within
// field f:
//   - characterId's current HP to every other in-map party member, and
//   - every other in-map party member's current HP back to characterId.
//
// The v83 PARTYDATA struct carries no HP, so a client's party HP gauges are
// populated exclusively by the PartyMemberHP packet. This is the canonical
// sync used both on map entry (spawn) and on party join. Without the join
// call, a player who joins a party while already standing in a map (and the
// existing members likewise already standing there) sees every other gauge
// stuck at 0 until that member's next HP change, because the spawn-time sync
// short-circuits for characters who were not yet in a party when they entered
// the map.
//
// Each announce is best-effort: members not present on this channel are
// silently skipped (the channel-scoped session lookup no-ops), and a failed
// per-member character fetch is logged and skipped. Returns nil when the
// character is not in a party.
func Sync(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, characterId uint32) error {
	cd, err := loadPartyCharacterFunc(l, ctx, characterId)
	if err != nil {
		return err
	}
	if !cd.InParty() {
		return nil
	}

	pmp := model.FixedProvider(cd.Party())
	imf := party.OtherMemberInMap(f, characterId)
	otherIds, err := party.MemberToMemberIdMapper(party.FilteredMemberProvider(imf)(pmp))()
	if err != nil {
		return err
	}

	ch := f.Channel()
	return model.ForEachSlice(model.FixedProvider(otherIds), func(oid uint32) error {
		// This character's HP to the other in-map member.
		if aErr := announceMemberHPFunc(l, ctx, wp, ch, oid, characterId, cd.Hp(), cd.MaxHp()); aErr != nil {
			l.WithError(aErr).Debugf("hpsync: failed to announce character [%d] HP to member [%d].", characterId, oid)
		}

		// The other in-map member's HP back to this character.
		oc, oErr := loadCharacterFunc(l, ctx, oid)
		if oErr != nil {
			if errors.Is(oErr, requests.ErrNotFound) {
				l.Warnf("hpsync: skipping party HP for stale character [%d].", oid)
				return nil
			}
			return oErr
		}
		return announceMemberHPFunc(l, ctx, wp, ch, characterId, oid, oc.Hp(), oc.MaxHp())
	}, model.ParallelExecute())
}
