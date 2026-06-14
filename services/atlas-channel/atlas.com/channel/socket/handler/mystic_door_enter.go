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

// doorsInFieldFunc lists doors whose AREA field matches f. This is the only
// door lookup atlas-doors exposes over REST (GetInField is keyed on the area
// field). Declared as a package var so tests can inject a fake.
var doorsInFieldFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model) ([]door.Model, error) {
	return door.NewProcessor(l, ctx).GetInField(f)
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
// on currentField) is authorized to use, and which is present on currentField.
//
// Limitation: atlas-doors only exposes GetInField (keyed on the door's AREA
// field) over REST, so the door is resolvable only when the requester is on the
// AREA side. Town-side resolution would require a by-owner REST route on
// atlas-doors that does not yet exist; until then a town-side enter request
// returns (Model{}, false) and the warp is skipped.
func findDoorOnMap(l logrus.FieldLogger, ctx context.Context, currentField field.Model, ownerId, requesterId uint32) (door.Model, bool) {
	ms, err := doorsInFieldFunc(l, ctx, currentField)
	if err != nil {
		l.WithError(err).Warnf("Unable to retrieve doors in field [%d] for mystic-door entry.", currentField.MapId())
		return door.Model{}, false
	}

	var found door.Model
	ok := false
	for _, m := range ms {
		if m.OwnerCharacterId() == ownerId {
			found = m
			ok = true
			break
		}
	}
	if !ok {
		return door.Model{}, false
	}

	// Confirm currentField is actually a side of this door.
	if _, sideOk := linkedDestination(found, currentField); !sideOk {
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
