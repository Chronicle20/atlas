package dragon

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func TestCreatedEventCarriesWideCoordinatesAndOwner(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	m := NewBuilder(4242).SetField(f).SetX(70000).SetY(-70000).SetStance(3).SetJobId(2214).Build()

	msgs, err := createdEventProvider(m)()
	if err != nil || len(msgs) != 1 {
		t.Fatalf("provider failed: %v %d", err, len(msgs))
	}

	var e StatusEvent[StatusEventCreatedBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatal(err)
	}
	if e.Type != EventDragonStatusCreated || e.OwnerCharacterId != 4242 {
		t.Fatalf("envelope mismatch: %+v", e)
	}
	if e.Body.X != 70000 || e.Body.Y != -70000 || e.Body.JobId != 2214 {
		t.Fatalf("body mismatch: %+v", e.Body)
	}
}

func TestMovedEventCarriesRawBlobOnly(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	m := NewBuilder(4242).SetField(f).Build()

	msgs, err := movedEventProvider(m, []byte{1, 2, 3})()
	if err != nil || len(msgs) != 1 {
		t.Fatalf("provider failed: %v %d", err, len(msgs))
	}
	var e StatusEvent[StatusEventMovedBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatal(err)
	}
	if e.Type != EventDragonStatusMoved || len(e.Body.RawMovement) != 3 {
		t.Fatalf("moved event mismatch: %+v", e)
	}
}

func TestDestroyedEventCarriesOwnerOnly(t *testing.T) {
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	m := NewBuilder(4242).SetField(f).Build()

	msgs, err := destroyedEventProvider(m)()
	if err != nil || len(msgs) != 1 {
		t.Fatalf("provider failed: %v %d", err, len(msgs))
	}
	var e StatusEvent[StatusEventDestroyedBody]
	if err := json.Unmarshal(msgs[0].Value, &e); err != nil {
		t.Fatal(err)
	}
	if e.Type != EventDragonStatusDestroyed || e.OwnerCharacterId != 4242 {
		t.Fatalf("destroyed event mismatch: %+v", e)
	}
}
