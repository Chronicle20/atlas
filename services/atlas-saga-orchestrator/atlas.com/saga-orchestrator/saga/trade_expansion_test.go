package saga

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
)

// trade_expansion_test.go — the escrow-at-staging expansion of trade_settlement
// and trade_unwind (design §5A.7, §5A.8).
//
// The expanders perform NO inventory reads: an escrowed asset has already left
// its owner's compartment, so the snapshot travels on the payload. That retires
// an entire family of tests this file used to carry — staged-slot-moved,
// same-template instance substitution, unparseable asset id — because every one
// of them guarded against the staged asset changing underneath the settlement,
// which escrow makes impossible. They were deleted rather than ported; there is
// nothing left for them to assert.

// escrowItem builds one escrowed-item payload entry whose snapshot is non-zero
// in every field group — equip stats, cash, expiry and pet — so a re-grant that
// dropped any of them shows up. The cash/expiry/pet groups are the ones the
// bespoke stat list this replaced never carried at all.
func escrowItem(escrowId uuid.UUID, assetId uint32, templateId uint32, quantity uint32, inventoryType int8) TradeEscrowItem {
	return TradeEscrowItem{
		EscrowId:      escrowId,
		InventoryType: inventory.Type(inventoryType),
		AssetId:       asset.Id(assetId),
		Snapshot: AssetSnapshot{
			Slot:           3,
			TemplateId:     templateId,
			Quantity:       quantity,
			Expiration:     escrowItemExpiration,
			CashId:         4815162342,
			Rechargeable:   200,
			WeaponAttack:   17,
			Slots:          7,
			Flag:           2,
			Owner:          "Chronicle",
			LevelType:      1,
			Level:          4,
			Experience:     1234,
			HammersApplied: 2,
			PetId:          909,
		},
	}
}

var escrowItemExpiration = time.Date(2031, 4, 5, 6, 7, 8, 0, time.UTC)

// tradeSettlementFixture is the canonical two-sided settlement: each side stages
// one item, side 0 also stages meso.
func tradeSettlementFixture() TradeSettlementPayload {
	return TradeSettlementPayload{
		TransactionId: uuid.New(),
		WorldId:       1,
		ChannelId:     1,
		RoomType:      3,
		Sides: [2]TradeSettlementSide{
			{
				CharacterId:   100,
				Items:         []TradeSettlementItem{escrowItem(uuid.New(), 55, 2000000, 5, 2)},
				MesoStaged:    10_000_000,
				MesoTax:       400_000,
				MesoDelivered: 9_600_000,
			},
			{
				CharacterId: 200,
				Items:       []TradeSettlementItem{escrowItem(uuid.New(), 77, 1302000, 1, 1)},
			},
		},
	}
}

func expandSettlement(t *testing.T, payload TradeSettlementPayload) []Step[any] {
	t.Helper()
	p := newTestExpansionProcessor(t)
	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, payload))
	require.NoError(t, err)
	return steps
}

func actionsOf(steps []Step[any]) []Action {
	out := make([]Action, 0, len(steps))
	for _, s := range steps {
		out = append(out, s.Action())
	}
	return out
}

// TestExpandTradeSettlementOrdersReleasesBeforeAccepts pins design §6.3's
// surviving ordering rule: every release precedes every accept, so a slot freed
// by an outgoing item is available to an incoming one and a failure in either
// release compensates before anything has been created.
func TestExpandTradeSettlementOrdersReleasesBeforeAccepts(t *testing.T) {
	steps := expandSettlement(t, tradeSettlementFixture())

	lastRelease, firstAccept := -1, -1
	for i, a := range actionsOf(steps) {
		if a == ReleaseFromTrade {
			lastRelease = i
		}
		if a == AcceptToCharacter && firstAccept == -1 {
			firstAccept = i
		}
	}
	require.NotEqual(t, -1, lastRelease, "no release emitted")
	require.NotEqual(t, -1, firstAccept, "no accept emitted")
	require.Less(t, lastRelease, firstAccept, "an accept was emitted before the last release")
}

