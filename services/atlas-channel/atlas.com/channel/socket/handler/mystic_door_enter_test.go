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
	})
	return m
}

// --- linkedDestination ---

func TestLinkedDestinationAreaSideWarpsToTown(t *testing.T) {
	d := testDoor(testOwnerId)
	got, ok := linkedDestination(d, testField(testAreaMapId))
	if !ok || got != testTownMapId {
		t.Fatalf("area side: want (%d,true) got (%d,%v)", testTownMapId, got, ok)
	}
}

func TestLinkedDestinationTownSideWarpsToArea(t *testing.T) {
	d := testDoor(testOwnerId)
	got, ok := linkedDestination(d, testField(testTownMapId))
	if !ok || got != testAreaMapId {
		t.Fatalf("town side: want (%d,true) got (%d,%v)", testAreaMapId, got, ok)
	}
}

func TestLinkedDestinationUnrelatedFieldFails(t *testing.T) {
	d := testDoor(testOwnerId)
	if _, ok := linkedDestination(d, testField(_map.Id(200000000))); ok {
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

	d, ok := findDoorOnMap(logrus.New(), context.Background(), testField(testAreaMapId), testOwnerId, testOwnerId)
	if !ok {
		t.Fatal("owner on area side: door must be found")
	}
	target, ok := linkedDestination(d, testField(testAreaMapId))
	if !ok || target != testTownMapId {
		t.Fatalf("owner on area side warps to town: got (%d,%v)", target, ok)
	}
}

func TestFindDoorOnMapPartyMemberOnAreaSide(t *testing.T) {
	restoreSeams := installLookupSeams(t, []door.Model{testDoor(testOwnerId)}, []uint32{testOwnerId, testMemberId})
	defer restoreSeams()

	if _, ok := findDoorOnMap(logrus.New(), context.Background(), testField(testAreaMapId), testOwnerId, testMemberId); !ok {
		t.Fatal("party member on area side: door must be found")
	}
}

func TestFindDoorOnMapStrangerRejected(t *testing.T) {
	restoreSeams := installLookupSeams(t, []door.Model{testDoor(testOwnerId)}, []uint32{testStranger})
	defer restoreSeams()

	if _, ok := findDoorOnMap(logrus.New(), context.Background(), testField(testAreaMapId), testOwnerId, testStranger); ok {
		t.Fatal("stranger must not resolve a door")
	}
}

func TestFindDoorOnMapNoDoorPresent(t *testing.T) {
	restoreSeams := installLookupSeams(t, nil, []uint32{testOwnerId})
	defer restoreSeams()

	if _, ok := findDoorOnMap(logrus.New(), context.Background(), testField(testAreaMapId), testOwnerId, testOwnerId); ok {
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

// installLookupSeams swaps doorsInFieldFunc + partyMemberIdsFunc for the test
// and returns a restore func.
func installLookupSeams(t *testing.T, doors []door.Model, members []uint32) func() {
	t.Helper()
	origDoors := doorsInFieldFunc
	origMembers := partyMemberIdsFunc
	doorsInFieldFunc = func(_ logrus.FieldLogger, _ context.Context, _ field.Model) ([]door.Model, error) {
		return doors, nil
	}
	partyMemberIdsFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) []uint32 {
		return members
	}
	return func() {
		doorsInFieldFunc = origDoors
		partyMemberIdsFunc = origMembers
	}
}
