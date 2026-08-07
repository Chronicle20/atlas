package tasks

import (
	"atlas-maps/kafka/message"
	"atlas-maps/mist"
	"atlas-maps/monster"
	"context"
	"strconv"
	"sync"
	"time"

	mistKafka "atlas-maps/kafka/message/mist"
	mapchar "atlas-maps/map/character"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const MistTickTask = "mist_tick_task"

// EnvCommandTopicCharacterBuff is the Kafka topic where APPLY-disease
// commands are published. Mirrors atlas-monsters' value (services
// communicate via topic-name only — no shared library import).
const EnvCommandTopicCharacterBuff = "COMMAND_TOPIC_CHARACTER_BUFF"

// EnvCommandTopicMonster is the Kafka topic where APPLY_STATUS commands are
// published. Mirrors atlas-channel's value (services communicate via
// topic-name only -- no shared library import).
const EnvCommandTopicMonster = "COMMAND_TOPIC_MONSTER"

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
	FromId   uint32 `json:"fromId"`
	SourceId int32  `json:"sourceId"`
	Level    byte   `json:"level"`
	// milliseconds — contract owner: atlas-buffs kafka/message/character/kafka.go (task-190)
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
			// MILLISECONDS. Contract owner:
			// services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go
			// (ApplyCommandBody.Duration). atlas-buffs has computed
			// expiresAt = now + duration*time.Millisecond since task-054
			// (197324e40, 2026-05-03).
			//
			// This REVERSES commit 11e07dfa7 ("mist tick publishes disease
			// duration in seconds"), which was correct against the pre-task-054
			// contract and was silently inverted by task-054 one day later. Do
			// not flip it back: tools/buff-duration-guard.sh fails CI on a
			// seconds-valued emitter. (task-190 FR-1.2 / FR-1.4)
			Duration: int32(m.DiseaseDuration().Milliseconds()),
			Changes:  []statChange{{Type: m.Disease(), Amount: m.DiseaseValue()}},
		},
	}
	return kafkaProducer.SingleMessageProvider(key, value)
}

// monsterCommand is the COMMAND_TOPIC_MONSTER envelope, mirrored from
// atlas-channel's kafka/message/monster/kafka.go Command[E].
type monsterCommand[E any] struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	MonsterId uint32     `json:"monsterId"`
	Type      string     `json:"type"`
	Body      E          `json:"body"`
}

// applyStatusBody is a byte-compatible mirror of atlas-channel's
// monster.ApplyStatusCommandBody (services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go:44-52),
// verified key-for-key and type-for-type against atlas-monsters' consumer
// body (services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go:53-61).
//
// COMMAND_TOPIC_MONSTER is a SHARED topic: every registered handler in
// atlas-monsters unmarshals every message on it, so a key that is
// same-named-but-narrower in a sibling command body produces decode-error
// spam on unrelated handlers (see the KILL/useSkill skillId byte-vs-uint32
// collision). Reuse the exact existing key set -- add nothing, rename
// nothing, retype nothing.
//
// Duration and TickInterval are MILLISECONDS.
type applyStatusBody struct {
	SourceType        string           `json:"sourceType"`
	SourceCharacterId uint32           `json:"sourceCharacterId"`
	SourceSkillId     uint32           `json:"sourceSkillId"`
	SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
	Statuses          map[string]int32 `json:"statuses"`
	Duration          uint32           `json:"duration"`
	TickInterval      uint32           `json:"tickInterval"`
}

