package character

import (
	"encoding/json"
	"testing"

	messagechar "atlas-channel/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

func TestSetHPCommandProvider(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(3), 100000000).Build()
	msgs, err := SetHPCommandProvider(f, 4242, 0xFFFF)()
	if err != nil {
		t.Fatalf("provider err: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}

	var cmd messagechar.Command[messagechar.SetHPCommandBody]
	if uErr := json.Unmarshal(msgs[0].Value, &cmd); uErr != nil {
		t.Fatalf("unmarshal: %v", uErr)
	}
	if cmd.Type != messagechar.CommandSetHP {
		t.Fatalf("Type = %q, want %q", cmd.Type, messagechar.CommandSetHP)
	}
	if cmd.CharacterId != 4242 {
		t.Fatalf("CharacterId = %d, want 4242", cmd.CharacterId)
	}
	if cmd.Body.ChannelId != channel.Id(3) {
		t.Fatalf("Body.ChannelId = %d, want 3", cmd.Body.ChannelId)
	}
	if cmd.Body.Amount != 0xFFFF {
		t.Fatalf("Body.Amount = %d, want 65535", cmd.Body.Amount)
	}
}

func TestRequestChangeMesoCommandProvider(t *testing.T) {
	f := field.NewBuilder(2, 1, 100000000).Build()
	msgs, err := RequestChangeMesoCommandProvider(f, 42, 42, "SKILL", -1500)()
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages=%d, want 1", len(msgs))
	}
	var cmd messagechar.Command[messagechar.RequestChangeMesoBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatal(err)
	}
	if cmd.Type != messagechar.CommandRequestChangeMeso {
		t.Errorf("type=%s, want %s", cmd.Type, messagechar.CommandRequestChangeMeso)
	}
	if cmd.CharacterId != 42 || cmd.WorldId != 2 {
		t.Errorf("envelope=%+v", cmd)
	}
	if cmd.Body.Amount != -1500 || cmd.Body.ActorId != 42 || cmd.Body.ActorType != "SKILL" || cmd.Body.ShowEffect {
		t.Errorf("body=%+v", cmd.Body)
	}
}

func TestRedeemStoredExperienceCommandProvider(t *testing.T) {
	f := field.NewBuilder(world.Id(1), channel.Id(2), 100000000).Build()
	msgs, err := RedeemStoredExperienceCommandProvider(f, 1234)()
	if err != nil {
		t.Fatalf("provider err: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}

	wantKey := producer.CreateKey(1234)
	if string(msgs[0].Key) != string(wantKey) {
		t.Errorf("key = %v, want %v", msgs[0].Key, wantKey)
	}

	var cmd messagechar.Command[messagechar.RedeemStoredExperienceCommandBody]
	if uErr := json.Unmarshal(msgs[0].Value, &cmd); uErr != nil {
		t.Fatalf("unmarshal: %v", uErr)
	}
	if cmd.Type != messagechar.CommandRedeemStoredExperience {
		t.Errorf("Type = %q, want %q", cmd.Type, messagechar.CommandRedeemStoredExperience)
	}
	if cmd.CharacterId != 1234 {
		t.Errorf("CharacterId = %d, want 1234", cmd.CharacterId)
	}
	if cmd.WorldId != world.Id(1) {
		t.Errorf("WorldId = %d, want 1", cmd.WorldId)
	}
	if cmd.Body.ChannelId != channel.Id(2) {
		t.Errorf("Body.ChannelId = %d, want 2", cmd.Body.ChannelId)
	}
}