// TestExpandTradeSettlementReleasesFromEscrowNotFromCharacters is the
// load-bearing shape assertion. The item is no longer in anyone's compartment,
// so a release_from_character here would target an asset that does not exist and
// fail the settlement of every trade.
func TestExpandTradeSettlementReleasesFromEscrowNotFromCharacters(t *testing.T) {
	payload := tradeSettlementFixture()
	steps := expandSettlement(t, payload)

	escrowIds := map[uuid.UUID]bool{
		payload.Sides[0].Items[0].EscrowId: false,
		payload.Sides[1].Items[0].EscrowId: false,
	}
	for _, s := range steps {
		require.NotEqual(t, ReleaseFromCharacter, s.Action(),
			"settlement must not release from a compartment — the asset is in escrow")
		if s.Action() != ReleaseFromTrade {
			continue
		}
		pl, ok := s.Payload().(ReleaseFromTradePayload)
		require.True(t, ok)
		_, known := escrowIds[pl.EscrowId]
		require.True(t, known, "release named an escrow row that was never staged")
		escrowIds[pl.EscrowId] = true
	}
	for id, released := range escrowIds {
		require.Truef(t, released, "escrow row %s was never released", id)
	}
}

// TestExpandTradeSettlementCrossesTheAssets pins that each side's items go to
// the OTHER side. A settlement that returned each item to its owner would be a
// no-op trade that still consumed the meso.
func TestExpandTradeSettlementCrossesTheAssets(t *testing.T) {
	payload := tradeSettlementFixture()
	steps := expandSettlement(t, payload)

	got := map[uint32]uint32{} // templateId -> recipient
	for _, s := range steps {
		if s.Action() != AcceptToCharacter {
			continue
		}
		pl, ok := s.Payload().(AcceptToCharacterPayload)
		require.True(t, ok)
		got[pl.TemplateId] = pl.CharacterId
	}
	require.Equal(t, uint32(200), got[2000000], "side 0's item must go to side 1")
	require.Equal(t, uint32(100), got[1302000], "side 1's item must go to side 0")
}

// TestExpandTradeSettlementEmitsNoNegativeAward is the single most important
// assertion in this file.
//
// Under escrow-at-staging the meso was debited when it was STAGED (design
// §5A.5). An expander that kept the old negative leg would debit it a second
// time at settlement and take twice the meso the player agreed to trade.
func TestExpandTradeSettlementEmitsNoNegativeAward(t *testing.T) {
	steps := expandSettlement(t, tradeSettlementFixture())
	for _, s := range steps {
		if s.Action() != AwardMesos {
			continue
		}
		pl, ok := s.Payload().(AwardMesosPayload)
		require.True(t, ok)
		require.Positivef(t, pl.Amount, "settlement emitted a negative meso award (%d) — the debit already happened at stage time", pl.Amount)
	}
}

// TestExpandTradeSettlementCreditsTheTaxedAmountToTheOtherSide pins §6.5: the
// receiver gets MesoDelivered, and the destroyed tax is simply never credited to
// anyone.
func TestExpandTradeSettlementCreditsTheTaxedAmountToTheOtherSide(t *testing.T) {
	steps := expandSettlement(t, tradeSettlementFixture())

	var credits []AwardMesosPayload
	for _, s := range steps {
		if s.Action() != AwardMesos {
			continue
		}
		pl, ok := s.Payload().(AwardMesosPayload)
		require.True(t, ok)
		credits = append(credits, pl)
	}
	require.Len(t, credits, 1, "only the side that staged meso produces a credit")
	require.Equal(t, uint32(200), credits[0].CharacterId, "the credit goes to the OTHER side")
	require.Equal(t, int32(9_600_000), credits[0].Amount, "the credit is the post-tax delivered amount")
}

// TestExpandTradeSettlementAcceptsTheStagedQuantity pins the partial-stack case:
// the snapshot's quantity is what was ESCROWED, so staging 5 of a 200 stack
// delivers 5.
func TestExpandTradeSettlementAcceptsTheStagedQuantity(t *testing.T) {
	steps := expandSettlement(t, tradeSettlementFixture())
	for _, s := range steps {
		if s.Action() != AcceptToCharacter {
			continue
		}
		pl, ok := s.Payload().(AcceptToCharacterPayload)
		require.True(t, ok)
		if pl.TemplateId == 2000000 {
			require.Equal(t, uint32(5), pl.AssetData.Quantity)
		}
	}
}

