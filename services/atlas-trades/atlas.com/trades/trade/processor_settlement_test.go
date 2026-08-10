package trade

import (
	"atlas-trades/configuration"
	inventorydata "atlas-trades/data/inventory"
	sagadata "atlas-trades/data/saga"
	"atlas-trades/kafka/message"
	compartmentmsg "atlas-trades/kafka/message/compartment"
	sagamsg "atlas-trades/kafka/message/saga"
	trademsg "atlas-trades/kafka/message/trade"
	"atlas-trades/ledger"
	"atlas-trades/settlement"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// --- harness -----------------------------------------------------------------

// stagedQuantity is what each side stages out of its slot-1 stack in the
// settlement harnesses. It is a PARTIAL stack, so the giver's slot survives the
// trade and the receiver's matching stack has room to merge into.
const stagedQuantity = uint16(5)

// testStagedRoom is an OPEN room between 100 and 200 with one staged item each.
func testStagedRoom(t *testing.T) (*ProcessorImpl, *emitted) {
	t.Helper()
	p, e := testOpenRoom(t)
	for _, id := range []character.Id{100, 200} {
		if err := p.PutItem(uuid.New(), id, byte(inventory.TypeValueUse), stagingSourceSlot, stagedQuantity, 1); err != nil {
			t.Fatalf("put item for %d: %v", id, err)
		}
	}
	return p, e
}

// confirmBoth drives both sides through CONFIRM, carrying the given CRC lists.
func confirmBoth(t *testing.T, p *ProcessorImpl, owner []trademsg.CrcEntry, visitor []trademsg.CrcEntry) {
	t.Helper()
	if err := p.Confirm(uuid.New(), 100, owner); err != nil {
		t.Fatalf("owner confirm: %v", err)
	}
	if err := p.Confirm(uuid.New(), 200, visitor); err != nil {
		t.Fatalf("visitor confirm: %v", err)
	}
}

// testConfirmedRoom is a room both sides have confirmed, awaiting attestation.
func testConfirmedRoom(t *testing.T) (*ProcessorImpl, *emitted) {
	t.Helper()
	p, e := testStagedRoom(t)
	confirmBoth(t, p, nil, nil)
	return p, e
}

// confirmCrc is the CRC list both sides send with TRADE_CONFIRM in the CRC
// harness. Its Data is the staged template, which is what the reference client
// derives the pair from.
func confirmCrc() []trademsg.CrcEntry {
	return []trademsg.CrcEntry{{Data: uint32(stagingTemplateId), Crc: 0x1234ABCD}}
}

// testConfirmedRoomWithCrc is testConfirmedRoom on a version whose
// TRADE_CONFIRM carries a CRC list.
func testConfirmedRoomWithCrc(t *testing.T) (*ProcessorImpl, *emitted) {
	t.Helper()
	p, e := testStagedRoom(t)
	confirmBoth(t, p, confirmCrc(), confirmCrc())
	return p, e
}

// testConfirmedRoomWithFullInventory is a confirmed room in which only 100
// staged, and `receiver` has no free slot and no stack of the incoming template
// to merge into.
func testConfirmedRoomWithFullInventory(t *testing.T, receiver character.Id) (*ProcessorImpl, *emitted) {
	t.Helper()
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, stagedQuantity, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}

	// The receiver's compartment is rebuilt full: capacity 3, three slots taken,
	// and NOT ONE of them holding the template it is about to be handed — so the
	// incoming item needs a slot of its own and there is none.
	assets := map[assetKey]inventorydata.Asset{
		{100, inventory.TypeValueUse, stagingSourceSlot}: inventorydata.NewAsset(stagingAssetId, stagingSourceSlot, stagingTemplateId, 100, 0),
	}
	for i := slot.Position(1); i <= 3; i++ {
		assets[assetKey{receiver, inventory.TypeValueUse, i}] = inventorydata.NewAsset(asset.Id(9000+i), i, item.Id(3000000+int(i)), 1, 0)
	}
	p.invp = &fakeInventory{assets: assets, capacity: 3}

	confirmBoth(t, p, nil, nil)
	return p, e
}

// testConfirmedRoomWithMeso is a confirmed room in which characterId staged
// `amount` mesos and neither side staged an item.
func testConfirmedRoomWithMeso(t *testing.T, characterId character.Id, amount uint32) (*ProcessorImpl, *emitted) {
	t.Helper()
	p, e := testOpenRoomWithMeso(t, characterId, amount*2)
	if err := p.AddMeso(uuid.New(), characterId, int32(amount)); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	if got := mesoStagedBy(t, p, characterId); got != amount {
		t.Fatalf("staged meso: got %d, want %d", got, amount)
	}
	confirmBoth(t, p, nil, nil)
	return p, e
}

// testSettlingRoom is a room whose settlement saga is in flight.
func testSettlingRoom(t *testing.T) (*ProcessorImpl, *emitted) {
	t.Helper()
	p, e := testConfirmedRoom(t)
	for _, id := range []character.Id{100, 200} {
		if err := p.Attest(uuid.New(), id, nil); err != nil {
			t.Fatalf("attest for %d: %v", id, err)
		}
	}
	room, ok := p.RoomForCharacter(100)
	if !ok {
		t.Fatal("the room did not survive attestation")
	}
	if room.State() != StateSettling {
		t.Fatalf("state: got %s, want %s", room.State(), StateSettling)
	}
	return p, e
}

// --- assertions ---------------------------------------------------------------

// assertEventOfType requires at least one status event of the given type.
func assertEventOfType(t *testing.T, e *emitted, eventType string) {
	t.Helper()
	if evs := statusEvents[json.RawMessage](t, e, eventType); len(evs) == 0 {
		t.Errorf("%s events: got 0, want at least 1", eventType)
	}
}

// assertCancelledWithReason requires exactly one CANCELLED carrying the given
// leaveReason key.
func assertCancelledWithReason(t *testing.T, e *emitted, reason string) {
	t.Helper()
	evs := statusEvents[trademsg.CancelledEventBody](t, e, trademsg.StatusTypeCancelled)
	if len(evs) != 1 {
		t.Fatalf("CANCELLED events: got %d, want 1", len(evs))
	}
	if evs[0].Body.Reason != reason {
		t.Errorf("cancel reason: got %s, want %s", evs[0].Body.Reason, reason)
	}
}

// collectSagas decodes every COMMAND_TOPIC_SAGA message. It decodes through the
// SHARED library's Saga, whose Step.UnmarshalJSON resolves the payload to its
// concrete type — decoding into a local struct would hand back a map[string]any
// and every payload-type assertion below would pass vacuously.
func collectSagas(t *testing.T, e *emitted) []sharedsaga.Saga {
	t.Helper()
	var out []sharedsaga.Saga
	for _, raw := range e.messages(t, sagamsg.EnvCommandTopic) {
		var s sharedsaga.Saga
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatalf("decode saga command: %v", err)
		}
		out = append(out, s)
	}
	return out
}

func assertNoSagaSubmitted(t *testing.T, e *emitted) {
	t.Helper()
	if sagas := collectSagas(t, e); len(sagas) != 0 {
		t.Errorf("saga commands: got %d, want 0", len(sagas))
	}
}

// settlementPayloadOf requires the saga to be the one-step trade_settlement
// composite and returns its CONCRETE payload.
func settlementPayloadOf(t *testing.T, s sharedsaga.Saga) sharedsaga.TradeSettlementPayload {
	t.Helper()
	if len(s.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(s.Steps))
	}
	payload, ok := s.Steps[0].Payload.(sharedsaga.TradeSettlementPayload)
	if !ok {
		t.Fatalf("payload type: got %T, want TradeSettlementPayload", s.Steps[0].Payload)
	}
	return payload
}

