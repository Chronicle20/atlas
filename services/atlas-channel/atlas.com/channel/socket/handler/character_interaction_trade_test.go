package handler

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	interactioncb "github.com/Chronicle20/atlas/libs/atlas-packet/interaction/clientbound"
	interaction2 "github.com/Chronicle20/atlas/libs/atlas-packet/interaction/serverbound"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func testLogger() logrus.FieldLogger {
	l, _ := testlog.NewNullLogger()
	return l
}

func testReader(b []byte) request.Reader {
	req := request.Request(b)
	return request.NewRequestReader(&req, 0)
}

// --- cross-family occupancy (design §2.1) ------------------------------------

// TestAnyMiniRoomOccupiedReportsAHit pins that a character already seated in
// one family blocks a trade-room create.
func TestAnyMiniRoomOccupiedReportsAHit(t *testing.T) {
	probes := []miniRoomOccupancyProbe{
		{family: "mini-game", probe: func() (bool, error) { return false, nil }},
		{family: "merchant", probe: func() (bool, error) { return true, nil }},
	}
	if !anyMiniRoomOccupied(testLogger(), 100, probes) {
		t.Error("occupied: got false, want true")
	}
}

// TestAnyMiniRoomOccupiedStopsAtTheFirstHit pins that the check short-circuits:
// once one family claims the character there is nothing a later probe can say,
// and every probe is a REST round trip.
func TestAnyMiniRoomOccupiedStopsAtTheFirstHit(t *testing.T) {
	later := 0
	probes := []miniRoomOccupancyProbe{
		{family: "mini-game", probe: func() (bool, error) { return true, nil }},
		{family: "merchant", probe: func() (bool, error) { later++; return false, nil }},
		{family: "trade", probe: func() (bool, error) { later++; return false, nil }},
	}
	if !anyMiniRoomOccupied(testLogger(), 100, probes) {
		t.Error("occupied: got false, want true")
	}
	if later != 0 {
		t.Errorf("probes after the first hit: got %d, want 0", later)
	}
}

// TestAnyMiniRoomOccupiedTreatsAFailedProbeAsFree pins the BEST-EFFORT posture
// (design §2.1): a transient read failure must not lock a player out of opening
// a trade, because each service still enforces its own single-room invariant
// authoritatively. It must also not abort the remaining probes.
func TestAnyMiniRoomOccupiedTreatsAFailedProbeAsFree(t *testing.T) {
	reached := false
	probes := []miniRoomOccupancyProbe{
		{family: "mini-game", probe: func() (bool, error) { return false, errors.New("boom") }},
		{family: "trade", probe: func() (bool, error) { reached = true; return false, nil }},
	}
	if anyMiniRoomOccupied(testLogger(), 100, probes) {
		t.Error("occupied: got true, want false")
	}
	if !reached {
		t.Error("a failed probe aborted the remaining probes")
	}
}

// TestAnyMiniRoomOccupiedWithNoHits pins the common case: a free character is
// not blocked.
func TestAnyMiniRoomOccupiedWithNoHits(t *testing.T) {
	probes := []miniRoomOccupancyProbe{
		{family: "mini-game", probe: func() (bool, error) { return false, nil }},
		{family: "merchant", probe: func() (bool, error) { return false, nil }},
		{family: "trade", probe: func() (bool, error) { return false, nil }},
	}
	if anyMiniRoomOccupied(testLogger(), 100, probes) {
		t.Error("occupied: got true, want false")
	}
}

// --- the CREATE arms' decision ------------------------------------------------

// TestTradeRoomCreateRefusesWhenTheCharacterIsAlreadyInAMiniRoom pins FR-1.2's
// reply path: a cross-family collision announces the enter error and emits
// NOTHING. The announced code is resolved from the tenant enterError table by
// CharacterInteractionEnterResultErrorBody, so the refusal names no byte here.
func TestTradeRoomCreateRefusesWhenTheCharacterIsAlreadyInAMiniRoom(t *testing.T) {
	refused := 0
	created := 0
	ok := tradeRoomCreate(testLogger(), 100,
		func(uint32) bool { return true },
		func() { refused++ },
		func() error { created++; return nil },
	)
	if ok {
		t.Error("created: got true, want false")
	}
	if refused != 1 {
		t.Errorf("refusals announced: got %d, want 1", refused)
	}
	if created != 0 {
		t.Errorf("commands emitted: got %d, want 0", created)
	}
}

