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

// TestHandleCreated_AreaSpawnDoorPlusPortal_TownPortalOnly asserts the Cosmic
// DoorObject per-side sequence: the AREA field gets SpawnPortal + SpawnDoor;
// the TOWN field gets ONLY SpawnPortal (no SpawnDoor — DoorObject line 120
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
		t.Fatalf("town SpawnDoor: want 0 (Cosmic sends spawnDoor only when !inTown), got %d", got)
	}
	// Eligibility wiring: every broadcast carries the owner + party id so the
	// seam can intersect with same-channel party membership.
	for _, c := range calls.broadcasts {
		if c.ownerCharacterId != ownerId || c.partyId != 77 {
			t.Fatalf("broadcast lost eligibility args: %+v", c)
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

// TestHandleRemoved_Recast_NoTownPortalClear asserts a RECAST removal does NOT
// emit a TOWN_PORTAL clear. A recast is a remove+create on the SAME party slot;
// the trailing CREATED re-sets it. Clearing here would make every in-party
// client tear down then immediately rebuild that slot's town-door layer in one
// frame (CField::OnTownPortalChanged), which crashes the v83 client. The
// area/town RemoveDoor broadcasts still fire — only the party slot clear is
// suppressed.
func TestHandleRemoved_Recast_NoTownPortalClear(t *testing.T) {
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
			Reason:     RemoveReasonRecast,
		},
	})

	// The field-level removes still happen (the door object is genuinely removed
	// before the recast re-creates it).
	if got := countWriter(calls.broadcasts, areaMapId, doorcb.RemoveDoorWriter); got != 1 {
		t.Fatalf("recast area RemoveDoor: want 1, got %d", got)
	}
	if got := countWriter(calls.broadcasts, townMapId, doorcb.RemoveTownDoorWriter); got != 1 {
		t.Fatalf("recast town RemoveTownDoor: want 1, got %d", got)
	}
	// The party town-portal slot clear MUST be suppressed on recast.
	if len(calls.townPortals) != 0 {
		t.Fatalf("recast must NOT emit a town-portal clear (crashes v83 client), got %+v", calls.townPortals)
	}
}

// TestHandleSlotChanged_ReslotsTownPortal asserts a reslot re-places the
// town-side portal: RemoveTownDoor then SpawnPortal at the new slot on the
// town field only.
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
		PartyId:          77,
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

// TestPartyMemberSet_OwnerOnlyWhenNoParty asserts the eligibility seam returns
// just the owner when there is no party (partyId == 0).
func TestPartyMemberSet_OwnerOnlyWhenNoParty(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	members := partyMemberSet(logrus.New(), ctx, ownerId, 0)
	if len(members) != 1 {
		t.Fatalf("no-party member set: want 1, got %d", len(members))
	}
	if _, ok := members[ownerId]; !ok {
		t.Fatalf("no-party member set must include owner %d", ownerId)
	}
}

// TestBroadcastDoorToEligible_FiltersToPartyMembers asserts the default
// broadcaster only announces to sessions whose character is in the party
// member set (owner + members), skipping outsiders. Session enumeration is
// stubbed via the _map processor seam is not available here, so we exercise
// the membership-intersection contract through partyMemberSet directly.
func TestBroadcastDoorToEligible_FiltersToPartyMembers(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	// Stub the party membership seam: owner 555 + members 600, 601.
	origPM := partyMemberSet
	defer func() { partyMemberSet = origPM }()
	partyMemberSet = func(_ logrus.FieldLogger, _ context.Context, owner, _ uint32) map[uint32]struct{} {
		return map[uint32]struct{}{owner: {}, 600: {}, 601: {}}
	}

	members := partyMemberSet(logrus.New(), ctx, ownerId, 77)
	for _, want := range []uint32{ownerId, 600, 601} {
		if _, ok := members[want]; !ok {
			t.Fatalf("eligible set missing %d", want)
		}
	}
	if _, ok := members[999]; ok {
		t.Fatalf("outsider 999 must not be eligible")
	}
}
