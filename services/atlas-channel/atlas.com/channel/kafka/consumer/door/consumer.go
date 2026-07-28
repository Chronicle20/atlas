package door

import (
	"atlas-channel/character/buff"
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/party"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"sync"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapc "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	doorcb "github.com/Chronicle20/atlas/libs/atlas-packet/door/clientbound"
	partycb "github.com/Chronicle20/atlas/libs/atlas-packet/party/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("door_status_event")(EnvEventTopicDoorStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(EnvEventTopicDoorStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleRemoved(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleSlotChanged(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// broadcastDoorToMap announces `enc` (writer `writerName`) to sessions in field
// `f`. The Mystic Door's area door (and its minimap portal) is a plain ranged
// map object — visible to EVERYONE in the map, exactly like the v83 client
// the door spawn sequence (no party filter). Party membership only gates door
// ENTRY and the partyPortal town-portal array (announceTownPortalToParty), not
// area visibility. forCharacterId != 0 still targets a single character (kept
// for any one-off targeted send); 0 broadcasts to every session in the map.
// Held as a package-level var so the test can stub session enumeration.
var broadcastDoorToEligible = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, ownerCharacterId, partyId, forCharacterId uint32, writerName string, enc packet.Encode) {
	ll := l.WithFields(logrus.Fields{
		"door_action": "broadcast", "writer": writerName, "map_id": uint32(f.MapId()),
		"owner": ownerCharacterId, "party_id": partyId, "for_character_id": forCharacterId,
	})
	// ForSessionsInMap runs the callback concurrently, so the recipient-list
	// append must be synchronized.
	var mu sync.Mutex
	sent := make([]uint32, 0)
	err := _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
		if forCharacterId != 0 && s.CharacterId() != forCharacterId {
			return nil
		}
		mu.Lock()
		sent = append(sent, s.CharacterId())
		mu.Unlock()
		return session.Announce(l)(ctx)(wp)(writerName)(enc)(s)
	})
	if err != nil {
		ll.WithError(err).Errorf("Unable to broadcast door packet [%s] to field [%d].", writerName, f.MapId())
		return
	}
	ll.WithField("recipients", sent).Infof("Broadcast [%s] in map [%d] to [%d] session(s).", writerName, uint32(f.MapId()), len(sent))
}

// announceTownPortalToParty sends the per-slot PARTY_OPERATION town-portal
// update to every online member of the owner's party. This is what makes a
// Mystic Door cast (or removed) while in a party appear/disappear in town: the
// v83 client renders town doors from the party town-portal array when in a
// party and ignores the solo SPAWN_PORTAL (CField::OnTownPortalChanged). The
// SPAWN_PORTAL emitted alongside remains the SOLO render path. Held as a
// package var so tests can stub party/session resolution.
var announceTownPortalToParty = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, sc server.Model, partyId uint32, slot byte, townMapId, targetMapId mapc.Id, x, y int16, clear bool) {
	ll := l.WithFields(logrus.Fields{
		"door_action": "town_portal_announce", "party_id": partyId, "slot": slot,
		"town_map_id": uint32(townMapId), "target_map_id": uint32(targetMapId),
		"x": x, "y": y, "clear": clear,
	})
	if partyId == 0 {
		ll.Debugf("TownPortal: skipping announce, owner not in a party.")
		return
	}
	// slot >= 6 makes the v83 client throw CDisconnectException in OnPartyResult
	// case 0x25 (@0xa3e31c). Log loudly rather than crash the recipients; this
	// should never happen for a 6-cap party and signals a slot-derivation bug.
	if slot >= 6 {
		ll.Errorf("TownPortal: REFUSING to send out-of-range slot [%d] (>=6 crashes the v83 client).", slot)
		return
	}
	p, err := party.NewProcessor(l, ctx).GetById(partyId)
	if err != nil {
		ll.WithError(err).Warnf("TownPortal: unable to resolve party [%d] for door owner.", partyId)
		return
	}
	var body packet.Encode
	if clear {
		body = partycb.PartyTownPortalClearBody(slot)
	} else {
		body = partycb.PartyTownPortalBody(slot, townMapId, targetMapId, x, y)
	}
	sp := session.NewProcessor(l, ctx)
	sent := make([]uint32, 0, len(p.Members()))
	for _, m := range p.Members() {
		_ = sp.IfPresentByCharacterId(sc.Channel())(m.Id(), func(s session.Model) error {
			sent = append(sent, s.CharacterId())
			return session.Announce(l)(ctx)(wp)(partycb.PartyOperationWriter)(body)(s)
		})
	}
	ll.WithField("recipients", sent).Infof("TownPortal: announced slot [%d] (clear=%t) to [%d] party member session(s).", slot, clear, len(sent))
}

