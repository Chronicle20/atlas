package kite

import (
	"atlas-kites/character"
	"atlas-kites/configuration"
	kiteMsg "atlas-kites/kafka/message/kite"
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
)

// discardLogger returns a real logrus.FieldLogger whose output is captured
// (and discarded) rather than nil. NewProcessorWithProvider is production
// code and must not special-case a nil logger -- passing an actual logger
// here means Create's refusal/contention logging runs for real in every
// test, the same as it would in production.
func discardLogger() logrus.FieldLogger {
	l, _ := logrustest.NewNullLogger()
	return l
}

// testProcessor builds a Processor against ctx's tenant with rec as the
// producer.Provider and cfg as the fixed tenant config, bypassing the real
// configuration.GetRegistry() (which would otherwise attempt a live
// atlas-tenants HTTP call). It uses the package-internal
// newProcessorWithConfig seam directly -- no exported, process-wide
// singleton mutator is added to the configuration package for this.
func testProcessor(ctx context.Context, rec *recorder, cfg configuration.Model) Processor {
	return newProcessorWithConfig(discardLogger(), ctx, rec.provider(), func(logrus.FieldLogger, context.Context, uuid.UUID) configuration.Model {
		return cfg
	})
}

// recorder is a producer.Provider that captures emitted messages per topic.
type recorder struct {
	mu   sync.Mutex
	msgs map[topic.Token][]kafka.Message
	fail bool
}

func newRecorder() *recorder { return &recorder{msgs: make(map[topic.Token][]kafka.Message)} }

func (r *recorder) provider() producer.Provider {
	return func(t topic.Token) producer.MessageProducer {
		return func(p model.Provider[[]kafka.Message]) error {
			if r.fail {
				return errors.New("emit failed")
			}
			ms, err := p()
			if err != nil {
				return err
			}
			r.mu.Lock()
			defer r.mu.Unlock()
			r.msgs[t] = append(r.msgs[t], ms...)
			return nil
		}
	}
}

func (r *recorder) count(t topic.Token) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.msgs[t])
}

func body() kiteMsg.CreateCommandBody {
	return kiteMsg.CreateCommandBody{Name: "Player", TemplateId: 5080000, Message: "congrats!", X: 320, Y: -140}
}

func TestCreateSucceedsAndEmitsCreated(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)
	rec := newRecorder()

	m, err := testProcessor(ctx, rec, configuration.DefaultConfig()).CreateAndEmit(testField(), 42, body())
	if err != nil {
		t.Fatalf("CreateAndEmit: %v", err)
	}
	if m.Id() == 0 || m.X() != 320 || m.Y() != -140 || m.Name() != "Player" {
		t.Errorf("unexpected model: %+v", m)
	}
	if rec.count(kiteMsg.EnvEventTopicStatus) != 1 {
		t.Errorf("emitted %d status events, want 1", rec.count(kiteMsg.EnvEventTopicStatus))
	}
	if ok, _ := getRegistry().Exists(ctx, 42); !ok {
		t.Error("kite not in registry after create")
	}
}

func TestCreateRejectsSecondKiteForSameCharacter(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)
	rec := newRecorder()
	p := testProcessor(ctx, rec, configuration.DefaultConfig())

	if _, err := p.CreateAndEmit(testField(), 42, body()); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := p.CreateAndEmit(testField(), 42, body())
	if !errors.Is(err, ErrAlreadyPlaced) {
		t.Fatalf("second create err = %v, want ErrAlreadyPlaced", err)
	}
	// One CREATED + one CREATION_FAILED.
	if rec.count(kiteMsg.EnvEventTopicStatus) != 2 {
		t.Errorf("emitted %d status events, want 2", rec.count(kiteMsg.EnvEventTopicStatus))
	}
}

func TestCreateRejectsBlockedMap(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)
	rec := newRecorder()

	fm := fieldWithMap(910000000)
	_, err := testProcessor(ctx, rec, configuration.DefaultConfig()).CreateAndEmit(fm, 42, body())
	if !errors.Is(err, ErrMapForbidden) {
		t.Fatalf("err = %v, want ErrMapForbidden", err)
	}
	if ok, _ := getRegistry().Exists(ctx, 42); ok {
		t.Error("a refused create must not insert into the registry")
	}
}

