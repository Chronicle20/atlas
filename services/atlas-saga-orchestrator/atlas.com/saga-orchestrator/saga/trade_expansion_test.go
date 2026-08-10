package saga

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// testAsset is one row of a stubbed inventory compartment: the fields
// expandTradeSettlement actually reads off compartment.AssetRestModel.
type testAsset struct {
	CharacterId   uint32
	InventoryType byte
	Slot          int16
	TemplateId    uint32
	Quantity      uint32
	Id            string
}

// testProcessorWithCompartments stands up an httptest inventory service that
// answers GET characters/{id}/inventory/compartments?type={t} from the supplied
// asset table, points INVENTORY_SERVICE_URL at it, and returns a
// *ProcessorImpl wired with a tenant context. The expansion functions reach
// inventory only through compartment.RequestCompartment, so no other processor
// dependency is needed.
func testProcessorWithCompartments(t *testing.T, assets []testAsset) *ProcessorImpl {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		characterId, err := characterIdFromCompartmentPath(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		invType, err := strconv.ParseUint(r.URL.Query().Get("type"), 10, 8)
		if err != nil {
			http.Error(w, "bad type: "+err.Error(), http.StatusBadRequest)
			return
		}
		var matched []testAsset
		for _, a := range assets {
			if a.CharacterId == characterId && a.InventoryType == byte(invType) {
				matched = append(matched, a)
			}
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(compartmentDoc(byte(invType), matched)))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	// The base URL is resolved per request, not at construction, so the shared
	// constructor can be reused verbatim once the env var is set.
	return newTestExpansionProcessor(t)
}

// characterIdFromCompartmentPath pulls {id} out of
// /characters/{id}/inventory/compartments.
func characterIdFromCompartmentPath(path string) (uint32, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 || parts[0] != "characters" {
		return 0, fmt.Errorf("unexpected compartment path %q", path)
	}
	id, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("unexpected character id in %q: %w", path, err)
	}
	return uint32(id), nil
}

