package channel

import (
	"atlas-transports/channel"
	channel2 "atlas-transports/kafka/message/channel"
	"bytes"
	"context"
	"testing"

	channel3 "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestHandleEventStatus_ChannelStarted(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	assert.NoError(t, err)

	// Create context with tenant
	ctx := tenant.WithContext(context.Background(), tenantModel)

	// Create logger
	l := logrus.New()
	l.SetOutput(&bytes.Buffer{}) // Suppress output during tests

	// Create the event
	event := channel2.StatusEvent{
		Type:      channel3.StatusTypeStarted,
		WorldId:   0,
		ChannelId: 1,
		IpAddress: "127.0.0.1",
		Port:      8484,
	}

	// Call the handler
	handleEventStatus(l, ctx, event)

	// Verify the channel was registered
	processor := channel.NewProcessor(l, ctx)
	channels := processor.GetAll()

	assert.Len(t, channels, 1, "Should have one channel registered")
	if len(channels) > 0 {
		assert.Equal(t, world.Id(0), channels[0].WorldId(), "World ID should match")
		assert.Equal(t, channel3.Id(1), channels[0].Id(), "Channel ID should match")
	}
}

func TestHandleEventStatus_ChannelShutdown(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	assert.NoError(t, err)

	// Create context with tenant
	ctx := tenant.WithContext(context.Background(), tenantModel)

	// Create logger
	l := logrus.New()
	l.SetOutput(&bytes.Buffer{}) // Suppress output during tests

	// First, register a channel
	startEvent := channel2.StatusEvent{
		Type:      channel3.StatusTypeStarted,
		WorldId:   0,
		ChannelId: 2,
		IpAddress: "127.0.0.1",
		Port:      8485,
	}
	handleEventStatus(l, ctx, startEvent)

	// Verify it was registered
	processor := channel.NewProcessor(l, ctx)
	channels := processor.GetAll()
	assert.Len(t, channels, 1, "Should have one channel registered")

	// Now send shutdown event
	shutdownEvent := channel2.StatusEvent{
		Type:      channel3.StatusTypeShutdown,
		WorldId:   0,
		ChannelId: 2,
		IpAddress: "127.0.0.1",
		Port:      8485,
	}
	handleEventStatus(l, ctx, shutdownEvent)

	// Verify the channel was unregistered
	channels = processor.GetAll()
	assert.Len(t, channels, 0, "Should have no channels registered after shutdown")
}

func TestHandleEventStatus_UnknownEventType(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	assert.NoError(t, err)

	// Create context with tenant
	ctx := tenant.WithContext(context.Background(), tenantModel)

	// Create a buffer to capture log output
	var logBuffer bytes.Buffer
	l := logrus.New()
	l.SetOutput(&logBuffer)
	l.SetLevel(logrus.ErrorLevel)

	// Create an event with unknown type
	event := channel2.StatusEvent{
		Type:      "UNKNOWN_TYPE",
		WorldId:   0,
		ChannelId: 1,
		IpAddress: "127.0.0.1",
		Port:      8484,
	}

	// Call the handler
	handleEventStatus(l, ctx, event)

	// Verify error was logged
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Unhandled event status", "Should log error for unknown event type")

	// Verify no channel was registered
	processor := channel.NewProcessor(l, ctx)
	channels := processor.GetAll()
	assert.Len(t, channels, 0, "Should have no channels registered for unknown event type")
}

func TestHandleEventStatus_MultipleChannels(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	assert.NoError(t, err)

	// Create context with tenant
	ctx := tenant.WithContext(context.Background(), tenantModel)

	// Create logger
	l := logrus.New()
	l.SetOutput(&bytes.Buffer{}) // Suppress output during tests

	// Register multiple channels
	events := []channel2.StatusEvent{
		{Type: channel3.StatusTypeStarted, WorldId: 0, ChannelId: 1, IpAddress: "127.0.0.1", Port: 8484},
		{Type: channel3.StatusTypeStarted, WorldId: 0, ChannelId: 2, IpAddress: "127.0.0.1", Port: 8485},
		{Type: channel3.StatusTypeStarted, WorldId: 1, ChannelId: 1, IpAddress: "127.0.0.1", Port: 8486},
	}

	for _, event := range events {
		handleEventStatus(l, ctx, event)
	}

	// Verify all channels were registered
	processor := channel.NewProcessor(l, ctx)
	channels := processor.GetAll()
	assert.Len(t, channels, 3, "Should have three channels registered")
}

func TestHandleEventStatus_DuplicateChannelIgnored(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	assert.NoError(t, err)

	// Create context with tenant
	ctx := tenant.WithContext(context.Background(), tenantModel)

	// Create logger
	l := logrus.New()
	l.SetOutput(&bytes.Buffer{}) // Suppress output during tests

	// Register the same channel twice
	event := channel2.StatusEvent{
		Type:      channel3.StatusTypeStarted,
		WorldId:   0,
		ChannelId: 5,
		IpAddress: "127.0.0.1",
		Port:      8484,
	}

	handleEventStatus(l, ctx, event)
	handleEventStatus(l, ctx, event)

	// Verify only one channel is registered
	processor := channel.NewProcessor(l, ctx)
	channels := processor.GetAll()
	assert.Len(t, channels, 1, "Should have only one channel registered despite duplicate events")
}
