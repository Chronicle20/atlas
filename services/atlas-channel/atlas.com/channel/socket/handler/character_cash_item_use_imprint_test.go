package handler

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testImprintTenant(t *testing.T, region string, major uint16, minor uint16) tenant.Model {
	t.Helper()
	te, err := tenant.Create(uuid.New(), region, major, minor)
	if err != nil {
		t.Fatal(err)
	}
	return te
}

// The classifier's 540 arm had an exact duplicate of the 5401 branch (dead code
// at lines 1132-1138 before task-227). Assert the arm the client actually
// implements, per derivation.md §3.
func TestCharacterImprintClassifierMatchesTheClient(t *testing.T) {
	cases := []struct {
		name   string
		tenant tenant.Model
		itemId item.Id
		want   CashSlotItemType
	}{
		{name: "5400 pre-v95", tenant: testImprintTenant(t, "GMS", 83, 1), itemId: 5400000, want: CashSlotItemType(52)},
		{name: "5400 v95", tenant: testImprintTenant(t, "GMS", 95, 1), itemId: 5400000, want: CashSlotItemType(53)},
		{name: "5401 pre-v95", tenant: testImprintTenant(t, "GMS", 83, 1), itemId: 5401000, want: CashSlotItemType(53)},
		{name: "5401 v95", tenant: testImprintTenant(t, "GMS", 95, 1), itemId: 5401000, want: CashSlotItemType(54)},
		{name: "unknown 540 prefix", tenant: testImprintTenant(t, "GMS", 83, 1), itemId: 5409000, want: CashSlotItemType(0)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := GetCashSlotItemType(tc.tenant)(tc.itemId); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

// The two sub-flow helpers must never collide on one version -- a collision
// routes a world transfer into the rename flow.
func TestImprintSubFlowTypesNeverCollide(t *testing.T) {
	for _, tn := range []tenant.Model{
		testImprintTenant(t, "GMS", 83, 1), testImprintTenant(t, "GMS", 95, 1), testImprintTenant(t, "JMS", 185, 1),
	} {
		if nameChangeCashSlotItemType(tn) == worldTransferCashSlotItemType(tn) {
			t.Fatalf("collision on %s v%d", tn.Region(), tn.MajorVersion())
		}
	}
}