// TestTradeRoomCreateEmitsForAFreeCharacter pins the common case, and that the
// occupancy probe is asked about the ACTING character.
func TestTradeRoomCreateEmitsForAFreeCharacter(t *testing.T) {
	var asked uint32
	refused := 0
	created := 0
	ok := tradeRoomCreate(testLogger(), 100,
		func(characterId uint32) bool { asked = characterId; return false },
		func() { refused++ },
		func() error { created++; return nil },
	)
	if !ok {
		t.Error("created: got false, want true")
	}
	if asked != 100 {
		t.Errorf("occupancy probed for character: got %d, want 100", asked)
	}
	if created != 1 {
		t.Errorf("commands emitted: got %d, want 1", created)
	}
	if refused != 0 {
		t.Errorf("refusals announced: got %d, want 0", refused)
	}
}

// TestTradeRoomCreateReportsAFailedEmit pins the branch the cash-trade open
// (nProc 0) keys on: it sends CREATE_ROOM then INVITE, and an invite for a room
// that was never created would address nothing. A failed emit must therefore
// answer false, and must not also announce a refusal — the client was not
// refused, the server failed.
func TestTradeRoomCreateReportsAFailedEmit(t *testing.T) {
	refused := 0
	ok := tradeRoomCreate(testLogger(), 100,
		func(uint32) bool { return false },
		func() { refused++ },
		func() error { return errors.New("kafka down") },
	)
	if ok {
		t.Error("created: got true, want false")
	}
	if refused != 0 {
		t.Errorf("refusals announced: got %d, want 0", refused)
	}
}

// --- the VISIT arm's trade-accept decision -------------------------------------

// TestTradeInviteAcceptForwardsWhenTheSerialOwnsATradeRoom pins the accept path
// the reference client actually uses: there is no dedicated trade-accept
// operation, so clicking "accept" on the invite sends mini-room VISIT (mode 4)
// with serialNumber = the invite's referenceId, which for trade is the room
// handle (= the owner's character id, design §2.3). The visit must be turned
// into an invite ACCEPT — the ONLY producer of the ACCEPTED status atlas-trades
// seats an invitee on — and must be consumed, not fanned out further.
func TestTradeInviteAcceptForwardsWhenTheSerialOwnsATradeRoom(t *testing.T) {
	var asked uint32
	accepted := 0
	consumed := tradeInviteAccept(testLogger(), 2, 1,
		func(ownerId uint32) (bool, error) { asked = ownerId; return true, nil },
		func() error { accepted++; return nil },
	)
	if !consumed {
		t.Error("consumed: got false, want true")
	}
	if asked != 1 {
		t.Errorf("room ownership probed for character: got %d, want 1", asked)
	}
	if accepted != 1 {
		t.Errorf("accepts emitted: got %d, want 1", accepted)
	}
}

// TestTradeInviteAcceptConsumesTheVisitEvenWhenTheEmitFails pins that a failed
// produce still consumes the visit. Falling through would hand the serial to
// atlas-mini-games, which owns no such room and answers ENTER_ERROR
// ROOM_CLOSED — "the room is already closed" on a trade the server knows is
// open.
func TestTradeInviteAcceptConsumesTheVisitEvenWhenTheEmitFails(t *testing.T) {
	if !tradeInviteAccept(testLogger(), 2, 1,
		func(uint32) (bool, error) { return true, nil },
		func() error { return errors.New("kafka down") },
	) {
		t.Error("consumed: got false, want true")
	}
}

// TestTradeInviteAcceptDeclinesAVisitThatNamesNoTradeRoom pins the other half:
// the balloon double-click join for a game room or shop must keep reaching the
// mini-game and merchant fan-out, and must not emit a trade accept.
func TestTradeInviteAcceptDeclinesAVisitThatNamesNoTradeRoom(t *testing.T) {
	accepted := 0
	if tradeInviteAccept(testLogger(), 2, 1,
		func(uint32) (bool, error) { return false, nil },
		func() error { accepted++; return nil },
	) {
		t.Error("consumed: got true, want false")
	}
	if accepted != 0 {
		t.Errorf("accepts emitted: got %d, want 0", accepted)
	}
}

// TestTradeInviteAcceptFallsThroughWhenTheProbeFails pins the same BEST-EFFORT
// posture the occupancy check takes: a transient atlas-trades read failure must
// not swallow a legitimate mini-game join, and must not emit an accept for an
// invite that may not exist.
func TestTradeInviteAcceptFallsThroughWhenTheProbeFails(t *testing.T) {
	accepted := 0
	if tradeInviteAccept(testLogger(), 2, 1,
		func(uint32) (bool, error) { return false, errors.New("boom") },
		func() error { accepted++; return nil },
	) {
		t.Error("consumed: got true, want false")
	}
	if accepted != 0 {
		t.Errorf("accepts emitted: got %d, want 0", accepted)
	}
}

