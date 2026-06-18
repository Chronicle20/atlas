package handler

import (
	"atlas-channel/door"
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	doorsb "github.com/Chronicle20/atlas/libs/atlas-packet/door/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const (
	testAreaMapId = _map.Id(910000000)
	testTownMapId = _map.Id(100000000)
	testOwnerId   = uint32(42)
	testMemberId  = uint32(99)
	testStranger  = uint32(7)
	testAreaX     = int16(1111)
	testAreaY     = int16(-222)
	testTownX     = int16(333)
	testTownY     = int16(-44)
)

func testField(mapId _map.Id) field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), mapId).Build()
}

func testDoor(owner uint32) door.Model {
	m, _ := door.Extract(door.RestModel{
		Id:               "door-1",
		AreaDoorId:       1,
		OwnerCharacterId: owner,
		WorldId:          world.Id(0),
		ChannelId:        channel.Id(1),
		MapId:            testAreaMapId,
		TownMapId:        testTownMapId,
		AreaX:            testAreaX,
		AreaY:            testAreaY,
		TownX:            testTownX,
		TownY:            testTownY,
	})
	return m
}

// --- linkedDestination ---

func TestLinkedDestinationAreaSideWarpsToTown(t *testing.T) {
	d := testDoor(testOwnerId)
	got, x, y, ok := linkedDestination(d, testField(testAreaMapId))
	if !ok || got != testTownMapId {
		t.Fatalf("area side: want (%d,true) got (%d,%v)", testTownMapId, got, ok)
	}
	if x != testTownX || y != testTownY {
		t.Fatalf("area side: want town pos (%d,%d) got (%d,%d)", testTownX, testTownY, x, y)
	}
}

func TestLinkedDestinationTownSideWarpsToArea(t *testing.T) {
	d := testDoor(testOwnerId)
	got, x, y, ok := linkedDestination(d, testField(testTownMapId))
	if !ok || got != testAreaMapId {
		t.Fatalf("town side: want (%d,true) got (%d,%v)", testAreaMapId, got, ok)
	}
	if x != testAreaX || y != testAreaY {
		t.Fatalf("town side: want area pos (%d,%d) got (%d,%d)", testAreaX, testAreaY, x, y)
	}
}

func TestLinkedDestinationUnrelatedFieldFails(t *testing.T) {
	d := testDoor(testOwnerId)
	if _, _, _, ok := linkedDestination(d, testField(_map.Id(200000000))); ok {
		t.Fatal("unrelated field must not resolve a destination")
	}
}

// --- authorizeDoorEntry ---

func TestAuthorizeOwner(t *testing.T) {
	if !authorizeDoorEntry(testOwnerId, testOwnerId, nil) {
		t.Fatal("owner must be authorized")
	}
}

func TestAuthorizePartyMember(t *testing.T) {
	if !authorizeDoorEntry(testOwnerId, testMemberId, []uint32{testOwnerId, testMemberId}) {
		t.Fatal("same-party member must be authorized")
	}
}

func TestAuthorizeStrangerRejected(t *testing.T) {
	if authorizeDoorEntry(testOwnerId, testStranger, []uint32{testStranger}) {
		t.Fatal("non-owner, non-party-member must be rejected")
	}
}

// --- findDoorOnMap (area side, seam-injected) ---

func TestFindDoorOnMapOwnerOnAreaSide(t *testing.T) {
	restoreSeams := installLookupSeams(t, []door.Model{testDoor(testOwnerId)}, []uint32{testOwnerId})
	defer restoreSeams()

	d, onSide, authorized := findDoorOnMap(logrus.New(), context.Background(), testField(testAreaMapId), testOwnerId, testOwnerId)
	if !onSide || !authorized {
		t.Fatal("owner on area side: door must be found and authorized")
	}
	target, _, _, ok := linkedDestination(d, testField(testAreaMapId))
	if !ok || target != testTownMapId {
		t.Fatalf("owner on area side warps to town: got (%d,%v)", target, ok)
	}
}

func TestFindDoorOnMapOwnerOnTownSide(t *testing.T) {
	restoreSeams := installLookupSeams(t, []door.Model{testDoor(testOwnerId)}, []uint32{testOwnerId})
	defer restoreSeams()

	// Requester standing on the TOWN map: the by-owner lookup resolves the door
	// via its TownMapId() side, and linkedDestination warps back to the AREA map.
	d, onSide, authorized := findDoorOnMap(logrus.New(), context.Background(), testField(testTownMapId), testOwnerId, testOwnerId)
	if !onSide || !authorized {
		t.Fatal("owner on town side: door must be found and authorized")
	}
	target, _, _, ok := linkedDestination(d, testField(testTownMapId))
	if !ok || target != testAreaMapId {
		t.Fatalf("owner on town side warps to area: got (%d,%v)", target, ok)
	}
}

