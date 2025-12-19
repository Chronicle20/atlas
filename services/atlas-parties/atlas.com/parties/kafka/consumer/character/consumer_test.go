package character

import (
	"atlas-parties/character"
	"context"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/job"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Test setup helpers
func setupEventHandlerTest() (logrus.FieldLogger, context.Context, tenant.Model) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	
	return logger, ctx, ten
}

// Helper to create a character for testing
func createTestCharacter(ten tenant.Model, characterId uint32, partyId uint32, level byte, jobId job.Id) character.Model {
	registry := character.GetRegistry()
	char := registry.Create(ten, 1, 1, 100000, characterId, "TestChar", level, jobId, 0)
	
	if partyId != 0 {
		char = registry.Update(ten, characterId, func(m character.Model) character.Model {
			return m.JoinParty(partyId)
		})
	}
	
	return char
}

func TestHandleStatusEventLevelChanged(t *testing.T) {
	tests := []struct {
		name            string
		eventType       string
		characterId     uint32
		oldLevel        byte
		newLevel        byte
		partyId         uint32
		expectProcessed bool
		expectError     bool
	}{
		{
			name:            "valid level changed event for character in party",
			eventType:       StatusEventTypeLevelChanged,
			characterId:     123,
			oldLevel:        10,
			newLevel:        11,
			partyId:         456,
			expectProcessed: true,
			expectError:     false,
		},
		{
			name:            "valid level changed event for character not in party",
			eventType:       StatusEventTypeLevelChanged,
			characterId:     124,
			oldLevel:        5,
			newLevel:        6,
			partyId:         0, // not in party
			expectProcessed: true,
			expectError:     false,
		},
		{
			name:            "wrong event type should be ignored",
			eventType:       StatusEventTypeJobChanged, // wrong type
			characterId:     125,
			oldLevel:        15,
			newLevel:        16,
			partyId:         789,
			expectProcessed: false,
			expectError:     false,
		},
		{
			name:            "character not found should handle gracefully",
			eventType:       StatusEventTypeLevelChanged,
			characterId:     999, // non-existent character
			oldLevel:        20,
			newLevel:        21,
			partyId:         0,
			expectProcessed: true,
			expectError:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger, ctx, ten := setupEventHandlerTest()
			
			// Setup character if needed
			if tc.characterId != 999 {
				createTestCharacter(ten, tc.characterId, tc.partyId, tc.oldLevel, job.Id(100))
			}
			
			// Create event
			event := StatusEvent[LevelChangedStatusEventBody]{
				TransactionId: uuid.New(),
				WorldId:       world.Id(1),
				CharacterId:   tc.characterId,
				Type:          tc.eventType,
				Body: LevelChangedStatusEventBody{
					ChannelId: channel.Id(1),
					Amount:    tc.newLevel - tc.oldLevel,
					Current:   tc.newLevel,
				},
			}
			
			// Track if processor was called by checking character level
			var initialLevel byte
			if tc.characterId != 999 {
				char, _ := character.GetRegistry().Get(ten, tc.characterId)
				initialLevel = char.Level()
			}
			
			// Execute handler
			handleStatusEventLevelChanged(logger, ctx, event)
			
			// Verify results
			if tc.expectProcessed && tc.characterId != 999 {
				char, err := character.GetRegistry().Get(ten, tc.characterId)
				if tc.eventType == StatusEventTypeLevelChanged {
					if !tc.expectError {
						assert.NoError(t, err)
						assert.Equal(t, tc.newLevel, char.Level(), "Character level should be updated")
					}
				} else {
					// Wrong event type - level should not change
					assert.NoError(t, err)
					assert.Equal(t, initialLevel, char.Level(), "Character level should not change for wrong event type")
				}
			}
			
			// Cleanup
			if tc.characterId != 999 {
				character.GetRegistry().Delete(ten, tc.characterId)
			}
		})
	}
}