// --- the inventory-type decode boundary --------------------------------------

// tradePutItemBytes builds a TRADE_PUT_ITEM body in the client's field order
// (OperationTradePutItem.Encode): compartment byte, int16 slot, uint16
// quantity, target-slot byte.
func tradePutItemBytes(inventoryType byte, sourceSlot int16, quantity uint16, targetSlot byte) []byte {
	b := make([]byte, 0, 6)
	b = append(b, inventoryType)
	b = binary.LittleEndian.AppendUint16(b, uint16(sourceSlot))
	b = binary.LittleEndian.AppendUint16(b, quantity)
	return append(b, targetSlot)
}

func decodeTradePutItem(t *testing.T, b []byte) interaction2.OperationTradePutItem {
	t.Helper()
	r := testReader(b)
	sp := &interaction2.OperationTradePutItem{}
	sp.Decode(testLogger(), context.Background())(&r, nil)
	return *sp
}

// TestTradeInventoryTypeAcceptsEveryCompartment pins that the narrowing does
// not reject a legitimate stage from any of the five compartments.
func TestTradeInventoryTypeAcceptsEveryCompartment(t *testing.T) {
	for _, want := range inventory.Types {
		sp := decodeTradePutItem(t, tradePutItemBytes(byte(want), 5, 100, 3))
		got, ok := tradeInventoryType(sp.InventoryType())
		if !ok {
			t.Errorf("compartment [%d]: rejected", want)
			continue
		}
		if got != want {
			t.Errorf("compartment [%d]: got %d", want, got)
		}
	}
}

// TestTradeInventoryTypeRejectsAByteAbove127 pins the int8 boundary: the wire
// field is an UNSIGNED byte and inventory.Type is a SIGNED int8, so 0xC8 would
// otherwise reach atlas-trades as compartment -56.
func TestTradeInventoryTypeRejectsAByteAbove127(t *testing.T) {
	sp := decodeTradePutItem(t, tradePutItemBytes(0xC8, 5, 100, 3))
	if sp.InventoryType() != 0xC8 {
		t.Fatalf("decoded compartment: got %d, want 200", sp.InventoryType())
	}
	if got, ok := tradeInventoryType(sp.InventoryType()); ok {
		t.Errorf("compartment 200: accepted as %d, want rejected", got)
	}
}

// TestTradeInventoryTypeRejectsOutOfRangeCompartments pins that the check is a
// membership test over the five real compartments, not just a sign test.
func TestTradeInventoryTypeRejectsOutOfRangeCompartments(t *testing.T) {
	for _, b := range []byte{0, 6, 127, 128, 255} {
		if _, ok := tradeInventoryType(b); ok {
			t.Errorf("compartment [%d]: accepted, want rejected", b)
		}
	}
}

// TestTradePutItemDecodesEveryFieldTheCommandCarries pins that the four fields
// the PUT_ITEM command forwards all survive the decode, including an equipped
// (negative) source slot.
func TestTradePutItemDecodesEveryFieldTheCommandCarries(t *testing.T) {
	sp := decodeTradePutItem(t, tradePutItemBytes(byte(inventory.TypeValueEquip), -11, 1, 2))
	if sp.InventoryType() != byte(inventory.TypeValueEquip) {
		t.Errorf("inventoryType: got %d, want 1", sp.InventoryType())
	}
	if sp.Slot() != -11 {
		t.Errorf("slot: got %d, want -11", sp.Slot())
	}
	if sp.Quantity() != 1 {
		t.Errorf("quantity: got %d, want 1", sp.Quantity())
	}
	if sp.TargetSlot() != 2 {
		t.Errorf("targetSlot: got %d, want 2", sp.TargetSlot())
	}
}

// --- the CRC attestation payload ---------------------------------------------

// crcBytes builds a confirm/transaction CRC block: a count byte followed by
// {data, crc} uint32 pairs.
func crcBytes(pairs [][2]uint32) []byte {
	b := []byte{byte(len(pairs))}
	for _, p := range pairs {
		b = binary.LittleEndian.AppendUint32(b, p[0])
		b = binary.LittleEndian.AppendUint32(b, p[1])
	}
	return b
}

