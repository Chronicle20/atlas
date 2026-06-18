package door

import (
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	doorcb "github.com/Chronicle20/atlas/libs/atlas-packet/door/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	areaMapId _map.Id = 100000000
	townMapId _map.Id = 104000000
	ownerId   uint32  = 555
)

// broadcastCall captures one broadcastDoorToEligible invocation.
type broadcastCall struct {
	mapId            _map.Id
	writerName       string
	ownerCharacterId uint32
	partyId          uint32
	forCharacterId   uint32
}

// townPortalCall captures one announceTownPortalToParty invocation.
type townPortalCall struct {
	partyId     uint32
	slot        byte
	townMapId   _map.Id
	targetMapId _map.Id
	x, y        int16
	clear       bool
}

// recordedCalls bundles the recorders for both broadcast seams.
type recordedCalls struct {
	broadcasts  []broadcastCall
	townPortals []townPortalCall
}

// withRecordingBroadcaster swaps the package-level broadcast seams for recording
// stubs so tests assert the wire effect of the door consumer without standing up
// REST mocks for ForSessionsInMap or party/session resolution.
func withRecordingBroadcaster(t *testing.T) (restore func(), calls *recordedCalls) {
	t.Helper()
	rec := &recordedCalls{}
	origB := broadcastDoorToEligible
	broadcastDoorToEligible = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, f field.Model, ownerCharacterId, partyId, forCharacterId uint32, writerName string, _ packet.Encode) {
		rec.broadcasts = append(rec.broadcasts, broadcastCall{
			mapId:            f.MapId(),
			writerName:       writerName,
			ownerCharacterId: ownerCharacterId,
			partyId:          partyId,
			forCharacterId:   forCharacterId,
		})
	}
	origT := announceTownPortalToParty
	announceTownPortalToParty = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ server.Model, partyId uint32, slot byte, townMapId, targetMapId _map.Id, x, y int16, clear bool) {
		rec.townPortals = append(rec.townPortals, townPortalCall{
			partyId: partyId, slot: slot, townMapId: townMapId, targetMapId: targetMapId, x: x, y: y, clear: clear,
		})
	}
	return func() { broadcastDoorToEligible = origB; announceTownPortalToParty = origT }, rec
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	ch := channelconst.NewModel(0, 1)
	return server.Register(tm, ch, "127.0.0.1", 8484)
}

func countWriter(calls []broadcastCall, mapId _map.Id, writerName string) int {
	n := 0
	for _, c := range calls {
		if c.mapId == mapId && c.writerName == writerName {
			n++
		}
	}
	return n
}

// TestHandleCreated_AreaSpawnDoorPlusPortal_TownPortalOnly asserts the v83 client
// door object per-side sequence: the AREA field gets SpawnPortal + SpawnDoor;
// the TOWN field gets ONLY SpawnPortal (no SpawnDoor — door object line 120
// guards spawnDoor behind !inTown()).
func TestHandleCreated_AreaSpawnDoorPlusPortal_TownPortalOnly(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingBroadcaster(t)
	defer restore()

	h := handleCreated(sc, nil)
	h(logrus.New(), ctx, StatusEvent[CreatedBody]{
		WorldId:          sc.WorldId(),
		ChannelId:        sc.ChannelId(),
		MapId:            areaMapId,
		Instance:         uuid.Nil,
		PairId:           1,
		OwnerCharacterId: ownerId,
		PartyId:          77,
		Type:             EventDoorStatusCreated,
		Body: CreatedBody{
			AreaDoorId:   10,
			TownDoorId:   20,
			TownMapId:    townMapId,
			Slot:         0,
			TownPortalId: 0x80,
			AreaX:        300,
			AreaY:        400,
			TownX:        -100,
			TownY:        -200,
			SkillId:      9101004,
			SkillLevel:   1,
		},
	})

	if got := countWriter(calls.broadcasts, areaMapId, doorcb.SpawnDoorWriter); got != 1 {
		t.Fatalf("area SpawnDoor: want 1, got %d", got)
	}
	if got := countWriter(calls.broadcasts, areaMapId, doorcb.SpawnPortalWriter); got != 1 {
		t.Fatalf("area SpawnPortal: want 1, got %d", got)
	}
	if got := countWriter(calls.broadcasts, townMapId, doorcb.SpawnPortalWriter); got != 1 {
		t.Fatalf("town SpawnPortal: want 1, got %d", got)
	}
	if got := countWriter(calls.broadcasts, townMapId, doorcb.SpawnDoorWriter); got != 0 {
		t.Fatalf("town SpawnDoor: want 0 (sends spawnDoor only when !inTown), got %d", got)
	}
	// Every broadcast carries the owner + party id (for logging/targeting); the
	// area door itself goes to every session in the map (no party filter).
	for _, c := range calls.broadcasts {
		if c.ownerCharacterId != ownerId || c.partyId != 77 {
			t.Fatalf("broadcast lost owner/party args: %+v", c)
		}
	}
	// PARTY town render: a partied cast also sets the member's town-portal slot
	// (the in-party town-door render source), townMapId + area targetMapId + area pos.
	if len(calls.townPortals) != 1 {
		t.Fatalf("town-portal set: want 1, got %d", len(calls.townPortals))
	}
	tp := calls.townPortals[0]
	if tp.clear || tp.partyId != 77 || tp.slot != 0 || tp.townMapId != townMapId ||
		tp.targetMapId != areaMapId || tp.x != 300 || tp.y != 400 {
		t.Fatalf("town-portal set payload: %+v", tp)
	}
}

