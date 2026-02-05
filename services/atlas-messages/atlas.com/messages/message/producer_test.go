package message

import (
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

// TestGeneralChatEventProvider tests the general chat event provider
func TestGeneralChatEventProvider(t *testing.T) {
	testCases := []struct {
		name        string
		worldId     world.Id
		channelId   channel.Id
		mapId       _map.Id
		actorId     uint32
		message     string
		balloonOnly bool
	}{
		{
			name:        "Standard general chat",
			worldId:     1,
			channelId:   1,
			mapId:       100000000,
			actorId:     12345,
			message:     "Hello world!",
			balloonOnly: false,
		},
		{
			name:        "Balloon only chat",
			worldId:     2,
			channelId:   3,
			mapId:       200000000,
			actorId:     67890,
			message:     "Balloon message",
			balloonOnly: true,
		},
		{
			name:        "Empty message",
			worldId:     1,
			channelId:   1,
			mapId:       100000000,
			actorId:     12345,
			message:     "",
			balloonOnly: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := field.NewBuilder(tc.worldId, tc.channelId, tc.mapId).Build()
			provider := generalChatEventProvider(f, tc.actorId, tc.message, tc.balloonOnly)

			// Provider should not be nil
			if provider == nil {
				t.Fatal("Expected provider to be non-nil")
			}

			// Execute the provider
			messages, err := provider()
			if err != nil {
				t.Fatalf("Provider returned error: %v", err)
			}

			// Should return exactly one message
			if len(messages) != 1 {
				t.Errorf("Expected 1 message, got %d", len(messages))
			}

			// Verify key is set (actor ID based)
			if len(messages) > 0 && len(messages[0].Key) == 0 {
				t.Error("Expected message key to be set")
			}
		})
	}
}

// TestMultiChatEventProvider tests the multi chat event provider
func TestMultiChatEventProvider(t *testing.T) {
	testCases := []struct {
		name       string
		worldId    world.Id
		channelId  channel.Id
		mapId      _map.Id
		actorId    uint32
		message    string
		chatType   string
		recipients []uint32
	}{
		{
			name:       "Party chat",
			worldId:    1,
			channelId:  1,
			mapId:      100000000,
			actorId:    12345,
			message:    "Party message",
			chatType:   "party",
			recipients: []uint32{12345, 12346, 12347},
		},
		{
			name:       "Guild chat",
			worldId:    1,
			channelId:  2,
			mapId:      200000000,
			actorId:    67890,
			message:    "Guild message",
			chatType:   "guild",
			recipients: []uint32{67890, 67891},
		},
		{
			name:       "Empty recipients",
			worldId:    1,
			channelId:  1,
			mapId:      100000000,
			actorId:    12345,
			message:    "Solo message",
			chatType:   "party",
			recipients: []uint32{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := field.NewBuilder(tc.worldId, tc.channelId, tc.mapId).Build()
			provider := multiChatEventProvider(f, tc.actorId, tc.message, tc.chatType, tc.recipients)

			if provider == nil {
				t.Fatal("Expected provider to be non-nil")
			}

			messages, err := provider()
			if err != nil {
				t.Fatalf("Provider returned error: %v", err)
			}

			if len(messages) != 1 {
				t.Errorf("Expected 1 message, got %d", len(messages))
			}
		})
	}
}

// TestWhisperChatEventProvider tests the whisper chat event provider
func TestWhisperChatEventProvider(t *testing.T) {
	testCases := []struct {
		name      string
		worldId   world.Id
		channelId channel.Id
		mapId     _map.Id
		actorId   uint32
		message   string
		recipient uint32
	}{
		{
			name:      "Standard whisper",
			worldId:   1,
			channelId: 1,
			mapId:     100000000,
			actorId:   12345,
			message:   "Private message",
			recipient: 67890,
		},
		{
			name:      "Self whisper",
			worldId:   1,
			channelId: 1,
			mapId:     100000000,
			actorId:   12345,
			message:   "Self message",
			recipient: 12345,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := field.NewBuilder(tc.worldId, tc.channelId, tc.mapId).Build()
			provider := whisperChatEventProvider(f, tc.actorId, tc.message, tc.recipient)

			if provider == nil {
				t.Fatal("Expected provider to be non-nil")
			}

			messages, err := provider()
			if err != nil {
				t.Fatalf("Provider returned error: %v", err)
			}

			if len(messages) != 1 {
				t.Errorf("Expected 1 message, got %d", len(messages))
			}
		})
	}
}