// TestTradeConfirmEntriesForwardsEveryPair pins that the attestation payload
// survives from the wire to the command, in order.
func TestTradeConfirmEntriesForwardsEveryPair(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	r := testReader(crcBytes([][2]uint32{{100, 200}, {300, 400}}))
	sp := &interaction2.OperationTradeConfirm{}
	sp.Decode(testLogger(), ctx)(&r, nil)

	entries := tradeConfirmEntries(sp.Entries())
	if len(entries) != 2 {
		t.Fatalf("entries: got %d, want 2", len(entries))
	}
	if entries[0].Data != 100 || entries[0].Crc != 200 {
		t.Errorf("entries[0]: got %+v, want {100 200}", entries[0])
	}
	if entries[1].Data != 300 || entries[1].Crc != 400 {
		t.Errorf("entries[1]: got %+v, want {300 400}", entries[1])
	}
}

// TestTradeConfirmEntriesIsEmptyWhereTheClientSendsNoCrc pins the GMS v79
// shape: tradeCrcPresent is false there, so the decoded list is empty and the
// command legitimately carries none — an empty list is not a dropped field.
func TestTradeConfirmEntriesIsEmptyWhereTheClientSendsNoCrc(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	r := testReader(nil)
	sp := &interaction2.OperationTradeConfirm{}
	sp.Decode(testLogger(), ctx)(&r, nil)

	entries := tradeConfirmEntries(sp.Entries())
	if entries == nil {
		t.Fatal("entries: got nil, want an empty non-nil slice")
	}
	if len(entries) != 0 {
		t.Errorf("entries: got %d, want 0", len(entries))
	}
}

// TestTransactionEntriesForwardsEveryPair pins the TRANSACTION half, which
// carries the same pairs through a distinct codec type.
func TestTransactionEntriesForwardsEveryPair(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	r := testReader(crcBytes([][2]uint32{{7, 8}}))
	sp := &interaction2.OperationTransaction{}
	sp.Decode(testLogger(), ctx)(&r, nil)

	entries := transactionEntries(sp.Entries())
	if len(entries) != 1 {
		t.Fatalf("entries: got %d, want 1", len(entries))
	}
	if entries[0].Data != 7 || entries[0].Crc != 8 {
		t.Errorf("entries[0]: got %+v, want {7 8}", entries[0])
	}
}

// --- the refusal's per-version gate (enterError) -------------------------------

// TestRefuseWithEnterErrorSkipsAnUnboundKey pins the crash-sentinel guard.
// CharacterInteractionEnterResultErrorBody resolves the refusal byte through
// ResolveCode(..., "enterError", key), which returns 99 on a miss — self
// documented in libs/atlas-packet/resolve.go as "will likely cause a client
// crash". The shipped templates make that reachable for trade: gms_v92 and
// gms_v12 bind no enterError table at all and jms_185 binds only two keys, so
// on those tenants every (or nearly every) trade refusal would have written 99.
// A refusal with no bound code must be dropped, not sent.
func TestRefuseWithEnterErrorSkipsAnUnboundKey(t *testing.T) {
	orig := interactionEnterErrorConfigured
	defer func() { interactionEnterErrorConfigured = orig }()

	var asked string
	interactionEnterErrorConfigured = func(_ logrus.FieldLogger, _ context.Context, key interactioncb.CharacterInteractionEnterErrorMode) bool {
		asked = key
		return false
	}

	announced := 0
	refuseWithEnterError(testLogger(), context.Background(), 100,
		interactioncb.CharacterInteractionEnterErrorModeOtherRequests,
		func() { announced++ },
	)
	if announced != 0 {
		t.Errorf("announces: got %d, want 0 — an unbound key must not reach ResolveCode's 99 sentinel", announced)
	}
	if asked != interactioncb.CharacterInteractionEnterErrorModeOtherRequests {
		t.Errorf("gate consulted for key %q, want %q", asked, interactioncb.CharacterInteractionEnterErrorModeOtherRequests)
	}
}

// TestRefuseWithEnterErrorSendsABoundKey is the other half: the gate must not
// swallow the refusal on a version that DOES bind the key (every gms template
// except v12/v92), or a colliding create would be silently ignored everywhere.
func TestRefuseWithEnterErrorSendsABoundKey(t *testing.T) {
	orig := interactionEnterErrorConfigured
	defer func() { interactionEnterErrorConfigured = orig }()
	interactionEnterErrorConfigured = func(logrus.FieldLogger, context.Context, interactioncb.CharacterInteractionEnterErrorMode) bool {
		return true
	}

	announced := 0
	refuseWithEnterError(testLogger(), context.Background(), 100,
		interactioncb.CharacterInteractionEnterErrorModeOtherRequests,
		func() { announced++ },
	)
	if announced != 1 {
		t.Errorf("announces: got %d, want 1", announced)
	}
}