// readLedger returns every entry either trading character appears on.
func readLedger(t *testing.T, p *ProcessorImpl) []ledger.Model {
	t.Helper()
	entries, err := ledger.NewProcessor(p.l, p.ctx, p.db).GetByCharacterId(100, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	return entries
}

// --- CONFIRM ------------------------------------------------------------------

// TestConfirmDoesNotBroadcastModeSeventeenOnFirstConfirm pins design §6.2:
// mode 17 auto-replies with an attestation, so sending it on the first confirm
// would let one side drive the other's attestation without its owner ever
// pressing Trade.
func TestConfirmDoesNotBroadcastModeSeventeenOnFirstConfirm(t *testing.T) {
	p, e := testStagedRoom(t)
	if err := p.Confirm(uuid.New(), 100, nil); err != nil {
		t.Fatalf("confirm: %v", err)
	}

	assertEventOfType(t, e, trademsg.StatusTypeParticipantConfirmed)
	assertNoEventOfType(t, e, trademsg.StatusTypeAttestationRequested)

	room, ok := p.RoomForCharacter(100)
	if !ok {
		t.Fatal("the room did not survive the confirm")
	}
	if room.State() != StateOpen {
		t.Errorf("state after one confirm: got %s, want %s", room.State(), StateOpen)
	}
}

// TestConfirmFreezesStaging pins FR-3.6 on the confirm path specifically: from
// the FIRST confirm the room refuses further staging from BOTH sides.
func TestConfirmFreezesStaging(t *testing.T) {
	p, e := testStagedRoom(t)
	if err := p.Confirm(uuid.New(), 100, nil); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	before := len(stagedItemsOf(t, p, 200))

	if err := p.PutItem(uuid.New(), 200, byte(inventory.TypeValueUse), 2, 1, 2); err != nil {
		t.Fatalf("put item: %v", err)
	}
	if err := p.AddMeso(uuid.New(), 200, 1_000); err != nil {
		t.Fatalf("add meso: %v", err)
	}

	if got := len(stagedItemsOf(t, p, 200)); got != before {
		t.Errorf("staged items: got %d, want the pre-confirm %d", got, before)
	}
	if got := mesoStagedBy(t, p, 200); got != 0 {
		t.Errorf("mesoStaged: got %d, want 0", got)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeMesoStaged)
}

// TestBothConfirmsEnterAwaitingAttestation pins design §3.1/§6.2.
func TestBothConfirmsEnterAwaitingAttestation(t *testing.T) {
	p, e := testConfirmedRoom(t)

	assertEventOfType(t, e, trademsg.StatusTypeAttestationRequested)
	room, ok := p.RoomForCharacter(100)
	if !ok {
		t.Fatal("the room did not survive the confirms")
	}
	if room.State() != StateAwaitingAttestation {
		t.Errorf("state: got %s, want %s", room.State(), StateAwaitingAttestation)
	}
	assertNoSagaSubmitted(t, e)
}

// TestDoubleConfirmFromOneSideIsIgnored pins that a repeated confirm cannot
// stand in for the counterparty's.
func TestDoubleConfirmFromOneSideIsIgnored(t *testing.T) {
	p, e := testStagedRoom(t)
	for i := 0; i < 2; i++ {
		if err := p.Confirm(uuid.New(), 100, nil); err != nil {
			t.Fatalf("confirm %d: %v", i, err)
		}
	}

	assertNoEventOfType(t, e, trademsg.StatusTypeAttestationRequested)
	if got := len(statusEvents[trademsg.ParticipantConfirmedEventBody](t, e, trademsg.StatusTypeParticipantConfirmed)); got != 1 {
		t.Errorf("PARTICIPANT_CONFIRMED events: got %d, want 1 — the second confirm must be dropped, not re-broadcast", got)
	}
	room, _ := p.RoomForCharacter(100)
	if room.State() != StateOpen {
		t.Errorf("state: got %s, want %s", room.State(), StateOpen)
	}
}

// TestConfirmOnASoloRoomDoesNotRequestAttestation pins that an unpaired room
// can never reach the attestation round trip: BothConfirmed counts seats, not
// just confirm flags.
func TestConfirmOnASoloRoomDoesNotRequestAttestation(t *testing.T) {
	p, e := testProcessor(t)
	if err := p.CreateRoom(uuid.New(), testField(t), 100, 3); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := p.Confirm(uuid.New(), 100, nil); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeAttestationRequested)
	assertNoSagaSubmitted(t, e)
}

// --- attestation ---------------------------------------------------------------

// TestBothAttestationsSubmitOneSaga pins design §6.3: exactly one saga, whose
// transactionId is the ledger idempotency key.
func TestBothAttestationsSubmitOneSaga(t *testing.T) {
	p, e := testConfirmedRoom(t)
	for _, id := range []character.Id{100, 200} {
		if err := p.Attest(uuid.New(), id, nil); err != nil {
			t.Fatalf("attest for %d: %v", id, err)
		}
	}

	sagas := collectSagas(t, e)
	if len(sagas) != 1 {
		t.Fatalf("sagas submitted: got %d, want 1", len(sagas))
	}
	if sagas[0].SagaType != sharedsaga.TradeTransaction {
		t.Errorf("sagaType: got %s, want %s", sagas[0].SagaType, sharedsaga.TradeTransaction)
	}
	if len(sagas[0].Steps) != 1 || sagas[0].Steps[0].Action != sharedsaga.TradeSettlement {
		t.Fatalf("steps: want one trade_settlement composite, got %+v", sagas[0].Steps)
	}

	room, ok := p.RoomForCharacter(100)
	if !ok {
		t.Fatal("the room did not survive attestation")
	}
	if room.State() != StateSettling {
		t.Errorf("state: got %s, want %s", room.State(), StateSettling)
	}
	if sagas[0].TransactionId != room.SettlementId() {
		t.Errorf("saga transactionId: got %s, want the room's settlement id %s", sagas[0].TransactionId, room.SettlementId())
	}
	payload := settlementPayloadOf(t, sagas[0])
	if payload.TransactionId != room.SettlementId() {
		t.Errorf("payload transactionId: got %s, want %s", payload.TransactionId, room.SettlementId())
	}
	if payload.Sides[0].CharacterId != 100 || payload.Sides[1].CharacterId != 200 {
		t.Errorf("payload sides: got %d and %d, want the owner (100) then the visitor (200)", payload.Sides[0].CharacterId, payload.Sides[1].CharacterId)
	}
	for i, side := range payload.Sides {
		if len(side.Items) != 1 {
			t.Fatalf("side %d items: got %d, want 1", i, len(side.Items))
		}
		if side.Items[0].AssetId != stagingAssetId || side.Items[0].Quantity != asset.Quantity(stagedQuantity) {
			t.Errorf("side %d item: got %+v, want asset %d quantity %d", i, side.Items[0], stagingAssetId, stagedQuantity)
		}
	}
}

// TestOneSidedAttestationDoesNotSettle pins that the second TRANSACTION is what
// releases the saga: a single side's attestation must not move the trade.
func TestOneSidedAttestationDoesNotSettle(t *testing.T) {
	p, e := testConfirmedRoom(t)
	if err := p.Attest(uuid.New(), 100, nil); err != nil {
		t.Fatalf("attest: %v", err)
	}
	assertNoSagaSubmitted(t, e)
	room, _ := p.RoomForCharacter(100)
	if room.State() != StateAwaitingAttestation {
		t.Errorf("state: got %s, want %s", room.State(), StateAwaitingAttestation)
	}
}

