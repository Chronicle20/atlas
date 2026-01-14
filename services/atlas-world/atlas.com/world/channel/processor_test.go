package channel_test

import (
	"atlas-world/channel"
	"atlas-world/test"
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func setupProcessor(t *testing.T) (channel.Processor, tenant.Model, func()) {
	t.Helper()
	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	processor := channel.NewProcessor(logger, ctx)
	tenant := test.CreateMockTenant(tenantId)

	cleanup := func() {
		// Clean up registered channels for this tenant
		servers := channel.GetChannelRegistry().ChannelServers(tenant)
		for _, s := range servers {
			_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, s.WorldId(), s.ChannelId())
		}
	}

	return processor, tenant, cleanup
}

func TestNewProcessor(t *testing.T) {
	processor, _, cleanup := setupProcessor(t)
	defer cleanup()

	if processor == nil {
		t.Fatal("Expected processor to be initialized")
	}
}

func TestAllProvider_Empty(t *testing.T) {
	processor, _, cleanup := setupProcessor(t)
	defer cleanup()

	channels, err := processor.AllProvider()()
	if err != nil {
		t.Fatalf("AllProvider() unexpected error: %v", err)
	}
	if len(channels) != 0 {
		t.Errorf("len(channels) = %d, want 0", len(channels))
	}
}

func TestAllProvider_WithChannels(t *testing.T) {
	processor, _, cleanup := setupProcessor(t)
	defer cleanup()

	// Register some channels
	_, _ = processor.Register(1, 0, "192.168.1.1", 8080, 0, 100)
	_, _ = processor.Register(1, 1, "192.168.1.2", 8081, 0, 100)
	_, _ = processor.Register(2, 0, "192.168.1.3", 8082, 0, 100)

	channels, err := processor.AllProvider()()
	if err != nil {
		t.Fatalf("AllProvider() unexpected error: %v", err)
	}
	if len(channels) != 3 {
		t.Errorf("len(channels) = %d, want 3", len(channels))
	}
}