// compartmentDoc renders a JSON:API compartment whose assets relationship is
// materialized from the included block, matching compartment.CompartmentRestModel.
func compartmentDoc(invType byte, assets []testAsset) string {
	refs := make([]map[string]any, 0, len(assets))
	included := make([]map[string]any, 0, len(assets))
	for _, a := range assets {
		refs = append(refs, map[string]any{"type": "assets", "id": a.Id})
		included = append(included, map[string]any{
			"type": "assets",
			"id":   a.Id,
			"attributes": map[string]any{
				"slot":       a.Slot,
				"templateId": a.TemplateId,
				"quantity":   a.Quantity,
				"owner":      "Chronicle",
			},
		})
	}
	doc := map[string]any{
		"data": map[string]any{
			"type":          "compartments",
			"id":            fmt.Sprintf("comp-%d", invType),
			"attributes":    map[string]any{"type": invType, "capacity": 24},
			"relationships": map[string]any{"assets": map[string]any{"data": refs}},
		},
		"included": included,
	}
	b, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// tradeCompartments is the shared fixture inventory: character 100 holds five
// of item 2000000 in USE slot 1, character 200 holds one 1302000 in EQUIP
// slot 3.
func tradeCompartments() []testAsset {
	return []testAsset{
		{CharacterId: 100, InventoryType: 2, Slot: 1, TemplateId: 2000000, Quantity: 5, Id: "55"},
		{CharacterId: 200, InventoryType: 1, Slot: 3, TemplateId: 1302000, Quantity: 1, Id: "77"},
	}
}

// tradeSettlementFixture stages the compartments above plus 10,000,000 meso
// from side A, already taxed at the default 4% tier (design §6.3 requires the
// tax to arrive RESOLVED — the orchestrator does no rate arithmetic).
func tradeSettlementFixture() TradeSettlementPayload {
	return TradeSettlementPayload{
		TransactionId: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		WorldId:       1,
		ChannelId:     1,
		RoomType:      3,
		Sides: [2]TradeSettlementSide{
			{
				CharacterId:   100,
				Items:         []TradeSettlementItem{{InventoryType: 2, SourceSlot: 1, AssetId: 55, TemplateId: 2000000, Quantity: 5}},
				MesoStaged:    10_000_000,
				MesoTax:       400_000,
				MesoDelivered: 9_600_000,
			},
			{
				CharacterId: 200,
				Items:       []TradeSettlementItem{{InventoryType: 1, SourceSlot: 3, AssetId: 77, TemplateId: 1302000, Quantity: 1}},
			},
		},
	}
}

// TestExpandTradeSettlementOrdersReleasesBeforeAccepts pins design §6.3: every
// release precedes every accept, so an outgoing item's slot is free for an
// incoming one and a release failure compensates before anything is created.
func TestExpandTradeSettlementOrdersReleasesBeforeAccepts(t *testing.T) {
	p := testProcessorWithCompartments(t, tradeCompartments())

	step := NewStep[any]("trade_settlement", Pending, TradeSettlement, tradeSettlementFixture())
	steps, err := p.expandTradeSettlement(step)
	require.NoError(t, err)

	lastRelease, firstAccept := -1, -1
	for i, s := range steps {
		switch s.Action() {
		case ReleaseFromCharacter:
			lastRelease = i
		case AcceptToCharacter:
			if firstAccept == -1 {
				firstAccept = i
			}
		}
	}
	require.NotEqualf(t, -1, lastRelease, "expected release steps, got %d steps", len(steps))
	require.NotEqualf(t, -1, firstAccept, "expected accept steps, got %d steps", len(steps))
	require.Lessf(t, lastRelease, firstAccept, "release at %d follows accept at %d; every release must precede every accept", lastRelease, firstAccept)
}

// TestExpandTradeSettlementReleasesMatchAccepts pins the atomicity of the swap
// at expansion time: every staged item contributes exactly one release AND one
// accept. A partial expansion that moved one side's goods without the other's
// would leave the counts unbalanced.
func TestExpandTradeSettlementReleasesMatchAccepts(t *testing.T) {
	p := testProcessorWithCompartments(t, tradeCompartments())
	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, tradeSettlementFixture()))
	require.NoError(t, err)

	releasedAssets := make(map[uint32]uint32)
	acceptedTemplates := make(map[uint32]int)
	for _, s := range steps {
		switch s.Action() {
		case ReleaseFromCharacter:
			pl, ok := s.Payload().(ReleaseFromCharacterPayload)
			require.True(t, ok, "release step payload must be ReleaseFromCharacterPayload, got %T", s.Payload())
			releasedAssets[pl.AssetId] = pl.Quantity
		case AcceptToCharacter:
			pl, ok := s.Payload().(AcceptToCharacterPayload)
			require.True(t, ok, "accept step payload must be AcceptToCharacterPayload, got %T", s.Payload())
			acceptedTemplates[pl.TemplateId]++
		}
	}
	require.Equal(t, map[uint32]uint32{55: 5, 77: 1}, releasedAssets)
	require.Equal(t, map[uint32]int{2000000: 1, 1302000: 1}, acceptedTemplates)
}

// TestExpandTradeSettlementCrossesTheAssets pins that A's items are accepted by
// B and vice versa — a same-side accept would be a no-op swap.
func TestExpandTradeSettlementCrossesTheAssets(t *testing.T) {
	p := testProcessorWithCompartments(t, tradeCompartments())
	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, tradeSettlementFixture()))
	require.NoError(t, err)

	seen := 0
	for _, s := range steps {
		if s.Action() != AcceptToCharacter {
			continue
		}
		pl, ok := s.Payload().(AcceptToCharacterPayload)
		require.True(t, ok, "accept step payload must be AcceptToCharacterPayload, got %T", s.Payload())
		switch pl.TemplateId {
		case 2000000: // staged by character 100
			require.Equal(t, uint32(200), pl.CharacterId, "item 2000000 must be accepted by 200")
			require.Equal(t, byte(2), pl.InventoryType)
			seen++
		case 1302000: // staged by character 200
			require.Equal(t, uint32(100), pl.CharacterId, "item 1302000 must be accepted by 100")
			require.Equal(t, byte(1), pl.InventoryType)
			seen++
		default:
			t.Fatalf("unexpected accepted template %d", pl.TemplateId)
		}
	}
	require.Equal(t, 2, seen, "both staged items must produce an accept step")
}

