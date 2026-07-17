package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	trpkt "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// installItemInSlotSeam swaps itemInSlotFunc for the test and returns a
// restore func (precedent: installTeleportRockRequestsSeam in
// teleport_rock_add_map_test.go / doorsByOwnerFunc in mystic_door_enter.go).
func installItemInSlotSeam(t *testing.T, matchSlot int16, matchTemplateId uint32) func() {
	t.Helper()
	orig := itemInSlotFunc
	itemInSlotFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32, slot int16) (uint32, error) {
		if slot != matchSlot {
			return 0, nil
		}
		return matchTemplateId, nil
	}
	return func() {
		itemInSlotFunc = orig
	}
}

type useRockCall struct {
	itemId item.Id
	target trpkt.Target
}

// installUseRockSeam swaps useRockFunc for the test and returns a captured
// call slice plus a restore func.
func installUseRockSeam(t *testing.T) (*[]useRockCall, func()) {
	t.Helper()
	orig := useRockFunc
	calls := &[]useRockCall{}
	useRockFunc = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer) func(s session.Model, itemId item.Id, target trpkt.Target) {
		return func(s session.Model, itemId item.Id, target trpkt.Target) {
			*calls = append(*calls, useRockCall{itemId: itemId, target: target})
		}
	}
	return calls, func() {
		useRockFunc = orig
	}
}

// newTeleportRockUseTestSession builds a session.Model with the given
// characterId (idiom: newTeleportRockTestSession in
// teleport_rock_add_map_test.go).
func newTeleportRockUseTestSession(t *testing.T, characterId uint32) (session.Model, func()) {
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
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
	updated := sp.SetField(sessionId, f)

	return updated, func() { session.ClearRegistryForTenant(ten.Id()) }
}

func TestTeleportRockUseHandleFunc_ValidInvokesUseRock(t *testing.T) {
	restoreSlot := installItemInSlotSeam(t, 2, 2320000)
	defer restoreSlot()
	calls, restoreUse := installUseRockSeam(t)
	defer restoreUse()

	const characterId = uint32(555)
	s, cleanup := newTeleportRockUseTestSession(t, characterId)
	defer cleanup()

	// TestUseByMapDecode payload (Task 3): slot=2, itemId=2320000, target
	// map=100000000, updateTime=42.
	raw := []byte{
		0x02, 0x00, // slot = 2
		0x80, 0x66, 0x23, 0x00, // itemId = 2320000
		0x00,                   // byName = 0
		0x00, 0xE1, 0xF5, 0x05, // mapId = 100000000
		0x2A, 0x00, 0x00, 0x00, // updateTime = 42
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	handlerFunc := TeleportRockUseHandleFunc(logrus.New(), context.Background(), nil)
	handlerFunc(s, &reader, map[string]interface{}{})

	if len(*calls) != 1 {
		t.Fatalf("useRockFunc call count = %d, want 1", len(*calls))
	}
	call := (*calls)[0]
	if call.itemId != item.Id(2320000) {
		t.Fatalf("useRockFunc itemId = %d, want 2320000", call.itemId)
	}
	if call.target.ByName() || call.target.TargetMap() != 100000000 {
		t.Fatalf("useRockFunc target = %+v, want map target 100000000", call.target)
	}
}

func TestTeleportRockUseHandleFunc_SlotMismatchNotInvoked(t *testing.T) {
	// itemInSlotFunc reports a different template id than the packet claims.
	restoreSlot := installItemInSlotSeam(t, 2, 2320001)
	defer restoreSlot()
	calls, restoreUse := installUseRockSeam(t)
	defer restoreUse()

	const characterId = uint32(555)
	s, cleanup := newTeleportRockUseTestSession(t, characterId)
	defer cleanup()

	raw := []byte{
		0x02, 0x00, // slot = 2
		0x80, 0x66, 0x23, 0x00, // itemId = 2320000
		0x00,                   // byName = 0
		0x00, 0xE1, 0xF5, 0x05, // mapId = 100000000
		0x2A, 0x00, 0x00, 0x00, // updateTime = 42
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	handlerFunc := TeleportRockUseHandleFunc(logrus.New(), context.Background(), nil)
	handlerFunc(s, &reader, map[string]interface{}{})

	if len(*calls) != 0 {
		t.Fatalf("useRockFunc call count = %d, want 0 on slot mismatch", len(*calls))
	}
}

func TestTeleportRockUseHandleFunc_AbsentTargetNotInvoked(t *testing.T) {
	restoreSlot := installItemInSlotSeam(t, 2, 2320000)
	defer restoreSlot()
	calls, restoreUse := installUseRockSeam(t)
	defer restoreUse()

	const characterId = uint32(555)
	s, cleanup := newTeleportRockUseTestSession(t, characterId)
	defer cleanup()

	// TestUseAbsentTargetIsInvalid payload (Task 3): slot=2, itemId=2320000,
	// only the trailing updateTime remains — no target payload.
	raw := []byte{
		0x02, 0x00,
		0x80, 0x66, 0x23, 0x00,
		0x2A, 0x00, 0x00, 0x00,
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	handlerFunc := TeleportRockUseHandleFunc(logrus.New(), context.Background(), nil)
	handlerFunc(s, &reader, map[string]interface{}{})

	if len(*calls) != 0 {
		t.Fatalf("useRockFunc call count = %d, want 0 on absent target payload", len(*calls))
	}
}

func TestTeleportRockUseHandleFunc_NonRockItemNotInvoked(t *testing.T) {
	restoreSlot := installItemInSlotSeam(t, 2, 1000000)
	defer restoreSlot()
	calls, restoreUse := installUseRockSeam(t)
	defer restoreUse()

	const characterId = uint32(555)
	s, cleanup := newTeleportRockUseTestSession(t, characterId)
	defer cleanup()

	// Same shape as the valid case, but itemId = 1000000 (not 232xxxx).
	raw := []byte{
		0x02, 0x00, // slot = 2
		0x40, 0x42, 0x0F, 0x00, // itemId = 1000000
		0x00,                   // byName = 0
		0x00, 0xE1, 0xF5, 0x05, // mapId = 100000000
		0x2A, 0x00, 0x00, 0x00, // updateTime = 42
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	handlerFunc := TeleportRockUseHandleFunc(logrus.New(), context.Background(), nil)
	handlerFunc(s, &reader, map[string]interface{}{})

	if len(*calls) != 0 {
		t.Fatalf("useRockFunc call count = %d, want 0 on non-rock item id", len(*calls))
	}
}
