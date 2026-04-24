package shops_test

import (
	"atlas-npc/commodities"
	"atlas-npc/data/consumable"
	"atlas-npc/shops"
	"atlas-npc/test"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// newConsumable builds a consumable.Model through its JSON representation so
// tests do not need to depend on internal struct fields.
func newConsumable(t *testing.T, id uint32, slotMax uint32, unitPrice float64) consumable.Model {
	t.Helper()
	raw := fmt.Sprintf(`{"id":%d,"slotMax":%d,"unitPrice":%f}`, id, slotMax, unitPrice)
	var m consumable.Model
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("failed to build consumable.Model: %v", err)
	}
	return m
}

func newCommodity(t *testing.T, npcId, templateId uint32) commodities.Model {
	t.Helper()
	cm, err := commodities.NewBuilder().
		SetId(uuid.New()).
		SetNpcId(npcId).
		SetTemplateId(templateId).
		Build()
	if err != nil {
		t.Fatalf("failed to build commodity: %v", err)
	}
	return cm
}

// TestRechargeablePartition verifies that rechargeables always trail
// non-rechargeables in the commodity list, regardless of the Recharger flag.
// This protects the client from the CShopDlg::SetShopDlg uninitialized-quantity
// read when a rechargeable sits in slot 0.
func TestRechargeablePartition(t *testing.T) {
	mockTenant := test.CreateDefaultMockTenant()
	ctx := tenant.WithContext(context.Background(), mockTenant)
	db := test.SetupTestDB(t, commodities.Migration, shops.Migration)
	defer test.CleanupTestDB(t, db)

	processor := shops.NewProcessor(logrus.New(), ctx, db)
	p, ok := processor.(*shops.ProcessorImpl)
	if !ok {
		t.Fatalf("expected *shops.ProcessorImpl")
	}
	tenantId := mockTenant.Id()

	// Bullets (2330000) and subis (2070000) are rechargeable.
	bullet := newConsumable(t, 2330000, 200, 0.9)
	subi := newConsumable(t, 2070000, 400, 0.5)
	cache := &mockConsumableCache{
		consumables: map[uuid.UUID][]consumable.Model{
			tenantId: {bullet, subi},
		},
	}
	originalCache := shops.GetConsumableCache()
	shops.SetConsumableCacheForTesting(cache)
	defer shops.SetConsumableCacheForTesting(originalCache)

	t.Run("NonRechargerShopMovesRechargeablesToEnd", func(t *testing.T) {
		npcId := uint32(2070001)
		// Simulate the JSON seed shape: rechargeables in slots 0-1 (the bug),
		// followed by non-rechargeables. The decorator must move the
		// rechargeables to the end.
		cms := []commodities.Model{
			newCommodity(t, npcId, 2330000), // rechargeable
			newCommodity(t, npcId, 2070000), // rechargeable
			newCommodity(t, npcId, 2061001), // non-rechargeable
			newCommodity(t, npcId, 2060001), // non-rechargeable
			newCommodity(t, npcId, 2030000), // non-rechargeable
		}
		m, err := shops.NewBuilder(npcId).SetRecharger(false).SetCommodities(cms).Build()
		if err != nil {
			t.Fatalf("failed to build shop model: %v", err)
		}

		result := p.RechargeableConsumablesDecorator(m)
		got := templateIds(result.Commodities())
		want := []uint32{2061001, 2060001, 2030000, 2070000, 2330000}
		if !equalUint32Slices(got, want) {
			t.Errorf("unexpected ordering\n got=%v\nwant=%v", got, want)
		}
	})

	t.Run("RechargeablesSortedByTemplateId", func(t *testing.T) {
		npcId := uint32(2070002)
		cms := []commodities.Model{
			newCommodity(t, npcId, 2330000), // rechargeable, higher id
			newCommodity(t, npcId, 2070000), // rechargeable, lower id
			newCommodity(t, npcId, 2010000), // non-rechargeable
		}
		m, err := shops.NewBuilder(npcId).SetRecharger(false).SetCommodities(cms).Build()
		if err != nil {
			t.Fatalf("failed to build shop model: %v", err)
		}

		result := p.RechargeableConsumablesDecorator(m)
		got := templateIds(result.Commodities())
		want := []uint32{2010000, 2070000, 2330000}
		if !equalUint32Slices(got, want) {
			t.Errorf("unexpected ordering\n got=%v\nwant=%v", got, want)
		}
	})

	t.Run("NonRechargeableRelativeOrderPreserved", func(t *testing.T) {
		npcId := uint32(2070003)
		// No rechargeables in input; order should be unchanged.
		cms := []commodities.Model{
			newCommodity(t, npcId, 2010003),
			newCommodity(t, npcId, 2010001),
			newCommodity(t, npcId, 2010002),
			newCommodity(t, npcId, 2010000),
		}
		m, err := shops.NewBuilder(npcId).SetRecharger(false).SetCommodities(cms).Build()
		if err != nil {
			t.Fatalf("failed to build shop model: %v", err)
		}

		result := p.RechargeableConsumablesDecorator(m)
		got := templateIds(result.Commodities())
		want := []uint32{2010003, 2010001, 2010002, 2010000}
		if !equalUint32Slices(got, want) {
			t.Errorf("unexpected ordering\n got=%v\nwant=%v", got, want)
		}
	})

	t.Run("RechargerShopInjectsMissingCatalogAtEnd", func(t *testing.T) {
		npcId := uint32(2070004)
		// Only non-rechargeables stored; the recharger shop should gain both
		// catalog entries and they should land at the end, sorted by template.
		cms := []commodities.Model{
			newCommodity(t, npcId, 2010001),
			newCommodity(t, npcId, 2010002),
		}
		m, err := shops.NewBuilder(npcId).SetRecharger(true).SetCommodities(cms).Build()
		if err != nil {
			t.Fatalf("failed to build shop model: %v", err)
		}

		result := p.RechargeableConsumablesDecorator(m)
		got := templateIds(result.Commodities())
		want := []uint32{2010001, 2010002, 2070000, 2330000}
		if !equalUint32Slices(got, want) {
			t.Errorf("unexpected ordering\n got=%v\nwant=%v", got, want)
		}
	})
}

func templateIds(cms []commodities.Model) []uint32 {
	ids := make([]uint32, len(cms))
	for i, c := range cms {
		ids[i] = c.TemplateId()
	}
	return ids
}

func equalUint32Slices(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