// TestAttestationTimeoutSettlesAnyway pins design §3.1: the attestation is
// defence in depth, not a liveness dependency. A client that never replies must
// not be able to wedge a trade.
func TestAttestationTimeoutSettlesAnyway(t *testing.T) {
	p, e := testConfirmedRoom(t)
	if err := p.Attest(uuid.New(), 100, nil); err != nil {
		t.Fatalf("attest: %v", err)
	}

	room, _ := p.RoomForCharacter(100)
	if err := p.ExpireAttestation(uuid.New(), room.Id()); err != nil {
		t.Fatalf("expire: %v", err)
	}

	if got := len(collectSagas(t, e)); got != 1 {
		t.Errorf("sagas submitted: got %d, want 1 — the attestation timeout did not settle the trade", got)
	}
}

// TestAttestationDeadlineIsArmedOnTheSecondConfirm pins that the deadline is
// real rather than a method nothing calls: after both confirms the timer fires
// on its own and settles the trade.
func TestAttestationDeadlineIsArmedOnTheSecondConfirm(t *testing.T) {
	p, e := testStagedRoom(t)
	p.cfg = fakeConfig{cfg: configuration.DefaultConfig().WithAttestationTimeout(10 * time.Millisecond)}
	confirmBoth(t, p, nil, nil)

	deadline := time.Now().Add(2 * time.Second)
	for {
		if len(collectSagas(t, e)) == 1 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the armed attestation deadline never settled the trade")
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// TestExpireAttestationDoesNothingOnceSettling pins the deadline's re-check: a
// timer that fires after both sides attested must not submit a second saga.
func TestExpireAttestationDoesNothingOnceSettling(t *testing.T) {
	p, e := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)

	if err := p.ExpireAttestation(uuid.New(), room.Id()); err != nil {
		t.Fatalf("expire: %v", err)
	}
	if got := len(collectSagas(t, e)); got != 1 {
		t.Errorf("sagas submitted: got %d, want the original 1", got)
	}
}

// realWindowCrc is the CRC list the reference client ACTUALLY sends with
// TRADE_CONFIRM once both sides have staged. CTradingRoomDlg::Trade @0x7c39a0
// walks BOTH dialog arrays (members 113 and 114), so the confirm carries every
// item in the window; CTradingRoomDlg::OnTrade @0x7c20bc walks member 114 only,
// so the attestation carries just the counterparty's. testStagedRoom stages
// stagingTemplateId on both sides and CItemInfo::GetItemCRC keys on the
// TEMPLATE, so the window's two pairs are identical.
func realWindowCrc() []trademsg.CrcEntry {
	return []trademsg.CrcEntry{
		{Data: uint32(stagingTemplateId), Crc: 0x1234ABCD},
		{Data: uint32(stagingTemplateId), Crc: 0x1234ABCD},
	}
}

// TestAttestationIsTheCounterpartyContributionNotTheWholeWindow is the
// regression test for the item-for-item trade. Both sides stage, so each
// client's confirm carries two pairs and its attestation carries one. Comparing
// the two lists directly refused every real two-sided trade with
// TRADE_CRC_FAILED -- the client's "the game file has been damaged" notice.
//
// It went unnoticed because a one-sided trade leaves the giver's attestation
// empty (which is lenient) and the receiver's window holding a single item,
// making the two lists coincidentally equal.
func TestAttestationIsTheCounterpartyContributionNotTheWholeWindow(t *testing.T) {
	p, e := testStagedRoom(t)
	confirmBoth(t, p, realWindowCrc(), realWindowCrc())

	if err := p.Attest(uuid.New(), 100, confirmCrc()); err != nil {
		t.Fatalf("owner attest: %v", err)
	}
	if err := p.Attest(uuid.New(), 200, confirmCrc()); err != nil {
		t.Fatalf("visitor attest: %v", err)
	}

	if got := len(collectSagas(t, e)); got != 1 {
		t.Fatalf("sagas submitted: got %d, want 1 — a faithful two-sided attestation must settle", got)
	}
}

// TestAttestationWithAnUnconfirmedCrcIsRefused pins that relaxing the length
// comparison did not relax the tamper check: a pair this side never reported at
// confirm time is still a mismatch, which is the case the check exists for.
func TestAttestationWithAnUnconfirmedCrcIsRefused(t *testing.T) {
	p, e := testStagedRoom(t)
	confirmBoth(t, p, realWindowCrc(), realWindowCrc())

	if err := p.Attest(uuid.New(), 100, []trademsg.CrcEntry{{Data: uint32(stagingTemplateId), Crc: 0xDEADBEEF}}); err != nil {
		t.Fatalf("owner attest: %v", err)
	}
	if err := p.Attest(uuid.New(), 200, confirmCrc()); err != nil {
		t.Fatalf("visitor attest: %v", err)
	}

	assertCancelledWithReason(t, e, ReasonTradeCrcFailed)
	assertNoSagaSubmitted(t, e)
}

// TestAttestationNamingMoreItemsThanTheCounterpartyStagedIsRefused pins the
// count half: the attestation must name exactly what the counterparty staged,
// so a client claiming to receive two items when one was staged is refused even
// though every pair it names is one it confirmed.
func TestAttestationNamingMoreItemsThanTheCounterpartyStagedIsRefused(t *testing.T) {
	p, e := testStagedRoom(t)
	confirmBoth(t, p, realWindowCrc(), realWindowCrc())

	if err := p.Attest(uuid.New(), 100, realWindowCrc()); err != nil {
		t.Fatalf("owner attest: %v", err)
	}
	if err := p.Attest(uuid.New(), 200, confirmCrc()); err != nil {
		t.Fatalf("visitor attest: %v", err)
	}

	assertCancelledWithReason(t, e, ReasonTradeCrcFailed)
	assertNoSagaSubmitted(t, e)
}

// TestCrcMismatchTearsDownWithStatusThirteen pins design §6.1 check 4.
func TestCrcMismatchTearsDownWithStatusThirteen(t *testing.T) {
	p, e := testConfirmedRoomWithCrc(t)
	if err := p.Attest(uuid.New(), 100, []trademsg.CrcEntry{{Data: uint32(stagingTemplateId), Crc: 0xDEADBEEF}}); err != nil {
		t.Fatalf("owner attest: %v", err)
	}
	if err := p.Attest(uuid.New(), 200, confirmCrc()); err != nil {
		t.Fatalf("visitor attest: %v", err)
	}

	assertCancelledWithReason(t, e, ReasonTradeCrcFailed)
	assertNoSagaSubmitted(t, e)
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("room survived a CRC mismatch")
	}
}

// TestMatchingCrcSettles guards the CRC check against over-reach: the identical
// list, reordered, is not a mismatch.
func TestMatchingCrcSettles(t *testing.T) {
	p, e := testConfirmedRoomWithCrc(t)
	for _, id := range []character.Id{100, 200} {
		if err := p.Attest(uuid.New(), id, confirmCrc()); err != nil {
			t.Fatalf("attest for %d: %v", id, err)
		}
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeCancelled)
	if got := len(collectSagas(t, e)); got != 1 {
		t.Errorf("sagas submitted: got %d, want 1", got)
	}
}