// TestExpandTradeSettlementSnapshotsBeforeRelease pins that the accept step
// carries the AssetData read from the owner's compartment BEFORE any release
// soft-deletes it — the quantity/owner on the snapshot come from the lookup,
// not from a zero value.
func TestExpandTradeSettlementSnapshotsBeforeRelease(t *testing.T) {
	p := testProcessorWithCompartments(t, tradeCompartments())
	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, tradeSettlementFixture()))
	require.NoError(t, err)

	for _, s := range steps {
		if s.Action() != AcceptToCharacter {
			continue
		}
		pl, ok := s.Payload().(AcceptToCharacterPayload)
		require.True(t, ok, "accept step payload must be AcceptToCharacterPayload, got %T", s.Payload())
		require.Equal(t, "Chronicle", pl.AssetData.Owner)
		if pl.TemplateId == 2000000 {
			require.Equal(t, uint32(5), pl.AssetData.Quantity)
		} else {
			require.Equal(t, uint32(1), pl.AssetData.Quantity)
		}
	}
}

// TestExpandTradeSettlementAcceptsOnlyTheStagedQuantity pins the partial-stack
// case the shared fixture cannot reach: it stages whole stacks, so the source
// asset's quantity and the staged quantity coincide and a snapshot that carried
// the SOURCE stack would look correct.
//
// The accept RECREATES the asset from AssetData, so AssetData.Quantity is what
// the recipient is awarded. Staging 1 of a 200 stack must release 1 and award
// 1 — awarding the snapshot's 200 mints 199 items out of nothing.
func TestExpandTradeSettlementAcceptsOnlyTheStagedQuantity(t *testing.T) {
	p := testProcessorWithCompartments(t, []testAsset{
		{CharacterId: 100, InventoryType: 2, Slot: 2, TemplateId: 2000005, Quantity: 200, Id: "2"},
	})
	payload := TradeSettlementPayload{
		TransactionId: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		RoomType:      3,
		Sides: [2]TradeSettlementSide{
			{
				CharacterId: 100,
				Items:       []TradeSettlementItem{{InventoryType: 2, SourceSlot: 2, AssetId: 2, TemplateId: 2000005, Quantity: 1}},
			},
			{CharacterId: 200},
		},
	}

	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, payload))
	require.NoError(t, err)

	releases, accepts := 0, 0
	for _, s := range steps {
		switch s.Action() {
		case ReleaseFromCharacter:
			pl, ok := s.Payload().(ReleaseFromCharacterPayload)
			require.True(t, ok, "release step payload must be ReleaseFromCharacterPayload, got %T", s.Payload())
			require.Equal(t, uint32(1), pl.Quantity, "release must take only the staged quantity")
			releases++
		case AcceptToCharacter:
			pl, ok := s.Payload().(AcceptToCharacterPayload)
			require.True(t, ok, "accept step payload must be AcceptToCharacterPayload, got %T", s.Payload())
			require.Equal(t, uint32(200), pl.CharacterId, "the staged item must cross to the other side")
			require.Equal(t, uint32(1), pl.AssetData.Quantity, "accept must award the STAGED quantity, not the source stack")
			accepts++
		}
	}
	require.Equal(t, 1, releases)
	require.Equal(t, 1, accepts)
}

// TestExpandTradeSettlementKeepsTheSnapshotsNonQuantityFields pins that
// overriding the quantity does not disturb the rest of the snapshot — the whole
// point of snapshotting is that cash ownership, expiry and rolled stats survive
// the transfer (FR-10.3).
func TestExpandTradeSettlementKeepsTheSnapshotsNonQuantityFields(t *testing.T) {
	p := testProcessorWithCompartments(t, []testAsset{
		{CharacterId: 100, InventoryType: 2, Slot: 2, TemplateId: 2000005, Quantity: 200, Id: "2"},
	})
	payload := TradeSettlementPayload{
		TransactionId: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		RoomType:      3,
		Sides: [2]TradeSettlementSide{
			{
				CharacterId: 100,
				Items:       []TradeSettlementItem{{InventoryType: 2, SourceSlot: 2, AssetId: 2, TemplateId: 2000005, Quantity: 1}},
			},
			{CharacterId: 200},
		},
	}

	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, payload))
	require.NoError(t, err)

	for _, s := range steps {
		if s.Action() != AcceptToCharacter {
			continue
		}
		pl, ok := s.Payload().(AcceptToCharacterPayload)
		require.True(t, ok, "accept step payload must be AcceptToCharacterPayload, got %T", s.Payload())
		require.Equal(t, "Chronicle", pl.AssetData.Owner)
		require.Equal(t, uint32(2000005), pl.TemplateId)
	}
}

