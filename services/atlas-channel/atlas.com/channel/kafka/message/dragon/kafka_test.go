package dragon

import (
	"encoding/json"
	"testing"
)

// The producer-side JSON is pinned literally here rather than imported from
// atlas-dragons: the two contracts live in separate modules, so importing one
// into the other's test would defeat the purpose. If atlas-dragons changes a
// json tag, this test fails and the mirror gets updated deliberately.
//
// Producer definitions: services/atlas-dragons/atlas.com/dragons/dragon/kafka.go
// and .../kafka/consumer/dragon/kafka.go.
func TestCreatedEventMirrorsProducerTags(t *testing.T) {
	raw := []byte(`{
		"worldId": 0, "channelId": 1, "mapId": 100000000,
		"instance": "00000000-0000-0000-0000-000000000000",
		"ownerCharacterId": 4242, "type": "CREATED",
		"body": {"x": 70000, "y": -70000, "stance": 3, "jobId": 2214}
	}`)

	var e StatusEvent[StatusEventCreatedBody]
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatal(err)
	}
	if e.OwnerCharacterId != 4242 || e.Type != EventDragonStatusCreated ||
		e.WorldId != 0 || e.ChannelId != 1 || e.MapId != 100000000 {
		t.Fatalf("envelope fields did not survive: %+v", e)
	}
	if e.Body.X != 70000 || e.Body.Y != -70000 || e.Body.Stance != 3 || e.Body.JobId != 2214 {
		t.Fatalf("created body did not survive: %+v", e.Body)
	}
}

func TestMovedEventMirrorsProducerTags(t *testing.T) {
	raw := []byte(`{
		"worldId": 0, "channelId": 1, "mapId": 100000000,
		"instance": "00000000-0000-0000-0000-000000000000",
		"ownerCharacterId": 4242, "type": "MOVED",
		"body": {"rawMovement": "AQID"}
	}`)

	var e StatusEvent[StatusEventMovedBody]
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatal(err)
	}
	if e.Type != EventDragonStatusMoved || len(e.Body.RawMovement) != 3 {
		t.Fatalf("moved body did not survive: %+v", e)
	}
}

func TestDestroyedEventMirrorsProducerTags(t *testing.T) {
	raw := []byte(`{
		"worldId": 0, "channelId": 1, "mapId": 100000000,
		"instance": "00000000-0000-0000-0000-000000000000",
		"ownerCharacterId": 4242, "type": "DESTROYED", "body": {}
	}`)

	var e StatusEvent[StatusEventDestroyedBody]
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatal(err)
	}
	if e.Type != EventDragonStatusDestroyed || e.OwnerCharacterId != 4242 {
		t.Fatalf("destroyed envelope did not survive: %+v", e)
	}
}

// The command direction is channel -> dragons, so this asserts what the
// CONSUMER in atlas-dragons will see when it decodes what we produce.
func TestMoveCommandMirrorsConsumerTags(t *testing.T) {
	c := Command[MoveCommandBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000,
		Type: CommandTypeMove,
		Body: MoveCommandBody{CharacterId: 4242, StartX: 100, StartY: -200, Stance: 3, RawMovement: []byte{1, 2, 3}},
	}
	b, err := json.Marshal(&c)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	body, ok := m["body"].(map[string]interface{})
	if !ok {
		t.Fatalf("no body object: %s", b)
	}
	for _, k := range []string{"characterId", "startX", "startY", "stance", "rawMovement"} {
		if _, present := body[k]; !present {
			t.Errorf("MoveCommandBody must serialize a %q key (atlas-dragons decodes it)", k)
		}
	}
}