// TestExpandTradeSettlementKeepsTheSnapshotStats pins FR-10.3: a delivered equip
// arrives with its scrolled stats, slots and owner intact, not as a bare
// template.
func TestExpandTradeSettlementKeepsTheSnapshotStats(t *testing.T) {
	steps := expandSettlement(t, tradeSettlementFixture())
	var seen bool
	for _, s := range steps {
		if s.Action() != AcceptToCharacter {
			continue
		}
		pl, ok := s.Payload().(AcceptToCharacterPayload)
		require.True(t, ok)
		require.Equal(t, uint16(17), pl.AssetData.WeaponAttack)
		require.Equal(t, uint16(7), pl.AssetData.Slots)
		require.Equal(t, uint16(2), pl.AssetData.Flag)
		require.Equal(t, "Chronicle", pl.AssetData.Owner)
		seen = true
	}
	require.True(t, seen)
}

// TestExpandTradeSettlementEmitsNoMesoStepsWhenNothingStaged pins the
// items-only trade: no meso staged, no award steps at all.
func TestExpandTradeSettlementEmitsNoMesoStepsWhenNothingStaged(t *testing.T) {
	payload := tradeSettlementFixture()
	payload.Sides[0].MesoStaged = 0
	payload.Sides[0].MesoTax = 0
	payload.Sides[0].MesoDelivered = 0

	for _, s := range expandSettlement(t, payload) {
		require.NotEqual(t, AwardMesos, s.Action())
	}
}

// TestExpandTradeSettlementRejectsSelfTrade pins the guard against a settlement
// that names one character on both sides — the crossing logic would hand the
// player their own items back while consuming the meso.
func TestExpandTradeSettlementRejectsSelfTrade(t *testing.T) {
	payload := tradeSettlementFixture()
	payload.Sides[1].CharacterId = payload.Sides[0].CharacterId

	p := newTestExpansionProcessor(t)
	_, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, payload))
	require.Error(t, err)
	require.Contains(t, err.Error(), "both sides")
}

// TestExpandTradeSettlementRejectsDeliveredMesoAboveInt32 pins the conversion
// guard: MesoDelivered is uint32 and AwardMesosPayload.Amount is int32, so a
// value above MaxInt32 would wrap and turn a credit into a debit.
func TestExpandTradeSettlementRejectsDeliveredMesoAboveInt32(t *testing.T) {
	payload := tradeSettlementFixture()
	payload.Sides[0].MesoDelivered = math.MaxInt32 + 1

	p := newTestExpansionProcessor(t)
	_, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, payload))
	require.Error(t, err)
	require.Contains(t, err.Error(), "int32 range")
}

// tradeUnwindFixture is one abandoned trade: both sides had staged, and both get
// everything back.
func tradeUnwindFixture() TradeUnwindPayload {
	return TradeUnwindPayload{
		TransactionId: uuid.New(),
		Items: []TradeUnwindItem{
			{OwnerId: 100, Item: escrowItem(uuid.New(), 55, 2000000, 5, 2)},
			{OwnerId: 200, Item: escrowItem(uuid.New(), 77, 1302000, 1, 1)},
		},
		Mesos: []TradeUnwindMeso{
			{CharacterId: 100, WorldId: 1, ChannelId: 1, Amount: 10_000_000},
		},
	}
}

func expandUnwind(t *testing.T, payload TradeUnwindPayload) []Step[any] {
	t.Helper()
	p := newTestExpansionProcessor(t)
	steps, err := p.expandTradeUnwind(NewStep[any]("trade_unwind", Pending, TradeUnwind, payload))
	require.NoError(t, err)
	return steps
}

// TestExpandTradeUnwindReturnsEachItemToItsOwner is the teardown invariant: an
// abandoned trade leaves both players exactly where they started. Crossing the
// items here — the settlement's behaviour — would execute a trade neither side
// confirmed.
func TestExpandTradeUnwindReturnsEachItemToItsOwner(t *testing.T) {
	payload := tradeUnwindFixture()
	steps := expandUnwind(t, payload)

	got := map[uint32]uint32{} // templateId -> recipient
	for _, s := range steps {
		if s.Action() != AcceptToCharacter {
			continue
		}
		pl, ok := s.Payload().(AcceptToCharacterPayload)
		require.True(t, ok)
		got[pl.TemplateId] = pl.CharacterId
	}
	require.Equal(t, uint32(100), got[2000000], "side 0's item must come back to side 0")
	require.Equal(t, uint32(200), got[1302000], "side 1's item must come back to side 1")
}