// TestAttestationAbsentOnLegacyVersionsIsNotAMismatch pins design §4.4: GMS <=
// v79 sends no CRC list at all, so an empty attestation must settle rather than
// read as a tampered one.
func TestAttestationAbsentOnLegacyVersionsIsNotAMismatch(t *testing.T) {
	p, e := testConfirmedRoomWithCrc(t)
	for _, id := range []character.Id{100, 200} {
		if err := p.Attest(uuid.New(), id, nil); err != nil {
			t.Fatalf("attest for %d: %v", id, err)
		}
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeCancelled)
	if got := len(collectSagas(t, e)); got != 1 {
		t.Errorf("sagas submitted: got %d, want 1", got)
	}
}

// --- settlement pre-checks ------------------------------------------------------

// TestPreCheckFailureTearsDownRatherThanKeepingTheRoom pins design §6.1's
// correction of PRD FR-4.9: CTradingRoomDlg::OnLeave closes the dialog before
// showing the notice, so there is no client state in which the room survives a
// status 8/9/13.
func TestPreCheckFailureTearsDownRatherThanKeepingTheRoom(t *testing.T) {
	p, e := testConfirmedRoomWithFullInventory(t, 200)
	for _, id := range []character.Id{100, 200} {
		if err := p.Attest(uuid.New(), id, nil); err != nil {
			t.Fatalf("attest for %d: %v", id, err)
		}
	}

	assertCancelledWithReason(t, e, ReasonTradeCannotCarry)
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("room survived a settlement pre-check failure")
	}
	assertNoSagaSubmitted(t, e)
}

// TestAFullInventoryStillMergesIntoAMatchingStack guards the free-slot check
// against over-reach: atlas-inventory's Accept merges into an existing stack
// with room, so a full compartment that already holds the incoming template is
// not a refusal.
func TestAFullInventoryStillMergesIntoAMatchingStack(t *testing.T) {
	p, e := testOpenRoom(t)
	if err := p.PutItem(uuid.New(), 100, byte(inventory.TypeValueUse), stagingSourceSlot, stagedQuantity, 1); err != nil {
		t.Fatalf("put item: %v", err)
	}
	// 200's compartment is full — but slot 1 holds the very template it is
	// about to receive, with room under the 200-unit stack ceiling.
	assets := map[assetKey]inventorydata.Asset{
		{100, inventory.TypeValueUse, stagingSourceSlot}: inventorydata.NewAsset(stagingAssetId, stagingSourceSlot, stagingTemplateId, 100, 0),
		{200, inventory.TypeValueUse, 1}:                 inventorydata.NewAsset(9001, 1, stagingTemplateId, 100, 0),
		{200, inventory.TypeValueUse, 2}:                 inventorydata.NewAsset(9002, 2, 3000001, 1, 0),
		{200, inventory.TypeValueUse, 3}:                 inventorydata.NewAsset(9003, 3, 3000002, 1, 0),
	}
	p.invp = &fakeInventory{assets: assets, capacity: 3}

	confirmBoth(t, p, nil, nil)
	for _, id := range []character.Id{100, 200} {
		if err := p.Attest(uuid.New(), id, nil); err != nil {
			t.Fatalf("attest for %d: %v", id, err)
		}
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeCancelled)
	if got := len(collectSagas(t, e)); got != 1 {
		t.Errorf("sagas submitted: got %d, want 1", got)
	}
}

// TestSettlementRefusesWhenTheStagedAssetIsGone pins design §6.1 check 3: the
// reservation is not observable from here, so what is verified is the state it
// exists to protect. A vanished asset is TRADE_FAILED (8), not CANNOT_CARRY.
func TestSettlementRefusesWhenTheStagedAssetIsGone(t *testing.T) {
	p, e := testConfirmedRoom(t)
	// The owner's staged stack disappears from their inventory entirely.
	delete(p.invp.(*fakeInventory).assets, assetKey{100, inventory.TypeValueUse, stagingSourceSlot})

	for _, id := range []character.Id{100, 200} {
		if err := p.Attest(uuid.New(), id, nil); err != nil {
			t.Fatalf("attest for %d: %v", id, err)
		}
	}
	assertCancelledWithReason(t, e, ReasonTradeFailed)
	assertNoSagaSubmitted(t, e)
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("room survived a lost reservation")
	}
}

// TestSettlementRefusesWhenTheStagedMesoIsNoLongerHeld pins design §6.1 check 2
// against a player who staged mesos and then spent them.
func TestSettlementRefusesWhenTheStagedMesoIsNoLongerHeld(t *testing.T) {
	p, e := testConfirmedRoomWithMeso(t, 100, 1_000_000)
	p.cp = &fakeCharacters{rows: map[character.Id]testCharacter{
		100: {Id: 100, Name: "Owner", Hp: 100, Level: 30, Meso: 1},
		200: {Id: 200, Name: "Guest", Hp: 100, Level: 30, Meso: 1_000_000},
	}}

	for _, id := range []character.Id{100, 200} {
		if err := p.Attest(uuid.New(), id, nil); err != nil {
			t.Fatalf("attest for %d: %v", id, err)
		}
	}
	assertCancelledWithReason(t, e, ReasonTradeCannotCarry)
	assertNoSagaSubmitted(t, e)
}

// TestSettlementTaxIsResolvedBeforeTheSagaLeaves pins design §6.3: the tax is
// computed in atlas-trades (it needs the tenant config) and passed to the
// orchestrator as resolved integers.
func TestSettlementTaxIsResolvedBeforeTheSagaLeaves(t *testing.T) {
	p, e := testConfirmedRoomWithMeso(t, 100, 10_000_000)
	for _, id := range []character.Id{100, 200} {
		if err := p.Attest(uuid.New(), id, nil); err != nil {
			t.Fatalf("attest for %d: %v", id, err)
		}
	}

	sagas := collectSagas(t, e)
	if len(sagas) != 1 {
		t.Fatalf("sagas submitted: got %d, want 1", len(sagas))
	}
	payload := settlementPayloadOf(t, sagas[0])
	if payload.Sides[0].MesoStaged != 10_000_000 {
		t.Errorf("mesoStaged: got %d, want 10000000", payload.Sides[0].MesoStaged)
	}
	if payload.Sides[0].MesoTax != 400_000 {
		t.Errorf("mesoTax: got %d, want 400000", payload.Sides[0].MesoTax)
	}
	if payload.Sides[0].MesoDelivered != 9_600_000 {
		t.Errorf("mesoDelivered: got %d, want 9600000", payload.Sides[0].MesoDelivered)
	}
}

// TestSettlementResolvesAStagedSlotThatMovedSinceTheStage pins the correction
// the CONFIRM freeze does NOT provide. Nothing gates an inventory move while
// the dialog is open, and the refresh only corrects on a tick boundary, so an
// item dragged in the last tick interval before CONFIRM would otherwise be
// handed to the orchestrator at its vacated slot — which the expander rejects
// as a different asset instance, turning a valid trade into LEAVE 8.
func TestSettlementResolvesAStagedSlotThatMovedSinceTheStage(t *testing.T) {
	p, e := testConfirmedRoom(t)
	p.invp.(*fakeInventory).relocate(100, inventory.TypeValueUse, stagingSourceSlot, 8)

	for _, id := range []character.Id{100, 200} {
		if err := p.Attest(uuid.New(), id, nil); err != nil {
			t.Fatalf("attest for %d: %v", id, err)
		}
	}

	sagas := collectSagas(t, e)
	if len(sagas) != 1 {
		t.Fatalf("sagas submitted: got %d, want 1", len(sagas))
	}
	payload := settlementPayloadOf(t, sagas[0])
	if got := payload.Sides[0].Items[0].SourceSlot; got != 8 {
		t.Errorf("settled sourceSlot: got %d, want the asset's CURRENT slot 8", got)
	}
	if got := stagedItemsOf(t, p, 100)[0].SourceSlot(); got != 8 {
		t.Errorf("staged sourceSlot: got %d, want the corrected 8 written back", got)
	}
}

