package recipe

import (
	"atlas-npc-conversations/conversation"
	"atlas-npc-conversations/test"
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func craftState(t *testing.T, stateId, itemId string, mats, qtys []uint32, mesos uint32, stimId uint32, stimFail float64) conversation.StateModel {
	t.Helper()
	// Ensure slices are non-nil and equal-length so the validated builder accepts
	// them. Tests that need truly invalid data (e.g. mismatch) should use
	// craftStateRaw instead.
	if mats == nil {
		mats = []uint32{4011000}
	}
	if qtys == nil {
		qtys = make([]uint32, len(mats))
		for i := range qtys {
			qtys[i] = 1
		}
	}
	caBuilder := conversation.NewCraftActionBuilder().
		SetItemId(itemId).
		SetMaterials(mats).
		SetQuantities(qtys).
		SetMesoCost(mesos).
		SetStimulatorId(stimId).
		SetStimulatorFailChance(stimFail).
		SetSuccessState("end").
		SetFailureState("end").
		SetMissingMaterialsState("end")
	ca, err := caBuilder.Build()
	if err != nil {
		t.Fatalf("craftAction build: %v", err)
	}
	state, err := conversation.NewStateBuilder().
		SetId(stateId).
		SetCraftAction(ca).
		Build()
	if err != nil {
		t.Fatalf("state build: %v", err)
	}
	return state
}

// craftStateRaw bypasses CraftActionBuilder validation to create states with
// deliberately invalid data (e.g. mismatched materials/quantities) for
// processor-level defensive-check tests.
func craftStateRaw(t *testing.T, stateId, itemId string, mats, qtys []uint32) conversation.StateModel {
	t.Helper()
	ca := conversation.NewCraftActionModelDirect(itemId, mats, qtys, 0, 0, 0, "end", "end", "end")
	state, err := conversation.NewStateBuilder().
		SetId(stateId).
		SetCraftAction(ca).
		Build()
	if err != nil {
		t.Fatalf("state build (raw): %v", err)
	}
	return state
}

func TestRebuildForConversation_HappyPath_InsertsRowPerCraftAction(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	tenantId := uuid.New()
	te, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	convId := uuid.New()
	states := []conversation.StateModel{
		craftState(t, "craft0", "1082007", []uint32{4011000}, []uint32{3}, 18000, 0, 0),
		craftState(t, "craft1", "1082008", []uint32{4011000, 4011001}, []uint32{2, 1}, 12000, 4020009, 0.10),
	}

	p := NewProcessor(l, ctx, db)
	res, err := p.RebuildForConversation(db.WithContext(ctx))(2040020, convId, states)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if res.Inserted != 2 || res.Skipped != 0 {
		t.Errorf("unexpected RebuildResult: %+v", res)
	}

	got, err := getByItemIdProvider(1082008)(db.WithContext(ctx))()
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(got) != 1 || got[0].StimulatorID != 4020009 {
		t.Errorf("expected stimulator row, got %+v", got)
	}
}

func TestRebuildForConversation_SkipsUnparseableItemId(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	tenantId := uuid.New()
	te, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	convId := uuid.New()
	states := []conversation.StateModel{
		craftState(t, "ok", "1082007", []uint32{4011000}, []uint32{3}, 18000, 0, 0),
		craftState(t, "bad", "notANumber", []uint32{4011000}, []uint32{3}, 18000, 0, 0),
	}

	p := NewProcessor(l, ctx, db)
	res, err := p.RebuildForConversation(db.WithContext(ctx))(2040020, convId, states)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if res.Inserted != 1 {
		t.Errorf("Inserted = %d, want 1", res.Inserted)
	}
	if res.Skipped != 1 || len(res.SkippedDetails) != 1 || res.SkippedDetails[0].StateId != "bad" {
		t.Errorf("Skipped not recorded: %+v", res)
	}
}

func TestRebuildForConversation_SkipsMaterialsQuantitiesMismatch(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	tenantId := uuid.New()
	te, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	convId := uuid.New()
	// craftStateRaw bypasses builder validation to produce a state with 2 mats
	// but only 1 qty — exercising the processor's defensive mismatch check.
	states := []conversation.StateModel{
		craftStateRaw(t, "mismatch", "1082007", []uint32{4011000, 4011001}, []uint32{3}),
	}

	p := NewProcessor(l, ctx, db)
	res, err := p.RebuildForConversation(db.WithContext(ctx))(2040020, convId, states)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if res.Inserted != 0 || res.Skipped != 1 || res.SkippedDetails[0].Reason == "" {
		t.Errorf("expected one skip with reason: %+v", res)
	}
}

func TestRebuildForConversation_ClearsExistingRowsForConversation(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	tenantId := uuid.New()
	te, _ := tenant.Create(tenantId, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	convId := uuid.New()
	p := NewProcessor(l, ctx, db)

	if _, err := p.RebuildForConversation(db.WithContext(ctx))(2040020, convId, []conversation.StateModel{
		craftState(t, "craftA", "1", []uint32{4011000}, []uint32{1}, 0, 0, 0),
		craftState(t, "craftB", "2", []uint32{4011000}, []uint32{1}, 0, 0, 0),
	}); err != nil {
		t.Fatalf("initial: %v", err)
	}

	if _, err := p.RebuildForConversation(db.WithContext(ctx))(2040020, convId, []conversation.StateModel{
		craftState(t, "craftA", "1", []uint32{4011000}, []uint32{1}, 0, 0, 0),
	}); err != nil {
		t.Fatalf("second rebuild: %v", err)
	}

	got, _ := getAllForTenant(db.WithContext(ctx))
	if len(got) != 1 || got[0].StateID != "craftA" {
		t.Errorf("expected only craftA after rebuild, got %+v", got)
	}
}

func TestTransform_RoundtripsRecipe(t *testing.T) {
	tenantId := uuid.New()
	convId := uuid.New()
	m, _ := NewBuilder().
		SetTenantId(tenantId).
		SetConversationId(convId).
		SetNpcId(2040020).
		SetStateId("craftWarrior0").
		SetItemId(1082007).
		SetMaterials([]Material{{ItemId: 4011000, Quantity: 3}, {ItemId: 4011001, Quantity: 2}}).
		SetMesoCost(18000).
		SetStimulatorId(4020009).
		SetStimulatorFailChance(0.1).
		Build()

	rm := Transform(m)
	if rm.GetID() != m.Id().String() {
		t.Errorf("id roundtrip: %q vs %q", rm.GetID(), m.Id().String())
	}
	if rm.NpcId != 2040020 || rm.ItemId != 1082007 {
		t.Errorf("attribute mismatch: %+v", rm)
	}
	if len(rm.Materials) != 2 || rm.Materials[1].Quantity != 2 {
		t.Errorf("materials: %+v", rm.Materials)
	}
	if rm.StimulatorId != 4020009 || rm.StimulatorFailChance != 0.1 {
		t.Errorf("stimulator fields wrong: %+v", rm)
	}
	if rm.GetName() != Resource {
		t.Errorf("resource name: %q want %q", rm.GetName(), Resource)
	}
}

func TestClearForTenant_RemovesOnlyActiveTenant(t *testing.T) {
	l, _ := logtest.NewNullLogger()
	tenantA := uuid.New()
	tenantB := uuid.New()
	teA, _ := tenant.Create(tenantA, "GMS", 83, 1)
	teB, _ := tenant.Create(tenantB, "GMS", 83, 1)
	ctxA := tenant.WithContext(context.Background(), teA)
	ctxB := tenant.WithContext(context.Background(), teB)
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	pA := NewProcessor(l, ctxA, db)
	pB := NewProcessor(l, ctxB, db)

	if _, err := pA.RebuildForConversation(db.WithContext(ctxA))(1, uuid.New(), []conversation.StateModel{craftState(t, "x", "100", nil, nil, 0, 0, 0)}); err != nil {
		t.Fatalf("seed A: %v", err)
	}
	if _, err := pB.RebuildForConversation(db.WithContext(ctxB))(2, uuid.New(), []conversation.StateModel{craftState(t, "y", "200", nil, nil, 0, 0, 0)}); err != nil {
		t.Fatalf("seed B: %v", err)
	}

	count, err := pA.ClearForTenant(db.WithContext(ctxA))
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if count != 1 {
		t.Errorf("ClearForTenant count = %d, want 1", count)
	}
	got, _ := getAllForTenant(db.WithContext(ctxB))
	if len(got) != 1 {
		t.Errorf("tenant B rows wrongly affected: %d remaining", len(got))
	}
}
