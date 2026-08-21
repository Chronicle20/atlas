package system_message

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

func TestShowHintCommandProvider_WireShape(t *testing.T) {
	ch := channel.NewModel(world.Id(0), channel.Id(1))
	tid := uuid.MustParse("11111111-2222-3333-4444-555555555555")

	msgs, err := ShowHintCommandProvider(tid, ch, 12345, "hint text", 0, 0)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	if !bytes.Equal(msgs[0].Key, producer.CreateKey(12345)) {
		t.Fatalf("unexpected key: %v", msgs[0].Key)
	}

	var cmd Command[ShowHintBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if cmd.TransactionId != tid {
		t.Errorf("expected transactionId %s, got %s", tid, cmd.TransactionId)
	}
	if cmd.WorldId != world.Id(0) {
		t.Errorf("expected worldId 0, got %d", cmd.WorldId)
	}
	if cmd.ChannelId != channel.Id(1) {
		t.Errorf("expected channelId 1, got %d", cmd.ChannelId)
	}
	if cmd.CharacterId != 12345 {
		t.Errorf("expected characterId 12345, got %d", cmd.CharacterId)
	}
	if cmd.Type != CommandShowHint {
		t.Errorf("expected type %s, got %s", CommandShowHint, cmd.Type)
	}
	if cmd.Body.Hint != "hint text" {
		t.Errorf("expected body.hint %q, got %q", "hint text", cmd.Body.Hint)
	}
	if cmd.Body.Width != 0 {
		t.Errorf("expected body.width 0, got %d", cmd.Body.Width)
	}
	if cmd.Body.Height != 0 {
		t.Errorf("expected body.height 0, got %d", cmd.Body.Height)
	}

	var raw map[string]any
	if err := json.Unmarshal(msgs[0].Value, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	for _, key := range []string{"transactionId", "worldId", "channelId", "characterId", "type", "body"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected key %q in raw JSON", key)
		}
	}
	body, ok := raw["body"].(map[string]any)
	if !ok {
		t.Fatalf("expected body to be an object, got %T", raw["body"])
	}
	for _, key := range []string{"hint", "width", "height"} {
		if _, ok := body[key]; !ok {
			t.Errorf("expected key %q in body", key)
		}
	}
}