func handleCreated(sc server.Model, wp writer.Producer) message.Handler[StatusEvent[CreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e StatusEvent[CreatedBody]) {
		if e.Type != EventDoorStatusCreated {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		b := e.Body
		l.WithFields(logrus.Fields{
			"door_action": "event_created", "owner": e.OwnerCharacterId, "party_id": e.PartyId,
			"for_character_id": e.ForCharacterId, "slot": b.Slot, "area_map_id": uint32(e.MapId),
			"town_map_id": uint32(b.TownMapId), "town_portal_id": b.TownPortalId,
			"area_x": b.AreaX, "area_y": b.AreaY, "town_x": b.TownX, "town_y": b.TownY,
		}).Infof("Door CREATED: owner [%d] party [%d] slot [%d] (forCharacter [%d]).", e.OwnerCharacterId, e.PartyId, b.Slot, e.ForCharacterId)
		areaField := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		townField := field.NewBuilder(e.WorldId, e.ChannelId, b.TownMapId).SetInstance(e.Instance).Build()

		// AREA field viewers (the door spawn sequence, areaDoor: from=area,
		// inTown=false): spawnPortal(area, town, townPortalPos) then
		// spawnDoor(owner, areaPos, launched=false-for-first-deploy).
		broadcastDoorToEligible(l, ctx, wp, areaField, e.OwnerCharacterId, e.PartyId, e.ForCharacterId,
			doorcb.SpawnPortalWriter, writer.SpawnPortalBody(e.MapId, b.TownMapId, b.TownX, b.TownY))
		broadcastDoorToEligible(l, ctx, wp, areaField, e.OwnerCharacterId, e.PartyId, e.ForCharacterId,
			doorcb.SpawnDoorWriter, writer.SpawnDoorBody(e.OwnerCharacterId, b.AreaX, b.AreaY, false))

		// TOWN field viewers (townDoor: from=town, inTown=true): ONLY
		// spawnPortal(town, area, areaPos) — NO spawnDoor (line 120 guards
		// spawnDoor behind !inTown()). SPAWN_PORTAL is the SOLO town render path.
		broadcastDoorToEligible(l, ctx, wp, townField, e.OwnerCharacterId, e.PartyId, e.ForCharacterId,
			doorcb.SpawnPortalWriter, writer.SpawnPortalBody(b.TownMapId, e.MapId, b.AreaX, b.AreaY))

		// PARTY town render path: a viewer in a party renders town doors from the
		// party town-portal array, not SPAWN_PORTAL — so set this member's slot for
		// every party member (wherever they are; the array is global client state).
		// ForCharacterId != 0 is a join/leave visibility delta already covered by the
		// JOIN/LEAVE PARTYDATA refresh, so only broadcast on a fresh cast.
		if e.ForCharacterId == 0 {
			announceTownPortalToParty(l, ctx, wp, sc, e.PartyId, b.Slot, b.TownMapId, e.MapId, b.AreaX, b.AreaY, false)
		}
	}
}

func handleRemoved(sc server.Model, wp writer.Producer) message.Handler[StatusEvent[RemovedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e StatusEvent[RemovedBody]) {
		if e.Type != EventDoorStatusRemoved {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		b := e.Body
		l.WithFields(logrus.Fields{
			"door_action": "event_removed", "owner": e.OwnerCharacterId, "party_id": e.PartyId,
			"for_character_id": e.ForCharacterId, "slot": b.Slot, "area_map_id": uint32(e.MapId),
			"town_map_id": uint32(b.TownMapId), "reason": b.Reason,
		}).Infof("Door REMOVED: owner [%d] party [%d] slot [%d] reason [%s] (forCharacter [%d]).", e.OwnerCharacterId, e.PartyId, b.Slot, b.Reason, e.ForCharacterId)

		areaField := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()

		// AREA viewers (areaDoor.sendDestroyData, inTown=false): removeDoor(owner).
		//
		// This MUST run even on a RECAST. The v83 client keys CTownPortalPool by
		// owner, and CTownPortalPool::OnTownPortalCreated (@0x7bd6c6) does NOT update
		// an existing entry in place — a repeat SpawnDoor for an owner whose door is
		// already open TOGGLES it: it loads the CDOOR "closing" sprite (which renders
		// below the platform) and RELEASES pLayerFrame. The next removal then crashes
		// in CTownPortalPool::OnTownPortalRemoved (@0x7be064), whose animate=0 path
		// dereferences pLayerFrame (`if (!pLayerFrame) _com_issue_error(0x80004003)` —
		// the E_POINTER seen live). So on a recast we destroy the old client entry
		// HERE; the trailing CREATED then builds a FRESH entry at the new position
		// (no existing entry -> no toggle). IDA-verified, v83 (port 13342).
		//
		// (The earlier "Remove+Spawn crashes" symptom was actually removing a door
		// that the deleted ReconcileParty re-spawns had already toggled; with that
		// gone, the recast remove+spawn is the correct, crash-free sequence.)
		broadcastDoorToEligible(l, ctx, wp, areaField, e.OwnerCharacterId, e.PartyId, e.ForCharacterId,
			doorcb.RemoveDoorWriter, writer.RemoveDoorBody(e.OwnerCharacterId))

		// On a RECAST the trailing CREATED re-establishes the town portal AND the
		// caster re-applies the Mystic Door buff, so STOP here: clearing the town
		// portal or cancelling the buff would fight the immediately-following re-cast
		// (a buff cancel landing after the re-apply would leave the door icon-less).
		if b.Reason == RemoveReasonRecast {
			return
		}

		townField := field.NewBuilder(e.WorldId, e.ChannelId, b.TownMapId).SetInstance(e.Instance).Build()

		// TOWN viewers (townDoor.sendDestroyData, inTown=true): removeDoor town=true
		// -> 8-byte SPAWN_PORTAL clear (RemoveTownDoor), NOT SpawnPortal(...,0,0).
		// This is the SOLO town clear; party members clear via the town-portal array.
		broadcastDoorToEligible(l, ctx, wp, townField, e.OwnerCharacterId, e.PartyId, e.ForCharacterId,
			doorcb.RemoveTownDoorWriter, writer.RemoveTownDoorBody())

		// PARTY town render path: clear this member's town-portal slot. See
		// handleCreated; only on a real removal broadcast (not a leave delta).
		if e.ForCharacterId == 0 {
			announceTownPortalToParty(l, ctx, wp, sc, e.PartyId, b.Slot, 0, 0, 0, 0, true)
			// Clear the owner's Mystic Door buff so the duration icon disappears
			// when the door is gone (expiry / leave-field / cancel). RECAST already
			// returned above, so a recast keeps the buff (the new cast refreshes it).
			_ = buff.NewProcessor(l, ctx).Cancel(areaField, e.OwnerCharacterId, int32(skillconst.PriestMysticDoorId))
		}
	}
}

func handleSlotChanged(sc server.Model, wp writer.Producer) message.Handler[StatusEvent[SlotChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e StatusEvent[SlotChangedBody]) {
		if e.Type != EventDoorStatusSlotChanged {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		b := e.Body
		l.WithFields(logrus.Fields{
			"door_action": "event_slot_changed", "owner": e.OwnerCharacterId, "party_id": e.PartyId,
			"for_character_id": e.ForCharacterId, "old_slot": b.OldSlot, "new_slot": b.NewSlot,
			"area_map_id": uint32(e.MapId), "town_map_id": uint32(b.TownMapId), "town_x": b.TownX, "town_y": b.TownY,
		}).Infof("Door SLOT_CHANGED: owner [%d] party [%d] [%d]->[%d].", e.OwnerCharacterId, e.PartyId, b.OldSlot, b.NewSlot)
		townField := field.NewBuilder(e.WorldId, e.ChannelId, b.TownMapId).SetInstance(e.Instance).Build()

		// SOLO town render path: reslot moves the town-side minimap portal indicator
		// for a viewer NOT in a party (e.g. a member who just left and reslotted back
		// to solo slot 0). the v83 client has no dedicated reslot packet, so emit
		// remove(town) + spawnPortal at the new slot for the town field. Party members
		// ignore this (CField::OnTownPortalChanged @0x5365c8 takes the party branch and
		// renders from the PARTYDATA aTownPortal array instead).
		broadcastDoorToEligible(l, ctx, wp, townField, e.OwnerCharacterId, e.PartyId, e.ForCharacterId,
			doorcb.RemoveTownDoorWriter, writer.RemoveTownDoorBody())
		broadcastDoorToEligible(l, ctx, wp, townField, e.OwnerCharacterId, e.PartyId, e.ForCharacterId,
			doorcb.SpawnPortalWriter, writer.SpawnPortalBody(b.TownMapId, e.MapId, b.TownX, b.TownY))

		// PARTY town render path: a reslot is ALWAYS driven by a party membership
		// change (join/left/expel — see atlas-doors handleJoined/Left/Expel), and the
		// channel party-status consumer already re-sends the full PARTYDATA (with every
		// member's door resolved via applyMemberDoor) on each of those events. That
		// full refresh re-renders the town-portal array self-consistently — every
		// member's door at their own party index. Emitting an incremental per-slot
		// TOWN_PORTAL(0x25) clear/set here is both redundant with that refresh AND
		// actively harmful: the OldSlot clear wipes whichever OTHER member now occupies
		// that array index (e.g. a member who stayed at slot 0 while this door moved
		// 0->1 emits no SLOT_CHANGED of its own, so is never restored), and the two
		// updates race across the door_status vs party_status topics. So do NOT touch
		// the party town-portal array on reslot — the PARTYDATA refresh owns it.
	}
}