// TestMessengerChatEventProvider tests the messenger chat event provider
func TestMessengerChatEventProvider(t *testing.T) {
	testCases := []struct {
		name       string
		worldId    world.Id
		channelId  channel.Id
		mapId      _map.Id
		actorId    uint32
		message    string
		recipients []uint32
	}{
		{
			name:       "Messenger with recipients",
			worldId:    1,
			channelId:  1,
			mapId:      100000000,
			actorId:    12345,
			message:    "Messenger message",
			recipients: []uint32{12346, 12347},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := field.NewBuilder(tc.worldId, tc.channelId, tc.mapId).Build()
			provider := messengerChatEventProvider(f, tc.actorId, tc.message, tc.recipients)

			if provider == nil {
				t.Fatal("Expected provider to be non-nil")
			}

			messages, err := provider()
			if err != nil {
				t.Fatalf("Provider returned error: %v", err)
			}

			if len(messages) != 1 {
				t.Errorf("Expected 1 message, got %d", len(messages))
			}
		})
	}
}

// TestPetChatEventProvider tests the pet chat event provider
func TestPetChatEventProvider(t *testing.T) {
	testCases := []struct {
		name      string
		worldId   world.Id
		channelId channel.Id
		mapId     _map.Id
		petId     uint32
		message   string
		ownerId   uint32
		petSlot   int8
		nType     byte
		nAction   byte
		balloon   bool
	}{
		{
			name:      "Pet chat with balloon",
			worldId:   1,
			channelId: 1,
			mapId:     100000000,
			petId:     98765,
			message:   "Pet says hello!",
			ownerId:   12345,
			petSlot:   0,
			nType:     1,
			nAction:   2,
			balloon:   true,
		},
		{
			name:      "Pet chat without balloon",
			worldId:   2,
			channelId: 2,
			mapId:     200000000,
			petId:     11111,
			message:   "Pet message",
			ownerId:   67890,
			petSlot:   1,
			nType:     0,
			nAction:   0,
			balloon:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := field.NewBuilder(tc.worldId, tc.channelId, tc.mapId).Build()
			provider := petChatEventProvider(f, tc.petId, tc.message, tc.ownerId, tc.petSlot, tc.nType, tc.nAction, tc.balloon)

			if provider == nil {
				t.Fatal("Expected provider to be non-nil")
			}

			messages, err := provider()
			if err != nil {
				t.Fatalf("Provider returned error: %v", err)
			}

			if len(messages) != 1 {
				t.Errorf("Expected 1 message, got %d", len(messages))
			}
		})
	}
}

// TestPinkTextChatEventProvider tests the pink text chat event provider
func TestPinkTextChatEventProvider(t *testing.T) {
	testCases := []struct {
		name       string
		worldId    world.Id
		channelId  channel.Id
		mapId      _map.Id
		actorId    uint32
		message    string
		recipients []uint32
	}{
		{
			name:       "Pink text to single recipient",
			worldId:    1,
			channelId:  1,
			mapId:      100000000,
			actorId:    0, // System actor
			message:    "You are in map 100000000",
			recipients: []uint32{12345},
		},
		{
			name:       "Pink text to multiple recipients",
			worldId:    1,
			channelId:  1,
			mapId:      100000000,
			actorId:    0,
			message:    "Server announcement",
			recipients: []uint32{12345, 67890, 11111},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := field.NewBuilder(tc.worldId, tc.channelId, tc.mapId).Build()
			provider := pinkTextChatEventProvider(f, tc.actorId, tc.message, tc.recipients)

			if provider == nil {
				t.Fatal("Expected provider to be non-nil")
			}

			messages, err := provider()
			if err != nil {
				t.Fatalf("Provider returned error: %v", err)
			}

			if len(messages) != 1 {
				t.Errorf("Expected 1 message, got %d", len(messages))
			}
		})
	}
}

// TestEventProviders_MessageKeyConsistency tests that message keys are consistent
func TestEventProviders_MessageKeyConsistency(t *testing.T) {
	// All event providers should create messages with consistent key format
	// Key is based on actor ID

	actorId := uint32(12345)
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()

	providers := []struct {
		name     string
		provider func() ([]byte, error)
	}{
		{
			name: "General chat",
			provider: func() ([]byte, error) {
				msgs, err := generalChatEventProvider(f, actorId, "test", false)()
				if err != nil || len(msgs) == 0 {
					return nil, err
				}
				return msgs[0].Key, nil
			},
		},
		{
			name: "Multi chat",
			provider: func() ([]byte, error) {
				msgs, err := multiChatEventProvider(f, actorId, "test", "party", []uint32{})()
				if err != nil || len(msgs) == 0 {
					return nil, err
				}
				return msgs[0].Key, nil
			},
		},
		{
			name: "Whisper chat",
			provider: func() ([]byte, error) {
				msgs, err := whisperChatEventProvider(f, actorId, "test", 67890)()
				if err != nil || len(msgs) == 0 {
					return nil, err
				}
				return msgs[0].Key, nil
			},
		},
	}

	var firstKey []byte
	for i, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			key, err := p.provider()
			if err != nil {
				t.Fatalf("Provider error: %v", err)
			}

			if i == 0 {
				firstKey = key
			} else {
				// Keys should be consistent for same actor
				if string(key) != string(firstKey) {
					t.Errorf("Expected consistent key for actor %d, got different keys", actorId)
				}
			}
		})
	}
}
