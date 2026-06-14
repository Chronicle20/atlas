package handler

import (
	"atlas-channel/door"
	"atlas-channel/party"
	"atlas-channel/portal"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	doorsb "github.com/Chronicle20/atlas/libs/atlas-packet/door/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// doorsByOwnerFunc lists the owner's live door(s) via the atlas-doors by-owner
// REST route. Because a door is keyed on its owner (not a single field), this
// resolves from EITHER side — the requester can be standing on the door's AREA
// field or its TOWN map. Declared as a package var so tests can inject a fake.
var doorsByOwnerFunc = func(l logrus.FieldLogger, ctx context.Context, ownerId uint32) ([]door.Model, error) {
	return door.NewProcessor(l, ctx).GetByOwner(ownerId)
}

// partyMemberIdsFunc returns the character ids of the party that the given
// character belongs to (empty slice when not in a party / lookup fails).
// Declared as a package var so tests can inject a fake.
var partyMemberIdsFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) []uint32 {
	pm, err := party.NewProcessor(l, ctx).GetByMemberId(characterId)
	if err != nil {
		return nil
	}
	ids := make([]uint32, 0, len(pm.Members()))
	for _, m := range pm.Members() {
		ids = append(ids, m.Id())
	}
	return ids
}

// warpFunc warps characterId on field f to targetMapId. Declared as a package
// var so tests can capture the warp target without a live Kafka producer.
var warpFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, targetMapId _map.Id) error {
	return portal.NewProcessor(l, ctx).Warp(f, characterId, targetMapId)
}

// playPortalSoundForSession announces the existing portal-sound simple-effect
// to the session. It reuses the same writer + body the system_message consumer
// uses for CommandPlayPortalSound — no new packet is introduced.
var playPortalSoundForSession = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model) {
	if wp == nil {
		return
	}
	_ = session.Announce(l)(ctx)(wp)(charcb.CharacterEffectWriter)(charpkt.CharacterPlayPortalSoundEffectEffectBody())(s)
}

// linkedDestination resolves the map a requester standing on currentField warps
// to when entering door d. A door spans an AREA field and a TOWN map:
//   - on the AREA side -> warp to the TOWN map
//   - on the TOWN side  -> warp to the AREA map
//
// The bool is false when currentField is neither side of the door.
func linkedDestination(d door.Model, currentField field.Model) (_map.Id, bool) {
	switch currentField.MapId() {
	case d.Field().MapId():
		return d.TownMapId(), true
	case d.TownMapId():
		return d.Field().MapId(), true
	default:
		return 0, false
	}
}

// authorizeDoorEntry reports whether requesterId may use a door owned by
// ownerId: they are the owner, or a current party member of the owner.
func authorizeDoorEntry(ownerId, requesterId uint32, requesterPartyMemberIds []uint32) bool {
	if requesterId == ownerId {
		return true
	}
	for _, id := range requesterPartyMemberIds {
		if id == ownerId {
			return true
		}
	}
	return false
}

// findDoorOnMap locates the door owned by ownerId that the requester (standing
// on currentField) is authorized to use, and of which currentField is a side.
//
// It fetches the owner's live door(s) by owner (not by field), so it resolves
// BIDIRECTIONALLY: the requester may be on the door's AREA field (warp to town)
// OR on its TOWN map (warp to area). A door's world/channel must match
// currentField, and currentField's map must be either the area map (area side)
// or the town map (town side). A character has at most one live door (recast
// replaces), but the route returns a slice, so 0/1/many are handled by picking
// the first door that matches the current field's world/channel + side.
func findDoorOnMap(l logrus.FieldLogger, ctx context.Context, currentField field.Model, ownerId, requesterId uint32) (door.Model, bool) {
	ms, err := doorsByOwnerFunc(l, ctx, ownerId)
	if err != nil {
		l.WithError(err).Warnf("Unable to retrieve doors for owner [%d] for mystic-door entry.", ownerId)
		return door.Model{}, false
	}

	var found door.Model
	ok := false
	for _, m := range ms {
		af := m.Field()
		// World/channel of the door's area field must match the requester.
		if af.WorldId() != currentField.WorldId() || af.ChannelId() != currentField.ChannelId() {
			continue
		}
		// currentField's map must be a side of this door: the area map (requester
		// on the AREA side) or the town map (requester on the TOWN side).
		if af.MapId() == currentField.MapId() || m.TownMapId() == currentField.MapId() {
			found = m
			ok = true
			break
		}
	}
	if !ok {
		return door.Model{}, false
	}

	if !authorizeDoorEntry(ownerId, requesterId, partyMemberIdsFunc(l, ctx, requesterId)) {
		return door.Model{}, false
	}
	return found, true
}

func MysticDoorEnterHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := doorsb.Enter{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		d, ok := findDoorOnMap(l, ctx, s.Field(), p.OwnerId(), s.CharacterId())
		if !ok {
			// Ineligible / no door present: skip the warp silently. The client
			// re-enables its own input after the request resolves with no warp.
			return
		}

		targetMapId, ok := linkedDestination(d, s.Field())
		if !ok {
			return
		}

		if err := warpFunc(l, ctx, s.Field(), s.CharacterId(), targetMapId); err != nil {
			l.WithError(err).Warnf("Mystic-door warp failed for character [%d].", s.CharacterId())
			return
		}

		playPortalSoundForSession(l, ctx, wp, s)
	}
}