// TestHandleCreated_WrongType_NoBroadcast guards against the handler firing
// for an unrelated status type delivered on the same topic.
func TestHandleCreated_WrongType_NoBroadcast(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingBroadcaster(t)
	defer restore()

	h := handleCreated(sc, nil)
	h(logrus.New(), ctx, StatusEvent[CreatedBody]{
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId(),
		Type:      EventDoorStatusRemoved, // wrong type for created handler
	})

	if len(calls.broadcasts) != 0 {
		t.Fatalf("wrong-type event: want 0 broadcasts, got %d", len(calls.broadcasts))
	}
}

// TestHandleCreated_OtherChannel_NoBroadcast asserts the per-channel guard
// (FR-6.5): an event for a different channel is ignored.
func TestHandleCreated_OtherChannel_NoBroadcast(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingBroadcaster(t)
	defer restore()

	h := handleCreated(sc, nil)
	h(logrus.New(), ctx, StatusEvent[CreatedBody]{
		WorldId:   sc.WorldId(),
		ChannelId: sc.ChannelId() + 9, // different channel
		MapId:     areaMapId,
		Type:      EventDoorStatusCreated,
		Body:      CreatedBody{TownMapId: townMapId},
	})

	if len(calls.broadcasts) != 0 {
		t.Fatalf("other-channel event: want 0 broadcasts, got %d", len(calls.broadcasts))
	}
}

// TestHandleRemoved_AreaRemoveDoor_TownRemoveTownDoor asserts removal sends
// RemoveDoor to the AREA field and RemoveTownDoor (8-byte SPAWN_PORTAL clear)
// to the TOWN field — NOT SpawnPortal.
func TestHandleRemoved_AreaRemoveDoor_TownRemoveTownDoor(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingBroadcaster(t)
	defer restore()

	h := handleRemoved(sc, nil)
	h(logrus.New(), ctx, StatusEvent[RemovedBody]{
		WorldId:          sc.WorldId(),
		ChannelId:        sc.ChannelId(),
		MapId:            areaMapId,
		Instance:         uuid.Nil,
		OwnerCharacterId: ownerId,
		PartyId:          77,
		Type:             EventDoorStatusRemoved,
		Body: RemovedBody{
			AreaDoorId: 10,
			TownDoorId: 20,
			TownMapId:  townMapId,
			Reason:     RemoveReasonExpiry,
		},
	})

	if got := countWriter(calls.broadcasts, areaMapId, doorcb.RemoveDoorWriter); got != 1 {
		t.Fatalf("area RemoveDoor: want 1, got %d", got)
	}
	if got := countWriter(calls.broadcasts, townMapId, doorcb.RemoveTownDoorWriter); got != 1 {
		t.Fatalf("town RemoveTownDoor: want 1, got %d", got)
	}
	if got := countWriter(calls.broadcasts, townMapId, doorcb.SpawnPortalWriter); got != 0 {
		t.Fatalf("town must NOT get SpawnPortal on removal (use RemoveTownDoor 8-byte clear), got %d", got)
	}
	// PARTY town render: removal clears the member's town-portal slot.
	if len(calls.townPortals) != 1 || !calls.townPortals[0].clear || calls.townPortals[0].partyId != 77 {
		t.Fatalf("town-portal clear: want 1 clear for party 77, got %+v", calls.townPortals)
	}
}

// TestHandleRemoved_Recast_NoRemovalPackets asserts a RECAST removal emits NO
// removal packets at all — not the area RemoveDoor, not the town RemoveTownDoor,
// not the party town-portal clear. A recast is a remove + immediate re-create of
// the SAME owner; the v83 client keys its door pool by owner and updates in place
// (CTownPortalPool::OnTownPortalCreated @0x7bd6c6), so the trailing CREATED fully
// refreshes the door. Emitting RemoveDoor runs the despawn animation
// (OnTownPortalRemoved @0x7be064) and the re-spawn lands on the same COM canvas
// layers in one frame — which crashes the client. IDA-verified (v83).
func TestHandleRemoved_Recast_NoRemovalPackets(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingBroadcaster(t)
	defer restore()

	h := handleRemoved(sc, nil)
	h(logrus.New(), ctx, StatusEvent[RemovedBody]{
		WorldId:          sc.WorldId(),
		ChannelId:        sc.ChannelId(),
		MapId:            areaMapId,
		Instance:         uuid.Nil,
		OwnerCharacterId: ownerId,
		PartyId:          77,
		Type:             EventDoorStatusRemoved,
		Body: RemovedBody{
			AreaDoorId: 10,
			TownDoorId: 20,
			TownMapId:  townMapId,
			Slot:       1,
			Reason:     RemoveReasonRecast,
		},
	})

	if len(calls.broadcasts) != 0 {
		t.Fatalf("recast must NOT emit field removal packets (despawn-then-respawn crashes v83), got %+v", calls.broadcasts)
	}
	if len(calls.townPortals) != 0 {
		t.Fatalf("recast must NOT emit a town-portal clear, got %+v", calls.townPortals)
	}
}

