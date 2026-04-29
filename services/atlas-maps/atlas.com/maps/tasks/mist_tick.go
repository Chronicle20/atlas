package tasks

import (
	"context"
	"sync"
	"time"

	"atlas-maps/kafka/message"
	mistKafka "atlas-maps/kafka/message/mist"
	"atlas-maps/kafka/producer"
	mapchar "atlas-maps/map/character"
	"atlas-maps/mist"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

const MistTickTask = "mist_tick_task"

// EnvCommandTopicCharacterBuff is the Kafka topic where APPLY-disease
// commands are published. Mirrors atlas-monsters' value (services
// communicate via topic-name only — no shared library import).
const EnvCommandTopicCharacterBuff = "COMMAND_TOPIC_CHARACTER_BUFF"

// PositionLookup resolves a character's current world coordinates. Injected
// as a seam so MistTick can be unit-tested without standing up the
// atlas-character REST client.
type PositionLookup func(ctx context.Context, characterId uint32) (x int16, y int16, err error)

// buffCommand is the Kafka envelope mirrored from atlas-monsters'
// disease.go. Defined locally to avoid a cross-service import.
type buffCommand[E any] struct {
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
	Instance    uuid.UUID  `json:"instance"`
	CharacterId uint32     `json:"characterId"`
	Type        string     `json:"type"`
	Body        E          `json:"body"`
}

type applyDiseaseBody struct {
	FromId   uint32       `json:"fromId"`
	SourceId int32        `json:"sourceId"`
	Level    byte         `json:"level"`
	Duration int32        `json:"duration"`
	Changes  []statChange `json:"changes"`
}

type statChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func applyDiseaseCommandProvider(m mist.Mist, characterId uint32) model.Provider[[]kafka.Message] {
	key := kafkaProducer.CreateKey(int(characterId))
	value := &buffCommand[applyDiseaseBody]{
		WorldId:     m.Field().WorldId(),
		ChannelId:   m.Field().ChannelId(),
		MapId:       m.Field().MapId(),
		Instance:    m.Field().Instance(),
		CharacterId: characterId,
		Type:        "APPLY",
		Body: applyDiseaseBody{
			FromId:   m.OwnerId(),
			SourceId: int32(m.SourceSkillId()),
			Level:    byte(m.SourceSkillLevel()),
			Duration: int32(m.DiseaseDuration() / time.Millisecond),
			Changes:  []statChange{{Type: m.Disease(), Amount: m.DiseaseValue()}},
		},
	}
	return kafkaProducer.SingleMessageProvider(key, value)
}

// MistTick is the periodic tick task that expires mists past their lifetime
// and re-applies the disease to characters currently inside the mist's
// bounding box. It is registered via tasks.Register in main.
type MistTick struct {
	l                logrus.FieldLogger
	interval         int
	posLookup        PositionLookup
	registry         *mist.Registry
	producerProvider func(ctx context.Context) producer.Provider
	processorFactory func(l logrus.FieldLogger, ctx context.Context, p producer.Provider, r *mist.Registry) mist.Processor
	charsInField     func(t tenant.Model, f field.Model) []uint32
}

// NewMistTick constructs a MistTick wired to the singleton mist registry
// and the standard producer provider. The supplied posLookup is the seam
// for fetching character world coordinates (atlas-character REST in
// production, fakes in tests).
func NewMistTick(l logrus.FieldLogger, interval int, posLookup PositionLookup) *MistTick {
	return &MistTick{
		l:         l,
		interval:  interval,
		posLookup: posLookup,
		registry:  mist.GetRegistry(),
		producerProvider: func(ctx context.Context) producer.Provider {
			return producer.ProviderImpl(l)(ctx)
		},
		processorFactory: mist.NewProcessorWithRegistry,
		charsInField: func(t tenant.Model, f field.Model) []uint32 {
			tctx := tenant.WithContext(context.Background(), t)
			ids, err := mapchar.NewProcessor(l, tctx).GetCharactersInMap(uuid.Nil, f)
			if err != nil {
				return nil
			}
			return ids
		},
	}
}

// Run is invoked once per tick by tasks.Register's loop. It fans out per
// tenant goroutines as described in FR-4.6.3.
func (r *MistTick) Run() {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(context.Background(), MistTickTask)
	defer span.End()
	r.runOnce(ctx)
}

// runOnce performs a single synchronous tick pass. Tests invoke this
// directly to deterministically observe the side effects without spawning
// goroutines.
func (r *MistTick) runOnce(ctx context.Context) {
	tenants := r.registry.GetTenants()
	var wg sync.WaitGroup
	for _, t := range tenants {
		t := t
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.processTenant(ctx, t)
		}()
	}
	wg.Wait()
}

func (r *MistTick) processTenant(ctx context.Context, t tenant.Model) {
	tctx := tenant.WithContext(ctx, t)
	prov := r.producerProvider(tctx)

	mists := r.registry.AllByTenant(t)
	for _, m := range mists {
		if m.Expired() {
			if _, err := r.processorFactory(r.l, tctx, prov, r.registry).Destroy(m.Id(), mistKafka.ReasonExpired); err != nil {
				r.l.WithError(err).Errorf("MistTick: failed to destroy expired mist [%s].", m.Id())
			}
			continue
		}
		if !m.ShouldTick() {
			continue
		}
		members := r.charsInField(t, m.Field())
		if len(members) == 0 {
			r.registry.UpdateLastTick(t, m.Id(), time.Now())
			continue
		}
		emitErr := message.Emit(prov)(func(buf *message.Buffer) error {
			for _, cid := range members {
				x, y, err := r.posLookup(tctx, cid)
				if err != nil {
					r.l.WithError(err).Debugf("MistTick: position fetch failed for character [%d].", cid)
					continue
				}
				if !m.Contains(x, y) {
					continue
				}
				if err := buf.Put(EnvCommandTopicCharacterBuff, applyDiseaseCommandProvider(m, cid)); err != nil {
					return err
				}
			}
			return nil
		})
		if emitErr != nil {
			r.l.WithError(emitErr).Errorf("MistTick: failed to emit apply-disease for mist [%s].", m.Id())
		}
		r.registry.UpdateLastTick(t, m.Id(), time.Now())
	}
}

// SleepTime reports the configured tick interval.
func (r *MistTick) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}