func TestFindDoorOnMapPartyMemberOnTownSide(t *testing.T) {
	restoreSeams := installLookupSeams(t, []door.Model{testDoor(testOwnerId)}, []uint32{testOwnerId, testMemberId})
	defer restoreSeams()

	if _, onSide, authorized := findDoorOnMap(logrus.New(), context.Background(), testField(testTownMapId), testOwnerId, testMemberId); !onSide || !authorized {
		t.Fatal("party member on town side: door must be found and authorized")
	}
}

func TestFindDoorOnMapUnrelatedFieldFails(t *testing.T) {
	restoreSeams := installLookupSeams(t, []door.Model{testDoor(testOwnerId)}, []uint32{testOwnerId})
	defer restoreSeams()

	// Owner has a door, but the requester is on a map that is neither the door's
	// area nor town side -> no resolution.
	if _, onSide, _ := findDoorOnMap(logrus.New(), context.Background(), testField(_map.Id(200000000)), testOwnerId, testOwnerId); onSide {
		t.Fatal("door must not resolve when current field is neither side")
	}
}

func TestFindDoorOnMapPartyMemberOnAreaSide(t *testing.T) {
	restoreSeams := installLookupSeams(t, []door.Model{testDoor(testOwnerId)}, []uint32{testOwnerId, testMemberId})
	defer restoreSeams()

	if _, onSide, authorized := findDoorOnMap(logrus.New(), context.Background(), testField(testAreaMapId), testOwnerId, testMemberId); !onSide || !authorized {
		t.Fatal("party member on area side: door must be found and authorized")
	}
}

func TestFindDoorOnMapStrangerRejected(t *testing.T) {
	restoreSeams := installLookupSeams(t, []door.Model{testDoor(testOwnerId)}, []uint32{testStranger})
	defer restoreSeams()

	// A stranger CAN see the door (it is on this map) but is NOT authorized to use it.
	if _, onSide, authorized := findDoorOnMap(logrus.New(), context.Background(), testField(testAreaMapId), testOwnerId, testStranger); !onSide || authorized {
		t.Fatal("stranger: door is on the map (onSide) but entry must not be authorized")
	}
}

func TestFindDoorOnMapNoDoorPresent(t *testing.T) {
	restoreSeams := installLookupSeams(t, nil, []uint32{testOwnerId})
	defer restoreSeams()

	if _, onSide, _ := findDoorOnMap(logrus.New(), context.Background(), testField(testAreaMapId), testOwnerId, testOwnerId); onSide {
		t.Fatal("no door present must not resolve")
	}
}

// --- decode pins the wire shape the handler consumes ---

func TestEnterDoorDecode(t *testing.T) {
	// ownerId 42 (LE uint32) + direction 1.
	raw := []byte{0x2A, 0x00, 0x00, 0x00, 0x01}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := doorsb.Enter{}
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})
	if p.OwnerId() != 42 {
		t.Fatalf("ownerId want 42 got %d", p.OwnerId())
	}
	if p.Direction() != 1 {
		t.Fatalf("direction want 1 got %d", p.Direction())
	}
	if p.Operation() != doorsb.EnterDoorHandle {
		t.Fatalf("operation want %q got %q", doorsb.EnterDoorHandle, p.Operation())
	}
}

func TestMysticDoorEnterHandleFuncSymbol(t *testing.T) {
	if MysticDoorEnterHandleFunc(logrus.New(), context.Background(), nil) == nil {
		t.Fatal("MysticDoorEnterHandleFunc returned nil closure")
	}
}

// installLookupSeams swaps doorsByOwnerFunc + partyMemberIdsFunc for the test
// and returns a restore func. The door lookup is now driven by a by-owner door
// list (resolvable from either the area or the town side).
func installLookupSeams(t *testing.T, doors []door.Model, members []uint32) func() {
	t.Helper()
	origDoors := doorsByOwnerFunc
	origMembers := partyMemberIdsFunc
	doorsByOwnerFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) ([]door.Model, error) {
		return doors, nil
	}
	partyMemberIdsFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) []uint32 {
		return members
	}
	return func() {
		doorsByOwnerFunc = origDoors
		partyMemberIdsFunc = origMembers
	}
}
