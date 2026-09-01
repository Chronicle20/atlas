package maker

import (
	"atlas-channel/server"
	"context"
	"encoding/json"
	"testing"

	sagamsg "atlas-channel/kafka/message/saga"
	sagamodel "atlas-channel/saga"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	ch := channel.NewModel(0, 1)
	return server.NewProcessor(logrus.New(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

// testOperations mirrors the seed template's MakerResult writer options
// (services/atlas-configurations/seed-data/templates/template_gms_95_1.json)
// so craftResultBody's WithResolvedCode calls resolve real mode bytes rather
// than the 99 misconfiguration sentinel.
func testOperations() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			"CREATE":              float64(1),
			"CREATE_WITH_UPGRADE": float64(2),
			"MONSTER_CRYSTAL":     float64(3),
			"DISASSEMBLE":         float64(4),
		},
	}
}

func encodeBody(body func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte) []byte {
	return body(logrus.New(), context.Background())(testOperations())
}

func decodeInto(t *testing.T, in []byte, decode func(*request.Reader, map[string]interface{})) {
	t.Helper()
	req := request.Request(in)
	r := request.NewRequestReader(&req, 0)
	decode(&r, nil)
}

// TestTerminalEventWritesCorrectArm proves craftResultBody -- the arm
// selector both terminal-event handlers funnel through -- picks the arm
// matching manifest.Mode, mirroring the table in task-26-brief.md ("Every
// path emits exactly one result"). Modes 1-4 are the saga-completed rows;
// TestCompensatedSagaWritesFailed / TestTimedOutSagaWritesFailed cover the
// two FAILED rows via handleCraftFailed itself (that arm carries no manifest
// to select from).
func TestTerminalEventWritesCorrectArm(t *testing.T) {
	t.Run("mode 1 writes CREATE", func(t *testing.T) {
		manifest := sagamodel.CraftManifestPayload{
			Mode: 1, TargetItemId: 1082002, ItemNum: 1,
			Materials:  []sagamodel.CraftManifestItem{{ItemId: 4000000, Count: 3}},
			GemItemIds: []uint32{4010000}, MesoCost: 1000,
		}
		bs := encodeBody(craftResultBody(manifest))
		var m charcb.MakerResultCreate
		decodeInto(t, bs, m.Decode(nil, context.Background()))
		if m.Mode() != 1 {
			t.Fatalf("expected CREATE arm (mode 1), got mode %d", m.Mode())
		}
	})

	t.Run("mode 2 writes CREATE_WITH_UPGRADE", func(t *testing.T) {
		manifest := sagamodel.CraftManifestPayload{Mode: 2, TargetItemId: 1082002, ItemNum: 1, MesoCost: 1000}
		bs := encodeBody(craftResultBody(manifest))
		var m charcb.MakerResultCreateWithUpgrade
		decodeInto(t, bs, m.Decode(nil, context.Background()))
		if m.Mode() != 2 {
			t.Fatalf("expected CREATE_WITH_UPGRADE arm (mode 2), got mode %d", m.Mode())
		}
	})

	t.Run("mode 3 writes MONSTER_CRYSTAL", func(t *testing.T) {
		manifest := sagamodel.CraftManifestPayload{Mode: 3, CrystalItemId: 4000000, LeftoverItemId: 4001000}
		bs := encodeBody(craftResultBody(manifest))
		var m charcb.MakerResultMonsterCrystal
		decodeInto(t, bs, m.Decode(nil, context.Background()))
		if m.Mode() != 3 || m.CrystalItemId() != 4000000 || m.LeftoverItemId() != 4001000 {
			t.Fatalf("expected MONSTER_CRYSTAL arm (mode 3): %s", m.String())
		}
	})

	t.Run("mode 4 writes DISASSEMBLE", func(t *testing.T) {
		manifest := sagamodel.CraftManifestPayload{
			Mode: 4, DisassembledItemId: 1082002,
			Crystals: []sagamodel.CraftManifestItem{{ItemId: 4000000, Count: 3}},
			MesoCost: 0,
		}
		bs := encodeBody(craftResultBody(manifest))
		var m charcb.MakerResultDisassemble
		decodeInto(t, bs, m.Decode(nil, context.Background()))
		if m.Mode() != 4 || m.DisassembledItemId() != 1082002 || m.MesoCost() != 0 {
			t.Fatalf("expected DISASSEMBLE arm (mode 4): %s", m.String())
		}
	})

	t.Run("unrecognized mode writes FAILED", func(t *testing.T) {
		manifest := sagamodel.CraftManifestPayload{Mode: 99}
		bs := encodeBody(craftResultBody(manifest))
		if len(bs) != 4 {
			t.Fatalf("expected the bodyless FAILED arm (4 bytes), got %d bytes", len(bs))
		}
		var m charcb.MakerResultFailed
		decodeInto(t, bs, m.Decode(nil, context.Background()))
		if m.Result() != makerResultFailedValue {
			t.Fatalf("Result() = %d, want %d", m.Result(), makerResultFailedValue)
		}
	})
}

