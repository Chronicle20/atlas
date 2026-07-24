package handler

import (
	"atlas-channel/character/teleportrock"
	"atlas-channel/session"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// fakeTeleportRockProcessor implements teleportrock.Processor and records the
// arguments each request method was invoked with, so the test can assert the
// register path pulls the map id from session state (never the wire).
type fakeTeleportRockProcessor struct {
	addMapCalls    []addMapCall
	removeMapCalls []removeMapCall
}

type addMapCall struct {
	field       field.Model
	characterId uint32
	vip         bool
}

type removeMapCall struct {
	worldId     world.Id
	characterId uint32
	mapId       _map.Id
	vip         bool
}

func (f *fakeTeleportRockProcessor) GetByCharacterId(_ uint32) (teleportrock.Model, error) {
	return teleportrock.Model{}, nil
}

func (f *fakeTeleportRockProcessor) RequestAddMap(fm field.Model, characterId uint32, vip bool) error {
	f.addMapCalls = append(f.addMapCalls, addMapCall{field: fm, characterId: characterId, vip: vip})
	return nil
}

func (f *fakeTeleportRockProcessor) RequestRemoveMap(worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	f.removeMapCalls = append(f.removeMapCalls, removeMapCall{worldId: worldId, characterId: characterId, mapId: mapId, vip: vip})
	return nil
}

var _ teleportrock.Processor = (*fakeTeleportRockProcessor)(nil)

// installTeleportRockRequestsSeam swaps teleportRockRequestsFunc for the test
// and returns a restore func (precedent: doorsByOwnerFunc in
// mystic_door_enter.go / mystic_door_enter_test.go).
func installTeleportRockRequestsSeam(t *testing.T, fake *fakeTeleportRockProcessor) func() {
	t.Helper()
	orig := teleportRockRequestsFunc
	teleportRockRequestsFunc = func(_ logrus.FieldLogger, _ context.Context) teleportrock.Processor {
		return fake
	}
	return func() {
		teleportRockRequestsFunc = orig
	}
}

// newTeleportRockTestSession builds a session.Model with the given characterId
// and field via the public session.Processor API (SetCharacterId / SetField),
// the same idiom used in kafka/consumer/map/consumer_test.go's
// addFieldSession and session/processor_test.go. World/channel must be 0 to
// match the zero-value session.NewSession leaves them at (SetField only
// overwrites mapId + instance on top of the existing field).
func newTeleportRockTestSession(t *testing.T, characterId uint32, mapId _map.Id) (session.Model, func()) {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), s)

	sp := session.NewProcessor(logrus.New(), ctx)
	sp.SetCharacterId(sessionId, characterId)
	f := field.NewBuilder(world.Id(0), channel.Id(0), mapId).Build()
	updated := sp.SetField(sessionId, f)

	return updated, func() { session.ClearRegistryForTenant(ten.Id()) }
}

func TestTeleportRockAddMapHandleFunc_RegisterUsesSessionMapId(t *testing.T) {
	fake := &fakeTeleportRockProcessor{}
	restore := installTeleportRockRequestsSeam(t, fake)
	defer restore()

	const characterId = uint32(555)
	const sessionMapId = _map.Id(102000000)
	s, cleanup := newTeleportRockTestSession(t, characterId, sessionMapId)
	defer cleanup()

	// Register packet: nType=1 (register), bCanTransferContinent=1 (VIP). No
	// map id follows on the wire for register.
	raw := []byte{0x01, 0x01}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	handlerFunc := TeleportRockAddMapHandleFunc(logrus.New(), context.Background(), nil)
	handlerFunc(s, &reader, map[string]interface{}{})

	if len(fake.addMapCalls) != 1 {
		t.Fatalf("RequestAddMap call count = %d, want 1", len(fake.addMapCalls))
	}
	if len(fake.removeMapCalls) != 0 {
		t.Fatalf("RequestRemoveMap call count = %d, want 0", len(fake.removeMapCalls))
	}
	call := fake.addMapCalls[0]
	if call.field.MapId() != sessionMapId {
		t.Fatalf("RequestAddMap mapId = %d, want the SESSION map id %d (must not come from the wire)", call.field.MapId(), sessionMapId)
	}
	if call.characterId != characterId {
		t.Fatalf("RequestAddMap characterId = %d, want %d", call.characterId, characterId)
	}
	if !call.vip {
		t.Fatal("RequestAddMap vip = false, want true")
	}
}

func TestTeleportRockAddMapHandleFunc_DeleteUsesWireMapId(t *testing.T) {
	fake := &fakeTeleportRockProcessor{}
	restore := installTeleportRockRequestsSeam(t, fake)
	defer restore()

	const characterId = uint32(777)
	const sessionMapId = _map.Id(102000000)
	const wireMapId = uint32(200000000)
	s, cleanup := newTeleportRockTestSession(t, characterId, sessionMapId)
	defer cleanup()

	// Delete packet: nType=0 (delete), bCanTransferContinent=0 (regular list),
	// dwTargetField=200000000 (LE uint32).
	raw := []byte{0x00, 0x00, 0x00, 0xC2, 0xEB, 0x0B}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	handlerFunc := TeleportRockAddMapHandleFunc(logrus.New(), context.Background(), nil)
	handlerFunc(s, &reader, map[string]interface{}{})

	if len(fake.removeMapCalls) != 1 {
		t.Fatalf("RequestRemoveMap call count = %d, want 1", len(fake.removeMapCalls))
	}
	if len(fake.addMapCalls) != 0 {
		t.Fatalf("RequestAddMap call count = %d, want 0", len(fake.addMapCalls))
	}
	call := fake.removeMapCalls[0]
	if call.mapId != _map.Id(wireMapId) {
		t.Fatalf("RequestRemoveMap mapId = %d, want the WIRE map id %d", call.mapId, wireMapId)
	}
	if call.characterId != characterId {
		t.Fatalf("RequestRemoveMap characterId = %d, want %d", call.characterId, characterId)
	}
	if call.vip {
		t.Fatal("RequestRemoveMap vip = true, want false")
	}
	if call.worldId != s.Field().WorldId() {
		t.Fatalf("RequestRemoveMap worldId = %d, want session worldId %d", call.worldId, s.Field().WorldId())
	}
}

func TestTeleportRockAddMapHandleFuncSymbol(t *testing.T) {
	if TeleportRockAddMapHandleFunc(logrus.New(), context.Background(), nil) == nil {
		t.Fatal("TeleportRockAddMapHandleFunc returned nil closure")
	}
}
