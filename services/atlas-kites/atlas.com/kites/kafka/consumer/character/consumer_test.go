package character

import (
	"atlas-kites/character"
	character2 "atlas-kites/kafka/message/character"
	kiteMsg "atlas-kites/kafka/message/kite"
	"atlas-kites/kite"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// recorder is a producer.Provider that captures emitted messages per topic.
// Duplicated from kite/processor_test.go's recorder (~25 lines) rather than
// promoted to an exported test-only seam on the kite package -- an earlier
// task in this plan had an exported test-only seam rejected in review as the
// anti-pattern the project's *_testhelpers.go ban exists to prevent.
type recorder struct {
	mu   sync.Mutex
	msgs map[string][]kafka.Message
	fail bool
}

func newRecorder() *recorder { return &recorder{msgs: make(map[string][]kafka.Message)} }

func (r *recorder) provider() producer.Provider {
	return func(t string) producer.MessageProducer {
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

func (r *recorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = make(map[string][]kafka.Message)
}

func (r *recorder) messages(topic string) []kafka.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.msgs[topic]
}

// writerFactory lets recorder double as the process-wide producer.Manager's
// WriterFactory. handleStatusEventMapChanged and friends build their
// kite.Processor via kite.NewProcessor, which resolves the real singleton
// producer rather than accepting an injected producer.Provider -- unlike
// kite/processor_test.go, where every Processor under test is built through
// the NewProcessorWithProvider seam. Capturing what the consumer emits
// therefore requires intercepting one layer down, at the Manager/Writer
// level, installed via producer.ConfigWriterFactory. The env-var token and
// the resolved topic name coincide in tests (topic.EnvProvider falls back to
// the raw token when the env var is unset), so both capture paths land in
// the same r.msgs map under the same key.
func (r *recorder) writerFactory() producer.WriterFactory {
	return func(topicName string) producer.Writer {
		return &recorderWriter{rec: r, topic: topicName}
	}
}

type recorderWriter struct {
	rec   *recorder
	topic string
}

func (w *recorderWriter) Topic() string { return w.topic }

func (w *recorderWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	w.rec.mu.Lock()
	defer w.rec.mu.Unlock()
	w.rec.msgs[w.topic] = append(w.rec.msgs[w.topic], msgs...)
	return nil
}

func (w *recorderWriter) Close() error { return nil }

func setup(t *testing.T) context.Context {
	t.Helper()
	s := miniredis.RunT(t)
	c := goredis.NewClient(&goredis.Options{Addr: s.Addr()})
	kite.InitRegistry(c)
	character.InitRegistry(c)
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

func nullLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

// A map change must destroy the kite against the OLD field, instance included.
// Capturing `of` before the index transition is what keeps the DESTROYED event
// from fanning out to the map the character just walked into.
func TestMapChangedDestroysAgainstOldFieldWithInstance(t *testing.T) {
	ctx := setup(t)
	l := nullLogger()
	rec := newRecorder() // same recording producer.Provider as kite/processor_test.go

	// handleStatusEventMapChanged's DestroyAndEmit call builds its Processor
	// via kite.NewProcessor, i.e. the real singleton producer.Manager -- so
	// rec must also be installed as that Manager's WriterFactory to observe
	// what the handler (as opposed to the seed call below) actually emits.
	producer.ResetInstance()
	producer.GetManager(producer.ConfigWriterFactory(rec.writerFactory()))

	oldInst := uuid.New()
	newInst := uuid.New()
	of := field.NewBuilder(0, 1, 104040000).SetInstance(oldInst).Build()

	if _, err := kite.NewProcessorWithProvider(l, ctx, rec.provider()).
		CreateAndEmit(of, 42, kiteMsg.CreateCommandBody{Name: "Player", TemplateId: 5080000, Message: "hi", X: 320, Y: -140}); err != nil {
		t.Fatalf("seed kite: %v", err)
	}
	rec.reset()

	handleStatusEventMapChanged(l, ctx, character2.StatusEvent[character2.StatusEventMapChangedBody]{
		WorldId:     0,
		CharacterId: 42,
		Type:        character2.EventCharacterStatusTypeMapChanged,
		Body: character2.StatusEventMapChangedBody{
			ChannelId:      1,
			OldMapId:       104040000,
			OldInstance:    oldInst,
			TargetMapId:    104040001,
			TargetInstance: newInst,
		},
	})

	if _, err := kite.NewProcessor(l, ctx).GetByCharacterId(42); err == nil {
		t.Error("kite survived the owner's map change")
	}

	msgs := rec.messages(kiteMsg.EnvEventTopicStatus)
	if len(msgs) != 1 {
		t.Fatalf("emitted %d status events, want 1 (DESTROYED)", len(msgs))
	}
	var ev kiteMsg.StatusEvent[kiteMsg.DestroyedStatusEventBody]
	if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.Type != kiteMsg.EventTopicStatusTypeDestroyed {
		t.Errorf("event type = %s, want DESTROYED", ev.Type)
	}
	if ev.MapId != 104040000 {
		t.Errorf("DESTROYED fanned out to map %d, want the OLD map 104040000", ev.MapId)
	}
	if ev.Instance != oldInst {
		t.Errorf("DESTROYED instance = %s, want the OLD instance %s", ev.Instance, oldInst)
	}
	if ev.Body.Reason != kiteMsg.DestroyReasonOwnerLeft {
		t.Errorf("reason = %s, want OWNER_LEFT", ev.Body.Reason)
	}
}

// The character-in-field index must key on the instance, so a character in an
// instanced copy of a map is not visible in the base copy's set. The chalkboards
// consumer this is modelled on drops the instance here and its instanced maps
// have therefore never replayed.
func TestLoginIndexesWithInstance(t *testing.T) {
	ctx := setup(t)
	l := nullLogger()

	inst := uuid.New()
	handleStatusEventLogin(l, ctx, character2.StatusEvent[character2.StatusEventLoginBody]{
		WorldId:     0,
		CharacterId: 42,
		Type:        character2.EventCharacterStatusTypeLogin,
		Body:        character2.StatusEventLoginBody{ChannelId: 1, MapId: 104040000, Instance: inst},
	})

	instanced := field.NewBuilder(0, 1, 104040000).SetInstance(inst).Build()
	base := field.NewBuilder(0, 1, 104040000).SetInstance(uuid.Nil).Build()

	got, err := character.NewProcessor(l, ctx).GetCharactersInMap(instanced)
	if err != nil {
		t.Fatalf("GetCharactersInMap(instanced): %v", err)
	}
	if len(got) != 1 || got[0] != 42 {
		t.Errorf("instanced field = %v, want [42]", got)
	}

	got, err = character.NewProcessor(l, ctx).GetCharactersInMap(base)
	if err != nil {
		t.Fatalf("GetCharactersInMap(base): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("base field saw the instanced character: %v", got)
	}
}