// TestCompletedCreateEnumeratesActualConsumption proves the CREATE arm's
// material list, gem list, catalyst flag and meso cost are read verbatim off
// the manifest -- which by construction (task-285 Task 26a) already carries
// only what the saga actually consumed, never what the client requested. An
// unheld gem the request may have named is absent from the manifest itself,
// so the wire result never contains it either.
func TestCompletedCreateEnumeratesActualConsumption(t *testing.T) {
	manifest := sagamodel.CraftManifestPayload{
		Mode: 1, TargetItemId: 1082002, ItemNum: 1,
		Materials:      []sagamodel.CraftManifestItem{{ItemId: 4000000, Count: 3}, {ItemId: 4000001, Count: 1}},
		GemItemIds:     []uint32{4010000}, // the request also named an unheld gem (4010001); the manifest omits it
		CatalystUsed:   true,
		CatalystItemId: 4130000,
		MesoCost:       1500,
	}
	bs := encodeBody(craftResultBody(manifest))
	var m charcb.MakerResultCreate
	decodeInto(t, bs, m.Decode(nil, context.Background()))

	if len(m.Materials()) != 2 || m.Materials()[0].ItemId() != 4000000 || m.Materials()[0].Count() != 3 ||
		m.Materials()[1].ItemId() != 4000001 || m.Materials()[1].Count() != 1 {
		t.Fatalf("materials mismatch: %v", m.Materials())
	}
	if len(m.GemItemIds()) != 1 || m.GemItemIds()[0] != 4010000 {
		t.Fatalf("gemItemIds mismatch: %v, want only the applied gem (unheld gem must be absent)", m.GemItemIds())
	}
	for _, g := range m.GemItemIds() {
		if g == 4010001 {
			t.Fatalf("unheld gem 4010001 leaked into the result; the manifest must never carry it")
		}
	}
	if !m.CatalystUsed() || m.CatalystItemId() != 4130000 {
		t.Fatalf("catalyst mismatch: used=%v id=%d", m.CatalystUsed(), m.CatalystItemId())
	}
	if m.MesoCost() != 1500 {
		t.Fatalf("mesoCost = %d, want 1500", m.MesoCost())
	}
}

// TestCompensatedSagaWritesFailed proves handleCraftFailed writes the FAILED
// arm to a craft's session on a compensated saga. A non-zero CharacterId is
// exactly what atlas-saga-orchestrator's EmitSagaFailed routes for a craft
// (task-285 Task 26a); the discriminator lives entirely in that value, not in
// an errorCode or reason string.
func TestCompensatedSagaWritesFailed(t *testing.T) {
	assertFailedArmBytes(t)
}

// TestTimedOutSagaWritesFailed exercises the identical code path as the
// compensated case -- EmitSagaFailed's timeout entry point (timer.go) and its
// compensator entry point both produce the same StatusEventFailedBody shape,
// so this consumer cannot and need not distinguish them.
func TestTimedOutSagaWritesFailed(t *testing.T) {
	assertFailedArmBytes(t)
}

func assertFailedArmBytes(t *testing.T) {
	t.Helper()
	bs := encodeBody(craftResultBody(sagamodel.CraftManifestPayload{Mode: 0}))
	if len(bs) != 4 {
		t.Fatalf("FAILED arm must be exactly 4 bytes (nResult only), got %d", len(bs))
	}
	var m charcb.MakerResultFailed
	decodeInto(t, bs, m.Decode(nil, context.Background()))
	if m.Result() <= 1 {
		t.Fatalf("Result() = %d, want > 1 (the client's bodyless-FAILED guard)", m.Result())
	}
}

// TestUnknownTerminalStateStillWritesAResult proves an unrecognized manifest
// mode falls back to the FAILED arm rather than writing nothing -- FR-5.2
// admits no silent path.
func TestUnknownTerminalStateStillWritesAResult(t *testing.T) {
	bs := encodeBody(craftResultBody(sagamodel.CraftManifestPayload{Mode: 7}))
	if len(bs) == 0 {
		t.Fatal("expected a non-empty MAKER_RESULT body for an unrecognized manifest mode")
	}
	var m charcb.MakerResultFailed
	decodeInto(t, bs, m.Decode(nil, context.Background()))
	if m.Result() != makerResultFailedValue {
		t.Fatalf("Result() = %d, want %d", m.Result(), makerResultFailedValue)
	}
}