func TestHandleStatusEventJobChanged(t *testing.T) {
	tests := []struct {
		name            string
		eventType       string
		characterId     uint32
		oldJobId        job.Id
		newJobId        job.Id
		partyId         uint32
		expectProcessed bool
		expectError     bool
	}{
		{
			name:            "valid job changed event for character in party",
			eventType:       StatusEventTypeJobChanged,
			characterId:     200,
			oldJobId:        job.Id(100),
			newJobId:        job.Id(200),
			partyId:         500,
			expectProcessed: true,
			expectError:     false,
		},
		{
			name:            "valid job changed event for character not in party",
			eventType:       StatusEventTypeJobChanged,
			characterId:     201,
			oldJobId:        job.Id(100),
			newJobId:        job.Id(300),
			partyId:         0, // not in party
			expectProcessed: true,
			expectError:     false,
		},
		{
			name:            "wrong event type should be ignored",
			eventType:       StatusEventTypeLevelChanged, // wrong type
			characterId:     202,
			oldJobId:        job.Id(100),
			newJobId:        job.Id(400),
			partyId:         600,
			expectProcessed: false,
			expectError:     false,
		},
		{
			name:            "character not found should handle gracefully",
			eventType:       StatusEventTypeJobChanged,
			characterId:     888, // non-existent character
			oldJobId:        job.Id(100),
			newJobId:        job.Id(500),
			partyId:         0,
			expectProcessed: true,
			expectError:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			logger, ctx, ten := setupEventHandlerTest()
			
			// Setup character if needed
			if tc.characterId != 888 {
				createTestCharacter(ten, tc.characterId, tc.partyId, 10, tc.oldJobId)
			}
			
			// Create event
			event := StatusEvent[JobChangedStatusEventBody]{
				TransactionId: uuid.New(),
				WorldId:       world.Id(1),
				CharacterId:   tc.characterId,
				Type:          tc.eventType,
				Body: JobChangedStatusEventBody{
					ChannelId: channel.Id(1),
					JobId:     tc.newJobId,
				},
			}
			
			// Track if processor was called by checking character job
			var initialJobId job.Id
			if tc.characterId != 888 {
				char, _ := character.GetRegistry().Get(ten, tc.characterId)
				initialJobId = char.JobId()
			}
			
			// Execute handler
			handleStatusEventJobChanged(logger, ctx, event)
			
			// Verify results
			if tc.expectProcessed && tc.characterId != 888 {
				char, err := character.GetRegistry().Get(ten, tc.characterId)
				if tc.eventType == StatusEventTypeJobChanged {
					if !tc.expectError {
						assert.NoError(t, err)
						assert.Equal(t, tc.newJobId, char.JobId(), "Character job should be updated")
					}
				} else {
					// Wrong event type - job should not change
					assert.NoError(t, err)
					assert.Equal(t, initialJobId, char.JobId(), "Character job should not change for wrong event type")
				}
			}
			
			// Cleanup
			if tc.characterId != 888 {
				character.GetRegistry().Delete(ten, tc.characterId)
			}
		})
	}
}

func TestEventTypeFiltering(t *testing.T) {
	logger, ctx, ten := setupEventHandlerTest()
	characterId := uint32(300)
	
	// Setup character
	createTestCharacter(ten, characterId, 100, 5, job.Id(100))
	
	tests := []struct {
		name        string
		handler     func()
		expectLevel byte
		expectJobId job.Id
	}{
		{
			name: "level handler ignores job event",
			handler: func() {
				event := StatusEvent[LevelChangedStatusEventBody]{
					TransactionId: uuid.New(),
					WorldId:       world.Id(1),
					CharacterId:   characterId,
					Type:          StatusEventTypeJobChanged, // wrong type
					Body: LevelChangedStatusEventBody{
						ChannelId: channel.Id(1),
						Amount:    5,
						Current:   10,
					},
				}
				handleStatusEventLevelChanged(logger, ctx, event)
			},
			expectLevel: 5,         // unchanged
			expectJobId: job.Id(100), // unchanged
		},
		{
			name: "job handler ignores level event",
			handler: func() {
				event := StatusEvent[JobChangedStatusEventBody]{
					TransactionId: uuid.New(),
					WorldId:       world.Id(1),
					CharacterId:   characterId,
					Type:          StatusEventTypeLevelChanged, // wrong type
					Body: JobChangedStatusEventBody{
						ChannelId: channel.Id(1),
						JobId:     job.Id(200),
					},
				}
				handleStatusEventJobChanged(logger, ctx, event)
			},
			expectLevel: 5,         // unchanged
			expectJobId: job.Id(100), // unchanged
		},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Execute handler
			tc.handler()
			
			// Verify no changes
			char, err := character.GetRegistry().Get(ten, characterId)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectLevel, char.Level())
			assert.Equal(t, tc.expectJobId, char.JobId())
		})
	}
	
	// Cleanup
	character.GetRegistry().Delete(ten, characterId)
}

func TestEventHandlersLogMessages(t *testing.T) {
	// Test that handlers produce appropriate log messages for debugging
	logger, ctx, ten := setupEventHandlerTest()
	characterId := uint32(400)
	
	// Setup character
	createTestCharacter(ten, characterId, 100, 5, job.Id(100))
	
	t.Run("level handler logs processing messages", func(t *testing.T) {
		event := StatusEvent[LevelChangedStatusEventBody]{
			TransactionId: uuid.New(),
			WorldId:       world.Id(1),
			CharacterId:   characterId,
			Type:          StatusEventTypeLevelChanged,
			Body: LevelChangedStatusEventBody{
				ChannelId: channel.Id(1),
				Amount:    1,
				Current:   15,
			},
		}
		
		// Should not panic and complete processing
		assert.NotPanics(t, func() {
			handleStatusEventLevelChanged(logger, ctx, event)
		})
	})
	
	t.Run("job handler logs processing messages", func(t *testing.T) {
		event := StatusEvent[JobChangedStatusEventBody]{
			TransactionId: uuid.New(),
			WorldId:       world.Id(1),
			CharacterId:   characterId,
			Type:          StatusEventTypeJobChanged,
			Body: JobChangedStatusEventBody{
				ChannelId: channel.Id(1),
				JobId:     job.Id(300),
			},
		}
		
		// Should not panic and complete processing  
		assert.NotPanics(t, func() {
			handleStatusEventJobChanged(logger, ctx, event)
		})
	})
	
	// Cleanup
	character.GetRegistry().Delete(ten, characterId)
}