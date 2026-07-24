package mount

import (
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"

	msg "atlas-mounts/kafka/message/mount"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func resolveSingle(t *testing.T, p model.Provider[[]kafka.Message]) kafka.Message {
	t.Helper()
	msgs, err := p()
	if err != nil {
		t.Fatalf("unexpected provider error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected exactly 1 message, got %d", len(msgs))
	}
	return msgs[0]
}

func TestEventProvidersKeyAndType(t *testing.T) {
	const characterId uint32 = 42
	wid := world.Id(1)
	body := msg.StatusEventBody{Level: 3, Exp: 150}

	cases := []struct {
		name     string
		provider model.Provider[[]kafka.Message]
		wantType string
	}{
		{"set", setEventProvider(wid, characterId, body), msg.StatusEventTypeSet},
		{"tick", tickEventProvider(wid, characterId, body), msg.StatusEventTypeTick},
		{"feed", feedEventProvider(wid, characterId, body), msg.StatusEventTypeFeed},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := resolveSingle(t, c.provider)

			if len(m.Key) != 8 {
				t.Fatalf("expected 8-byte key, got %d bytes", len(m.Key))
			}
			if got := binary.LittleEndian.Uint32(m.Key); got != characterId {
				t.Errorf("expected key to encode characterId %d, got %d", characterId, got)
			}

			got := string(m.Value)
			for _, want := range []string{
				`"worldId":1`,
				`"characterId":42`,
				`"type":"` + c.wantType + `"`,
			} {
				if !strings.Contains(got, want) {
					t.Errorf("expected marshaled JSON to contain %q, got %s", want, got)
				}
			}

			var decoded msg.StatusEvent[msg.StatusEventBody]
			if err := json.Unmarshal(m.Value, &decoded); err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if decoded.Type != c.wantType {
				t.Errorf("expected decoded type %q, got %q", c.wantType, decoded.Type)
			}
		})
	}
}
