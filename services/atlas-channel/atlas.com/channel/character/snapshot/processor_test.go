package snapshot

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/inventory"
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"

	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type fetchCounts struct{ core, inv, skills, buffs int }

// installFetchSeams wires happy-path fakes and returns a counter. Callers
// override individual seams after this for failure cases.
func installFetchSeams(t *testing.T, characterId uint32) *fetchCounts {
	t.Helper()
	counts := &fetchCounts{}
	prevCore, prevInv, prevSkills, prevBuffs := coreFetchFn, inventoryFetchFn, skillsFetchFn, buffsFetchFn
	coreFetchFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (character.Model, error) {
		counts.core++
		return testCore(t, id), nil
	}
	inventoryFetchFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (inventory.Model, error) {
		counts.inv++
		inv, _, _ := testInventory(t, id)
		return inv, nil
	}
	skillsFetchFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) ([]skill.Model, error) {
		counts.skills++
		return []skill.Model{skill.NewModelBuilder(skillconst.Id(3121004)).SetLevel(10).MustBuild()}, nil
	}
	buffsFetchFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) ([]buff.Model, error) {
		counts.buffs++
		return []buff.Model{}, nil
	}
	t.Cleanup(func() {
		coreFetchFn, inventoryFetchFn, skillsFetchFn, buffsFetchFn = prevCore, prevInv, prevSkills, prevBuffs
	})
	return counts
}

// restoreFetchSeams registers a cleanup that restores today's package-level
// fetch seams to their pre-test values. Use in tests that assign the seams
// directly (rather than via installFetchSeams) so a fetch-failure fake
// never leaks into a later test.
func restoreFetchSeams(t *testing.T) {
	t.Helper()
	prevCore, prevInv, prevSkills, prevBuffs := coreFetchFn, inventoryFetchFn, skillsFetchFn, buffsFetchFn
	t.Cleanup(func() {
		coreFetchFn, inventoryFetchFn, skillsFetchFn, buffsFetchFn = prevCore, prevInv, prevSkills, prevBuffs
	})
}

func newTestProcessor(t *testing.T) (*Processor, tenant.Model) {
	t.Helper()
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	return NewProcessor(logrus.New(), ctx), tm
}

func TestProcessor_FirstGetPopulatesLazily_SecondGetIsZeroRest(t *testing.T) {
	resetRegistryForTest(t)
	p, _ := newTestProcessor(t)
	counts := installFetchSeams(t, 7)

	m, err := p.Get(7)
	if err != nil {
		t.Fatalf("first get: %v", err)
	}
	if m.Id() != 7 || len(m.Skills()) != 1 || len(m.Inventory().Consumable().Assets()) != 1 {
		t.Fatalf("first get returned undecorated model: %+v", m)
	}
	if counts.core != 1 || counts.inv != 1 || counts.skills != 1 {
		t.Fatalf("first get must fetch each component once: %+v", counts)
	}

	if _, err = p.Get(7); err != nil {
		t.Fatalf("second get: %v", err)
	}
	if counts.core != 1 || counts.inv != 1 || counts.skills != 1 {
		t.Fatalf("second get performed REST: %+v", counts)
	}
}

func TestProcessor_ComposedMatchesDecoratorPath(t *testing.T) {
	// FR-4.6 seed: the snapshot-composed model must equal the model built by
	// today's decorator chain for the same inputs.
	resetRegistryForTest(t)
	p, _ := newTestProcessor(t)
	installFetchSeams(t, 7)

	got, err := p.Get(7)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	inv, _, _ := testInventory(t, 7)
	want := testCore(t, 7).
		SetInventory(inv).
		SetSkills([]skill.Model{skill.NewModelBuilder(skillconst.Id(3121004)).SetLevel(10).MustBuild()})

	if got.Level() != want.Level() || got.JobId() != want.JobId() ||
		got.X() != want.X() || got.Y() != want.Y() {
		t.Fatalf("core mismatch: got %+v want %+v", got, want)
	}
	gw, gok := got.Equipment().Get("weapon")
	ww, wok := want.Equipment().Get("weapon")
	if gok != wok || (gok && gw.Equipable != nil) != (wok && ww.Equipable != nil) {
		t.Fatalf("equipment derivation mismatch")
	}
	if len(got.Skills()) != len(want.Skills()) || got.Skills()[0].Level() != want.Skills()[0].Level() {
		t.Fatalf("skills mismatch")
	}
	ga, wa := got.Inventory().Consumable().Assets(), want.Inventory().Consumable().Assets()
	if len(ga) != len(wa) || ga[0].TemplateId() != wa[0].TemplateId() || ga[0].Quantity() != wa[0].Quantity() || ga[0].Slot() != wa[0].Slot() {
		t.Fatalf("consumable assets mismatch: %+v vs %+v", ga, wa)
	}
}