// TestExpandTradeSettlementMesoIsAsymmetric pins design §6.5: the giver is
// deducted the FULL staged amount and the receiver credited the POST-TAX
// amount, so the tax is destroyed rather than moved.
func TestExpandTradeSettlementMesoIsAsymmetric(t *testing.T) {
	p := testProcessorWithCompartments(t, tradeCompartments())
	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, tradeSettlementFixture()))
	require.NoError(t, err)

	var deducted, credited int32
	var deductedFrom, creditedTo uint32
	for _, s := range steps {
		if s.Action() != AwardMesos {
			continue
		}
		pl, ok := s.Payload().(AwardMesosPayload)
		require.True(t, ok, "meso step payload must be AwardMesosPayload, got %T", s.Payload())
		if pl.Amount < 0 {
			deducted += -pl.Amount
			deductedFrom = pl.CharacterId
		} else {
			credited += pl.Amount
			creditedTo = pl.CharacterId
		}
	}
	// Fixture stages 10,000,000 from side A at the 4% tier.
	require.Equal(t, int32(10_000_000), deducted)
	require.Equal(t, int32(9_600_000), credited)
	require.Equal(t, int32(400_000), deducted-credited, "the tax must be destroyed, not credited")
	require.Equal(t, uint32(100), deductedFrom, "the staging side is deducted")
	require.Equal(t, uint32(200), creditedTo, "the other side is credited")
}

// TestExpandTradeSettlementEmitsNoMesoStepsWhenNothingStaged pins that an
// item-only trade produces no award_mesos steps at all.
func TestExpandTradeSettlementEmitsNoMesoStepsWhenNothingStaged(t *testing.T) {
	p := testProcessorWithCompartments(t, tradeCompartments())
	fixture := tradeSettlementFixture()
	fixture.Sides[0].MesoStaged, fixture.Sides[0].MesoTax, fixture.Sides[0].MesoDelivered = 0, 0, 0
	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, fixture))
	require.NoError(t, err)
	for _, s := range steps {
		require.NotEqualf(t, AwardMesos, s.Action(), "unexpected award_mesos step for an item-only trade: %+v", s.Payload())
	}
}

// TestExpandTradeSettlementRejectsSelfTrade pins that a payload naming the same
// character on both sides is refused rather than expanded into a self-swap.
func TestExpandTradeSettlementRejectsSelfTrade(t *testing.T) {
	p := testProcessorWithCompartments(t, tradeCompartments())
	fixture := tradeSettlementFixture()
	fixture.Sides[1].CharacterId = fixture.Sides[0].CharacterId
	_, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, fixture))
	require.ErrorContains(t, err, "both sides")
}

// TestExpandTradeSettlementFailsWhenStagedSlotMoved pins the revalidation
// (design §5.3): if the staged slot no longer holds the staged template the
// expansion fails and NO steps are produced, so neither side's goods move.
func TestExpandTradeSettlementFailsWhenStagedSlotMoved(t *testing.T) {
	moved := tradeCompartments()
	moved[1].TemplateId = 1302001 // character 200's equip slot now holds something else
	p := testProcessorWithCompartments(t, moved)
	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, tradeSettlementFixture()))
	require.Error(t, err)
	require.Nil(t, steps, "a failed expansion must not emit a partial step list")
}

// TestExpandTradeSettlementFailsOnInstanceSubstitution pins the instance check:
// the staged slot holding the RIGHT template but a DIFFERENT asset instance must
// fail. Reservations block move/merge/drop, but a same-template substitution
// would otherwise pass the template comparison — and two equips of one template
// carry different scrolled stats.
func TestExpandTradeSettlementFailsOnInstanceSubstitution(t *testing.T) {
	swapped := tradeCompartments()
	swapped[1].Id = "78" // same slot, same template, different instance
	p := testProcessorWithCompartments(t, swapped)
	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, tradeSettlementFixture()))
	require.ErrorContains(t, err, "expected staged instance")
	require.Nil(t, steps)
}

