package tenant_test

import (
	"atlas-tenants/tenant"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv(string(tenant.EventTopicTenantStatus), string(tenant.EventTopicTenantStatus))
	os.Exit(m.Run())
}
