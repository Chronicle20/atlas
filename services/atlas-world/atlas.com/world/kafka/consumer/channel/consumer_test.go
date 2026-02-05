package channel_test

import (
	"atlas-world/channel"
	consumer "atlas-world/kafka/consumer/channel"
	message "atlas-world/kafka/message/channel"
	"atlas-world/test"
	"testing"

	channel2 "github.com/Chronicle20/atlas-constants/channel"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func setupTest(t *testing.T) (logrus.FieldLogger, *logtest.Hook, func()) {
	t.Helper()
	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	cleanup := func() {
		hook.Reset()
	}

	return logger, hook, cleanup
}

func TestHandleEventStatus_Started(t *testing.T) {
	logger, _, cleanup := setupTest(t)
	defer cleanup()

	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)
	tenant := test.CreateMockTenant(tenantId)

	event := message.StatusEvent{
		Type:            channel2.StatusTypeStarted,
		WorldId:         1,
		ChannelId:       0,
		IpAddress:       "192.168.1.100",
		Port:            8080,
		CurrentCapacity: 25,
		MaxCapacity:     100,
	}

	// Use the exported test helper function
	consumer.HandleEventStatusForTest(logger, ctx, event)

	// Verify channel was registered
	registry := channel.GetChannelRegistry()
	ch, err := registry.ChannelServer(tenant, 1, 0)
	if err != nil {
		t.Fatalf("Channel should have been registered, got error: %v", err)
	}

	if ch.WorldId() != 1 {
		t.Errorf("ch.WorldId() = %d, want 1", ch.WorldId())
	}
	if ch.ChannelId() != 0 {
		t.Errorf("ch.ChannelId() = %d, want 0", ch.ChannelId())
	}
	if ch.IpAddress() != "192.168.1.100" {
		t.Errorf("ch.IpAddress() = %s, want 192.168.1.100", ch.IpAddress())
	}
	if ch.Port() != 8080 {
		t.Errorf("ch.Port() = %d, want 8080", ch.Port())
	}
	if ch.CurrentCapacity() != 25 {
		t.Errorf("ch.CurrentCapacity() = %d, want 25", ch.CurrentCapacity())
	}
	if ch.MaxCapacity() != 100 {
		t.Errorf("ch.MaxCapacity() = %d, want 100", ch.MaxCapacity())
	}

	// Cleanup
	_ = registry.RemoveByWorldAndChannel(tenant, 1, 0)
}

func TestHandleEventStatus_Shutdown(t *testing.T) {
	logger, _, cleanup := setupTest(t)
	defer cleanup()

	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)
	tenant := test.CreateMockTenant(tenantId)

	// First register a channel
	channelProcessor := channel.NewProcessor(logger, ctx)
	_, err := channelProcessor.Register(2, 1, "192.168.1.200", 9000, 0, 50)
	if err != nil {
		t.Fatalf("Failed to register channel: %v", err)
	}

	// Verify it exists
	registry := channel.GetChannelRegistry()
	_, err = registry.ChannelServer(tenant, 2, 1)
	if err != nil {
		t.Fatalf("Channel should exist before shutdown event")
	}

	// Send shutdown event
	event := message.StatusEvent{
		Type:      channel2.StatusTypeShutdown,
		WorldId:   2,
		ChannelId: 1,
		IpAddress: "192.168.1.200",
		Port:      9000,
	}

	consumer.HandleEventStatusForTest(logger, ctx, event)

	// Verify channel was unregistered
	_, err = registry.ChannelServer(tenant, 2, 1)
	if err == nil {
		t.Error("Channel should have been unregistered")
	}
}

func TestHandleEventStatus_UnknownType(t *testing.T) {
	logger, hook, cleanup := setupTest(t)
	defer cleanup()

	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)

	event := message.StatusEvent{
		Type:      "UNKNOWN_TYPE",
		WorldId:   1,
		ChannelId: 0,
	}

	consumer.HandleEventStatusForTest(logger, ctx, event)

	// Should have logged an error
	found := false
	for _, entry := range hook.Entries {
		if entry.Level == logrus.ErrorLevel {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected error log for unknown event type")
	}
}

func TestHandleEventStatus_MultipleStarted(t *testing.T) {
	logger, _, cleanup := setupTest(t)
	defer cleanup()

	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)
	tenant := test.CreateMockTenant(tenantId)

	// Register first channel
	event1 := message.StatusEvent{
		Type:        channel2.StatusTypeStarted,
		WorldId:     1,
		ChannelId:   0,
		IpAddress:   "192.168.1.1",
		Port:        8080,
		MaxCapacity: 100,
	}
	consumer.HandleEventStatusForTest(logger, ctx, event1)

	// Register second channel
	event2 := message.StatusEvent{
		Type:        channel2.StatusTypeStarted,
		WorldId:     1,
		ChannelId:   1,
		IpAddress:   "192.168.1.2",
		Port:        8081,
		MaxCapacity: 100,
	}
	consumer.HandleEventStatusForTest(logger, ctx, event2)

	// Verify both channels exist
	registry := channel.GetChannelRegistry()
	channels := registry.ChannelServers(tenant)

	if len(channels) != 2 {
		t.Errorf("len(channels) = %d, want 2", len(channels))
	}

	// Cleanup
	_ = registry.RemoveByWorldAndChannel(tenant, 1, 0)
	_ = registry.RemoveByWorldAndChannel(tenant, 1, 1)
}

func TestHandleEventStatus_StartedThenShutdown(t *testing.T) {
	logger, _, cleanup := setupTest(t)
	defer cleanup()

	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)
	tenant := test.CreateMockTenant(tenantId)

	// Start channel
	startEvent := message.StatusEvent{
		Type:        channel2.StatusTypeStarted,
		WorldId:     3,
		ChannelId:   0,
		IpAddress:   "192.168.1.50",
		Port:        7000,
		MaxCapacity: 200,
	}
	consumer.HandleEventStatusForTest(logger, ctx, startEvent)

	// Verify exists
	registry := channel.GetChannelRegistry()
	_, err := registry.ChannelServer(tenant, 3, 0)
	if err != nil {
		t.Fatal("Channel should exist after start event")
	}

	// Shutdown channel
	shutdownEvent := message.StatusEvent{
		Type:      channel2.StatusTypeShutdown,
		WorldId:   3,
		ChannelId: 0,
		IpAddress: "192.168.1.50",
		Port:      7000,
	}
	consumer.HandleEventStatusForTest(logger, ctx, shutdownEvent)

	// Verify removed
	_, err = registry.ChannelServer(tenant, 3, 0)
	if err == nil {
		t.Error("Channel should be removed after shutdown event")
	}
}

func TestHandleEventStatus_ShutdownNonExistent(t *testing.T) {
	logger, _, cleanup := setupTest(t)
	defer cleanup()

	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)

	// Try to shutdown a channel that doesn't exist
	// This should not panic
	event := message.StatusEvent{
		Type:      channel2.StatusTypeShutdown,
		WorldId:   99,
		ChannelId: 99,
		IpAddress: "192.168.1.1",
		Port:      8080,
	}

	// Should not panic
	consumer.HandleEventStatusForTest(logger, ctx, event)
}
