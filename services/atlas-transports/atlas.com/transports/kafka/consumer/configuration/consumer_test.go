package configuration

import (
	"context"
	"strings"
	"testing"

	configuration2 "atlas-transports/kafka/message/configuration"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// A header-less event resolves the zero tenant. The handler must log an
// ERROR and perform NO registry mutation — the registries are package
// singletons that were never initialised in this test, so any call into
// them would panic. A passing test therefore proves no ClearTenant ran.
func TestHandleConfigurationStatus_NilTenantSkipsReload(t *testing.T) {
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.ErrorLevel)

	zero, err := tenant.Create(uuid.Nil, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), zero)

	handleConfigurationStatus(l, ctx, configuration2.StatusEvent{
		Type:         "INSTANCE_ROUTE_UPDATED",
		ResourceType: "instance-route",
		ResourceId:   "temple-of-time-return-flight",
	})

	var logged bool
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.ErrorLevel && strings.Contains(e.Message, "without tenant headers") {
			logged = true
		}
	}
	if !logged {
		t.Fatalf("expected an ERROR naming the missing tenant headers; got %v", hook.AllEntries())
	}
}
