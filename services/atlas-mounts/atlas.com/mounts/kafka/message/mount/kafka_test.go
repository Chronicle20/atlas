package mount

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestStatusEventMarshal(t *testing.T) {
	e := StatusEvent[StatusEventBody]{
		WorldId:     world.Id(1),
		CharacterId: 42,
		Type:        StatusEventTypeFeed,
		Body: StatusEventBody{
			Level:     3,
			Exp:       150,
			Tiredness: 0,
			LevelUp:   true,
			TooTired:  false,
		},
	}

	b, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	got := string(b)
	for _, want := range []string{
		`"worldId":1`,
		`"characterId":42`,
		`"type":"FEED"`,
		`"level":3`,
		`"exp":150`,
		`"tiredness":0`,
		`"levelUp":true`,
		`"tooTired":false`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected marshaled JSON to contain %q, got %s", want, got)
		}
	}
}

func TestStatusEventTypeConstants(t *testing.T) {
	cases := map[string]string{
		StatusEventTypeSet:  "SET",
		StatusEventTypeTick: "TICK",
		StatusEventTypeFeed: "FEED",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("expected type constant %q, got %q", want, got)
		}
	}
	if EnvStatusEventTopic != "EVENT_TOPIC_MOUNT_STATUS" {
		t.Errorf("unexpected topic env const: %s", EnvStatusEventTopic)
	}
}
