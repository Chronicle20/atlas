package trade

import (
	"atlas-trades/configuration"
	inventorydata "atlas-trades/data/inventory"
	sagadata "atlas-trades/data/saga"
	"atlas-trades/escrow"
	"atlas-trades/kafka/message"
	sagamsg "atlas-trades/kafka/message/saga"
	trademsg "atlas-trades/kafka/message/trade"
	"atlas-trades/ledger"
	"atlas-trades/settlement"
	"context"
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

// testStagedRoom is an OPEN room between 100 and 200 with one CONFIRMED staged
// item each — the escrow row written and the staging saga terminal, which is
// what a settlement needs. A merely pending stage would settle short.
func testStagedRoom(t *testing.T) (*ProcessorImpl, *emitted) {
	t.Helper()
	p, e := testOpenRoom(t)
	for _, id := range []character.Id{100, 200} {
		stageOne(t, p, id, stagingSourceSlot, stagedQuantity, 1)
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
	stageOne(t, p, 100, stagingSourceSlot, stagedQuantity, 1)

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
//
// The stake is driven all the way through its award_mesos round trip, because
// under escrow-at-staging AddMeso only ARMS it: the room's mesoStaged does not
// advance until the debit reports terminal, and the escrow row is what a
// teardown or a failed settlement would refund from.
func testConfirmedRoomWithMeso(t *testing.T, characterId character.Id, amount uint32) (*ProcessorImpl, *emitted) {
	t.Helper()
	p, e := testOpenRoomWithMeso(t, characterId, amount*2)
	if err := p.AddMeso(uuid.New(), characterId, int32(amount)); err != nil {
		t.Fatalf("add meso: %v", err)
	}
	stakes := sagasWithAction(t, e, sharedsaga.AwardMesos)
	if len(stakes) != 1 {
		t.Fatalf("award_mesos sagas: got %d, want 1", len(stakes))
	}
	if _, err := p.MesoStageSucceeded(uuid.New(), stakes[0].TransactionId); err != nil {
		t.Fatalf("meso stake succeeded: %v", err)
	}
	room, _ := p.RoomForCharacter(characterId)
	escrowOf(t, p).setMeso(room.Id(), characterId, amount)
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

// sagasWithAction narrows collectSagas to the one-step sagas whose step carries
// the given action. Staging is a saga too now, so "how many sagas were
// submitted" is no longer a question with a single meaningful answer: a
// settlement assertion has to say WHICH saga it means, or a stage counts toward
// it and every count is off by the number of items the harness staged.
func sagasWithAction(t *testing.T, e *emitted, action sharedsaga.Action) []sharedsaga.Saga {
	t.Helper()
	var out []sharedsaga.Saga
	for _, s := range collectSagas(t, e) {
		if len(s.Steps) == 1 && s.Steps[0].Action == action {
			out = append(out, s)
		}
	}
	return out
}

func settlementSagas(t *testing.T, e *emitted) []sharedsaga.Saga {
	t.Helper()
	return sagasWithAction(t, e, sharedsaga.TradeSettlement)
}

func stageSagas(t *testing.T, e *emitted) []sharedsaga.Saga {
	t.Helper()
	return sagasWithAction(t, e, sharedsaga.TransferToTrade)
}

func unwindSagas(t *testing.T, e *emitted) []sharedsaga.Saga {
	t.Helper()
	return sagasWithAction(t, e, sharedsaga.TradeUnwind)
}

func assertNoSagaOfAction(t *testing.T, e *emitted, action sharedsaga.Action) {
	t.Helper()
	if sagas := sagasWithAction(t, e, action); len(sagas) != 0 {
		t.Errorf("%s sagas: got %d, want 0", action, len(sagas))
	}
}

// assertNoSettlementSubmitted requires that no trade_settlement composite left
// the service. Staging and unwind sagas are deliberately not counted.
func assertNoSettlementSubmitted(t *testing.T, e *emitted) {
	t.Helper()
	assertNoSagaOfAction(t, e, sharedsaga.TradeSettlement)
}

func assertNoStageSubmitted(t *testing.T, e *emitted) {
	t.Helper()
	assertNoSagaOfAction(t, e, sharedsaga.TransferToTrade)
}

func assertNoUnwindSubmitted(t *testing.T, e *emitted) {
	t.Helper()
	assertNoSagaOfAction(t, e, sharedsaga.TradeUnwind)
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

// stagePayloadOf requires the saga to be the one-step transfer_to_trade
// composite and returns its CONCRETE payload.
func stagePayloadOf(t *testing.T, s sharedsaga.Saga) sharedsaga.TransferToTradePayload {
	t.Helper()
	if len(s.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(s.Steps))
	}
	payload, ok := s.Steps[0].Payload.(sharedsaga.TransferToTradePayload)
	if !ok {
		t.Fatalf("payload type: got %T, want TransferToTradePayload", s.Steps[0].Payload)
	}
	return payload
}

// unwindPayloadOf requires the saga to be the one-step trade_unwind composite
// and returns its CONCRETE payload.
func unwindPayloadOf(t *testing.T, s sharedsaga.Saga) sharedsaga.TradeUnwindPayload {
	t.Helper()
	if len(s.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(s.Steps))
	}
	payload, ok := s.Steps[0].Payload.(sharedsaga.TradeUnwindPayload)
	if !ok {
		t.Fatalf("payload type: got %T, want TradeUnwindPayload", s.Steps[0].Payload)
	}
	return payload
}

// awardMesosPayloadOf requires the saga to be the bare award_mesos a meso stake
// submits and returns its CONCRETE payload.
func awardMesosPayloadOf(t *testing.T, s sharedsaga.Saga) sharedsaga.AwardMesosPayload {
	t.Helper()
	if len(s.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(s.Steps))
	}
	payload, ok := s.Steps[0].Payload.(sharedsaga.AwardMesosPayload)
	if !ok {
		t.Fatalf("payload type: got %T, want AwardMesosPayload", s.Steps[0].Payload)
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
	assertNoSettlementSubmitted(t, e)
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
	assertNoSettlementSubmitted(t, e)
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

	sagas := settlementSagas(t, e)
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
		if side.Items[0].AssetId != stagingAssetId || side.Items[0].Snapshot.Quantity != uint32(stagedQuantity) {
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
	assertNoSettlementSubmitted(t, e)
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

	if got := len(settlementSagas(t, e)); got != 1 {
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
		if len(settlementSagas(t, e)) == 1 {
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
	if got := len(settlementSagas(t, e)); got != 1 {
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

	if got := len(settlementSagas(t, e)); got != 1 {
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
	assertNoSettlementSubmitted(t, e)
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
	assertNoSettlementSubmitted(t, e)
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
	assertNoSettlementSubmitted(t, e)
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
	if got := len(settlementSagas(t, e)); got != 1 {
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
	if got := len(settlementSagas(t, e)); got != 1 {
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
	assertNoSettlementSubmitted(t, e)
}

// TestAFullInventoryStillMergesIntoAMatchingStack guards the free-slot check
// against over-reach: atlas-inventory's Accept merges into an existing stack
// with room, so a full compartment that already holds the incoming template is
// not a refusal.
func TestAFullInventoryStillMergesIntoAMatchingStack(t *testing.T) {
	p, e := testOpenRoom(t)
	stageOne(t, p, 100, stagingSourceSlot, stagedQuantity, 1)
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
	if got := len(settlementSagas(t, e)); got != 1 {
		t.Errorf("sagas submitted: got %d, want 1", got)
	}
}

// TestSettlementRefusesWhenAStagedItemHasNoEscrowRow is the escrow-era
// successor to design §6.1's check 3.
//
// Under reserve-at-staging the staged asset was still in its owner's
// compartment, so the check re-read the compartment to confirm the asset had
// not been spent out from under a hold that could lapse. Custody removes that
// failure class — nobody can touch an escrowed asset and custody does not
// expire — but it introduces a sharper one: a staged item whose row is NOT
// there means the row was removed under a settlement that is already
// committing. Skipping it would settle the trade minus one item, so the giver
// loses it and the receiver never gets it. It is an error, not a value to skip.
func TestSettlementRefusesWhenAStagedItemHasNoEscrowRow(t *testing.T) {
	p, e := testConfirmedRoom(t)
	// The owner's custody row disappears between the confirm and the settle.
	escrowOf(t, p).release(stagedItemsOf(t, p, 100)[0].EscrowId())

	if err := p.Attest(uuid.New(), 100, nil); err != nil {
		t.Fatalf("owner attest: %v", err)
	}
	if err := p.Attest(uuid.New(), 200, nil); err == nil {
		t.Fatal("visitor attest: expected the missing escrow row to surface an error rather than settle short")
	}

	assertNoSettlementSubmitted(t, e)
	// The room is closed rather than left wedged in SETTLING, by the same
	// recovery that covers an unwritable settlement record.
	if _, ok := p.RoomForCharacter(100); ok {
		t.Error("the room survived a settlement it could not build a payload for")
	}
	assertCancelledWithReason(t, e, ReasonTradeFailed)
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
	assertNoSettlementSubmitted(t, e)
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

	sagas := settlementSagas(t, e)
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

// TestSettlementPayloadCarriesTheEscrowRowsSnapshot pins where the settlement
// payload's items come from (design §5A.7). The ROOM knows which escrow rows a
// side staged and nothing more; only the custody row carries the stat snapshot,
// and the orchestrator can no longer look the asset up because the asset is not
// in anybody's compartment any more.
//
// The escrow id is the load-bearing field: every expanded step names it, so a
// payload that carried the room's view instead would release nothing.
func TestSettlementPayloadCarriesTheEscrowRowsSnapshot(t *testing.T) {
	p, e := testConfirmedRoom(t)
	want := map[character.Id]uuid.UUID{
		100: stagedItemsOf(t, p, 100)[0].EscrowId(),
		200: stagedItemsOf(t, p, 200)[0].EscrowId(),
	}

	for _, id := range []character.Id{100, 200} {
		if err := p.Attest(uuid.New(), id, nil); err != nil {
			t.Fatalf("attest for %d: %v", id, err)
		}
	}

	sagas := settlementSagas(t, e)
	if len(sagas) != 1 {
		t.Fatalf("sagas submitted: got %d, want 1", len(sagas))
	}
	payload := settlementPayloadOf(t, sagas[0])
	for _, side := range payload.Sides {
		if len(side.Items) != 1 {
			t.Fatalf("side %d items: got %d, want 1", side.CharacterId, len(side.Items))
		}
		if got := side.Items[0].EscrowId; got != want[side.CharacterId] {
			t.Errorf("side %d escrowId: got %s, want the staged item's custody row %s", side.CharacterId, got, want[side.CharacterId])
		}
		snap := side.Items[0].Snapshot
		if snap.WeaponAttack != escrowStatWeaponAttack || snap.Slots != escrowStatSlots {
			t.Errorf("side %d stat snapshot: got weaponAttack %d slots %d, want the escrow row's %d / %d", side.CharacterId, snap.WeaponAttack, snap.Slots, escrowStatWeaponAttack, escrowStatSlots)
		}
		if snap.Owner != escrowOwnerName {
			t.Errorf("side %d owner: got %q, want the escrow row's %q", side.CharacterId, snap.Owner, escrowOwnerName)
		}
		// Cash serial, expiry and pet id travel on the same snapshot. A
		// settlement that lost them would hand the receiver a degraded item —
		// the defect the bespoke stat list this replaced actually had.
		if snap.CashId != escrowCashId || !snap.Expiration.Equal(escrowExpiration) || snap.PetId != escrowPetId {
			t.Errorf("side %d cash/expiry/pet state: got %+v, want cashId %d expiration %s petId %d", side.CharacterId, snap, escrowCashId, escrowExpiration, escrowPetId)
		}
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

// TestSettlementSuccessUnwindsNothing pins the asymmetry in completeSettlement
// that replaces the old "cancel both holds" obligation.
//
// A SUCCESSFUL settlement's expanded release_from_trade steps have already
// emptied the escrow, so an unwind here would be a second delivery of assets
// that are no longer in custody — the orchestrator would fail every leg, or
// worse, succeed against rows a compensation had restored. Only failure unwinds
// (see TestSettlementFailureUnwindsTheEscrowBackToBothOwners).
func TestSettlementSuccessUnwindsNothing(t *testing.T) {
	p, e := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)
	// The saga's releases emptied the custody store on its way to success.
	for _, id := range []character.Id{100, 200} {
		escrowOf(t, p).release(stagedItemsOf(t, p, id)[0].EscrowId())
	}

	if err := p.SettlementSucceeded(uuid.New(), room.SettlementId()); err != nil {
		t.Fatalf("settle: %v", err)
	}
	assertNoUnwindSubmitted(t, e)
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
}

// TestSettlementFailureUnwindsTheEscrowBackToBothOwners pins the other half of
// completeSettlement's asymmetry. A failed settlement has been compensated by
// the orchestrator, which restored every custody row it had released — so both
// players' items are sitting in escrow with no trade left to deliver them, and
// somebody has to hand them back.
//
// It reads the escrow FRESH rather than replaying the settlement record's own
// item list: the record is a snapshot taken at submission, while the escrow is
// what the compensation actually managed to restore. Refunding the snapshot
// would create items whose restore failed out of nothing.
func TestSettlementFailureUnwindsTheEscrowBackToBothOwners(t *testing.T) {
	p, e := testSettlingRoom(t)
	room, _ := p.RoomForCharacter(100)
	want := map[uuid.UUID]character.Id{
		stagedItemsOf(t, p, 100)[0].EscrowId(): 100,
		stagedItemsOf(t, p, 200)[0].EscrowId(): 200,
	}

	if err := p.SettlementFailed(uuid.New(), room.SettlementId(), "ACCEPT_FAILED"); err != nil {
		t.Fatalf("fail: %v", err)
	}

	unwinds := unwindSagas(t, e)
	if len(unwinds) != 1 {
		t.Fatalf("trade_unwind sagas: got %d, want 1", len(unwinds))
	}
	payload := unwindPayloadOf(t, unwinds[0])
	if len(payload.Items) != 2 {
		t.Fatalf("unwind items: got %d, want one per escrowed item (2)", len(payload.Items))
	}
	for _, it := range payload.Items {
		owner, ok := want[it.Item.EscrowId]
		if !ok {
			t.Errorf("unwind returned escrow row %s, which was never staged", it.Item.EscrowId)
			continue
		}
		if it.OwnerId != owner {
			t.Errorf("unwind ownerId for %s: got %d, want %d", it.Item.EscrowId, it.OwnerId, owner)
		}
		delete(want, it.Item.EscrowId)
	}
	if len(want) != 0 {
		t.Errorf("escrow rows left in custody after a failed settlement: %v", want)
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
		if side.Items()[0].EscrowId() == uuid.Nil {
			t.Error("recorded item carries no escrow id; a reconciled settlement could never name the custody row it moves")
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

// --- cancel vs settle, raced across the pre-check's REST reads -----------------

// beginSettlingOnce drives the room to SETTLING the FIRST time a compartment is
// read, which is the real window: the settlement pre-checks read both sides'
// compartments over REST between reading the room and ending it, and the
// attestation deadline runs settle from an independent goroutine that no Kafka
// partition ordering serialises against the teardown consumer.
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

// TestSettlementRefusalLosesToASettlementThatWinsDuringItsReads pins
// teardownRoom's compare-and-set through the one window escrow-at-staging still
// leaves open. A pre-check refusal runs both sides' compartment reads before it
// ends the room, and another trigger can reach SETTLING inside that window; an
// unconditional removal there would delete the room the in-flight saga's
// terminal status must find AND unwind the custody it is about to deliver.
//
// (The plain-teardown twin of this test is gone with the reads it depended on:
// teardown no longer resolves anything over REST before claiming the room, so
// there is no longer a window to land a settlement in. FR-6.5 on that path is
// covered by TestCancelLosesToSettlement and
// TestTeardownCharacterLosesToSettlement.)
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
	assertNoUnwindSubmitted(t, e)
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
	assertNoUnwindSubmitted(t, e)
	if got := len(readLedger(t, p)); got != 0 {
		t.Errorf("ledger entries: got %d, want 0 — the losing delivery recorded a trade the winner owns", got)
	}
}

// --- the attestation deadline vs a failed emit -----------------------------------

// TestConfirmArmsTheDeadlineEvenWhenTheEmitFails pins the wedge a failed emit
// would otherwise leave. The registry swap to AWAITING_ATTESTATION is in-memory
// and is NOT rolled back by the enclosing transaction, so a room whose confirm
// failed to publish still sits in that state — with no mode 17 ever reaching
// the clients, no attestation possible, and both sides' items sitting in
// custody with nothing left to move them. Only the deadline can end it.
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
	assertNoSettlementSubmitted(t, e)
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

	// A SUCCESSFUL settlement's releases already emptied the custody store, so
	// the reconciled completion must not unwind on top of them.
	assertNoUnwindSubmitted(t, e)
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
	// The escrow must still go home: the room that would have unwound it is
	// gone, so the durable record is the only thing that knows the room id.
	unwinds := unwindSagas(t, e)
	if len(unwinds) != 1 {
		t.Fatalf("trade_unwind sagas: got %d, want 1", len(unwinds))
	}
	if got := len(unwindPayloadOf(t, unwinds[0]).Items); got != 2 {
		t.Errorf("unwind items: got %d, want one per staged item (2)", got)
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

// failingSagaSubmitter refuses to buffer the SETTLEMENT command, which is the
// second of the two ways the settle path can fail AFTER the room is already
// SETTLING.
//
// It refuses only Settle, and that is deliberate rather than lazy: the recovery
// this fake exists to exercise closes the trade by submitting a trade_unwind,
// so a fake that refused every action would break the recovery too and the test
// would pin nothing but a second-order failure. Embedding the real submitter
// leaves the other three actions working.
type failingSagaSubmitter struct {
	sagaSubmitter
	err error
}

func (f *failingSagaSubmitter) Settle(_ *message.Buffer) func(transactionId uuid.UUID, payload sharedsaga.TradeSettlementPayload) error {
	return func(_ uuid.UUID, _ sharedsaga.TradeSettlementPayload) error {
		return f.err
	}
}

// assertSettlementAbandoned requires the room to have been closed rather than
// left in a state nothing can act on: SETTLING refuses teardown (FR-6.5), has
// had its attestation deadline cancelled, and — with the command rolled back —
// has no durable record for the reconciler to find it by.
//
// Closing it also has to RETURN the custody. Nothing was settled, so both
// players' items are still escrowed, and a room that vanishes without unwinding
// keeps them.
func assertSettlementAbandoned(t *testing.T, p *ProcessorImpl, e *emitted) {
	t.Helper()
	if room, ok := p.RoomForCharacter(100); ok {
		t.Errorf("the room survived a failed settlement submission in state %s; nothing can act on it", room.State())
	}
	assertCancelledWithReason(t, e, ReasonTradeFailed)
	unwinds := unwindSagas(t, e)
	if len(unwinds) != 1 {
		t.Fatalf("trade_unwind sagas: got %d, want 1", len(unwinds))
	}
	if got := len(unwindPayloadOf(t, unwinds[0]).Items); got != 2 {
		t.Errorf("unwind items: got %d, want one per staged item (2)", got)
	}
	assertNoSettlementSubmitted(t, e)
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
	p.sgp = &failingSagaSubmitter{sagaSubmitter: p.sgp, err: errors.New("saga topic unavailable")}

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
	p.sgp = &failingSagaSubmitter{sagaSubmitter: p.sgp, err: errors.New("saga topic unavailable")}
	room, _ := p.RoomForCharacter(100)

	if err := p.ExpireAttestation(uuid.New(), room.Id()); err == nil {
		t.Fatal("expire: expected the refused saga submission to surface an error")
	}

	assertSettlementAbandoned(t, p, e)
}

// compile-time assurance the settlement fake satisfies the seam it stands in for.
var _ sagaSubmitter = (*failingSagaSubmitter)(nil)

// --- the escrow meso a settled trade leaves behind ------------------------------
//
// The custody rows read below are the REAL ones. The escrowStore is faked in this
// harness, but the meso half of staging is not injected at all — addMeso and
// resolveMesoStake build an escrow.Processor over the command's own transaction —
// so a room that staged meso has genuine rows in this database, which is exactly
// what a boot sweep would find.

// escrowMesoRowsOf reads the room's committed escrow meso rows straight out of
// the database, bypassing the faked read seam.
func escrowMesoRowsOf(t *testing.T, p *ProcessorImpl, roomId uuid.UUID) []escrow.MesoModel {
	t.Helper()
	rows, err := escrow.MesosByRoom(p.db, p.t.Id())(roomId)
	if err != nil {
		t.Fatalf("read escrow meso rows: %v", err)
	}
	return rows
}

// testSettlingRoomWithMeso is testConfirmedRoomWithMeso driven through both
// attestations, so its settlement saga is in flight and only the terminal status
// is missing.
func testSettlingRoomWithMeso(t *testing.T, characterId character.Id, amount uint32) (*ProcessorImpl, *emitted) {
	t.Helper()
	p, e := testConfirmedRoomWithMeso(t, characterId, amount)
	for _, id := range []character.Id{100, 200} {
		if err := p.Attest(uuid.New(), id, nil); err != nil {
			t.Fatalf("attest for %d: %v", id, err)
		}
	}
	return p, e
}

// bootSweep runs ReconcileEscrow exactly as ReconcileAtBoot would: the exclusion
// set is built from the settlement records that are still unresolved, so a room
// whose settlement has completed is deliberately NOT shielded from the sweep.
func bootSweep(t *testing.T, p *ProcessorImpl) {
	t.Helper()
	ctx := context.Background()
	inFlight, err := settlement.Unresolved(ctx, p.db)
	if err != nil {
		t.Fatalf("read unresolved settlements: %v", err)
	}
	owned := make(map[uuid.UUID]struct{}, len(inFlight))
	for _, s := range inFlight {
		owned[s.RoomId()] = struct{}{}
	}
	if err := ReconcileEscrow(reconcileLogger(), ctx, p.db, owned); err != nil {
		t.Fatalf("boot escrow sweep: %v", err)
	}
}

// TestSettlementSuccessDischargesTheEscrowedMeso pins the meso counterpart of the
// release_from_trade steps the expanded settlement saga emits for every item.
//
// The saga's meso leg is a bare award_mesos CREDIT to the receiver; nothing in it
// touches the giver's escrow row. A row that survives its own settlement is meso
// the boot sweep will hand back to a player who has already given it away.
func TestSettlementSuccessDischargesTheEscrowedMeso(t *testing.T) {
	const staked = uint32(5000)
	p, _ := testSettlingRoomWithMeso(t, 100, staked)
	room, _ := p.RoomForCharacter(100)
	roomId := room.Id()

	if err := p.SettlementSucceeded(uuid.New(), room.SettlementId()); err != nil {
		t.Fatalf("settle: %v", err)
	}

	for _, r := range escrowMesoRowsOf(t, p, roomId) {
		t.Errorf("a settled trade left [%d] meso in escrow for character [%d]; the next boot sweep refunds it to a giver who has already paid, minting %d meso", r.Amount(), r.OwnerId(), r.Amount())
	}
}

// TestBootSweepAfterASuccessfulSettlementRefundsNothing states the same defect as
// the behaviour it actually caused. The settlement record is gone by then — it is
// what the sweep's exclusion set is built from — so nothing but an empty escrow
// stands between a settled room and a second payout.
func TestBootSweepAfterASuccessfulSettlementRefundsNothing(t *testing.T) {
	const staked = uint32(5000)
	// The sweep resolves each owner's CURRENT field over REST before it can
	// refund them. Without this an unreachable owner would be SKIPPED, and the
	// test would pass on a service outage rather than on the discharge.
	serveLocations(t, map[character.Id][2]byte{100: {1, 1}, 200: {1, 1}})

	p, e := testSettlingRoomWithMeso(t, 100, staked)
	room, _ := p.RoomForCharacter(100)

	if err := p.SettlementSucceeded(uuid.New(), room.SettlementId()); err != nil {
		t.Fatalf("settle: %v", err)
	}
	bootSweep(t, p)

	for _, u := range unwindSagas(t, e) {
		for _, m := range unwindPayloadOf(t, u).Mesos {
			t.Errorf("the boot sweep refunded [%d] meso to character [%d] after a SUCCESSFUL settlement", m.Amount, m.CharacterId)
		}
	}
}

// TestSettledMesoIsConservedAcrossTheBootSweep is the whole invariant in one
// place: over stage → settle → boot sweep the giver parts with exactly what they
// staked and gets nothing back, and the receiver is credited exactly what the
// settlement said it would deliver. The difference is the tax, which is burned.
func TestSettledMesoIsConservedAcrossTheBootSweep(t *testing.T) {
	const staked = uint32(5000)
	serveLocations(t, map[character.Id][2]byte{100: {1, 1}, 200: {1, 1}})

	p, e := testSettlingRoomWithMeso(t, 100, staked)
	room, _ := p.RoomForCharacter(100)

	sagas := settlementSagas(t, e)
	if len(sagas) != 1 {
		t.Fatalf("trade_settlement sagas: got %d, want 1", len(sagas))
	}
	payload := settlementPayloadOf(t, sagas[0])
	if payload.Sides[0].CharacterId != 100 {
		t.Fatalf("settlement side 0: got character %d, want the giver 100", payload.Sides[0].CharacterId)
	}
	delivered := payload.Sides[0].MesoDelivered
	if delivered == 0 || delivered > staked {
		t.Fatalf("delivered meso: got %d, want a non-zero amount no larger than the staked %d", delivered, staked)
	}

	if err := p.SettlementSucceeded(uuid.New(), room.SettlementId()); err != nil {
		t.Fatalf("settle: %v", err)
	}
	bootSweep(t, p)

	// The stake's own award_mesos legs are what actually moved the giver's meso.
	var debited int32
	for _, s := range sagasWithAction(t, e, sharedsaga.AwardMesos) {
		a := awardMesosPayloadOf(t, s)
		if a.CharacterId != uint32(100) {
			t.Errorf("award_mesos for character %d; only the giver stakes here", a.CharacterId)
			continue
		}
		debited += a.Amount
	}
	if debited != -int32(staked) {
		t.Errorf("net meso moved by the giver's stakes: got %d, want %d", debited, -int32(staked))
	}

	var refunded uint32
	for _, u := range unwindSagas(t, e) {
		for _, m := range unwindPayloadOf(t, u).Mesos {
			refunded += m.Amount
		}
	}
	if refunded != 0 {
		t.Errorf("meso refunded to the giver after a successful trade: got %d, want 0 — the trade delivered %d and the giver was debited %d", refunded, delivered, staked)
	}
}

// TestSettlementSuccessKeepsARowWhoseStakeIsStillInFlight is the guard on the
// discharge, and it is the reason the row is ZEROED before it is conditionally
// deleted rather than deleted outright.
//
// The stake armed here is one whose award_mesos was still in flight when the
// settlement completed — armed before the confirm froze staging, terminal status
// not yet delivered. Its debit has landed on the player and the trade did not
// carry it, so the row is the only record from which it can be handed back.
func TestSettlementSuccessKeepsARowWhoseStakeIsStillInFlight(t *testing.T) {
	const staked = uint32(5000)
	const inFlightTotal = uint32(9000)
	const inFlightDelta = int32(4000)

	p, e := testSettlingRoomWithMeso(t, 100, staked)
	room, _ := p.RoomForCharacter(100)
	roomId := room.Id()
	stakeId := uuid.New()
	if err := escrow.ArmMesoStake(p.db, p.t)(roomId, 100, stakeId, inFlightTotal, inFlightDelta); err != nil {
		t.Fatalf("arm the in-flight stake: %v", err)
	}

	if err := p.SettlementSucceeded(uuid.New(), room.SettlementId()); err != nil {
		t.Fatalf("settle: %v", err)
	}

	rows := escrowMesoRowsOf(t, p, roomId)
	if len(rows) != 1 {
		t.Fatalf("escrow meso rows after settlement: got %d, want the one carrying the in-flight stake", len(rows))
	}
	if rows[0].Amount() != 0 {
		t.Errorf("settled row amount: got %d, want 0 — the settlement delivered it", rows[0].Amount())
	}
	if rows[0].PendingStakeId() != stakeId {
		t.Errorf("pending stake id: got %s, want the armed %s", rows[0].PendingStakeId(), stakeId)
	}

	// And the stake still resolves. Its room is gone, so the debit it moved has
	// nowhere to go but back to the player it was taken from.
	if _, err := p.MesoStageSucceeded(uuid.New(), stakeId); err != nil {
		t.Fatalf("resolve the in-flight stake: %v", err)
	}
	unwinds := unwindSagas(t, e)
	if len(unwinds) != 1 {
		t.Fatalf("trade_unwind sagas: got %d, want 1 refunding the orphaned stake", len(unwinds))
	}
	mesos := unwindPayloadOf(t, unwinds[0]).Mesos
	if len(mesos) != 1 || mesos[0].CharacterId != 100 || mesos[0].Amount != uint32(inFlightDelta) {
		t.Fatalf("orphaned stake refund: got %+v, want %d back to character 100", mesos, inFlightDelta)
	}
	if rows := escrowMesoRowsOf(t, p, roomId); len(rows) != 0 {
		t.Errorf("escrow meso rows after the stake resolved: got %d, want 0", len(rows))
	}
}

// TestSettlementFailureStillRefundsTheEscrowedMeso pins the other side of the
// asymmetry the discharge introduces. Only SUCCESS discharges; a failed
// settlement has been compensated, the meso is still escrowed with no trade left
// to deliver it, and it must go home.
func TestSettlementFailureStillRefundsTheEscrowedMeso(t *testing.T) {
	const staked = uint32(5000)
	p, e := testSettlingRoomWithMeso(t, 100, staked)
	room, _ := p.RoomForCharacter(100)
	roomId := room.Id()

	if err := p.SettlementFailed(uuid.New(), room.SettlementId(), "ACCEPT_FAILED"); err != nil {
		t.Fatalf("fail: %v", err)
	}

	unwinds := unwindSagas(t, e)
	if len(unwinds) != 1 {
		t.Fatalf("trade_unwind sagas: got %d, want 1", len(unwinds))
	}
	mesos := unwindPayloadOf(t, unwinds[0]).Mesos
	if len(mesos) != 1 || mesos[0].CharacterId != 100 || mesos[0].Amount != staked {
		t.Fatalf("refund after a failed settlement: got %+v, want the staked %d back to character 100", mesos, staked)
	}
	if rows := escrowMesoRowsOf(t, p, roomId); len(rows) != 0 {
		t.Errorf("escrow meso rows after a refunded failure: got %d, want 0", len(rows))
	}
}
