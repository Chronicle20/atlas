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
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	msgs, err := warpCommandProvider(f, 4242, _map.Id(926120410), "st00")()
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
	if env.Body.CharacterId != 4242 {
		t.Errorf("CharacterId=%d, want 4242", env.Body.CharacterId)
	}
	if env.Body.TargetMapId != _map.Id(926120410) {
		t.Errorf("TargetMapId=%d, want 926120410", env.Body.TargetMapId)
	}
	if env.Body.TargetPortalName != "st00" {
		t.Errorf("TargetPortalName=%s, want st00", env.Body.TargetPortalName)
	}
}

func TestWarpCommandProviderOmitsEmptyPortalName(t *testing.T) {
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	msgs, err := warpCommandProvider(f, 4242, _map.Id(926120410), "")()
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
}

func TestSendMessageProviderShape(t *testing.T) {
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	msgs, err := sendMessageProvider(f, 4242, "PINK_TEXT", "You have been banished.")()
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
	if env.CharacterId != 4242 {
		t.Errorf("CharacterId=%d, want 4242", env.CharacterId)
	}
	if env.Body.MessageType != "PINK_TEXT" {
		t.Errorf("MessageType=%s, want PINK_TEXT", env.Body.MessageType)
	}
	if env.Body.Message != "You have been banished." {
		t.Errorf("Message=%s, want %q", env.Body.Message, "You have been banished.")
	}
}

func TestModelBuilderSetBanish(t *testing.T) {
	b := information.Banish{Message: "Get out.", MapId: 926120410, PortalName: "st00"}
	m := information.NewModelBuilder().SetBanish(b).Build()
	if m.Banish() != b {
		t.Errorf("Banish()=%+v, want %+v", m.Banish(), b)
	}
}