// TestHandleSlotChanged_ReslotsTownPortal asserts a reslot re-places the
// town-side portal: RemoveTownDoor then SpawnPortal at the new slot on the
// town field only. PartyId is 0 so this is a pure solo-path test (the party
// block early-returns in announceTownPortalToParty when partyId == 0).
func TestHandleSlotChanged_ReslotsTownPortal(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingBroadcaster(t)
	defer restore()

	h := handleSlotChanged(sc, nil)
	h(logrus.New(), ctx, StatusEvent[SlotChangedBody]{
		WorldId:          sc.WorldId(),
		ChannelId:        sc.ChannelId(),
		MapId:            areaMapId,
		Instance:         uuid.Nil,
		OwnerCharacterId: ownerId,
		PartyId:          0,
		Type:             EventDoorStatusSlotChanged,
		Body: SlotChangedBody{
			TownMapId: townMapId,
			OldSlot:   0,
			NewSlot:   1,
			TownX:     -150,
			TownY:     -250,
		},
	})

	if got := countWriter(calls.broadcasts, townMapId, doorcb.RemoveTownDoorWriter); got != 1 {
		t.Fatalf("reslot town RemoveTownDoor: want 1, got %d", got)
	}
	if got := countWriter(calls.broadcasts, townMapId, doorcb.SpawnPortalWriter); got != 1 {
		t.Fatalf("reslot town SpawnPortal: want 1, got %d", got)
	}
	if got := countWriter(calls.broadcasts, areaMapId, doorcb.SpawnDoorWriter); got != 0 {
		t.Fatalf("reslot must not touch the area door, got %d area SpawnDoor", got)
	}
}

// TestHandleSlotChanged_PartyTownPortalNotTouched asserts that a reslot event for
// a partied owner (PartyId != 0, ForCharacterId == 0) does NOT emit any incremental
// party town-portal (TOWN_PORTAL/0x25) calls. A reslot is always driven by a party
// membership change (join/left/expel), and the channel party-status consumer already
// re-sends the full PARTYDATA — with every member's door resolved via applyMemberDoor —
// on each of those events, re-rendering the town-portal array self-consistently.
// Emitting a per-slot clear/set here was both redundant and harmful: the OldSlot clear
// wiped whichever OTHER member occupies that array index (a member who stayed at its
// slot emits no SLOT_CHANGED of its own, so was never restored), and the two updates
// raced across the door_status vs party_status topics. The solo town SpawnPortal is
// still emitted (a leaver reslotting back to solo slot 0 renders via the solo branch).
func TestHandleSlotChanged_PartyTownPortalNotTouched(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	restore, calls := withRecordingBroadcaster(t)
	defer restore()

	h := handleSlotChanged(sc, nil)
	h(logrus.New(), ctx, StatusEvent[SlotChangedBody]{
		WorldId:          sc.WorldId(),
		ChannelId:        sc.ChannelId(),
		MapId:            areaMapId,
		Instance:         uuid.Nil,
		OwnerCharacterId: ownerId,
		PartyId:          77,
		ForCharacterId:   0,
		Type:             EventDoorStatusSlotChanged,
		Body: SlotChangedBody{
			TownMapId: townMapId,
			OldSlot:   0,
			NewSlot:   1,
			TownX:     -150,
			TownY:     -250,
			AreaX:     300,
			AreaY:     400,
		},
	})

	// No incremental party town-portal (0x25) calls: the PARTYDATA refresh owns the
	// in-party town render.
	if len(calls.townPortals) != 0 {
		t.Fatalf("reslot must NOT emit incremental party town-portal calls (PARTYDATA refresh owns it), got %d: %+v",
			len(calls.townPortals), calls.townPortals)
	}

	// The solo town render path is still emitted (RemoveTownDoor + SpawnPortal on the
	// town field) — party members ignore it; a leaver who reslotted to solo renders from it.
	if got := countWriter(calls.broadcasts, townMapId, doorcb.RemoveTownDoorWriter); got != 1 {
		t.Fatalf("reslot town RemoveTownDoor: want 1, got %d", got)
	}
	if got := countWriter(calls.broadcasts, townMapId, doorcb.SpawnPortalWriter); got != 1 {
		t.Fatalf("reslot town SpawnPortal: want 1, got %d", got)
	}
}

// NOTE: the former TestPartyMemberSet_* / TestBroadcastDoorToEligible_FiltersToPartyMembers
// tests were removed: the area door is no longer party-filtered. Per the v83 client
// (the door spawn sequence) it is a plain map object shown to EVERY session in
// the map; party membership only gates entry and the partyPortal town-portal
// array (announceTownPortalToParty, still covered above).
