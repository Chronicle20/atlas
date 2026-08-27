package snapshot

import (
	"sync"
	"testing"
	"time"

	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/compartment"
	"atlas-channel/inventory"

	invconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// newTestRegistry bypasses the singleton for test isolation.
func newTestRegistry() *Registry {
	return &Registry{perTenant: map[uuid.UUID]map[uint32]*entry{}}
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func testCore(t *testing.T, id uint32) character.Model {
	t.Helper()
	return character.NewModelBuilder().
		SetId(id).SetLevel(30).SetJobId(312).
		SetMp(500).SetMaxMp(800).SetX(100).SetY(-50).
		MustBuild()
}

func testInventory(t *testing.T, characterId uint32) (inventory.Model, uuid.UUID, asset.Model) {
	t.Helper()
	compId := uuid.New()
	a := asset.NewModelBuilder(9001, compId, 2060000).SetSlot(2).SetQuantity(400).MustBuild()
	comp := compartment.NewBuilder(compId, characterId, invconst.TypeValueUse, 96).AddAsset(a).MustBuild()
	inv := inventory.NewBuilder(characterId).
		SetEquipable(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueEquip, 96).MustBuild()).
		SetConsumable(comp).
		SetSetup(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueSetup, 96).MustBuild()).
		SetEtc(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueETC, 96).MustBuild()).
		SetCash(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueCash, 96).MustBuild()).
		MustBuild()
	return inv, compId, a
}

// populate drives a full backfill so the entry is valid for mutation tests.
func populate(t *testing.T, r *Registry, tm tenant.Model, characterId uint32) ComponentView {
	t.Helper()
	v := r.View(tm, characterId)
	if v.CoreValid || v.SkillsValid || v.InvValid || v.BuffsValid || v.PosValid {
		t.Fatalf("fresh entry must start all-invalid: %+v", v)
	}
	if !r.BackfillCore(tm, characterId, testCore(t, characterId), v.CoreGen) {
		t.Fatalf("core backfill rejected")
	}
	sk := skill.NewModelBuilder(skillconst.Id(3121004)).SetLevel(10).MustBuild()
	if !r.BackfillSkills(tm, characterId, []skill.Model{sk}, v.SkillsGen) {
		t.Fatalf("skills backfill rejected")
	}
	inv, _, _ := testInventory(t, characterId)
	if !r.BackfillInventory(tm, characterId, inv, v.InvGen) {
		t.Fatalf("inventory backfill rejected")
	}
	if !r.BackfillBuffs(tm, characterId, []buff.Model{}, v.BuffsGen) {
		t.Fatalf("buffs backfill rejected")
	}
	return r.View(tm, characterId)
}

func TestRegistry_MutatorsNoOpWhenEntryAbsent(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	r.SetLevel(tm, 7, 31)
	r.InvalidateCore(tm, 7)
	r.SetPosition(tm, 7, 1, 2)
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.perTenant) != 0 {
		t.Fatalf("mutators must never create entries")
	}
}

func TestRegistry_ViewCreatesAllInvalidEntryAndBackfillValidates(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	v := populate(t, r, tm, 7)
	if !v.CoreValid || !v.SkillsValid || !v.InvValid || !v.BuffsValid {
		t.Fatalf("all components must be valid after backfill: %+v", v)
	}
	if v.PosValid {
		t.Fatalf("position starts invalid until fed by movement")
	}
}

func TestRegistry_StaleBackfillDiscarded(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	v := r.View(tm, 7)
	// A concurrent invalidation (any mutation) bumps the gen…
	r.InvalidateCore(tm, 7)
	// …so the in-flight backfill recorded at v.CoreGen must be discarded.
	if r.BackfillCore(tm, 7, testCore(t, 7), v.CoreGen) {
		t.Fatalf("stale backfill must be discarded after a concurrent mutation")
	}
	after := r.View(tm, 7)
	if after.CoreValid {
		t.Fatalf("discarded backfill must leave the component invalid")
	}
}

func TestRegistry_ApplyStatChanged_CompleteValuesAppliesInPlace(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	// JSON round-trip delivers float64.
	r.ApplyStatChanged(tm, 7, []stat.Type{stat.TypeMp}, map[string]interface{}{"mp": float64(463)})
	v := r.View(tm, 7)
	if !v.CoreValid {
		t.Fatalf("complete values must not invalidate")
	}
	if v.Core.Mp() != 463 {
		t.Fatalf("mp not applied: %d", v.Core.Mp())
	}
	// Redelivery is idempotent (absolute value).
	r.ApplyStatChanged(tm, 7, []stat.Type{stat.TypeMp}, map[string]interface{}{"mp": float64(463)})
	if got := r.View(tm, 7); got.Core.Mp() != 463 {
		t.Fatalf("redelivery corrupted mp: %d", got.Core.Mp())
	}
}