// --- terminal saga status --------------------------------------------------------

// TestCancelLosesToSettlement pins FR-6.5 / design §3.3: once the room is
// SETTLING, a cancel is recorded and ignored; the saga's terminal status
// produces the client's LEAVE.
func TestCancelLosesToSettlement(t *testing.T) {
	p, e := testSettlingRoom(t)
	if err := p.TeardownCharacter(uuid.New(), 100, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}

	assertNoEventOfType(t, e, trademsg.StatusTypeCancelled)
	if _, ok := p.RoomForCharacter(100); !ok {
		t.Error("a cancel tore down a settling room")
	}
}

// TestSettlementSuccessWritesTheLedgerAndEmitsSettled pins FR-5.6 / FR-7.1 and
// design §6.4's ordering: SETTLED is emitted only after the saga reports
// terminal success.
func TestSettlementSuccessWritesTheLedgerAndEmitsSettled(t *testing.T) {
	p, e := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)

	if err := p.SettlementSucceeded(uuid.New(), room.SettlementId()); err != nil {
		t.Fatalf("settle: %v", err)
	}

	assertEventOfType(t, e, trademsg.StatusTypeSettled)
	entries := readLedger(t, p)
	if len(entries) != 1 {
		t.Fatalf("ledger entries: got %d, want 1", len(entries))
	}
	if entries[0].TransactionId() != room.SettlementId() {
		t.Errorf("ledger transactionId: got %s, want the settlement saga's %s", entries[0].TransactionId(), room.SettlementId())
	}
	if len(entries[0].Sides()) != 2 {
		t.Fatalf("ledger sides: got %d, want 2", len(entries[0].Sides()))
	}
	for _, side := range entries[0].Sides() {
		if len(side.Items()) != 1 {
			t.Errorf("ledger side %d items: got %d, want 1", side.CharacterId(), len(side.Items()))
		}
	}
	settled := statusEvents[trademsg.SettledEventBody](t, e, trademsg.StatusTypeSettled)
	if len(settled) != 1 {
		t.Fatalf("SETTLED events: got %d, want 1", len(settled))
	}
	if settled[0].Body.LedgerEntryId != entries[0].Id() {
		t.Errorf("SETTLED ledgerEntryId: got %s, want %s", settled[0].Body.LedgerEntryId, entries[0].Id())
	}
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("room survived settlement")
	}
}

// TestSettlementSuccessCancelsBothHolds pins the obligation a successful trade
// creates. TradeSettlementItem carries no reservation id, so the saga cannot
// release the holds, and atlas-inventory's Release does not touch its
// reservation registry — an uncancelled hold sits on the giver's now-emptied
// slot for the rest of its TTL, refusing that slot's merge, drop and any fresh
// reserve.
func TestSettlementSuccessCancelsBothHolds(t *testing.T) {
	p, e := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)
	want := map[uuid.UUID]character.Id{}
	for _, id := range []character.Id{100, 200} {
		for _, i := range stagedItemsOf(t, p, id) {
			want[i.ReservationId()] = id
		}
	}
	if len(want) != 2 {
		t.Fatalf("staged reservations: got %d, want 2", len(want))
	}

	if err := p.SettlementSucceeded(uuid.New(), room.SettlementId()); err != nil {
		t.Fatalf("settle: %v", err)
	}

	cancels := compartmentCommands[compartmentmsg.CancelReservationCommandBody](t, e, compartmentmsg.CommandCancelReservation)
	if len(cancels) != 2 {
		t.Fatalf("CANCEL_RESERVATION commands: got %d, want one per staged item (2)", len(cancels))
	}
	for _, c := range cancels {
		if _, ok := want[c.TransactionId]; !ok {
			t.Errorf("cancelled reservation %s was never staged", c.TransactionId)
			continue
		}
		delete(want, c.TransactionId)
	}
	if len(want) != 0 {
		t.Errorf("reservations left uncancelled after a SUCCESSFUL trade: %v", want)
	}
}

// TestSecondCompletedDeliveryFindsNoRecordAndDrops pins what actually stops a
// redelivered COMPLETED: the DURABLE RECORD is gone, because the first delivery
// deleted it. The room is gone by then too, but the room is not what decides —
// after a restart there is no room at all, and the record still has to be the
// thing that stops the second delivery.
//
// (ledger.Record's own idempotency is pinned in the ledger package; the same
// arbiter under a CONCURRENT second delivery is pinned by
// TestSettlementSuccessDoesNotReEmitWhenAnotherDeliveryResolvedTheRecord.)
func TestSecondCompletedDeliveryFindsNoRecordAndDrops(t *testing.T) {
	p, e := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)

	for i := 0; i < 2; i++ {
		if err := p.SettlementSucceeded(uuid.New(), room.SettlementId()); err != nil {
			t.Fatalf("settle %d: %v", i, err)
		}
	}
	if got := len(readLedger(t, p)); got != 1 {
		t.Errorf("ledger entries: got %d, want 1", got)
	}
	if got := len(statusEvents[trademsg.SettledEventBody](t, e, trademsg.StatusTypeSettled)); got != 1 {
		t.Errorf("SETTLED events: got %d, want 1", got)
	}
	assertSettlementResolved(t, p, room.SettlementId())
}

// TestSettlementFailureEmitsStatusEightAndWritesNoLedgerRow pins FR-5.3 and
// FR-7.3: a failed trade is observable via logs and metrics only.
func TestSettlementFailureEmitsStatusEightAndWritesNoLedgerRow(t *testing.T) {
	p, e := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)

	if err := p.SettlementFailed(uuid.New(), room.SettlementId(), "NOT_ENOUGH_MESOS"); err != nil {
		t.Fatalf("fail: %v", err)
	}

	assertCancelledWithReason(t, e, ReasonTradeFailed)
	if entries := readLedger(t, p); len(entries) != 0 {
		t.Errorf("ledger entries: got %d, want 0 for a failed trade", len(entries))
	}
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("room survived a failed settlement")
	}
	if got := len(compartmentCommands[compartmentmsg.CancelReservationCommandBody](t, e, compartmentmsg.CommandCancelReservation)); got != 2 {
		t.Errorf("CANCEL_RESERVATION commands: got %d, want one per staged item (2)", got)
	}
}

// TestTerminalStatusIsKeyedByTheSettlementRecord pins the identity the whole
// terminal path now hangs on. The FAILED body names ONE character — the failed
// expanded step's — which is not a role, and the room may not exist at all
// after a restart, so the settlement transaction id is the only usable handle.
func TestTerminalStatusIsKeyedByTheSettlementRecord(t *testing.T) {
	p, _ := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)

	s, err := settlement.NewProcessor(p.l, p.ctx, p.db).GetByTransactionId(room.SettlementId())
	if err != nil {
		t.Fatalf("the submitted settlement was not recorded durably: %v", err)
	}
	if s.RoomId() != room.Id() {
		t.Errorf("recorded roomId: got %s, want %s", s.RoomId(), room.Id())
	}
	if s.OwnerId() != 100 || s.VisitorId() != 200 {
		t.Errorf("recorded participants: got %d and %d, want 100 and 200", s.OwnerId(), s.VisitorId())
	}
	if len(s.Sides()) != 2 {
		t.Fatalf("recorded sides: got %d, want 2", len(s.Sides()))
	}
	if s.Sides()[0].CharacterId() != 100 || s.Sides()[1].CharacterId() != 200 {
		t.Errorf("recorded side order: got %d then %d, want the owner then the visitor", s.Sides()[0].CharacterId(), s.Sides()[1].CharacterId())
	}
	for _, side := range s.Sides() {
		if len(side.Items()) != 1 {
			t.Fatalf("recorded items for %d: got %d, want 1", side.CharacterId(), len(side.Items()))
		}
		if side.Items()[0].ReservationId() == uuid.Nil {
			t.Error("recorded item carries no reservation id; a reconciled settlement could never cancel the hold")
		}
	}
}

