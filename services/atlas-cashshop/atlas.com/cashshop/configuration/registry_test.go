package configuration

import (
	"atlas-cashshop/configuration/tenant"
	"atlas-cashshop/configuration/tenant/cashshop"
	"atlas-cashshop/configuration/tenant/cashshop/surprise"
	"context"
	"testing"

	"github.com/google/uuid"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// An unconfigured tenant must still be able to open the stock box: the
// tenant-config fetch failing returns a zero RestModel (see GetTenantConfig),
// so the 5222000 default has to live in the accessor.
func TestGetSurpriseBoxTemplateIdsDefaultsTo5222000(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ids := GetSurpriseBoxTemplateIds(l, context.Background(), uuid.New())
	if len(ids) != 1 || ids[0] != 5222000 {
		t.Fatalf("ids = %v, want [5222000]", ids)
	}
}

func TestGetSurpriseBoxTemplateIdsUsesConfiguredList(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	tenantId := uuid.New()
	// Seed the memoized cache directly, the way the other registry tests do.
	mu.Lock()
	tenantConfig[tenantId] = tenant.RestModel{
		CashShop: cashshop.RestModel{
			Surprise: surprise.RestModel{BoxTemplateIds: []uint32{5222000, 5222002}},
		},
	}
	mu.Unlock()
	ids := GetSurpriseBoxTemplateIds(l, context.Background(), tenantId)
	if len(ids) != 2 || ids[0] != 5222000 || ids[1] != 5222002 {
		t.Fatalf("ids = %v, want [5222000 5222002]", ids)
	}
}
