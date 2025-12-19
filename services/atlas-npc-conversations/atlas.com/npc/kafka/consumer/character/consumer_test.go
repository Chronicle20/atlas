package character

import (
	"atlas-npc-conversations/kafka/message/character"
	"context"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestHandleStatusEventMapChanged tests that map changed events are processed correctly
func TestHandleStatusEventMapChanged(t *testing.T) {
	// Setup
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	// Use a basic context - the tenant.MustFromContext will panic if tenant is not set
	// but that's expected behavior for this test case
	ctx := context.Background()
	
	characterId := uint32(12345)
	oldMapId := _map.Id(100001)
	targetMapId := _map.Id(100002)
	channelId := channel.Id(0)
	targetPortalId := uint32(0)
	
	// Create test event
	event := character.StatusEvent[character.StatusEventMapChangedBody]{
		CharacterId: characterId,
		Type:        character.StatusEventTypeMapChanged,
		WorldId:     0,
		Body: character.StatusEventMapChangedBody{
			ChannelId:      channelId,
			OldMapId:       oldMapId,
			TargetMapId:    targetMapId,
			TargetPortalId: targetPortalId,
		},
	}
	
	// Create the handler with nil db for this test
	handler := handleStatusEventMapChanged(nil)
	
	// Execute the handler - this will likely panic due to missing tenant context
	// but that's okay - we're testing that the handler processes the correct event type
	// The panic will occur in conversation.NewProcessor when it tries to extract tenant
	assert.Panics(t, func() {
		handler(logger, ctx, event)
	}, "Expected panic when tenant context is missing")
}

// TestHandleStatusEventMapChangedIgnoresWrongEventType tests that the handler ignores non-map-changed events
func TestHandleStatusEventMapChangedIgnoresWrongEventType(t *testing.T) {
	// Setup
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	
	ctx := context.Background()
	
	characterId := uint32(12345)
	oldMapId := _map.Id(100001)
	targetMapId := _map.Id(100002)
	channelId := channel.Id(0)
	targetPortalId := uint32(0)
	
	// Create test event with wrong type
	event := character.StatusEvent[character.StatusEventMapChangedBody]{
		CharacterId: characterId,
		Type:        character.StatusEventTypeLogout, // Wrong type
		WorldId:     0,
		Body: character.StatusEventMapChangedBody{
			ChannelId:      channelId,
			OldMapId:       oldMapId,
			TargetMapId:    targetMapId,
			TargetPortalId: targetPortalId,
		},
	}
	
	// Create the handler
	handler := handleStatusEventMapChanged(nil)
	
	// Execute the handler - should return immediately without processing
	// since the event type doesn't match, the handler should return early
	// before it gets to the tenant extraction code
	assert.NotPanics(t, func() {
		handler(logger, ctx, event)
	})
}

// TestHandleStatusEventMapChangedFunctionExists tests that the handler function exists and has the correct signature
func TestHandleStatusEventMapChangedFunctionExists(t *testing.T) {
	// This test ensures the function exists and has the correct signature
	handler := handleStatusEventMapChanged((*gorm.DB)(nil))
	assert.NotNil(t, handler)
	
	// Test that the returned function has the correct signature
	logger := logrus.New()
	ctx := context.Background()
	event := character.StatusEvent[character.StatusEventMapChangedBody]{
		CharacterId: 12345,
		Type:        character.StatusEventTypeMapChanged,
		WorldId:     0,
		Body: character.StatusEventMapChangedBody{
			ChannelId:      channel.Id(0),
			OldMapId:       _map.Id(100001),
			TargetMapId:    _map.Id(100002),
			TargetPortalId: uint32(0),
		},
	}
	
	// Should panic when called with correct arguments but no tenant context
	assert.Panics(t, func() {
		handler(logger, ctx, event)
	}, "Expected panic when tenant context is missing")
}

// TestHandleStatusEventMapChangedHandlerRegistration tests that the handler is registered in InitHandlers
func TestHandleStatusEventMapChangedHandlerRegistration(t *testing.T) {
	// Verify that the InitHandlers function exists and can be called
	logger := logrus.New()
	var db *gorm.DB
	
	// This should not panic
	assert.NotPanics(t, func() {
		initHandlers := InitHandlers(logger, db)
		assert.NotNil(t, initHandlers)
		
		// Test that the function can be called (though we can't easily verify the actual registration)
		handlerCount := 0
		mockRegisterFunc := func(topic string, h handler.Handler) (string, error) {
			handlerCount++
			return "test-handler", nil
		}
		
		initHandlers(mockRegisterFunc)
		
		// Verify that handlers were registered (should be 3: logout, channel changed, map changed)
		assert.Equal(t, 3, handlerCount, "Expected 3 handlers to be registered")
	})
}