// TestSettlementStatusForAnUnknownRoomIsIgnored pins that a redelivered or
// foreign terminal status is a no-op rather than an error.
func TestSettlementStatusForAnUnknownRoomIsIgnored(t *testing.T) {
	p, e := testStagedRoom(t)
	if err := p.SettlementSucceeded(uuid.New(), uuid.New()); err != nil {
		t.Fatalf("settle: %v", err)
	}
	if err := p.SettlementFailed(uuid.New(), uuid.New(), "UNKNOWN"); err != nil {
		t.Fatalf("fail: %v", err)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeSettled)
	assertNoEventOfType(t, e, trademsg.StatusTypeCancelled)
	if got := len(readLedger(t, p)); got != 0 {
		t.Errorf("ledger entries: got %d, want 0", got)
	}
}

// --- cancel vs settle, raced across the teardown's REST reads ------------------

// beginSettlingOnce drives the room to SETTLING the FIRST time a compartment is
// read, which is the real window: every teardown resolves its staged slots over
// REST between reading the room and ending it, and the attestation deadline
// runs settle from an independent goroutine that no Kafka partition ordering
// serialises against the teardown consumer.
func beginSettlingOnce(t *testing.T, p *ProcessorImpl, roomId uuid.UUID) func() {
	t.Helper()
	var once sync.Once
	return func() {
		once.Do(func() {
			if _, err := p.reg.Update(p.t, roomId, func(cur Room) (Room, error) {
				return cur.WithState(StateSettling).WithSettlementId(uuid.New()), nil
			}); err != nil {
				t.Errorf("move to settling: %v", err)
			}
		})
	}
}

// TestTeardownLosesToASettlementThatWinsDuringItsReads pins FR-6.5 through the
// window a plain state read leaves open. Without the compare-and-set the
// teardown would cancel both holds the in-flight saga is about to consume AND
// delete the room its terminal status must find — so the swap would execute
// with no ledger row and no SETTLED, while both clients had already seen
// LEAVE 2.
func TestTeardownLosesToASettlementThatWinsDuringItsReads(t *testing.T) {
	p, e := testConfirmedRoom(t)
	room, _ := p.RoomForCharacter(100)
	p.invp.(*fakeInventory).onGetCompartment = beginSettlingOnce(t, p, room.Id())

	if err := p.TeardownCharacter(uuid.New(), 200, ReasonTradeCancelled); err != nil {
		t.Fatalf("teardown: %v", err)
	}

	assertNoEventOfType(t, e, trademsg.StatusTypeCancelled)
	assertNoCompartmentCommandOfType(t, e, compartmentmsg.CommandCancelReservation)
	survivor, ok := p.RoomForCharacter(100)
	if !ok {
		t.Fatal("the teardown deleted a room whose settlement had already started")
	}
	if survivor.State() != StateSettling {
		t.Errorf("state: got %s, want %s", survivor.State(), StateSettling)
	}
}

// TestSettlementRefusalLosesToASettlementThatWinsDuringItsReads pins the same
// compare-and-set on the OTHER teardown caller: a pre-check refusal also runs
// REST reads before it ends the room, and another trigger can reach SETTLING
// inside that window.
func TestSettlementRefusalLosesToASettlementThatWinsDuringItsReads(t *testing.T) {
	p, e := testConfirmedRoomWithFullInventory(t, 200)
	room, _ := p.RoomForCharacter(100)
	p.invp.(*fakeInventory).onGetCompartment = beginSettlingOnce(t, p, room.Id())

	for _, id := range []character.Id{100, 200} {
		if err := p.Attest(uuid.New(), id, nil); err != nil {
			t.Fatalf("attest for %d: %v", id, err)
		}
	}

	assertNoEventOfType(t, e, trademsg.StatusTypeCancelled)
	assertNoCompartmentCommandOfType(t, e, compartmentmsg.CommandCancelReservation)
	if _, ok := p.RoomForCharacter(100); !ok {
		t.Error("a pre-check refusal deleted a room whose settlement had already started")
	}
}

// TestSettlementSuccessDoesNotReEmitWhenAnotherDeliveryResolvedTheRecord pins
// the arbiter. The durable settlement record — not the room — decides who owns
// a terminal outcome, because after a restart there IS no room to race over:
// whoever deletes the record emits, and everyone else drops.
//
// The stale room left standing here is exactly what reconciliation leaves
// behind when it completes a settlement whose room this process still holds
// from a previous delivery, so the room must not be able to authorise a second
// SETTLED on its own.
func TestSettlementSuccessDoesNotReEmitWhenAnotherDeliveryResolvedTheRecord(t *testing.T) {
	p, e := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)

	// The competing delivery resolves the record, leaving the room behind.
	won, err := settlement.NewProcessor(p.l, p.ctx, p.db).Resolve(room.SettlementId())
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !won {
		t.Fatal("the settlement record was not there to resolve")
	}

	if err = p.SettlementSucceeded(uuid.New(), room.SettlementId()); err != nil {
		t.Fatalf("settle: %v", err)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeSettled)
	assertNoCompartmentCommandOfType(t, e, compartmentmsg.CommandCancelReservation)
	if got := len(readLedger(t, p)); got != 0 {
		t.Errorf("ledger entries: got %d, want 0 — the losing delivery recorded a trade the winner owns", got)
	}
}

// --- the attestation deadline vs a failed emit -----------------------------------

// TestConfirmArmsTheDeadlineEvenWhenTheEmitFails pins the wedge a failed emit
// would otherwise leave. The registry swap to AWAITING_ATTESTATION is in-memory
// and is NOT rolled back by the enclosing transaction, so a room whose confirm
// failed to publish still sits in that state — with no mode 17 ever reaching
// the clients, no attestation possible, and RefreshReservations keeping both
// sides' holds alive indefinitely. Only the deadline can end it.
func TestConfirmArmsTheDeadlineEvenWhenTheEmitFails(t *testing.T) {
	p, _ := testStagedRoom(t)
	room, _ := p.RoomForCharacter(100)
	// Break the outbox so every buffered message fails to publish.
	if err := p.db.Migrator().DropTable(&outbox.Entity{}); err != nil {
		t.Fatalf("drop outbox: %v", err)
	}

	if err := p.Confirm(uuid.New(), 100, nil); err == nil {
		t.Fatal("owner confirm: expected the broken outbox to surface an error")
	}
	if err := p.Confirm(uuid.New(), 200, nil); err == nil {
		t.Fatal("visitor confirm: expected the broken outbox to surface an error")
	}

	updated, ok := p.RoomForCharacter(100)
	if !ok {
		t.Fatal("the room did not survive the confirms")
	}
	if updated.State() != StateAwaitingAttestation {
		t.Fatalf("state: got %s, want %s — the in-memory swap is not rolled back", updated.State(), StateAwaitingAttestation)
	}
	if !p.timers.isArmed(p.t, room.Id()) {
		t.Error("no attestation deadline armed: the room is wedged in AWAITING_ATTESTATION for good")
	}
}

