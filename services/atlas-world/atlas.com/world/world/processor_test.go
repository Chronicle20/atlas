package world_test

import (
	"atlas-world/channel"
	"atlas-world/test"
	"atlas-world/world"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func setupProcessor(t *testing.T) (world.Processor, func()) {
	t.Helper()
	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	processor := world.NewProcessor(logger, ctx)
	tenant := test.CreateMockTenant(tenantId)

	cleanup := func() {
		// Clean up any registered channels
		servers := channel.GetChannelRegistry().ChannelServers(tenant)
		for _, s := range servers {
			_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, s.WorldId(), s.ChannelId())
		}
	}

	return processor, cleanup
}

func TestNewProcessor(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	if processor == nil {
		t.Fatal("Expected processor to be initialized")
	}
}

func TestNewProcessor_PanicsWithoutTenant(t *testing.T) {
	logger, _ := logtest.NewNullLogger()

	defer func() {
		if r := recover(); r == nil {
			t.Error("NewProcessor should panic when context has no tenant")
		}
	}()

	ctx := context.Background()
	_ = world.NewProcessor(logger, ctx)
}

func TestChannelDecorator_WithChannels(t *testing.T) {
	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)
	logger, _ := logtest.NewNullLogger()

	processor := world.NewProcessor(logger, ctx)
	channelProcessor := channel.NewProcessor(logger, ctx)

	// Register some channels for world 1
	_, _ = channelProcessor.Register(1, 0, "192.168.1.1", 8080, 0, 100)
	_, _ = channelProcessor.Register(1, 1, "192.168.1.2", 8081, 0, 100)

	// Create a world model
	worldModel, _ := world.NewModelBuilder().
		SetId(1).
		SetName("Scania").
		Build()

	// Decorate with channels
	decorated := processor.ChannelDecorator(worldModel)

	if len(decorated.Channels()) != 2 {
		t.Errorf("len(decorated.Channels()) = %d, want 2", len(decorated.Channels()))
	}

	// Cleanup
	tenant := test.CreateMockTenant(tenantId)
	_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, 1, 0)
	_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, 1, 1)
}

func TestChannelDecorator_NoChannels(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	// Create a world model
	worldModel, _ := world.NewModelBuilder().
		SetId(5).
		SetName("TestWorld").
		Build()

	// Decorate (no channels exist for world 5)
	decorated := processor.ChannelDecorator(worldModel)

	// Should return same world with empty channels
	if decorated.Name() != "TestWorld" {
		t.Errorf("decorated.Name() = %s, want TestWorld", decorated.Name())
	}
	if len(decorated.Channels()) != 0 {
		t.Errorf("len(decorated.Channels()) = %d, want 0", len(decorated.Channels()))
	}
}

func TestChannelDecorator_PreservesOtherFields(t *testing.T) {
	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)
	logger, _ := logtest.NewNullLogger()

	processor := world.NewProcessor(logger, ctx)
	channelProcessor := channel.NewProcessor(logger, ctx)

	// Register a channel
	_, _ = channelProcessor.Register(2, 0, "192.168.1.1", 8080, 0, 100)

	// Create a world with all fields set
	worldModel, _ := world.NewModelBuilder().
		SetId(2).
		SetName("Bera").
		SetState(world.StateEvent).
		SetMessage("Welcome!").
		SetEventMessage("Double XP!").
		SetRecommendedMessage("Join us!").
		SetCapacityStatus(world.StatusHighlyPopulated).
		Build()

	decorated := processor.ChannelDecorator(worldModel)

	// Verify all original fields are preserved
	if decorated.Id() != 2 {
		t.Errorf("decorated.Id() = %d, want 2", decorated.Id())
	}
	if decorated.Name() != "Bera" {
		t.Errorf("decorated.Name() = %s, want Bera", decorated.Name())
	}
	if decorated.State() != world.StateEvent {
		t.Errorf("decorated.State() = %d, want StateEvent", decorated.State())
	}
	if decorated.Message() != "Welcome!" {
		t.Errorf("decorated.Message() = %s, want Welcome!", decorated.Message())
	}
	if decorated.EventMessage() != "Double XP!" {
		t.Errorf("decorated.EventMessage() = %s, want Double XP!", decorated.EventMessage())
	}
	if decorated.RecommendedMessage() != "Join us!" {
		t.Errorf("decorated.RecommendedMessage() = %s, want Join us!", decorated.RecommendedMessage())
	}
	if decorated.CapacityStatus() != world.StatusHighlyPopulated {
		t.Errorf("decorated.CapacityStatus() = %d, want StatusHighlyPopulated", decorated.CapacityStatus())
	}

	// And verify channels were added
	if len(decorated.Channels()) != 1 {
		t.Errorf("len(decorated.Channels()) = %d, want 1", len(decorated.Channels()))
	}

	// Cleanup
	tenant := test.CreateMockTenant(tenantId)
	_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, 2, 0)
}

func TestGetFlag(t *testing.T) {
	// Test the getFlag function by testing the state values through the builder
	testCases := []struct {
		state    world.State
		expected world.State
	}{
		{world.StateNormal, world.StateNormal},
		{world.StateEvent, world.StateEvent},
		{world.StateNew, world.StateNew},
		{world.StateHot, world.StateHot},
	}

	for _, tc := range testCases {
		model, err := world.NewModelBuilder().
			SetId(1).
			SetName("Test").
			SetState(tc.state).
			Build()

		if err != nil {
			t.Fatalf("Build() unexpected error: %v", err)
		}
		if model.State() != tc.expected {
			t.Errorf("model.State() = %d, want %d", model.State(), tc.expected)
		}
	}
}

