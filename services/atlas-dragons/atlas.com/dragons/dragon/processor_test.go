package dragon

import (
	"atlas-dragons/character"
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// stubCharacters returns a fixed character, or ErrNotFound when notFound is set.
type stubCharacters struct {
	m        character.Model
	notFound bool
	calls    int
}

func (s *stubCharacters) GetById(characterId uint32) (character.Model, error) {
	s.calls++
	if s.notFound {
		return character.Model{}, requests.ErrNotFound
	}
	return s.m, nil
}

type capturedEvent struct {
	topic string
}

func newTestProcessor(t *testing.T, cs *stubCharacters) (*ProcessorImpl, tenant.Model, context.Context, *[]capturedEvent) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	registry = newRegistry(rc) // package-level singleton used by the processor

	ten, err := tenant.Create(uuid.New(), "GMS", 95, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	var emitted []capturedEvent
	p := &ProcessorImpl{
		l: logrus.New(), ctx: ctx, t: ten,
		characters: cs,
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			if _, err := provider(); err != nil {
				return err
			}
			emitted = append(emitted, capturedEvent{topic: topic})
			return nil
		},
	}
	return p, ten, ctx, &emitted
}

// buildCharacter constructs the stub's return value through the character
// Builder added in Step 2 — no test-only constructor, no *_testhelpers.go.
func buildCharacter(t *testing.T, id uint32, jobId job.Id, x, y int16) character.Model {
	t.Helper()
	return character.NewBuilder(id).SetJobId(jobId).SetX(x).SetY(y).Build()
}

func TestCreateIsANoOpForANonDragonJob(t *testing.T) {
	cs := &stubCharacters{m: buildCharacter(t, 1, 2001, 10, 20)}
	p, ten, ctx, emitted := newTestProcessor(t, cs)

	if err := p.Create(testField(), 1); err != nil {
		t.Fatal(err)
	}
	if len(*emitted) != 0 {
		t.Fatalf("Evan beginner must not spawn a dragon, emitted %v", *emitted)
	}
	if ok, _ := GetRegistry().Exists(ctx, ten, 1); ok {
		t.Fatal("no dragon must be stored for a non-dragon job")
	}
}

func TestCreateStoresAndEmitsOnceThenIsIdempotent(t *testing.T) {
	cs := &stubCharacters{m: buildCharacter(t, 42, 2214, 100, -200)}
	p, ten, ctx, emitted := newTestProcessor(t, cs)

	if err := p.Create(testField(), 42); err != nil {
		t.Fatal(err)
	}
	if err := p.Create(testField(), 42); err != nil {
		t.Fatal(err)
	}

	if len(*emitted) != 1 {
		t.Fatalf("a redelivered CREATE must emit exactly one CREATED, got %d", len(*emitted))
	}
	m, err := GetRegistry().Get(ctx, ten, 42)
	if err != nil || m.X() != 100 || m.Y() != -200 || m.JobId() != 2214 {
		t.Fatalf("stored dragon mismatch: %v %+v", err, m)
	}
}

func TestCreateIsANoOpWhenTheCharacterIsGone(t *testing.T) {
	cs := &stubCharacters{notFound: true}
	p, _, _, emitted := newTestProcessor(t, cs)

	if err := p.Create(testField(), 99); err != nil {
		t.Fatalf("a 404 means the character is gone, not a fetch failure: %v", err)
	}
	if len(*emitted) != 0 {
		t.Fatalf("no event for a missing character, got %v", *emitted)
	}
}

func TestDestroyEmitsOnceAndIsANoOpForAnAbsentDragon(t *testing.T) {
	cs := &stubCharacters{m: buildCharacter(t, 42, 2214, 0, 0)}
	p, _, _, emitted := newTestProcessor(t, cs)
	_ = p.Create(testField(), 42)

	if err := p.Destroy(42); err != nil {
		t.Fatal(err)
	}
	if err := p.Destroy(42); err != nil {
		t.Fatalf("destroying an absent dragon must be a no-op, got %v", err)
	}
	// 1 CREATED + 1 DESTROYED
	if len(*emitted) != 2 {
		t.Fatalf("expected exactly one DESTROYED, total emitted %d", len(*emitted))
	}
}

func TestMoveWithoutADragonIsRejectedAndCreatesNothing(t *testing.T) {
	cs := &stubCharacters{m: buildCharacter(t, 42, 2214, 0, 0)}
	p, ten, ctx, emitted := newTestProcessor(t, cs)

	if err := p.Move(42, 5, 6, 1, []byte{1, 2, 3, 4}); err != nil {
		t.Fatalf("a move for a dragonless character must be a logged no-op, got %v", err)
	}
	if len(*emitted) != 0 {
		t.Fatalf("no MOVED event without a dragon, got %v", *emitted)
	}
	if ok, _ := GetRegistry().Exists(ctx, ten, 42); ok {
		t.Fatal("Move must not create a dragon as a side effect")
	}
}

func TestMoveUpdatesPositionAndEmits(t *testing.T) {
	cs := &stubCharacters{m: buildCharacter(t, 42, 2214, 0, 0)}
	p, ten, ctx, emitted := newTestProcessor(t, cs)
	_ = p.Create(testField(), 42)

	if err := p.Move(42, 111, -222, 4, []byte{9, 9}); err != nil {
		t.Fatal(err)
	}
	m, err := GetRegistry().Get(ctx, ten, 42)
	if err != nil || m.X() != 111 || m.Y() != -222 || m.Stance() != 4 {
		t.Fatalf("position not updated: %v %+v", err, m)
	}
	if len(*emitted) != 2 {
		t.Fatalf("expected CREATED + MOVED, got %d", len(*emitted))
	}
}