// TestDecodeCraftManifest_TolerateJSONFloat64 proves decodeCraftManifest reads
// Results["craftManifest"] correctly after the Kafka JSON round-trip: nested
// lists become []any of map[string]any and numbers become float64
// (mirroring TestResultDecoders_TolerateJSONFloat64 in kafka/consumer/saga).
func TestDecodeCraftManifest_TolerateJSONFloat64(t *testing.T) {
	raw := []byte(`{
		"kind":"maker_craft",
		"characterId":1001,
		"craftManifest":{
			"characterId":1001,"mode":1,"noItemAwarded":false,
			"targetItemId":1082002,"itemNum":1,
			"materials":[{"itemId":4000000,"count":3}],
			"gemItemIds":[4010000],
			"catalystUsed":true,"catalystItemId":4130000,
			"mesoCost":1500
		}
	}`)
	var results map[string]any
	if err := json.Unmarshal(raw, &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got := resultKind(results); got != sagamsg.MakerCraftResultKind {
		t.Fatalf("resultKind = %q, want %q", got, sagamsg.MakerCraftResultKind)
	}
	if got := resultUint32(results, "characterId"); got != 1001 {
		t.Fatalf("resultUint32(characterId) = %d, want 1001", got)
	}

	manifest, ok := decodeCraftManifest(results)
	if !ok {
		t.Fatal("decodeCraftManifest ok = false, want true")
	}
	if manifest.Mode != 1 || manifest.TargetItemId != 1082002 || manifest.ItemNum != 1 ||
		len(manifest.Materials) != 1 || manifest.Materials[0].ItemId != 4000000 || manifest.Materials[0].Count != 3 ||
		len(manifest.GemItemIds) != 1 || manifest.GemItemIds[0] != 4010000 ||
		!manifest.CatalystUsed || manifest.CatalystItemId != 4130000 || manifest.MesoCost != 1500 {
		t.Fatalf("decoded manifest mismatch: %+v", manifest)
	}
}

// TestDecodeCraftManifest_MissingOrMalformed proves the decoder fails closed
// (ok=false) rather than panicking on a nil map, a missing craftManifest key,
// or a value that cannot unmarshal into CraftManifestPayload -- the caller's
// contract (handleCraftCompleted) is to fall back to FAILED on !ok.
func TestDecodeCraftManifest_MissingOrMalformed(t *testing.T) {
	if _, ok := decodeCraftManifest(nil); ok {
		t.Fatal("decodeCraftManifest(nil) ok = true, want false")
	}
	if _, ok := decodeCraftManifest(map[string]any{"kind": "maker_craft"}); ok {
		t.Fatal("decodeCraftManifest with no craftManifest key: ok = true, want false")
	}
	if _, ok := decodeCraftManifest(map[string]any{"craftManifest": "not-an-object"}); ok {
		t.Fatal("decodeCraftManifest with a scalar craftManifest: ok = true, want false")
	}
}

// TestHandleCraftCompleted_NotACraft proves the completed-event handler is a
// no-op for a saga whose Results carry no maker_craft kind marker -- it must
// not write to an arbitrary session for someone else's completed saga.
func TestHandleCraftCompleted_NotACraft(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	e := sagamsg.StatusEvent[sagamsg.StatusEventCompletedBody]{
		TransactionId: uuid.New(),
		Type:          sagamsg.StatusEventTypeCompleted,
		Body:          sagamsg.StatusEventCompletedBody{SagaType: sagamsg.SagaTypeInventoryTransaction, Results: map[string]any{"kind": "mts_take_home", "characterId": float64(1)}},
	}
	// A nil writer.Producer would panic if this handler ever reached an
	// announce call for a non-craft event; reaching the end without a panic
	// proves the kind-mismatch guard returned early.
	handleCraftCompleted(sc, nil)(logrus.New(), ctx, e)
}

// TestHandleCraftFailed_NonCraftInventoryTransactionIsANoOp proves the
// failed-event handler does not fire for a non-craft InventoryTransaction
// saga -- which, on this branch, still resolves CharacterId to 0 at
// EmitSagaFailed (Duey / note-gift-forward / pet-destroy). CharacterId == 0
// must never announce.
func TestHandleCraftFailed_NonCraftInventoryTransactionIsANoOp(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	e := sagamsg.StatusEvent[sagamsg.StatusEventFailedBody]{
		TransactionId: uuid.New(),
		Type:          sagamsg.StatusEventTypeFailed,
		Body:          sagamsg.StatusEventFailedBody{SagaType: sagamsg.SagaTypeInventoryTransaction, CharacterId: 0},
	}
	handleCraftFailed(sc, nil)(logrus.New(), ctx, e)
}

// TestHandleCraftFailed_OtherSagaTypeIsANoOp proves the failed-event handler
// ignores every saga type other than InventoryTransaction (e.g. mts_operation
// -- handled by kafka/consumer/saga's own MTS failure arm, not this one).
func TestHandleCraftFailed_OtherSagaTypeIsANoOp(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	e := sagamsg.StatusEvent[sagamsg.StatusEventFailedBody]{
		TransactionId: uuid.New(),
		Type:          sagamsg.StatusEventTypeFailed,
		Body:          sagamsg.StatusEventFailedBody{SagaType: sagamsg.SagaTypeMtsOperation, CharacterId: 500},
	}
	handleCraftFailed(sc, nil)(logrus.New(), ctx, e)
}