func TestGetByWorld(t *testing.T) {
	processor, _, cleanup := setupProcessor(t)
	defer cleanup()

	// Register channels in different worlds
	_, _ = processor.Register(1, 0, "192.168.1.1", 8080, 0, 100)
	_, _ = processor.Register(1, 1, "192.168.1.2", 8081, 0, 100)
	_, _ = processor.Register(2, 0, "192.168.1.3", 8082, 0, 100)

	// Get channels for world 1
	channels, err := processor.GetByWorld(1)
	if err != nil {
		t.Fatalf("GetByWorld() unexpected error: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("len(channels) = %d, want 2", len(channels))
	}

	for _, ch := range channels {
		if ch.WorldId() != 1 {
			t.Errorf("channel.WorldId() = %d, want 1", ch.WorldId())
		}
	}
}

func TestGetByWorld_Empty(t *testing.T) {
	processor, _, cleanup := setupProcessor(t)
	defer cleanup()

	// Register channels in world 1 only
	_, _ = processor.Register(1, 0, "192.168.1.1", 8080, 0, 100)

	// Get channels for world 2 (none)
	channels, err := processor.GetByWorld(2)
	if err != nil {
		t.Fatalf("GetByWorld() unexpected error: %v", err)
	}
	if len(channels) != 0 {
		t.Errorf("len(channels) = %d, want 0", len(channels))
	}
}

func TestByWorldProvider(t *testing.T) {
	processor, _, cleanup := setupProcessor(t)
	defer cleanup()

	_, _ = processor.Register(1, 0, "192.168.1.1", 8080, 0, 100)
	_, _ = processor.Register(2, 0, "192.168.1.2", 8081, 0, 100)

	provider := processor.ByWorldProvider(1)
	channels, err := provider()
	if err != nil {
		t.Fatalf("ByWorldProvider() unexpected error: %v", err)
	}
	if len(channels) != 1 {
		t.Errorf("len(channels) = %d, want 1", len(channels))
	}
}

func TestGetById(t *testing.T) {
	processor, _, cleanup := setupProcessor(t)
	defer cleanup()

	registered, _ := processor.Register(1, 2, "192.168.1.1", 8080, 50, 100)

	ch, err := processor.GetById(1, 2)
	if err != nil {
		t.Fatalf("GetById() unexpected error: %v", err)
	}
	if ch.Id() != registered.Id() {
		t.Errorf("GetById() returned different channel")
	}
	if ch.WorldId() != 1 {
		t.Errorf("ch.WorldId() = %d, want 1", ch.WorldId())
	}
	if ch.ChannelId() != 2 {
		t.Errorf("ch.ChannelId() = %d, want 2", ch.ChannelId())
	}
}

func TestGetById_NotFound(t *testing.T) {
	processor, _, cleanup := setupProcessor(t)
	defer cleanup()

	_, err := processor.GetById(99, 99)
	if err == nil {
		t.Error("GetById() expected error for non-existent channel")
	}
}

func TestByIdProvider(t *testing.T) {
	processor, _, cleanup := setupProcessor(t)
	defer cleanup()

	registered, _ := processor.Register(1, 0, "192.168.1.1", 8080, 0, 100)

	provider := processor.ByIdProvider(1, 0)
	ch, err := provider()
	if err != nil {
		t.Fatalf("ByIdProvider() unexpected error: %v", err)
	}
	if ch.Id() != registered.Id() {
		t.Errorf("ByIdProvider() returned different channel")
	}
}

func TestByIdProvider_NotFound(t *testing.T) {
	processor, _, cleanup := setupProcessor(t)
	defer cleanup()

	provider := processor.ByIdProvider(99, 99)
	_, err := provider()
	if err == nil {
		t.Error("ByIdProvider() expected error for non-existent channel")
	}
}

func TestProcessor_Register(t *testing.T) {
	processor, _, cleanup := setupProcessor(t)
	defer cleanup()

	ch, err := processor.Register(1, 0, "192.168.1.1", 8080, 50, 100)
	if err != nil {
		t.Fatalf("Register() unexpected error: %v", err)
	}
	if ch.WorldId() != 1 {
		t.Errorf("ch.WorldId() = %d, want 1", ch.WorldId())
	}
	if ch.ChannelId() != 0 {
		t.Errorf("ch.ChannelId() = %d, want 0", ch.ChannelId())
	}
	if ch.IpAddress() != "192.168.1.1" {
		t.Errorf("ch.IpAddress() = %s, want 192.168.1.1", ch.IpAddress())
	}
	if ch.Port() != 8080 {
		t.Errorf("ch.Port() = %d, want 8080", ch.Port())
	}
	if ch.CurrentCapacity() != 50 {
		t.Errorf("ch.CurrentCapacity() = %d, want 50", ch.CurrentCapacity())
	}
	if ch.MaxCapacity() != 100 {
		t.Errorf("ch.MaxCapacity() = %d, want 100", ch.MaxCapacity())
	}
	if ch.Id() == uuid.Nil {
		t.Error("ch.Id() should not be zero UUID")
	}
}

func TestProcessor_Register_InvalidInput(t *testing.T) {
	processor, _, cleanup := setupProcessor(t)
	defer cleanup()

	// Invalid port
	_, err := processor.Register(1, 0, "192.168.1.1", 0, 0, 100)
	if err == nil {
		t.Error("Register() expected error for invalid port")
	}

	// Empty IP
	_, err = processor.Register(1, 0, "", 8080, 0, 100)
	if err == nil {
		t.Error("Register() expected error for empty IP address")
	}

	// Zero max capacity
	_, err = processor.Register(1, 0, "192.168.1.1", 8080, 0, 0)
	if err == nil {
		t.Error("Register() expected error for zero max capacity")
	}
}

func TestUnregister(t *testing.T) {
	processor, _, cleanup := setupProcessor(t)
	defer cleanup()

	_, _ = processor.Register(1, 0, "192.168.1.1", 8080, 0, 100)

	err := processor.Unregister(1, 0)
	if err != nil {
		t.Fatalf("Unregister() unexpected error: %v", err)
	}

	// Verify channel is unregistered
	_, err = processor.GetById(1, 0)
	if err == nil {
		t.Error("Channel should have been unregistered")
	}
}

func TestUnregister_NotFound(t *testing.T) {
	processor, _, cleanup := setupProcessor(t)
	defer cleanup()

	err := processor.Unregister(99, 99)
	if err == nil {
		t.Error("Unregister() expected error for non-existent channel")
	}
}

func TestByWorldFilter(t *testing.T) {
	filter := channel.ByWorldFilter(5)

	// Create a channel for world 5
	ch1, _ := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetWorldId(5).
		SetIpAddress("192.168.1.1").
		SetPort(8080).
		SetMaxCapacity(100).
		Build()

	// Create a channel for world 3
	ch2, _ := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetWorldId(3).
		SetIpAddress("192.168.1.2").
		SetPort(8081).
		SetMaxCapacity(100).
		Build()

	if !filter(ch1) {
		t.Error("ByWorldFilter(5) should return true for channel in world 5")
	}
	if filter(ch2) {
		t.Error("ByWorldFilter(5) should return false for channel in world 3")
	}
}

func TestProcessorTenantIsolation(t *testing.T) {
	logger, _ := logtest.NewNullLogger()

	// Create two different tenants
	tenant1Id := uuid.New()
	tenant2Id := uuid.New()
	ctx1 := test.CreateTestContextWithTenant(tenant1Id)
	ctx2 := test.CreateTestContextWithTenant(tenant2Id)

	processor1 := channel.NewProcessor(logger, ctx1)
	processor2 := channel.NewProcessor(logger, ctx2)

	// Register channel with tenant 1
	ch1, _ := processor1.Register(1, 0, "192.168.1.1", 8080, 0, 100)

	// Register channel with tenant 2
	ch2, _ := processor2.Register(1, 0, "192.168.1.2", 8081, 0, 100)

	// Verify tenant 1 only sees their channel
	channels1, _ := processor1.AllProvider()()
	if len(channels1) != 1 {
		t.Errorf("Tenant 1 should have 1 channel, got %d", len(channels1))
	}
	if len(channels1) > 0 && channels1[0].Id() != ch1.Id() {
		t.Error("Tenant 1 should see their own channel")
	}

	// Verify tenant 2 only sees their channel
	channels2, _ := processor2.AllProvider()()
	if len(channels2) != 1 {
		t.Errorf("Tenant 2 should have 1 channel, got %d", len(channels2))
	}
	if len(channels2) > 0 && channels2[0].Id() != ch2.Id() {
		t.Error("Tenant 2 should see their own channel")
	}

	// Cleanup
	tenant1 := test.CreateMockTenant(tenant1Id)
	tenant2 := test.CreateMockTenant(tenant2Id)
	_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant1, 1, 0)
	_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant2, 1, 0)
}

func TestNewProcessor_ExtractsTenantFromContext(t *testing.T) {
	logger, _ := logtest.NewNullLogger()

	// Test that NewProcessor works with a valid context
	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)

	processor := channel.NewProcessor(logger, ctx)
	if processor == nil {
		t.Fatal("NewProcessor should return a valid processor")
	}

	// Register and verify it works
	ch, err := processor.Register(1, 0, "192.168.1.1", 8080, 0, 100)
	if err != nil {
		t.Fatalf("Register() unexpected error: %v", err)
	}

	// Cleanup
	tenant := test.CreateMockTenant(tenantId)
	_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, ch.WorldId(), ch.ChannelId())
}

func TestNewProcessor_PanicsWithoutTenant(t *testing.T) {
	logger, _ := logtest.NewNullLogger()

	defer func() {
		if r := recover(); r == nil {
			t.Error("NewProcessor should panic when context has no tenant")
		}
	}()

	// Create context without tenant
	ctx := context.Background()
	_ = channel.NewProcessor(logger, ctx)
}