// TestExpandTradeUnwindRefundsTheFullEscrowedMeso pins the difference from
// settlement: a refund is UNTAXED. Taxing an abandoned trade would charge the
// player for a trade that never happened.
func TestExpandTradeUnwindRefundsTheFullEscrowedMeso(t *testing.T) {
	steps := expandUnwind(t, tradeUnwindFixture())

	var refunds []AwardMesosPayload
	for _, s := range steps {
		if s.Action() != AwardMesos {
			continue
		}
		pl, ok := s.Payload().(AwardMesosPayload)
		require.True(t, ok)
		refunds = append(refunds, pl)
	}
	require.Len(t, refunds, 1)
	require.Equal(t, uint32(100), refunds[0].CharacterId, "the refund goes back to the staging character")
	require.Equal(t, int32(10_000_000), refunds[0].Amount, "the refund is the FULL escrowed amount, untaxed")
}

// TestExpandTradeUnwindOrdersReleasesBeforeAccepts mirrors the settlement rule.
func TestExpandTradeUnwindOrdersReleasesBeforeAccepts(t *testing.T) {
	steps := expandUnwind(t, tradeUnwindFixture())

	lastRelease, firstAccept := -1, -1
	for i, a := range actionsOf(steps) {
		if a == ReleaseFromTrade {
			lastRelease = i
		}
		if a == AcceptToCharacter && firstAccept == -1 {
			firstAccept = i
		}
	}
	require.NotEqual(t, -1, lastRelease)
	require.NotEqual(t, -1, firstAccept)
	require.Less(t, lastRelease, firstAccept)
}

// TestExpandTradeUnwindSkipsZeroRefunds pins that a participant who staged no
// meso produces no award step at all, rather than a zero-amount one that
// atlas-character would have to decide what to do with.
func TestExpandTradeUnwindSkipsZeroRefunds(t *testing.T) {
	payload := tradeUnwindFixture()
	payload.Mesos = append(payload.Mesos, TradeUnwindMeso{CharacterId: 200, WorldId: 1, ChannelId: 1, Amount: 0})

	var awards int
	for _, s := range expandUnwind(t, payload) {
		if s.Action() == AwardMesos {
			awards++
		}
	}
	require.Equal(t, 1, awards)
}

// TestTradeSettlementStepUnmarshalsToConcretePayload pins that
// Step[any].UnmarshalJSON has a real TradeSettlement case. Its default arm
// (model.go) decodes ANY unregistered action into map[string]any and assigns it
// through any(payload).(T), an assertion that always succeeds for Step[any] —
// so a forgotten case is completely silent. Asserting the CONCRETE payload type
// is the only thing that catches it.
func TestTradeSettlementStepUnmarshalsToConcretePayload(t *testing.T) {
	raw := []byte(`{
		"stepId": "trade_settlement",
		"status": "pending",
		"action": "trade_settlement",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"worldId": 1,
			"channelId": 1,
			"roomType": 3,
			"sides": [
				{"characterId": 100, "items": [{"escrowId": "22222222-2222-2222-2222-222222222222", "inventoryType": 2, "sourceSlot": 1, "assetId": 55, "templateId": 2000000, "quantity": 5}], "mesoStaged": 10000000, "mesoTax": 400000, "mesoDelivered": 9600000},
				{"characterId": 200, "items": [{"escrowId": "33333333-3333-3333-3333-333333333333", "inventoryType": 1, "sourceSlot": 3, "assetId": 77, "templateId": 1302000, "quantity": 1}], "mesoStaged": 0, "mesoTax": 0, "mesoDelivered": 0}
			]
		}
	}`)

	var step Step[any]
	require.NoError(t, step.UnmarshalJSON(raw))
	require.Equal(t, TradeSettlement, step.Action())

	pl, ok := step.Payload().(TradeSettlementPayload)
	require.Truef(t, ok, "payload decoded as %T, not TradeSettlementPayload — the switch fell through to the map[string]any default arm", step.Payload())
	require.Equal(t, uuid.MustParse("11111111-1111-1111-1111-111111111111"), pl.TransactionId)
	require.Equal(t, uuid.MustParse("22222222-2222-2222-2222-222222222222"), pl.Sides[0].Items[0].EscrowId)
	require.Equal(t, uint32(9_600_000), pl.Sides[0].MesoDelivered)
}

