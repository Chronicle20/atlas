package channel_test

import (
	"atlas-world/channel"
	"atlas-world/test"
	"sync"
	"testing"
	"time"

	channelConstant "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

func createTestChannel(t *testing.T, worldId world.Id, channelId channelConstant.Id, ipAddress string, port int) channel.Model {
	t.Helper()
	m, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetWorldId(worldId).
		SetChannelId(channelId).
		SetIpAddress(ipAddress).
		SetPort(port).
		SetMaxCapacity(100).
		SetCreatedAt(time.Now()).
		Build()
	if err != nil {
		t.Fatalf("Failed to create test channel: %v", err)
	}
	return m
}

func createUniqueTenant(t *testing.T) tenant.Model {
	t.Helper()
	return test.CreateMockTenant(uuid.New())
}

func TestGetChannelRegistry_Singleton(t *testing.T) {
	reg1 := channel.GetChannelRegistry()
	reg2 := channel.GetChannelRegistry()

	if reg1 != reg2 {
		t.Error("GetChannelRegistry() should return the same instance")
	}
}

func TestRegister(t *testing.T) {
	registry := channel.GetChannelRegistry()
	tenant := createUniqueTenant(t)
	ch := createTestChannel(t, 1, 0, "192.168.1.1", 8080)

	result := registry.Register(tenant, ch)

	if result.Id() != ch.Id() {
		t.Errorf("Register() returned channel with different ID")
	}

	// Verify channel is in registry
	servers := registry.ChannelServers(tenant)
	found := false
	for _, s := range servers {
		if s.Id() == ch.Id() {
			found = true
			break
		}
	}
	if !found {
		t.Error("Registered channel not found in ChannelServers()")
	}
}

func TestRegister_UpdatesExisting(t *testing.T) {
	registry := channel.GetChannelRegistry()
	tenant := createUniqueTenant(t)

	ch1 := createTestChannel(t, 1, 0, "192.168.1.1", 8080)
	registry.Register(tenant, ch1)

	// Register another channel with same world/channel but different ID
	ch2 := createTestChannel(t, 1, 0, "192.168.1.2", 8081)
	registry.Register(tenant, ch2)

	// Should only have one channel for world 1, channel 0
	servers := registry.ChannelServers(tenant)
	count := 0
	for _, s := range servers {
		if s.WorldId() == 1 && s.ChannelId() == 0 {
			count++
			// Should have the second channel's values
			if s.Id() != ch2.Id() {
				t.Error("Registry should have updated to new channel")
			}
		}
	}
	if count != 1 {
		t.Errorf("Expected 1 channel for world 1 channel 0, got %d", count)
	}
}

func TestChannelServers_Empty(t *testing.T) {
	registry := channel.GetChannelRegistry()
	tenant := createUniqueTenant(t)

	servers := registry.ChannelServers(tenant)

	if len(servers) != 0 {
		t.Errorf("ChannelServers() for new tenant should be empty, got %d", len(servers))
	}
}

func TestChannelServers_MultipleChannels(t *testing.T) {
	registry := channel.GetChannelRegistry()
	tenant := createUniqueTenant(t)

	ch1 := createTestChannel(t, 1, 0, "192.168.1.1", 8080)
	ch2 := createTestChannel(t, 1, 1, "192.168.1.2", 8081)
	ch3 := createTestChannel(t, 2, 0, "192.168.1.3", 8082)

	registry.Register(tenant, ch1)
	registry.Register(tenant, ch2)
	registry.Register(tenant, ch3)

	servers := registry.ChannelServers(tenant)

	if len(servers) != 3 {
		t.Errorf("ChannelServers() should return 3 channels, got %d", len(servers))
	}
}

func TestChannelServer_Found(t *testing.T) {
	registry := channel.GetChannelRegistry()
	tenant := createUniqueTenant(t)

	ch := createTestChannel(t, 1, 2, "192.168.1.1", 8080)
	registry.Register(tenant, ch)

	result, err := registry.ChannelServer(tenant, channelConstant.NewModel(1, 2))

	if err != nil {
		t.Fatalf("ChannelServer() unexpected error: %v", err)
	}
	if result.Id() != ch.Id() {
		t.Error("ChannelServer() returned wrong channel")
	}
}

func TestChannelServer_NotFound_World(t *testing.T) {
	registry := channel.GetChannelRegistry()
	tenant := createUniqueTenant(t)

	ch := createTestChannel(t, 1, 0, "192.168.1.1", 8080)
	registry.Register(tenant, ch)

	_, err := registry.ChannelServer(tenant, channelConstant.NewModel(99, 0))

	if err != channel.ErrChannelNotFound {
		t.Errorf("ChannelServer() error = %v, want ErrChannelNotFound", err)
	}
}

func TestChannelServer_NotFound_Channel(t *testing.T) {
	registry := channel.GetChannelRegistry()
	tenant := createUniqueTenant(t)

	ch := createTestChannel(t, 1, 0, "192.168.1.1", 8080)
	registry.Register(tenant, ch)

	_, err := registry.ChannelServer(tenant, channelConstant.NewModel(1, 99))

	if err != channel.ErrChannelNotFound {
		t.Errorf("ChannelServer() error = %v, want ErrChannelNotFound", err)
	}
}

func TestRemoveByWorldAndChannel_Success(t *testing.T) {
	registry := channel.GetChannelRegistry()
	tenant := createUniqueTenant(t)

	ch := createTestChannel(t, 1, 0, "192.168.1.1", 8080)
	registry.Register(tenant, ch)

	err := registry.RemoveByWorldAndChannel(tenant, channelConstant.NewModel(1, 0))

	if err != nil {
		t.Fatalf("RemoveByWorldAndChannel() unexpected error: %v", err)
	}

	// Verify channel is removed
	_, err = registry.ChannelServer(tenant, channelConstant.NewModel(1, 0))
	if err != channel.ErrChannelNotFound {
		t.Error("Channel should have been removed")
	}
}

func TestRemoveByWorldAndChannel_NotFound_World(t *testing.T) {
	registry := channel.GetChannelRegistry()
	tenant := createUniqueTenant(t)

	err := registry.RemoveByWorldAndChannel(tenant, channelConstant.NewModel(99, 0))

	if err != channel.ErrChannelNotFound {
		t.Errorf("RemoveByWorldAndChannel() error = %v, want ErrChannelNotFound", err)
	}
}

func TestRemoveByWorldAndChannel_NotFound_Channel(t *testing.T) {
	registry := channel.GetChannelRegistry()
	tenant := createUniqueTenant(t)

	ch := createTestChannel(t, 1, 0, "192.168.1.1", 8080)
	registry.Register(tenant, ch)

	err := registry.RemoveByWorldAndChannel(tenant, channelConstant.NewModel(1, 99))

	if err != channel.ErrChannelNotFound {
		t.Errorf("RemoveByWorldAndChannel() error = %v, want ErrChannelNotFound", err)
	}
}

func TestTenants(t *testing.T) {
	registry := channel.GetChannelRegistry()
	tenant1 := createUniqueTenant(t)
	tenant2 := createUniqueTenant(t)

	ch1 := createTestChannel(t, 1, 0, "192.168.1.1", 8080)
	ch2 := createTestChannel(t, 1, 0, "192.168.1.2", 8081)

	registry.Register(tenant1, ch1)
	registry.Register(tenant2, ch2)

	tenants := registry.Tenants()

	// Should contain both tenants (among potentially others from other tests)
	found1, found2 := false, false
	for _, t := range tenants {
		if t.Id() == tenant1.Id() {
			found1 = true
		}
		if t.Id() == tenant2.Id() {
			found2 = true
		}
	}

	if !found1 {
		t.Error("Tenants() should include tenant1")
	}
	if !found2 {
		t.Error("Tenants() should include tenant2")
	}
}

func TestTenantIsolation(t *testing.T) {
	registry := channel.GetChannelRegistry()
	tenant1 := createUniqueTenant(t)
	tenant2 := createUniqueTenant(t)

	ch1 := createTestChannel(t, 1, 0, "192.168.1.1", 8080)
	ch2 := createTestChannel(t, 1, 0, "192.168.1.2", 8081)

	registry.Register(tenant1, ch1)
	registry.Register(tenant2, ch2)

	// Tenant 1 should only see their channel
	servers1 := registry.ChannelServers(tenant1)
	for _, s := range servers1 {
		if s.Id() == ch2.Id() {
			t.Error("Tenant 1 should not see tenant 2's channel")
		}
	}

	// Tenant 2 should only see their channel
	servers2 := registry.ChannelServers(tenant2)
	for _, s := range servers2 {
		if s.Id() == ch1.Id() {
			t.Error("Tenant 2 should not see tenant 1's channel")
		}
	}

	// Tenant 1 cannot get tenant 2's channel directly
	result, err := registry.ChannelServer(tenant1, channelConstant.NewModel(1, 0))
	if err != nil {
		t.Fatalf("Tenant 1 should have a channel at world 1 channel 0")
	}
	if result.Id() == ch2.Id() {
		t.Error("Tenant 1 should not get tenant 2's channel")
	}
}

func TestConcurrentAccess(t *testing.T) {
	registry := channel.GetChannelRegistry()
	tenant := createUniqueTenant(t)

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent registrations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			ch := createTestChannel(t, world.Id(idx%10), channelConstant.Id(idx/10), "192.168.1.1", 8080+idx)
			registry.Register(tenant, ch)
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = registry.ChannelServers(tenant)
		}()
	}
	wg.Wait()

	// Concurrent mixed operations
	wg.Add(numGoroutines * 2)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			ch := createTestChannel(t, world.Id(idx%5), channelConstant.Id(idx%10), "192.168.1.1", 9000+idx)
			registry.Register(tenant, ch)
		}(i)
		go func() {
			defer wg.Done()
			_ = registry.ChannelServers(tenant)
		}()
	}
	wg.Wait()

	// Test should complete without deadlock or panic
}

func TestConcurrentTenantAccess(t *testing.T) {
	registry := channel.GetChannelRegistry()

	var wg sync.WaitGroup
	numTenants := 20
	numOps := 50

	// Multiple tenants doing concurrent operations
	for i := 0; i < numTenants; i++ {
		tenant := createUniqueTenant(t)
		wg.Add(numOps)
		for j := 0; j < numOps; j++ {
			go func(opIdx int) {
				defer wg.Done()
				ch := createTestChannel(t, world.Id(opIdx%5), channelConstant.Id(opIdx/5), "192.168.1.1", 8080+opIdx)
				registry.Register(tenant, ch)
				_ = registry.ChannelServers(tenant)
			}(j)
		}
	}
	wg.Wait()

	// Test should complete without deadlock or panic
}