func TestProcessor_PerComponentFallbackOnlyRefetchesInvalidComponent(t *testing.T) {
	resetRegistryForTest(t)
	p, tm := newTestProcessor(t)
	counts := installFetchSeams(t, 7)

	if _, err := p.Get(7); err != nil {
		t.Fatalf("populate: %v", err)
	}
	GetRegistry().InvalidateSkills(tm, 7)

	if _, err := p.Get(7); err != nil {
		t.Fatalf("refetch: %v", err)
	}
	if counts.skills != 2 {
		t.Fatalf("skills must refetch after invalidation: %+v", counts)
	}
	if counts.core != 1 || counts.inv != 1 {
		t.Fatalf("valid components must NOT refetch: %+v", counts)
	}
}

func TestProcessor_FallbackFailureSurfacesError(t *testing.T) {
	// FR-3.4 (core only): a core REST fallback failure surfaces exactly as
	// today's error path (character.ProcessorImpl.GetById propagates the
	// requests.Provider error) — the snapshot never converts it into
	// stale-success.
	resetRegistryForTest(t)
	p, tm := newTestProcessor(t)
	counts := installFetchSeams(t, 7)

	if _, err := p.Get(7); err != nil {
		t.Fatalf("populate: %v", err)
	}
	GetRegistry().InvalidateCore(tm, 7)

	wantErr := errors.New("core service down")
	coreFetchFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (character.Model, error) {
		return character.Model{}, wantErr
	}
	if _, err := p.Get(7); !errors.Is(err, wantErr) {
		t.Fatalf("core fallback failure must propagate, got %v", err)
	}
	_ = counts
}

func TestProcessor_InventoryFallbackFailureServesDegradedModel(t *testing.T) {
	// Controller ruling R14: matches today's character.ProcessorImpl.
	// InventoryDecorator, which swallows a fetch error and returns the model
	// unchanged rather than failing the whole read. Failing on the very
	// first read (nothing ever populated) isolates the degraded shape from
	// any stale-but-valid prior value.
	resetRegistryForTest(t)
	p, _ := newTestProcessor(t)
	restoreFetchSeams(t)

	wantErr := errors.New("inventory service down")
	coreFetchFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (character.Model, error) {
		return testCore(t, id), nil
	}
	inventoryFetchFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (inventory.Model, error) {
		return inventory.Model{}, wantErr
	}
	skillsFetchFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) ([]skill.Model, error) {
		return []skill.Model{skill.NewModelBuilder(skillconst.Id(3121004)).SetLevel(10).MustBuild()}, nil
	}

	m, err := p.Get(7)
	if err != nil {
		t.Fatalf("inventory fallback failure must not error, got %v", err)
	}
	if len(m.Inventory().Consumable().Assets()) != 0 {
		t.Fatalf("degraded model must not carry the failed inventory: %+v", m.Inventory())
	}
	if m.Id() != 7 || len(m.Skills()) != 1 {
		t.Fatalf("degraded model must still carry the composed core and skills: %+v", m)
	}
}