// TestSettlementRefusesWhenTheSlotMaxCannotBeRead pins design §7 on the
// settlement path's second atlas-data read: an unreadable stack ceiling makes
// the free-slot count unknowable, and assuming room is how a trade overflows an
// inventory.
func TestSettlementRefusesWhenTheSlotMaxCannotBeRead(t *testing.T) {
	p, e := testConfirmedRoom(t)
	p.idp = &fakeItemData{blocked: make(map[item.Id]bool), slotMaxErr: errors.New("atlas-data unreachable")}

	for _, id := range []character.Id{100, 200} {
		if err := p.Attest(uuid.New(), id, nil); err != nil {
			t.Fatalf("attest for %d: %v", id, err)
		}
	}
	assertCancelledWithReason(t, e, ReasonTradeCannotCarry)
	assertNoSagaSubmitted(t, e)
}

// TestTerminalStatusForATradeThatNeverSubmittedIsIgnored pins that a COMPLETED
// can only close a trade that actually submitted a settlement. A room still
// awaiting attestation has minted no settlement id and written no record, so
// the status resolves nothing: the id it carries is uuid.Nil, which the record
// lookup must refuse rather than match.
func TestTerminalStatusForATradeThatNeverSubmittedIsIgnored(t *testing.T) {
	p, e := testConfirmedRoom(t)
	room, _ := p.RoomForCharacter(100)
	if room.SettlementId() != uuid.Nil {
		t.Fatalf("settlementId before submission: got %s, want the zero id", room.SettlementId())
	}

	if err := p.SettlementSucceeded(uuid.New(), room.SettlementId()); err != nil {
		t.Fatalf("settle: %v", err)
	}
	assertNoEventOfType(t, e, trademsg.StatusTypeSettled)
	if got := len(readLedger(t, p)); got != 0 {
		t.Errorf("ledger entries: got %d, want 0", got)
	}
	if _, ok := p.RoomForCharacter(100); !ok {
		t.Error("a stray COMPLETED tore down a trade that had not submitted a settlement")
	}
}

// --- restart reconciliation -------------------------------------------------------

// fakeSagaOutcome stands in for atlas-saga-orchestrator's GET
// /sagas/{transactionId}. err models the two indistinguishable unknowns — the
// orchestrator unreachable, and a saga it has not consumed yet (a 404) — which
// must never be read as either terminal answer.
type fakeSagaOutcome struct {
	outcome sagadata.Outcome
	err     error
	calls   int
}

func (f *fakeSagaOutcome) Outcome(_ uuid.UUID) (sagadata.Outcome, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	return f.outcome, nil
}

// restart drops every trace of the in-memory registry for the test's tenant,
// which is what a process restart does to a live trade room. The durable
// settlement record and the outbox survive it, exactly as they would.
func restart(t *testing.T, p *ProcessorImpl) {
	t.Helper()
	for _, room := range p.reg.All(p.t) {
		p.reg.Remove(p.t, room.Id())
	}
	if got := len(p.reg.All(p.t)); got != 0 {
		t.Fatalf("rooms surviving the restart: got %d, want 0", got)
	}
}

// TestReconcileCompletesASettlementWhoseRoomTheRestartLost pins the hole this
// whole record exists for. The saga lives in atlas-saga-orchestrator and keeps
// running across an atlas-trades restart, so a trade that FULLY EXECUTED used
// to write no ledger row and emit no SETTLED — contradicting FR-7.1.
func TestReconcileCompletesASettlementWhoseRoomTheRestartLost(t *testing.T) {
	p, e := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)
	settlementId := room.SettlementId()
	restart(t, p)

	p.sagad = &fakeSagaOutcome{outcome: sagadata.OutcomeSucceeded}
	if err := p.ReconcileSettlements(); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	entries := readLedger(t, p)
	if len(entries) != 1 {
		t.Fatalf("ledger entries: got %d, want 1 — the executed trade left no audit record", len(entries))
	}
	if entries[0].TransactionId() != settlementId {
		t.Errorf("ledger transactionId: got %s, want the settlement saga's %s", entries[0].TransactionId(), settlementId)
	}
	if len(entries[0].Sides()) != 2 {
		t.Errorf("ledger sides: got %d, want 2", len(entries[0].Sides()))
	}

	settled := statusEvents[trademsg.SettledEventBody](t, e, trademsg.StatusTypeSettled)
	if len(settled) != 1 {
		t.Fatalf("SETTLED events: got %d, want 1", len(settled))
	}
	if settled[0].RoomId != room.Id() || settled[0].OwnerId != 100 || settled[0].VisitorId != 200 {
		t.Errorf("SETTLED envelope: got room %s owner %d visitor %d, want the record's %s / 100 / 200", settled[0].RoomId, settled[0].OwnerId, settled[0].VisitorId, room.Id())
	}
	if settled[0].Body.LedgerEntryId != entries[0].Id() {
		t.Errorf("SETTLED ledgerEntryId: got %s, want %s", settled[0].Body.LedgerEntryId, entries[0].Id())
	}

	// The holds must still be released: the room that would have cancelled them
	// is gone, so the record is the only thing that knows they exist.
	if got := len(compartmentCommands[compartmentmsg.CancelReservationCommandBody](t, e, compartmentmsg.CommandCancelReservation)); got != 2 {
		t.Errorf("CANCEL_RESERVATION commands: got %d, want one per staged item (2)", got)
	}
	assertSettlementResolved(t, p, settlementId)
}

// TestReconcileIsIdempotent pins that reconciliation is safe to run repeatedly
// — every boot runs it, and a boot loop must not double-credit or double-emit.
func TestReconcileIsIdempotent(t *testing.T) {
	p, e := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)
	restart(t, p)

	p.sagad = &fakeSagaOutcome{outcome: sagadata.OutcomeSucceeded}
	for i := 0; i < 2; i++ {
		if err := p.ReconcileSettlements(); err != nil {
			t.Fatalf("reconcile %d: %v", i, err)
		}
	}

	if got := len(readLedger(t, p)); got != 1 {
		t.Errorf("ledger entries: got %d, want 1", got)
	}
	if got := len(statusEvents[trademsg.SettledEventBody](t, e, trademsg.StatusTypeSettled)); got != 1 {
		t.Errorf("SETTLED events: got %d, want 1", got)
	}
	assertSettlementResolved(t, p, room.SettlementId())
}

// TestReconcileClosesASettlementWhoseSagaFailed pins the other terminal answer:
// a compensated saga produces LEAVE 8 and NO ledger row (FR-7.3).
func TestReconcileClosesASettlementWhoseSagaFailed(t *testing.T) {
	p, e := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)
	restart(t, p)

	p.sagad = &fakeSagaOutcome{outcome: sagadata.OutcomeFailed}
	if err := p.ReconcileSettlements(); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	assertCancelledWithReason(t, e, ReasonTradeFailed)
	if got := len(readLedger(t, p)); got != 0 {
		t.Errorf("ledger entries: got %d, want 0 for a failed trade", got)
	}
	if got := len(compartmentCommands[compartmentmsg.CancelReservationCommandBody](t, e, compartmentmsg.CommandCancelReservation)); got != 2 {
		t.Errorf("CANCEL_RESERVATION commands: got %d, want one per staged item (2)", got)
	}
	assertSettlementResolved(t, p, room.SettlementId())
}

