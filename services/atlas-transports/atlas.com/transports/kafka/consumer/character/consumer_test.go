package character

import (
	character2 "atlas-transports/kafka/message/character"
	"atlas-transports/transport"
	"bytes"
	"context"
	"testing"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestHandleEventStatus_NonLogoutEventIgnored(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	assert.NoError(t, err)

	// Create context with tenant
	ctx := tenant.WithContext(context.Background(), tenantModel)

	// Create logger with buffer to verify no errors
	var logBuffer bytes.Buffer
	l := logrus.New()
	l.SetOutput(&logBuffer)

	// Create a non-logout event (e.g., login event)
	event := character2.StatusEvent[character2.LogoutStatusEventBody]{
		WorldId:     0,
		CharacterId: 12345,
		Type:        "LOGIN", // Not a logout event
		Body: character2.LogoutStatusEventBody{
			ChannelId: 1,
			MapId:     100000000,
		},
	}

	// Call the handler - should return immediately without processing
	handleEventStatus(l, ctx, event)

	// Verify no errors were logged (handler should return early)
	logOutput := logBuffer.String()
	assert.Empty(t, logOutput, "Should not log anything for non-logout events")
}

func TestHandleEventStatus_LogoutFromNonTransportMap(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	assert.NoError(t, err)

	// Create context with tenant
	ctx := tenant.WithContext(context.Background(), tenantModel)

	// Create logger
	var logBuffer bytes.Buffer
	l := logrus.New()
	l.SetOutput(&logBuffer)
	l.SetLevel(logrus.DebugLevel)

	// Create a logout event from a non-transport map (no routes registered)
	event := character2.StatusEvent[character2.LogoutStatusEventBody]{
		WorldId:     0,
		CharacterId: 12345,
		Type:        character2.StatusEventTypeLogout,
		Body: character2.LogoutStatusEventBody{
			ChannelId: 1,
			MapId:     100000000, // Regular map, not a transport map
		},
	}

	// Call the handler - should process but find no matching route
	handleEventStatus(l, ctx, event)

	// The handler should log the logout attempt
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "logged out", "Should log the logout event")
}

func TestHandleEventStatus_LogoutFromTransportMap(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	assert.NoError(t, err)

	// Create context with tenant
	ctx := tenant.WithContext(context.Background(), tenantModel)

	// Create logger
	var logBuffer bytes.Buffer
	l := logrus.New()
	l.SetOutput(&logBuffer)
	l.SetLevel(logrus.DebugLevel)

	// First, set up a route in the transport registry
	stagingMapId := _map.Id(200090000)
	route, err := transport.NewBuilder("Test Ferry").
		SetStartMapId(_map.Id(101000300)).
		SetStagingMapId(stagingMapId).
		SetEnRouteMapIds([]_map.Id{_map.Id(200090100)}).
		SetDestinationMapId(_map.Id(200000100)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	assert.NoError(t, err)

	// Add the route to the registry using the processor
	processor := transport.NewProcessor(l, ctx)
	err = processor.AddTenant([]transport.Model{route}, []transport.SharedVesselModel{})
	assert.NoError(t, err)

	// Create a logout event from the staging map (transport map)
	event := character2.StatusEvent[character2.LogoutStatusEventBody]{
		WorldId:     0,
		CharacterId: 12345,
		Type:        character2.StatusEventTypeLogout,
		Body: character2.LogoutStatusEventBody{
			ChannelId: 1,
			MapId:     uint32(stagingMapId),
		},
	}

	// Call the handler - should process and attempt to warp character
	handleEventStatus(l, ctx, event)

	// The handler should log the logout attempt
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "logged out", "Should log the logout event")
}

func TestHandleEventStatus_LogoutFromEnRouteMap(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	assert.NoError(t, err)

	// Create context with tenant
	ctx := tenant.WithContext(context.Background(), tenantModel)

	// Create logger
	var logBuffer bytes.Buffer
	l := logrus.New()
	l.SetOutput(&logBuffer)
	l.SetLevel(logrus.DebugLevel)

	// First, set up a route in the transport registry
	enRouteMapId := _map.Id(200090100)
	route, err := transport.NewBuilder("Test Ferry EnRoute").
		SetStartMapId(_map.Id(101000300)).
		SetStagingMapId(_map.Id(200090000)).
		SetEnRouteMapIds([]_map.Id{enRouteMapId}).
		SetDestinationMapId(_map.Id(200000100)).
		SetBoardingWindowDuration(5 * time.Minute).
		SetPreDepartureDuration(2 * time.Minute).
		SetTravelDuration(10 * time.Minute).
		SetCycleInterval(30 * time.Minute).
		Build()
	assert.NoError(t, err)

	// Add the route to the registry using the processor
	processor := transport.NewProcessor(l, ctx)
	err = processor.AddTenant([]transport.Model{route}, []transport.SharedVesselModel{})
	assert.NoError(t, err)

	// Create a logout event from an en-route map
	event := character2.StatusEvent[character2.LogoutStatusEventBody]{
		WorldId:     0,
		CharacterId: 12345,
		Type:        character2.StatusEventTypeLogout,
		Body: character2.LogoutStatusEventBody{
			ChannelId: 1,
			MapId:     uint32(enRouteMapId),
		},
	}

	// Call the handler - should process and attempt to warp character
	handleEventStatus(l, ctx, event)

	// The handler should log the logout attempt
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "logged out", "Should log the logout event")
}

func TestHandleEventStatus_MultipleEventTypes(t *testing.T) {
	// Create a unique tenant for this test
	tenantId := uuid.New()
	tenantModel, err := tenant.Register(tenantId, "GMS", 83, 1)
	assert.NoError(t, err)

	// Create context with tenant
	ctx := tenant.WithContext(context.Background(), tenantModel)

	// Create logger
	l := logrus.New()
	l.SetOutput(&bytes.Buffer{})

	// Test various non-logout event types
	nonLogoutTypes := []string{"LOGIN", "LEVEL_UP", "MAP_CHANGE", "DISCONNECT", ""}

	for _, eventType := range nonLogoutTypes {
		t.Run("EventType_"+eventType, func(t *testing.T) {
			var logBuffer bytes.Buffer
			l := logrus.New()
			l.SetOutput(&logBuffer)

			event := character2.StatusEvent[character2.LogoutStatusEventBody]{
				WorldId:     0,
				CharacterId: 12345,
				Type:        eventType,
				Body: character2.LogoutStatusEventBody{
					ChannelId: 1,
					MapId:     100000000,
				},
			}

			// Should not panic and should return early
			handleEventStatus(l, ctx, event)

			// Should not log anything for non-logout events
			assert.Empty(t, logBuffer.String(), "Should not log for event type: %s", eventType)
		})
	}
}