// TestExpandTradeSettlementFailsOnUnparseableAssetId pins that an asset id the
// expander cannot represent is a loud error rather than a silent release of
// asset 0. The reachable shape is an id outside uint32 (the JSON:API layer
// already rejects a non-numeric id at unmarshal, before the expander sees it).
func TestExpandTradeSettlementFailsOnUnparseableAssetId(t *testing.T) {
	garbled := tradeCompartments()
	garbled[0].Id = "4294967296" // 2^32 — parses as an int, overflows uint32
	p := testProcessorWithCompartments(t, garbled)
	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, tradeSettlementFixture()))
	require.ErrorContains(t, err, "unparseable asset id")
	require.Nil(t, steps)
}

// TestExpandTradeSettlementRejectsMesoAboveInt32 pins the overflow guard:
// AwardMesosPayload.Amount is int32, so a staged amount above MaxInt32 would
// wrap on conversion and turn the giver's deduction into a credit.
func TestExpandTradeSettlementRejectsMesoAboveInt32(t *testing.T) {
	p := testProcessorWithCompartments(t, tradeCompartments())

	overflow := tradeSettlementFixture()
	overflow.Sides[0].MesoStaged = math.MaxInt32 + 1
	steps, err := p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, overflow))
	require.ErrorContains(t, err, "exceeds int32 range")
	require.Nil(t, steps)

	// The boundary value itself is accepted, and its deduction stays negative.
	atLimit := tradeSettlementFixture()
	atLimit.Sides[0].MesoStaged = math.MaxInt32
	atLimit.Sides[0].MesoDelivered = math.MaxInt32
	steps, err = p.expandTradeSettlement(NewStep[any]("trade_settlement", Pending, TradeSettlement, atLimit))
	require.NoError(t, err)
	var sawDeduction bool
	for _, s := range steps {
		if s.Action() != AwardMesos {
			continue
		}
		pl, ok := s.Payload().(AwardMesosPayload)
		require.True(t, ok, "meso step payload must be AwardMesosPayload, got %T", s.Payload())
		if pl.CharacterId == 100 {
			require.Equal(t, int32(-math.MaxInt32), pl.Amount)
			sawDeduction = true
		}
	}
	require.True(t, sawDeduction)
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
				{"characterId": 100, "items": [{"inventoryType": 2, "sourceSlot": 1, "assetId": 55, "templateId": 2000000, "quantity": 5}], "mesoStaged": 10000000, "mesoTax": 400000, "mesoDelivered": 9600000},
				{"characterId": 200, "items": [{"inventoryType": 1, "sourceSlot": 3, "assetId": 77, "templateId": 1302000, "quantity": 1}], "mesoStaged": 0, "mesoTax": 0, "mesoDelivered": 0}
			]
		}
	}`)

	var step Step[any]
	require.NoError(t, step.UnmarshalJSON(raw))
	require.Equal(t, TradeSettlement, step.Action())

	pl, ok := step.Payload().(TradeSettlementPayload)
	require.Truef(t, ok, "payload decoded as %T, not TradeSettlementPayload — the switch fell through to the map[string]any default arm", step.Payload())
	require.Equal(t, uuid.MustParse("11111111-1111-1111-1111-111111111111"), pl.TransactionId)
	require.Equal(t, uint32(100), uint32(pl.Sides[0].CharacterId))
	require.Equal(t, uint32(200), uint32(pl.Sides[1].CharacterId))
	require.Len(t, pl.Sides[0].Items, 1)
	require.Equal(t, uint32(9_600_000), pl.Sides[0].MesoDelivered)
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
	require.Equal(t, "NOT_ENOUGH_MESOS", DetermineErrorCode(s, NewStep[any]("award_mesos_deduct_100", Failed, AwardMesos, AwardMesosPayload{CharacterId: 100, Amount: -10})))
	require.Equal(t, "UNKNOWN", DetermineErrorCode(s, NewStep[any]("release_from_character_100_55", Failed, ReleaseFromCharacter, ReleaseFromCharacterPayload{CharacterId: 100})))
}