// TestTradeUnwindStepUnmarshalsToConcretePayload is the same guard for the
// teardown composite.
func TestTradeUnwindStepUnmarshalsToConcretePayload(t *testing.T) {
	raw := []byte(`{
		"stepId": "trade_unwind",
		"status": "pending",
		"action": "trade_unwind",
		"payload": {
			"transactionId": "11111111-1111-1111-1111-111111111111",
			"items": [{"ownerId": 100, "item": {"escrowId": "22222222-2222-2222-2222-222222222222", "inventoryType": 2, "sourceSlot": 1, "assetId": 55, "templateId": 2000000, "quantity": 5}}],
			"mesos": [{"characterId": 100, "worldId": 1, "channelId": 1, "amount": 500}]
		}
	}`)

	var step Step[any]
	require.NoError(t, step.UnmarshalJSON(raw))
	require.Equal(t, TradeUnwind, step.Action())

	pl, ok := step.Payload().(TradeUnwindPayload)
	require.Truef(t, ok, "payload decoded as %T, not TradeUnwindPayload", step.Payload())
	require.Len(t, pl.Items, 1)
	require.Equal(t, uint32(100), uint32(pl.Items[0].OwnerId))
	require.Equal(t, uint32(500), pl.Mesos[0].Amount)
}

// TestExtractCharacterIdForTradeSettlement pins that a trade_settlement step
// surfaces a participant rather than the 0 that the extractor's default arm
// returns for unknown payloads.
func TestExtractCharacterIdForTradeSettlement(t *testing.T) {
	step := NewStep[any]("trade_settlement", Pending, TradeSettlement, tradeSettlementFixture())
	require.Equal(t, uint32(100), ExtractCharacterId(step))
}

// TestDetermineErrorCodeForTradeTransaction pins the trade branch of
// DetermineErrorCode: the expanded steps' failures carry a diagnostic code
// (atlas-trades collapses all of them into LEAVE 8).
func TestDetermineErrorCodeForTradeTransaction(t *testing.T) {
	s, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(TradeTransaction).
		SetInitiatedBy("test").
		AddStep("trade_settlement", Pending, TradeSettlement, tradeSettlementFixture()).
		Build()
	require.NoError(t, err)

	require.Equal(t, "INVENTORY_FULL", DetermineErrorCode(s, NewStep[any]("accept_to_character_200_55", Failed, AcceptToCharacter, AcceptToCharacterPayload{CharacterId: 200})))
	require.Equal(t, "NOT_ENOUGH_MESOS", DetermineErrorCode(s, NewStep[any]("award_mesos_credit_200", Failed, AwardMesos, AwardMesosPayload{CharacterId: 200, Amount: 10})))
	require.Equal(t, "UNKNOWN", DetermineErrorCode(s, NewStep[any]("release_from_trade_100_55", Failed, ReleaseFromTrade, ReleaseFromTradePayload{})))
}

// stepIdsContain is a small readability helper for assertions on emitted ids.
func stepIdsContain(steps []Step[any], substr string) bool {
	for _, s := range steps {
		if strings.Contains(s.StepId(), substr) {
			return true
		}
	}
	return false
}

// TestExpandedTradeStepIdsNameTheirSubject pins that emitted step ids stay
// human-diagnosable: a release names the owner and asset, an accept names the
// recipient. Failures are reported by step id, so an id that named neither
// would make a production failure untraceable to a player.
func TestExpandedTradeStepIdsNameTheirSubject(t *testing.T) {
	steps := expandSettlement(t, tradeSettlementFixture())
	require.True(t, stepIdsContain(steps, "release_from_trade_100_55"))
	require.True(t, stepIdsContain(steps, "accept_to_character_200_55"))
}
