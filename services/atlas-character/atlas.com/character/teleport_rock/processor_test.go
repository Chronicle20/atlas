package teleport_rock

import (
	"atlas-character/kafka/message"
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	teleportrock2 "atlas-character/kafka/message/teleportrock"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestMain installs a no-op producer writer for the whole package. Add/Remove
// (unlike the buffer-taking AddMap/RemoveMap) drive message.Emit with the
// real producer.ProviderImpl, which would otherwise dial the (unreachable in
// tests) BOOTSTRAP_SERVERS broker and retry for ~42s per call before failing.
// See libs/atlas-kafka/producer/producertest for the documented convention.
func TestMain(m *testing.M) {
	producertest.InstallNoop()
	os.Exit(m.Run())
}

func testContext(t *testing.T) context.Context {
	m, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), m)
}

// bufferedTypes extracts the status-event types buffered on a topic so tests
// can assert LIST_UPDATED vs ERROR without a live producer. mb.GetAll()
// (atlas-character/kafka/message.Buffer) returns a map keyed by topic env
// string to the []kafka.Message buffered for that topic.
func addMap(t *testing.T, p Processor, mb *message.Buffer, characterId uint32, mapId _map.Id, vip bool) {
	t.Helper()
	if err := p.AddMap(mb)(uuid.New(), 0, characterId, mapId, vip); err != nil {
		t.Fatalf("AddMap: %v", err)
	}
}

func assertBuffered(t *testing.T, mb *message.Buffer, wantType string, wantReason string) {
	t.Helper()
	all := mb.GetAll()
	msgs := all[teleportrock2.EnvEventTopicStatus]
	var matches int
	for _, km := range msgs {
		var ev teleportrock2.StatusEvent[json.RawMessage]
		if err := json.Unmarshal(km.Value, &ev); err != nil {
			t.Fatalf("unmarshal status event: %v", err)
		}
		if ev.Type != wantType {
			continue
		}
		matches++
		if wantReason != "" {
			var body teleportrock2.ErrorStatusBody
			if err := json.Unmarshal(ev.Body, &body); err != nil {
				t.Fatalf("unmarshal error body: %v", err)
			}
			if body.Reason != wantReason {
				t.Fatalf("reason: got %q want %q", body.Reason, wantReason)
			}
		}
	}
	if matches != 1 {
		t.Fatalf("expected exactly one %s event, got %d (buffer: %+v)", wantType, matches, all)
	}
}

func TestAddMapPersistsAndBuffersListUpdated(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	mb := message.NewBuffer()
	addMap(t, p, mb, 42, 100000000, false)

	m, err := p.GetByCharacterId(42)
	if err != nil {
		t.Fatalf("GetByCharacterId: %v", err)
	}
	if len(m.Regular()) != 1 || m.Regular()[0] != 100000000 {
		t.Fatalf("regular list: %v", m.Regular())
	}
	assertBuffered(t, mb, teleportrock2.StatusEventTypeListUpdated, "")
}

func TestAddMapRejectsIneligible(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	for _, mapId := range []_map.Id{4000000, 909000000} { // sub-9-digit; x09 block
		mb := message.NewBuffer()
		addMap(t, p, mb, 42, mapId, false)
		assertBuffered(t, mb, teleportrock2.StatusEventTypeError, teleportrock2.ErrorReasonMapNotAllowed)
	}
	m, _ := p.GetByCharacterId(42)
	if len(m.Regular()) != 0 {
		t.Fatalf("nothing should persist: %v", m.Regular())
	}
}

func TestAddMapRejectsDuplicate(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	addMap(t, p, message.NewBuffer(), 42, 100000000, false)
	mb := message.NewBuffer()
	addMap(t, p, mb, 42, 100000000, false)
	assertBuffered(t, mb, teleportrock2.StatusEventTypeError, teleportrock2.ErrorReasonDuplicate)
}

func TestAddMapRejectsWhenFull(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	maps := []_map.Id{100000000, 101000000, 102000000, 103000000, 104000000}
	for _, m := range maps {
		addMap(t, p, message.NewBuffer(), 42, m, false)
	}
	mb := message.NewBuffer()
	addMap(t, p, mb, 42, 105000000, false)
	assertBuffered(t, mb, teleportrock2.StatusEventTypeError, teleportrock2.ErrorReasonListFull)

	// VIP list is independent: same character can still add there (10 cap).
	mb = message.NewBuffer()
	addMap(t, p, mb, 42, 105000000, true)
	assertBuffered(t, mb, teleportrock2.StatusEventTypeListUpdated, "")
}

func TestRemoveMapCompactsAndBuffersListUpdated(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	for _, m := range []_map.Id{100000000, 101000000, 102000000} {
		addMap(t, p, message.NewBuffer(), 42, m, false)
	}
	mb := message.NewBuffer()
	if err := p.RemoveMap(mb)(uuid.New(), 0, 42, 101000000, false); err != nil {
		t.Fatalf("RemoveMap: %v", err)
	}
	m, _ := p.GetByCharacterId(42)
	if len(m.Regular()) != 2 || m.Regular()[0] != 100000000 || m.Regular()[1] != 102000000 {
		t.Fatalf("compaction failed: %v", m.Regular())
	}
	assertBuffered(t, mb, teleportrock2.StatusEventTypeListUpdated, "")
}

func TestRemoveMapNotFound(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	mb := message.NewBuffer()
	if err := p.RemoveMap(mb)(uuid.New(), 0, 42, 100000000, false); err != nil {
		t.Fatalf("RemoveMap: %v", err)
	}
	assertBuffered(t, mb, teleportrock2.StatusEventTypeError, teleportrock2.ErrorReasonNotFound)
}

func TestAddReturnsUpdatedModelOnSuccess(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	m, err := p.Add(uuid.New(), 0, 42, 100000000, false)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(m.Regular()) != 1 || m.Regular()[0] != 100000000 {
		t.Fatalf("regular list = %v, want [100000000]", m.Regular())
	}
}

func TestAddReturnsTypedErrors(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	// ineligible map (0 fails EligibleForRegistration)
	if _, err := p.Add(uuid.New(), 0, 42, 0, false); !errors.Is(err, ErrMapNotAllowed) {
		t.Fatalf("ineligible: got %v, want ErrMapNotAllowed", err)
	}
	// duplicate
	if _, err := p.Add(uuid.New(), 0, 42, 100000000, false); err != nil {
		t.Fatalf("seed add: %v", err)
	}
	if _, err := p.Add(uuid.New(), 0, 42, 100000000, false); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate: got %v, want ErrDuplicate", err)
	}
	// remove-not-present
	if _, err := p.Remove(uuid.New(), 0, 42, 200000000, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("remove absent: got %v, want ErrNotFound", err)
	}
}

func TestAddReturnsListFull(t *testing.T) {
	db := testDatabase(t)
	ctx := testContext(t)
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db)

	// Fill the regular list to capacity (5) with eligible maps.
	for _, mid := range []_map.Id{100000000, 101000000, 102000000, 103000000, 104000000} {
		if _, err := p.Add(uuid.New(), 0, 42, mid, false); err != nil {
			t.Fatalf("seed add %d: %v", mid, err)
		}
	}
	if _, err := p.Add(uuid.New(), 0, 42, 105000000, false); !errors.Is(err, ErrListFull) {
		t.Fatalf("full: got %v, want ErrListFull", err)
	}
}