func TestRegistry_ApplyStatChanged_MissingOrUnappliableValueInvalidates(t *testing.T) {
	cases := []struct {
		name    string
		updates []stat.Type
		values  map[string]interface{}
	}{
		{"nil values", []stat.Type{stat.TypeMp}, nil},
		{"missing key", []stat.Type{stat.TypeMp, stat.TypeHp}, map[string]interface{}{"mp": float64(1)}},
		{"non-numeric", []stat.Type{stat.TypeMp}, map[string]interface{}{"mp": "463"}},
		{"available_sp unappliable", []stat.Type{stat.TypeAvailableSP}, map[string]interface{}{"available_sp": float64(3)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestRegistry()
			tm := newTestTenant(t)
			populate(t, r, tm, 7)
			r.ApplyStatChanged(tm, 7, tc.updates, tc.values)
			if v := r.View(tm, 7); v.CoreValid {
				t.Fatalf("must invalidate core")
			}
		})
	}
}

func TestRegistry_ApplyStatChanged_EmptyUpdatesIsNoOp(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	before := populate(t, r, tm, 7)
	r.ApplyStatChanged(tm, 7, []stat.Type{}, nil)
	after := r.View(tm, 7)
	if !after.CoreValid || after.CoreGen != before.CoreGen {
		t.Fatalf("empty updates must not mutate: before=%+v after=%+v", before, after)
	}
}

func TestRegistry_ApplyStatChanged_PetSnIsSkipped(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	r.ApplyStatChanged(tm, 7, []stat.Type{stat.TypePetSn1}, nil)
	if v := r.View(tm, 7); !v.CoreValid {
		t.Fatalf("PET_SN updates are not core fields and must be skipped, not invalidate")
	}
}

func TestRegistry_SetLevelAndExperience_ApplyWhenValid(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	r.SetLevel(tm, 7, 31)
	r.SetExperience(tm, 7, 123456)
	v := r.View(tm, 7)
	if v.Core.Level() != 31 || v.Core.Experience() != 123456 {
		t.Fatalf("level/exp not applied: %d/%d", v.Core.Level(), v.Core.Experience())
	}
}

func TestRegistry_InPlaceUpdateOnInvalidComponentOnlyBumpsGen(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	r.InvalidateCore(tm, 7)
	g := r.View(tm, 7).CoreGen
	r.SetLevel(tm, 7, 99)
	v := r.View(tm, 7)
	if v.CoreValid {
		t.Fatalf("in-place update must not validate a stale component")
	}
	if v.CoreGen == g {
		t.Fatalf("in-place update on invalid component must still bump gen (kills in-flight backfill)")
	}
}

func TestRegistry_SkillUpsertRemoveAndCooldownPreserved(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	up := skill.NewModelBuilder(skillconst.Id(3121004)).SetLevel(20).SetMasterLevel(30).MustBuild()
	r.UpsertSkill(tm, 7, up)
	v := r.View(tm, 7)
	if len(v.Skills) != 1 || v.Skills[0].Level() != 20 || v.Skills[0].MasterLevel() != 30 {
		t.Fatalf("upsert mismatch: %+v", v.Skills)
	}
	newSkill := skill.NewModelBuilder(skillconst.Id(3121002)).SetLevel(1).MustBuild()
	r.UpsertSkill(tm, 7, newSkill)
	if v = r.View(tm, 7); len(v.Skills) != 2 {
		t.Fatalf("insert mismatch: %+v", v.Skills)
	}
	r.RemoveSkill(tm, 7, skillconst.Id(3121004))
	if v = r.View(tm, 7); len(v.Skills) != 1 || v.Skills[0].Id() != skillconst.Id(3121002) {
		t.Fatalf("remove mismatch: %+v", v.Skills)
	}
}

func TestRegistry_AssetLifecycle(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	v := populate(t, r, tm, 7)
	compId := v.Inv.Consumable().Id()

	// Quantity absolute (idempotent).
	r.SetAssetQuantity(tm, 7, 9001, 380)
	r.SetAssetQuantity(tm, 7, 9001, 380)
	got := r.View(tm, 7)
	a, ok := got.Inv.Consumable().FindById(9001)
	if !ok || a.Quantity() != 380 {
		t.Fatalf("quantity not applied: %+v ok=%v", a, ok)
	}

	// Slot absolute by AssetId.
	r.SetAssetSlot(tm, 7, 9001, 5)
	got = r.View(tm, 7)
	a, _ = got.Inv.Consumable().FindById(9001)
	if a.Slot() != 5 {
		t.Fatalf("slot not applied: %d", a.Slot())
	}

	// Upsert (full replace by AssetId) + insert of a new asset.
	repl := asset.NewModelBuilder(9001, compId, 2061000).SetSlot(5).SetQuantity(111).MustBuild()
	r.UpsertAsset(tm, 7, compId, repl)
	ins := asset.NewModelBuilder(9002, compId, 2060001).SetSlot(6).SetQuantity(200).MustBuild()
	r.UpsertAsset(tm, 7, compId, ins)
	got = r.View(tm, 7)
	if len(got.Inv.Consumable().Assets()) != 2 {
		t.Fatalf("expected 2 assets: %+v", got.Inv.Consumable().Assets())
	}
	a, _ = got.Inv.Consumable().FindById(9001)
	if a.TemplateId() != 2061000 || a.Quantity() != 111 {
		t.Fatalf("upsert replace mismatch: %+v", a)
	}

	// Upsert into an unknown compartment invalidates instead of guessing.
	r.UpsertAsset(tm, 7, uuid.New(), ins)
	if got = r.View(tm, 7); got.InvValid {
		t.Fatalf("unknown compartment must invalidate inventory")
	}

	// Refill, then delete.
	inv2, _, _ := testInventory(t, 7)
	if !r.BackfillInventory(tm, 7, inv2, got.InvGen) {
		t.Fatalf("re-backfill rejected")
	}
	r.RemoveAsset(tm, 7, 9001)
	got = r.View(tm, 7)
	if _, ok = got.Inv.Consumable().FindById(9001); ok {
		t.Fatalf("delete must remove the asset")
	}
}

