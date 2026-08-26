package monster

import (
	"atlas-monsters/monster/information"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"

	system_message "atlas-monsters/kafka/message/system_message"
)

func TestWarpCommandProviderCarriesPortalName(t *testing.T) {
	tests := []struct {
		name        string
		characterId uint32
		targetMapId _map.Id
		portalName  string
	}{
		{
			name:        "carries the WZ portal name",
			characterId: 4242,
			targetMapId: _map.Id(926120410),
			portalName:  "st00",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
			msgs, err := warpCommandProvider(f, tc.characterId, tc.targetMapId, tc.portalName)()
			if err != nil {
				t.Fatalf("provider error: %v", err)
			}
			if len(msgs) != 1 {
				t.Fatalf("expected 1 message, got %d", len(msgs))
			}
			var env struct {
				Type string   `json:"type"`
				Body warpBody `json:"body"`
			}
			if err := json.Unmarshal(msgs[0].Value, &env); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if env.Type != "WARP" {
				t.Errorf("type=%s, want WARP", env.Type)
			}
			if env.Body.CharacterId != tc.characterId {
				t.Errorf("CharacterId=%d, want %d", env.Body.CharacterId, tc.characterId)
			}
			if env.Body.TargetMapId != tc.targetMapId {
				t.Errorf("TargetMapId=%d, want %d", env.Body.TargetMapId, tc.targetMapId)
			}
			if env.Body.TargetPortalName != tc.portalName {
				t.Errorf("TargetPortalName=%s, want %s", env.Body.TargetPortalName, tc.portalName)
			}
		})
	}
}

func TestWarpCommandProviderOmitsEmptyPortalName(t *testing.T) {
	tests := []struct {
		name        string
		characterId uint32
		targetMapId _map.Id
		portalName  string
	}{
		{
			name:        "empty portal name omitted from wire",
			characterId: 4242,
			targetMapId: _map.Id(926120410),
			portalName:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
			msgs, err := warpCommandProvider(f, tc.characterId, tc.targetMapId, tc.portalName)()
			if err != nil {
				t.Fatalf("provider error: %v", err)
			}
			if len(msgs) != 1 {
				t.Fatalf("expected 1 message, got %d", len(msgs))
			}
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal(msgs[0].Value, &envelope); err != nil {
				t.Fatalf("decode envelope: %v", err)
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal(envelope["body"], &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if _, ok := body["targetPortalName"]; ok {
				t.Errorf("targetPortalName present in body when empty; body=%v", body)
			}
		})
	}
}

func TestSendMessageProviderShape(t *testing.T) {
	tests := []struct {
		name        string
		characterId uint32
		messageType string
		message     string
	}{
		{
			name:        "pink-text banish message",
			characterId: 4242,
			messageType: "PINK_TEXT",
			message:     "You have been banished.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
			msgs, err := sendMessageProvider(f, tc.characterId, tc.messageType, tc.message)()
			if err != nil {
				t.Fatalf("provider error: %v", err)
			}
			if len(msgs) != 1 {
				t.Fatalf("expected 1 message, got %d", len(msgs))
			}
			var env system_message.Command[system_message.SendMessageBody]
			if err := json.Unmarshal(msgs[0].Value, &env); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if env.Type != system_message.CommandSendMessage {
				t.Errorf("type=%s, want %s", env.Type, system_message.CommandSendMessage)
			}
			if env.WorldId != world.Id(1) {
				t.Errorf("WorldId=%d, want 1", env.WorldId)
			}
			if env.ChannelId != channel.Id(2) {
				t.Errorf("ChannelId=%d, want 2", env.ChannelId)
			}
			if env.CharacterId != tc.characterId {
				t.Errorf("CharacterId=%d, want %d", env.CharacterId, tc.characterId)
			}
			if env.Body.MessageType != tc.messageType {
				t.Errorf("MessageType=%s, want %s", env.Body.MessageType, tc.messageType)
			}
			if env.Body.Message != tc.message {
				t.Errorf("Message=%s, want %q", env.Body.Message, tc.message)
			}
		})
	}
}

func TestModelBuilderSetBanish(t *testing.T) {
	tests := []struct {
		name   string
		banish information.Banish
	}{
		{
			name:   "banish fields round-trip through the builder",
			banish: information.Banish{Message: "Get out.", MapId: 926120410, PortalName: "st00"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := information.NewModelBuilder().SetBanish(tc.banish).Build()
			if m.Banish() != tc.banish {
				t.Errorf("Banish()=%+v, want %+v", m.Banish(), tc.banish)
			}
		})
	}
}
