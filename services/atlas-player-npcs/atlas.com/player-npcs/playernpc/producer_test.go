package playernpc

import (
	msg "atlas-player-npcs/kafka/message/playernpc"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

var emitted *producertest.Capture

func TestMain(m *testing.M) {
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
}

func buildEmitterTestNpc(t *testing.T, id uuid.UUID, characterId uint32) Model {
	t.Helper()
	m, err := NewBuilder().
		SetId(id).
		SetCharacterId(characterId).
		SetName("Statue").
		SetWorldId(0).
		SetMapId(102000004).
		SetScriptId(1).
		SetObjectId(7).
		SetX(100).
		SetCy(200).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected err = %v", err)
	}
	return m
}

func TestNewEmitter(t *testing.T) {
	l := testLogger()
	ctx := context.Background()
	emit := NewEmitter(l, ctx)

	t.Run("DEPLOYED publishes the full resource to the status topic", func(t *testing.T) {
		emitted.Reset()
		m := buildEmitterTestNpc(t, uuid.New(), 42)
		emit(Event{Type: EventTypeDeployed, WorldId: 0, MapId: 102000004, Models: []Model{m}})

		msgs := emitted.Messages(msg.EnvEventTopicStatus)
		if len(msgs) != 1 {
			t.Fatalf("emitted messages = %d, want 1", len(msgs))
		}
		var e msg.StatusEvent[msg.StatusModel]
		if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if e.Type != msg.EventTypeDeployed || e.Body.Id != m.Id() || e.Body.ObjectId != 7 {
			t.Errorf("DEPLOYED event = %+v", e)
		}
	})

	t.Run("REMOVED publishes id/objectId/mapId/worldId", func(t *testing.T) {
		emitted.Reset()
		m := buildEmitterTestNpc(t, uuid.New(), 42)
		emit(Event{Type: EventTypeRemoved, WorldId: 0, MapId: 102000004, Models: []Model{m}})

		msgs := emitted.Messages(msg.EnvEventTopicStatus)
		if len(msgs) != 1 {
			t.Fatalf("emitted messages = %d, want 1", len(msgs))
		}
		var e msg.StatusEvent[msg.StatusRemovedBody]
		if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if e.Type != msg.EventTypeRemoved || e.Body.Id != m.Id() || e.Body.ObjectId != 7 || e.Body.MapId != 102000004 {
			t.Errorf("REMOVED event = %+v", e)
		}
	})

	t.Run("REPOSITIONED carries every occupant in one message", func(t *testing.T) {
		emitted.Reset()
		m1 := buildEmitterTestNpc(t, uuid.New(), 1)
		m2 := buildEmitterTestNpc(t, uuid.New(), 2)
		emit(Event{Type: EventTypeRepositioned, WorldId: 0, MapId: 102000004, Models: []Model{m1, m2}})

		msgs := emitted.Messages(msg.EnvEventTopicStatus)
		if len(msgs) != 1 {
			t.Fatalf("emitted messages = %d, want 1", len(msgs))
		}
		var e msg.StatusEvent[msg.StatusRepositionedBody]
		if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if e.Type != msg.EventTypeRepositioned || len(e.Body.Npcs) != 2 {
			t.Errorf("REPOSITIONED event = %+v", e)
		}
	})

	t.Run("unknown event type does not publish and does not panic", func(t *testing.T) {
		emitted.Reset()
		emit(Event{Type: "BOGUS"})
		if msgs := emitted.Messages(msg.EnvEventTopicStatus); len(msgs) != 0 {
			t.Errorf("emitted messages = %d, want 0", len(msgs))
		}
	})
}