func TestRegistry_AssetMutationDoesNotAliasPriorReads(t *testing.T) {
	// A composed/read model handed out earlier must not observe later
	// mutations (value-copy semantics; guards the shared-map/slice hazard in
	// inventory.CloneModel / compartment.CloneModel).
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	before := r.View(tm, 7)
	beforeQty := before.Inv.Consumable().Assets()[0].Quantity()
	r.SetAssetQuantity(tm, 7, 9001, 42)
	if got := before.Inv.Consumable().Assets()[0].Quantity(); got != beforeQty {
		t.Fatalf("prior read mutated in place: %d -> %d", beforeQty, got)
	}
}

func TestRegistry_BuffUpsertRemoveBySourceId(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	b := buff.NewBuff(3111004, 20, 60_000, nil, timeNowForTest(), timeNowForTest().Add(60_000_000_000), false)
	r.UpsertBuff(tm, 7, b)
	r.UpsertBuff(tm, 7, b) // redelivery: replace, not duplicate
	v := r.View(tm, 7)
	if len(v.Buffs) != 1 {
		t.Fatalf("upsert must replace by sourceId: %+v", v.Buffs)
	}
	r.RemoveBuff(tm, 7, 3111004)
	if v = r.View(tm, 7); len(v.Buffs) != 0 {
		t.Fatalf("remove mismatch: %+v", v.Buffs)
	}
}

func TestRegistry_PositionOverlayAndComposedCache(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	if _, ok := r.ComposedIfValid(tm, 7); !ok {
		t.Fatalf("all-valid entry must compose")
	}
	r.SetPosition(tm, 7, 333, -444)
	m, ok := r.ComposedIfValid(tm, 7)
	if !ok {
		t.Fatalf("compose after position update")
	}
	if m.X() != 333 || m.Y() != -444 {
		t.Fatalf("position overlay not applied: %d/%d", m.X(), m.Y())
	}
	// Composition preserves the decorated shape: inventory + skills present.
	if len(m.Skills()) != 1 || len(m.Inventory().Consumable().Assets()) != 1 {
		t.Fatalf("composed model missing decorations")
	}
	r.InvalidatePosition(tm, 7)
	m, ok = r.ComposedIfValid(tm, 7)
	if !ok {
		t.Fatalf("pos-invalid entry still composes from core X/Y")
	}
	if m.X() != 100 || m.Y() != -50 {
		t.Fatalf("expected core REST X/Y when position invalid: %d/%d", m.X(), m.Y())
	}
}

func TestRegistry_ComposedNotServedWhenComponentInvalid(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	populate(t, r, tm, 7)
	r.InvalidateInventory(tm, 7)
	if _, ok := r.ComposedIfValid(tm, 7); ok {
		t.Fatalf("must not serve composed model over an invalid component")
	}
}

func TestRegistry_EvictAndTenantIsolation(t *testing.T) {
	r := newTestRegistry()
	t1 := newTestTenant(t)
	t2 := newTestTenant(t)
	populate(t, r, t1, 7)
	populate(t, r, t2, 7)
	r.Evict(t1, 7)
	if _, ok := r.ComposedIfValid(t1, 7); ok {
		t.Fatalf("evicted entry must be gone")
	}
	if _, ok := r.ComposedIfValid(t2, 7); !ok {
		t.Fatalf("t2 must survive t1 evict")
	}
	r.EvictTenant(t2.Id())
	if _, ok := r.ComposedIfValid(t2, 7); ok {
		t.Fatalf("tenant evict must drop entries")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := newTestRegistry()
	tm := newTestTenant(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				id := uint32(j%5 + 1)
				v := r.View(tm, id)
				_ = r.BackfillCore(tm, id, testCore(t, id), v.CoreGen)
				r.ApplyStatChanged(tm, id, []stat.Type{stat.TypeMp}, map[string]interface{}{"mp": float64(j)})
				r.SetPosition(tm, id, int16(j), int16(-j))
				_, _ = r.ComposedIfValid(tm, id)
				if j%50 == 0 {
					r.Evict(tm, id)
				}
			}
		}(i)
	}
	wg.Wait()
}

func timeNowForTest() time.Time { return time.Now() }