// applyStatusCommandProvider builds one APPLY_STATUS command for a monster
// standing inside the mist. Keyed on the monster unique id so it lands on the
// same partition as every other command for that monster.
//
// The POISON magnitude is intentionally the mist's DiseaseValue, which is 0
// for a player-cast mist: atlas-monsters computes poison damage as
// maxHP/(70 - sourceSkillLevel) (monster/status_task.go calculatePoisonDamage)
// and never reads the magnitude for POISON. VENOM is the status that does.
func applyStatusCommandProvider(m mist.Mist, monsterUniqueId uint32) model.Provider[[]kafka.Message] {
	key := kafkaProducer.CreateKey(int(monsterUniqueId))
	value := &monsterCommand[applyStatusBody]{
		WorldId:   m.Field().WorldId(),
		ChannelId: m.Field().ChannelId(),
		MapId:     m.Field().MapId(),
		Instance:  m.Field().Instance(),
		MonsterId: monsterUniqueId,
		Type:      "APPLY_STATUS",
		Body: applyStatusBody{
			SourceType:        "PLAYER_SKILL",
			SourceCharacterId: m.OwnerId(),
			SourceSkillId:     m.SourceSkillId(),
			SourceSkillLevel:  m.SourceSkillLevel(),
			Statuses:          map[string]int32{m.Disease(): m.DiseaseValue()},
			Duration:          uint32(m.DiseaseDuration().Milliseconds()),
			TickInterval:      uint32(m.TickInterval().Milliseconds()),
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
	monstersInRect   func(ctx context.Context, m mist.Mist) ([]monster.RestModel, error)
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
		monstersInRect: func(ctx context.Context, m mist.Mist) ([]monster.RestModel, error) {
			x1, y1, x2, y2 := m.Rect()
			// limit 0 == no cap. ctx is already tenant-decorated by
			// processTenant, so the REST call is tenant-scoped (NFR-3).
			return monster.NewProcessor(l, ctx).GetInMapRect(m.Field(), x1, y1, x2, y2, 0)
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
		routine.Go(r.l, ctx, func(_ context.Context) {
			defer wg.Done()
			r.processTenant(ctx, t)
		})
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
		switch m.TargetKind() {
		case mistKafka.TargetKindMonster:
			r.tickMonsters(tctx, prov, t, m)
		default:
			// Empty target kind normalizes to CHARACTER in mist.Create; the
			// default arm also covers any mist built directly by a test.
			r.tickCharacters(tctx, prov, t, m)
		}
		r.registry.UpdateLastTick(t, m.Id(), time.Now())
	}
}

// tickCharacters applies the mist's status to every character in the field
// whose position falls inside the mist's bounding box. This is the original
// (pre-task-200) mist tick body, extracted verbatim -- the monster AREA_POISON
// path must behave identically before and after (NFR-5).
func (r *MistTick) tickCharacters(ctx context.Context, prov producer.Provider, t tenant.Model, m mist.Mist) {
	members := r.charsInField(t, m.Field())
	if len(members) == 0 {
		return
	}
	emitErr := message.Emit(prov)(func(buf *message.Buffer) error {
		for _, cid := range members {
			x, y, err := r.posLookup(ctx, cid)
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
}

// tickMonsters applies the mist's damage-over-time status to every monster the
// atlas-monsters rect endpoint reports inside the mist's bounding box.
//
// The endpoint is authoritative for containment -- this does NOT re-filter
// with Mist.Contains. Double-filtering would mask an endpoint bug and would
// diverge if the two rect conventions (inclusive vs exclusive edges) ever
// differed. One authority per question.
func (r *MistTick) tickMonsters(ctx context.Context, prov producer.Provider, t tenant.Model, m mist.Mist) {
	monsters, err := r.monstersInRect(ctx, m)
	if err != nil {
		r.l.WithError(err).Errorf("MistTick: monster rect lookup failed for mist [%s]; skipping this mist's tick.", m.Id())
		return
	}
	if len(monsters) == 0 {
		r.l.Debugf("MistTick: mist [%s] found 0 monsters in rect.", m.Id())
		return
	}
	emitErr := message.Emit(prov)(func(buf *message.Buffer) error {
		applied := 0
		for _, rm := range monsters {
			uniqueId, cErr := strconv.Atoi(rm.Id)
			if cErr != nil {
				r.l.WithError(cErr).Warnf("MistTick: unparseable monster id [%s] for mist [%s].", rm.Id, m.Id())
				continue
			}
			if pErr := buf.Put(EnvCommandTopicMonster, applyStatusCommandProvider(m, uint32(uniqueId))); pErr != nil {
				return pErr
			}
			applied++
		}
		r.l.Debugf("MistTick: mist [%s] applied [%s] to %d of %d monsters in rect.", m.Id(), m.Disease(), applied, len(monsters))
		return nil
	})
	if emitErr != nil {
		r.l.WithError(emitErr).Errorf("MistTick: failed to emit apply-status for mist [%s].", m.Id())
	}
}

// SleepTime reports the configured tick interval.
func (r *MistTick) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