// TestReconcileLeavesAnUnknownOutcomeAlone pins the rule that matters most: a
// trade that MAY have executed is never reported to the players as
// unsuccessful. An unreachable orchestrator and a saga it has not consumed yet
// are indistinguishable from here, and both leave the record for the next boot
// or for the live status event.
func TestReconcileLeavesAnUnknownOutcomeAlone(t *testing.T) {
	p, e := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)
	restart(t, p)

	p.sagad = &fakeSagaOutcome{err: errors.New("atlas-saga-orchestrator unreachable")}
	if err := p.ReconcileSettlements(); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	assertNoEventOfType(t, e, trademsg.StatusTypeSettled)
	assertNoEventOfType(t, e, trademsg.StatusTypeCancelled)
	if got := len(readLedger(t, p)); got != 0 {
		t.Errorf("ledger entries: got %d, want 0", got)
	}
	if _, err := settlement.NewProcessor(p.l, p.ctx, p.db).GetByTransactionId(room.SettlementId()); err != nil {
		t.Errorf("the settlement record was consumed on an unknown outcome: %v", err)
	}
}

// TestReconcileLeavesARunningSagaAlone pins the third answer: a saga with
// pending steps has not settled anything yet, so completing it would report an
// outcome that has not happened.
func TestReconcileLeavesARunningSagaAlone(t *testing.T) {
	p, e := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)
	restart(t, p)

	p.sagad = &fakeSagaOutcome{outcome: sagadata.OutcomeRunning}
	if err := p.ReconcileSettlements(); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	assertNoEventOfType(t, e, trademsg.StatusTypeSettled)
	assertNoEventOfType(t, e, trademsg.StatusTypeCancelled)
	if _, err := settlement.NewProcessor(p.l, p.ctx, p.db).GetByTransactionId(room.SettlementId()); err != nil {
		t.Errorf("the settlement record was consumed for a saga that is still running: %v", err)
	}
}

// TestTerminalStatusAfterARestartCompletesFromTheRecord pins the OTHER half of
// the same hole: the terminal event is redelivered after the restart (the
// consumer group resumes from its committed offset), and with no room it must
// still settle the trade rather than drop.
func TestTerminalStatusAfterARestartCompletesFromTheRecord(t *testing.T) {
	p, e := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)
	restart(t, p)

	if err := p.SettlementSucceeded(uuid.New(), room.SettlementId()); err != nil {
		t.Fatalf("settle: %v", err)
	}
	if got := len(readLedger(t, p)); got != 1 {
		t.Errorf("ledger entries: got %d, want 1", got)
	}
	if got := len(statusEvents[trademsg.SettledEventBody](t, e, trademsg.StatusTypeSettled)); got != 1 {
		t.Errorf("SETTLED events: got %d, want 1", got)
	}
	assertSettlementResolved(t, p, room.SettlementId())
}

// assertSettlementResolved requires the durable record to be gone, so
// unfinished settlements cannot accumulate.
func assertSettlementResolved(t *testing.T, p *ProcessorImpl, settlementId uuid.UUID) {
	t.Helper()
	if _, err := settlement.NewProcessor(p.l, p.ctx, p.db).GetByTransactionId(settlementId); err == nil {
		t.Error("the settlement record survived its terminal outcome; unresolved settlements would accumulate")
	}
}

// --- a settlement that never leaves the process ------------------------------------

// failingSagaSubmitter refuses to buffer the saga command, which is the second
// of the two ways the settle path can fail AFTER the room is already SETTLING.
type failingSagaSubmitter struct {
	err error
}

func (f *failingSagaSubmitter) Settle(_ *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.TradeSettlementPayload) error {
	return func(_ uuid.UUID, _ sharedsaga.TradeSettlementPayload) error {
		return f.err
	}
}

// assertSettlementAbandoned requires the room to have been closed rather than
// left in a state nothing can act on: SETTLING refuses teardown (FR-6.5), is
// skipped by the reservation refresh, has had its attestation deadline
// cancelled, and — with the command rolled back — has no durable record for the
// reconciler to find it by.
func assertSettlementAbandoned(t *testing.T, p *ProcessorImpl, e *emitted) {
	t.Helper()
	if room, ok := p.RoomForCharacter(100); ok {
		t.Errorf("the room survived a failed settlement submission in state %s; nothing can act on it", room.State())
	}
	assertCancelledWithReason(t, e, ReasonTradeFailed)
	if got := len(compartmentCommands[compartmentmsg.CancelReservationCommandBody](t, e, compartmentmsg.CommandCancelReservation)); got != 2 {
		t.Errorf("CANCEL_RESERVATION commands: got %d, want one per staged item (2)", got)
	}
	assertNoSagaSubmitted(t, e)
}

// TestSettleFailingToRecordTheSettlementDoesNotWedgeTheRoom pins the first
// post-CAS failure: the durable record cannot be written. The transaction rolls
// back — no record, no saga — but the registry swap to SETTLING is in-memory
// and does NOT roll back with it.
func TestSettleFailingToRecordTheSettlementDoesNotWedgeTheRoom(t *testing.T) {
	p, e := testConfirmedRoom(t)
	if err := p.db.Migrator().DropTable(&settlement.Entry{}); err != nil {
		t.Fatalf("drop settlements: %v", err)
	}

	if err := p.Attest(uuid.New(), 100, nil); err != nil {
		t.Fatalf("owner attest: %v", err)
	}
	if err := p.Attest(uuid.New(), 200, nil); err == nil {
		t.Fatal("visitor attest: expected the unwritable settlement record to surface an error")
	}

	assertSettlementAbandoned(t, p, e)
}

// TestSettleFailingToSubmitTheSagaDoesNotWedgeTheRoom pins the second post-CAS
// failure: the saga command cannot be buffered. Same shape, different trigger —
// and the same wedge if the transition is not undone.
func TestSettleFailingToSubmitTheSagaDoesNotWedgeTheRoom(t *testing.T) {
	p, e := testConfirmedRoom(t)
	p.sgp = &failingSagaSubmitter{err: errors.New("saga topic unavailable")}

	if err := p.Attest(uuid.New(), 100, nil); err != nil {
		t.Fatalf("owner attest: %v", err)
	}
	if err := p.Attest(uuid.New(), 200, nil); err == nil {
		t.Fatal("visitor attest: expected the refused saga submission to surface an error")
	}

	assertSettlementAbandoned(t, p, e)
	if _, err := settlement.NewProcessor(p.l, p.ctx, p.db).GetByTransactionId(uuid.Nil); err == nil {
		t.Error("a settlement record survived the rolled-back submission")
	}
}

// TestExpireAttestationFailingToSubmitDoesNotWedgeTheRoom pins the same
// recovery on the deadline path, which reaches settle from its own goroutine
// and would otherwise wedge a room nobody is waiting on a command for.
func TestExpireAttestationFailingToSubmitDoesNotWedgeTheRoom(t *testing.T) {
	p, e := testConfirmedRoom(t)
	p.sgp = &failingSagaSubmitter{err: errors.New("saga topic unavailable")}
	room, _ := p.RoomForCharacter(100)

	if err := p.ExpireAttestation(uuid.New(), room.Id()); err == nil {
		t.Fatal("expire: expected the refused saga submission to surface an error")
	}

	assertSettlementAbandoned(t, p, e)
}

// compile-time assurance the settlement fake satisfies the seam it stands in for.
var _ settlementSubmitter = (*failingSagaSubmitter)(nil)