func TestProcessor_SkillsFallbackFailureServesDegradedModel(t *testing.T) {
	// Controller ruling R14: matches today's character.ProcessorImpl.
	// SkillModelDecorator, which swallows a fetch error and returns the
	// model unchanged rather than failing the whole read. Failing on the
	// very first read (nothing ever populated) isolates the degraded shape
	// from any stale-but-valid prior value.
	resetRegistryForTest(t)
	p, _ := newTestProcessor(t)
	restoreFetchSeams(t)

	wantErr := errors.New("skills service down")
	coreFetchFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (character.Model, error) {
		return testCore(t, id), nil
	}
	inventoryFetchFn = func(_ logrus.FieldLogger, _ context.Context, id uint32) (inventory.Model, error) {
		inv, _, _ := testInventory(t, id)
		return inv, nil
	}
	skillsFetchFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) ([]skill.Model, error) {
		return nil, wantErr
	}

	m, err := p.Get(7)
	if err != nil {
		t.Fatalf("skills fallback failure must not error, got %v", err)
	}
	if len(m.Skills()) != 0 {
		t.Fatalf("degraded model must not carry the failed skills: %+v", m.Skills())
	}
	if m.Id() != 7 || len(m.Inventory().Consumable().Assets()) != 1 {
		t.Fatalf("degraded model must still carry the composed core and inventory: %+v", m)
	}
}

func TestProcessor_StaleBackfillStillServesThisCaller(t *testing.T) {
	// Design §4.4: if an event bumps the gen mid-fetch, the backfill is
	// discarded but the fetched value is returned to THIS caller (it is
	// exactly what REST would have returned today).
	resetRegistryForTest(t)
	p, tm := newTestProcessor(t)
	installFetchSeams(t, 7)
	if _, err := p.Get(7); err != nil {
		t.Fatalf("populate: %v", err)
	}
	GetRegistry().InvalidateSkills(tm, 7)

	prev := skillsFetchFn
	skillsFetchFn = func(l logrus.FieldLogger, ctx context.Context, id uint32) ([]skill.Model, error) {
		// Concurrent event arrives while the fetch is in flight.
		GetRegistry().InvalidateSkills(tm, 7)
		return prev(l, ctx, id)
	}
	m, err := p.Get(7)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(m.Skills()) != 1 {
		t.Fatalf("caller must still receive the fetched skills: %+v", m.Skills())
	}
	if v := GetRegistry().View(tm, 7); v.SkillsValid {
		t.Fatalf("discarded backfill must leave the component invalid for the next read")
	}
}

func TestProcessor_PositionOverlayServesLocalFold(t *testing.T) {
	resetRegistryForTest(t)
	p, tm := newTestProcessor(t)
	installFetchSeams(t, 7)
	if _, err := p.Get(7); err != nil {
		t.Fatalf("populate: %v", err)
	}
	GetRegistry().SetPosition(tm, 7, 555, -666)
	m, err := p.Get(7)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if m.X() != 555 || m.Y() != -666 {
		t.Fatalf("position overlay not served: %d/%d", m.X(), m.Y())
	}
}

func TestProcessor_GetBuffs_LazySeedThenEventMaintained(t *testing.T) {
	resetRegistryForTest(t)
	p, tm := newTestProcessor(t)
	counts := installFetchSeams(t, 7)

	bs, err := p.GetBuffs(7)
	if err != nil {
		t.Fatalf("first buffs: %v", err)
	}
	if len(bs) != 0 || counts.buffs != 1 {
		t.Fatalf("first buffs read must seed via REST: %+v %+v", bs, counts)
	}

	b := buff.NewBuff(3111004, 20, 60_000, nil, timeNowForTest(), timeNowForTest().Add(60_000_000_000), false)
	GetRegistry().UpsertBuff(tm, 7, b)
	bs, err = p.GetBuffs(7)
	if err != nil {
		t.Fatalf("second buffs: %v", err)
	}
	if len(bs) != 1 || counts.buffs != 1 {
		t.Fatalf("second buffs read must be event-served, zero REST: %+v %+v", bs, counts)
	}
}

func TestProcessor_GetBuffs_SelfExpiresPastExpiresAt(t *testing.T) {
	// event-coverage.md §5: a lost EXPIRED event degrades to at most the
	// buff's natural duration — reads self-filter expired entries.
	resetRegistryForTest(t)
	p, tm := newTestProcessor(t)
	installFetchSeams(t, 7)
	if _, err := p.GetBuffs(7); err != nil {
		t.Fatalf("seed: %v", err)
	}
	expired := buff.NewBuff(3111004, 20, 1, nil, timeNowForTest().Add(-2_000_000_000), timeNowForTest().Add(-1_000_000_000), false)
	GetRegistry().UpsertBuff(tm, 7, expired)
	bs, err := p.GetBuffs(7)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(bs) != 0 {
		t.Fatalf("expired buffs must be filtered: %+v", bs)
	}
}
