package character

import (
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

func TestCommand_ChangeMap_Serialization(t *testing.T) {
	cmd := Command[ChangeMapBody]{
		TransactionId: uuid.MustParse("12345678-1234-5678-1234-567812345678"),
		WorldId:       world.Id(1),
		CharacterId:   42,
		Type:          CommandChangeMap,
		Body: ChangeMapBody{
			ChannelId: channel.Id(2),
			MapId:     _map.Id(100000000),
			Instance:  uuid.Nil,
			PortalId:  0,
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("Failed to marshal command: %v", err)
	}

	var decoded Command[ChangeMapBody]
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	if decoded.Type != CommandChangeMap {
		t.Errorf("Expected Type %s, got %s", CommandChangeMap, decoded.Type)
	}
	if decoded.Body.MapId != _map.Id(100000000) {
		t.Errorf("Expected MapId 100000000, got %d", decoded.Body.MapId)
	}
	if decoded.Body.Instance != uuid.Nil {
		t.Errorf("Expected Instance Nil, got %v", decoded.Body.Instance)
	}
}

func TestCommandTypeConstant_ChangeMap(t *testing.T) {
	if CommandChangeMap != "CHANGE_MAP" {
		t.Errorf("Expected CommandChangeMap to be 'CHANGE_MAP', got '%s'", CommandChangeMap)
	}
	if EnvCommandTopic != "COMMAND_TOPIC_CHARACTER" {
		t.Errorf("Expected EnvCommandTopic to be 'COMMAND_TOPIC_CHARACTER', got '%s'", EnvCommandTopic)
	}
}
