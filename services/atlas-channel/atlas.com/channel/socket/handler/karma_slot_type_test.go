package handler

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// testTenantVariants returns one {name, tenant} per configured tenant version
// this package's tests already exercise (see e.g.
// TestCurrencySackTypeIsNineteenOnEveryVersion in
// character_cash_item_use_meso_sack_test.go, and libs/atlas-packet/test's
// Variants): gms v48/61/72/79/83/84/87/92/95 and jms v185. Reused by both
// tests in this file so the canonical version set is defined once.
func testTenantVariants(t *testing.T) []struct {
	name   string
	tenant tenant.Model
} {
	t.Helper()
	versions := []struct {
		name   string
		region string
		major  uint16
		minor  uint16
	}{
		{"GMS v48", "GMS", 48, 1},
		{"GMS v61", "GMS", 61, 1},
		{"GMS v72", "GMS", 72, 1},
		{"GMS v79", "GMS", 79, 1},
		{"GMS v83", "GMS", 83, 1},
		{"GMS v84", "GMS", 84, 1},
		{"GMS v87", "GMS", 87, 1},
		{"GMS v92", "GMS", 92, 1},
		{"GMS v95", "GMS", 95, 1},
		{"JMS v185", "JMS", 185, 1},
	}
	out := make([]struct {
		name   string
		tenant tenant.Model
	}, 0, len(versions))
	for _, v := range versions {
		out = append(out, struct {
			name   string
			tenant tenant.Model
		}{name: v.name, tenant: mustTenant(t, v.region, v.major, v.minor)})
	}
	return out
}

// TestKarmaAndSealResolversAreDisjoint is the FR-2.4 regression guard, and it is
// not ceremony: pre-95, CashSlotItemTypeSealTimed is 64 and so is the GMS >= 95
// karma type. The two arms are disjoint today ONLY because the seal arm
// recomputes itself to 65 at GMS >= 95. A version-scoped resolver on each side
// makes the disjointness structural; this test is what keeps it that way.
func TestKarmaAndSealResolversAreDisjoint(t *testing.T) {
	for _, v := range testTenantVariants(t) {
		t.Run(v.name, func(t *testing.T) {
			karma := karmaScissorsCashSlotItemType(v.tenant)
			seal := sealTimedCashSlotItemType(v.tenant)
			if karma == seal {
				t.Fatalf("karma and seal cash-slot types collide at %s: both %d", v.name, karma)
			}
		})
	}
}

// TestGetCashSlotItemTypeFor552Unchanged: rewriting the bare `category == 552`
// branch to use the named constant must not change a single returned value.
func TestGetCashSlotItemTypeFor552Unchanged(t *testing.T) {
	for _, v := range testTenantVariants(t) {
		t.Run(v.name, func(t *testing.T) {
			want := CashSlotItemType(63)
			if v.tenant.Region() == "GMS" && v.tenant.MajorVersion() >= 95 {
				want = CashSlotItemType(64)
			}
			if got := GetCashSlotItemType(v.tenant)(item.Id(5520000)); got != want {
				t.Fatalf("GetCashSlotItemType(5520000) = %d, want %d", got, want)
			}
			if got := GetCashSlotItemType(v.tenant)(item.Id(5520001)); got != want {
				t.Fatalf("GetCashSlotItemType(5520001) = %d, want %d", got, want)
			}
		})
	}
}