func TestCreateRejectsOverlongMessage(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)
	rec := newRecorder()

	b := body()
	b.Message = string(make([]byte, 183))
	_, err := testProcessor(ctx, rec, configuration.DefaultConfig()).CreateAndEmit(testField(), 42, b)
	if !errors.Is(err, ErrMessageTooLong) {
		t.Fatalf("err = %v, want ErrMessageTooLong", err)
	}
}

func TestCreateRollsBackRegistryWhenEmitFails(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)
	rec := newRecorder()
	rec.fail = true

	if _, err := testProcessor(ctx, rec, configuration.DefaultConfig()).CreateAndEmit(testField(), 42, body()); err == nil {
		t.Fatal("CreateAndEmit should fail when the emit fails")
	}
	if ok, _ := getRegistry().Exists(ctx, 42); ok {
		t.Error("registry must not retain a kite whose CREATED event was never emitted")
	}
}

func TestDestroyRemovesAndEmits(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)
	rec := newRecorder()
	p := testProcessor(ctx, rec, configuration.DefaultConfig())

	if _, err := p.CreateAndEmit(testField(), 42, body()); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := p.DestroyAndEmit(42, kiteMsg.DestroyReasonOwnerLeft); err != nil {
		t.Fatalf("DestroyAndEmit: %v", err)
	}
	if ok, _ := getRegistry().Exists(ctx, 42); ok {
		t.Error("kite still present after destroy")
	}
	if rec.count(kiteMsg.EnvEventTopicStatus) != 2 {
		t.Errorf("emitted %d status events, want 2 (created + destroyed)", rec.count(kiteMsg.EnvEventTopicStatus))
	}
}

// TestConcurrentCreateEnforcesPerMapCapAcrossCharacters is the FR-5.2 proof:
// the command topic is keyed on characterId, so one character's own commands
// are already totally ordered by partition -- the per-map cap is the one
// invariant that is NOT safe by construction, because two DIFFERENT
// characters placing into the same full-but-for-one map land on different
// partitions and could otherwise both observe count < maxPerMap before either
// inserts. With maxPerMap: 1 and two characters racing, exactly one must
// succeed and the other must see ErrMapFull -- never both succeeding (over
// cap) and never both refusing (lock starvation/bug).
func TestConcurrentCreateEnforcesPerMapCapAcrossCharacters(t *testing.T) {
	testRegistry(t)
	ctx, _ := testContext(t)

	f := testField()
	l := discardLogger()

	// Both characters must already be tracked as "in the field" for
	// InMapModelProvider's character-index x kite-ownership composition to
	// count the winner's kite when the loser's cap check runs.
	cp := character.NewProcessor(l, ctx)
	cp.Enter(f, 42)
	cp.Enter(f, 43)

	rec := newRecorder()
	p := newProcessorWithConfig(l, ctx, rec.provider(), func(logrus.FieldLogger, context.Context, uuid.UUID) configuration.Model {
		return configuration.Extract(configuration.RestModel{MaxPerMap: 1})
	})

	type outcome struct {
		characterId uint32
		err         error
	}
	results := make(chan outcome, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	for _, cid := range []uint32{42, 43} {
		cid := cid
		routine.Go(l, ctx, func(_ context.Context) {
			defer wg.Done()
			_, err := p.CreateAndEmit(f, cid, body())
			results <- outcome{characterId: cid, err: err}
		})
	}
	wg.Wait()
	close(results)

	var succeeded, mapFull int
	for r := range results {
		switch {
		case r.err == nil:
			succeeded++
		case errors.Is(r.err, ErrMapFull):
			mapFull++
		default:
			t.Errorf("character [%d]: unexpected error %v", r.characterId, r.err)
		}
	}
	if succeeded != 1 || mapFull != 1 {
		t.Fatalf("succeeded=%d mapFull=%d, want exactly 1 and 1", succeeded, mapFull)
	}
}