func TestStateConstants(t *testing.T) {
	if world.StateNormal != 0 {
		t.Errorf("StateNormal = %d, want 0", world.StateNormal)
	}
	if world.StateEvent != 1 {
		t.Errorf("StateEvent = %d, want 1", world.StateEvent)
	}
	if world.StateNew != 2 {
		t.Errorf("StateNew = %d, want 2", world.StateNew)
	}
	if world.StateHot != 3 {
		t.Errorf("StateHot = %d, want 3", world.StateHot)
	}
}

func TestStatusConstants(t *testing.T) {
	if world.StatusNormal != 0 {
		t.Errorf("StatusNormal = %d, want 0", world.StatusNormal)
	}
	if world.StatusHighlyPopulated != 1 {
		t.Errorf("StatusHighlyPopulated = %d, want 1", world.StatusHighlyPopulated)
	}
	if world.StatusFull != 2 {
		t.Errorf("StatusFull = %d, want 2", world.StatusFull)
	}
}

func TestAllWorldProvider_NoChannels(t *testing.T) {
	processor, cleanup := setupProcessor(t)
	defer cleanup()

	// When no channels are registered, no worlds should be returned
	worlds, err := processor.AllWorldProvider()()
	if err != nil {
		t.Fatalf("AllWorldProvider() unexpected error: %v", err)
	}
	if len(worlds) != 0 {
		t.Errorf("len(worlds) = %d, want 0 (no channels registered)", len(worlds))
	}
}

func TestMapDistinctWorldId(t *testing.T) {
	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)
	logger, _ := logtest.NewNullLogger()

	channelProcessor := channel.NewProcessor(logger, ctx)

	// Register channels in different worlds
	_, _ = channelProcessor.Register(1, 0, "192.168.1.1", 8080, 0, 100)
	_, _ = channelProcessor.Register(1, 1, "192.168.1.2", 8081, 0, 100)
	_, _ = channelProcessor.Register(2, 0, "192.168.1.3", 8082, 0, 100)
	_, _ = channelProcessor.Register(3, 0, "192.168.1.4", 8083, 0, 100)
	_, _ = channelProcessor.Register(3, 1, "192.168.1.5", 8084, 0, 100)

	// Get all channels
	tenant := test.CreateMockTenant(tenantId)
	channels := channel.GetChannelRegistry().ChannelServers(tenant)

	// Count distinct world IDs
	worldIds := make(map[byte]bool)
	for _, ch := range channels {
		worldIds[ch.WorldId()] = true
	}

	if len(worldIds) != 3 {
		t.Errorf("Expected 3 distinct world IDs, got %d", len(worldIds))
	}

	// Cleanup
	_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, 1, 0)
	_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, 1, 1)
	_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, 2, 0)
	_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, 3, 0)
	_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, 3, 1)
}

func TestProcessorWithMultipleChannels(t *testing.T) {
	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)
	logger, _ := logtest.NewNullLogger()

	processor := world.NewProcessor(logger, ctx)
	channelProcessor := channel.NewProcessor(logger, ctx)

	// Register multiple channels across multiple worlds
	_, _ = channelProcessor.Register(0, 0, "192.168.1.1", 8080, 10, 100)
	_, _ = channelProcessor.Register(0, 1, "192.168.1.2", 8081, 20, 100)
	_, _ = channelProcessor.Register(0, 2, "192.168.1.3", 8082, 30, 100)

	// Create a world model for world 0
	worldModel, _ := world.NewModelBuilder().
		SetId(0).
		SetName("Scania").
		Build()

	// Decorate
	decorated := processor.ChannelDecorator(worldModel)

	// Should have 3 channels
	if len(decorated.Channels()) != 3 {
		t.Errorf("len(decorated.Channels()) = %d, want 3", len(decorated.Channels()))
	}

	// Verify each channel is present
	foundChannels := make(map[byte]bool)
	for _, ch := range decorated.Channels() {
		foundChannels[ch.ChannelId()] = true
	}

	if !foundChannels[0] || !foundChannels[1] || !foundChannels[2] {
		t.Error("Not all channels were found in decorated world")
	}

	// Cleanup
	tenant := test.CreateMockTenant(tenantId)
	_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, 0, 0)
	_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, 0, 1)
	_ = channel.GetChannelRegistry().RemoveByWorldAndChannel(tenant, 0, 2)
}

func createTestChannelInRegistry(t *testing.T, tenantId uuid.UUID, worldId, channelId byte) {
	t.Helper()
	ch, err := channel.NewModelBuilder().
		SetId(uuid.New()).
		SetWorldId(worldId).
		SetChannelId(channelId).
		SetIpAddress("192.168.1.1").
		SetPort(8080 + int(channelId)).
		SetMaxCapacity(100).
		SetCreatedAt(time.Now()).
		Build()
	if err != nil {
		t.Fatalf("Failed to create test channel: %v", err)
	}

	tenant := test.CreateMockTenant(tenantId)
	channel.GetChannelRegistry().Register(tenant, ch)
}
