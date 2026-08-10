package handler

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
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
