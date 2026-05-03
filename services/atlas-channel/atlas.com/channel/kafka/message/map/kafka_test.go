package _map

import (
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

func TestStatusEvent_MapTimerStarted_Serialization(t *testing.T) {
	event := StatusEvent[MapTimerStarted]{
		TransactionId: uuid.MustParse("12345678-1234-5678-1234-567812345678"),
		WorldId:       world.Id(1),
		ChannelId:     channel.Id(2),
		MapId:         _map.Id(100000000),
		Instance:      uuid.Nil,
		Type:          EventTopicMapStatusTypeMapTimerStarted,
		Body: MapTimerStarted{
			CharacterId: 12345,
			Seconds:     600,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded StatusEvent[MapTimerStarted]
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if decoded.Type != EventTopicMapStatusTypeMapTimerStarted {
		t.Errorf("Type mismatch")
	}
	if decoded.Body.CharacterId != 12345 || decoded.Body.Seconds != 600 {
		t.Errorf("Body mismatch")
	}
}

func TestEventTypeConstant_MapTimerStarted(t *testing.T) {
	if EventTopicMapStatusTypeMapTimerStarted != "MAP_TIMER_STARTED" {
		t.Errorf("Expected 'MAP_TIMER_STARTED', got '%s'", EventTopicMapStatusTypeMapTimerStarted)
	}
}
