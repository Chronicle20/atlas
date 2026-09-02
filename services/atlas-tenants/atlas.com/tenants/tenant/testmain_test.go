package tenant_test

import (
	"atlas-tenants/tenant"
	"os"
	"testing"
)

// resolvedTenantStatusTopic is deliberately different from the token name.
// Setting the token to its own value would mask exactly the defect this
// package regressed on: a token that is never resolved through the
// environment still "works" when the resolved name equals the token.
const resolvedTenantStatusTopic = "tenant-status-test"

func TestMain(m *testing.M) {
	_ = os.Setenv(string(tenant.EventTopicTenantStatus), resolvedTenantStatusTopic)
	os.Exit(m.Run())
}
