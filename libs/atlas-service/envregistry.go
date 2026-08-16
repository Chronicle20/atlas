package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
)

// envRegistryConfig carries WithEnvironmentRegistry's argument through to
// Bootstrap.
type envRegistryConfig struct {
	serviceName string
}

// WithEnvironmentRegistry makes Bootstrap subscribe to the environment
// status topic (EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS), project it
// into an env.MapRegistry, install it with env.SetRegistry, and AND its
// catch-up into Runtime.Ready(). Every service's main.go in Phase C passes
// exactly this one option.
func WithEnvironmentRegistry(serviceName string) Option {
	return func(c *bootstrapConfig) {
		c.envRegistry = &envRegistryConfig{serviceName: serviceName}
	}
}

// startEnvironmentRegistry installs the process-wide env.Registry and, when
// a topic is configured, starts the projection subscriber and ANDs its
// catch-up gate into Ready(). With no topic configured the registry stays
// empty and every query answers the FR-1.8 legacy value; this is not fatal,
// since main runs this way until Phase F rolls the topic out everywhere.
func (r *Runtime) startEnvironmentRegistry(c *envRegistryConfig) {
	topic := os.Getenv("EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS")
	reg := env.NewMapRegistry(env.Self(), time.Now)
	env.SetRegistry(reg)
	if topic == "" {
		r.logger.Warn("environment registry: EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS unset; running in legacy single-environment mode")
		return
	}
	s := &envSubscriber{registry: reg, caughtUp: newEnvCaughtUp(), topic: topic}
	// Per-process group id so each container start replays the full
	// compacted log from FirstOffset; a shared group would resume from the
	// previous run's committed offset and leave the in-memory registry
	// empty until the next write.
	groupId := fmt.Sprintf("%s - environments - %s", c.serviceName, uuid.New().String())
	if err := s.Start(r.tdm.Context(), r.logger, r.tdm.WaitGroup(), groupId); err != nil {
		r.logger.WithError(err).Fatal("Unable to start environment registry subscriber.")
	}
	r.envCaughtUp = s.caughtUp
	r.gates = append(r.gates, s.caughtUp.CaughtUpNow)
}

// envMutator is the subset of *env.MapRegistry the subscriber needs in
// order to apply the projected log. Kept as an interface (rather than a
// concrete *env.MapRegistry field) so tests can exercise handle() against
// the same registry type production code uses without a Kafka broker.
type envMutator interface {
	Apply(env.Record)
	ApplyTombstone(env.Id)
	Observe(time.Time)
}

// envSubscriber is the consumer-side wiring for the environment-status
// topic. It mirrors services/atlas-channel/.../configuration/projection
// structurally: snapshot end offsets, register one consumer replaying from
// FirstOffset under a per-process group id, decode envelopes, and apply
// them to the registry.
type envSubscriber struct {
	registry envMutator
	caughtUp *envCaughtUp
	topic    string

	// readEndOffsets resolves the topic's replayable end offsets. Defaults
	// to consumer.ReadReplayableEndOffsets when nil; overridable in tests so
	// the outage-vs-missing-topic branch in resolveStartOffsets can be
	// exercised without a live broker (DialContext has a 10s timeout, far
	// too slow for a unit test to hit deliberately).
	readEndOffsets func(ctx context.Context, brokers []string, topic string) (map[int]int64, error)
}

// resolveStartOffsets reads the topic's replayable end offsets and decides
// how a resolution failure should affect the caught-up gate. The two
// conditions read collapses into one error return by consumer package
// (libs/atlas-kafka/consumer/offsets.go): a topic that does not exist yet
// (kafka.UnknownTopicOrPartition), and everything else — DialContext
// failing, a partition leader unreachable, i.e. a broker outage at boot.
// Those are NOT the same condition and must not be treated the same:
//
//   - Missing topic: a legitimate empty projection (nothing has been
//     published yet, e.g. before the first PR environment is ever
//     created). Safe to proceed with an empty snapshot — SetEndOffsets({})
//     iterates zero partitions and the gate flips immediately, correctly,
//     since there is nothing to catch up on.
//   - Anything else: indistinguishable from an outage. Collapsing this to
//     an empty snapshot (as services/atlas-channel/.../configuration/projection's
//     offsetsOrEmpty does, and as this function's first version also did)
//     flips the readiness gate before the consumer is even registered:
//     /readyz reports Ready with an empty registry, and every query answers
//     the FR-1.8 legacy default ("the local deployment owns it") — a
//     plausible-looking wrong answer, not an error. Fail closed instead:
//     propagate the error so the gate never flips and Start Fatals (mirrors
//     the existing RegisterHandler-failure Fatal a few lines below — this
//     is not a new failure mode, just the same one triggered earlier).
//
// This is a deliberate divergence from the atlas-channel sibling's
// offsetsOrEmpty, which the brief told this file to mirror; see task-20
// fix-round-1 in .superpowers/sdd/plan/task-20-report.md for why. Do not
// "fix" this back to match — the sibling has the same latent gap and is
// tracked separately (see the report).
func resolveStartOffsets(ctx context.Context, brokers []string, topic string,
	read func(ctx context.Context, brokers []string, topic string) (map[int]int64, error),
	l logrus.FieldLogger,
) (map[int]int64, error) {
	offsets, err := read(ctx, brokers, topic)
	if err == nil {
		return offsets, nil
	}
	if errors.Is(err, kafka.UnknownTopicOrPartition) {
		l.WithField("topic", topic).Info("envregistry.topic_not_yet_created; starting with an empty snapshot")
		return map[int]int64{}, nil
	}
	return nil, fmt.Errorf("envregistry: reading end offsets for %q: %w", topic, err)
}

// Start snapshots the topic's end offset (giving caughtUp a bar to clear),
// then registers a consumer + handler on the shared atlas-kafka manager.
// Returns an error — without registering any consumer — when the end
// offsets could not be resolved for a reason other than the topic not
// existing yet (see resolveStartOffsets); the caller (startEnvironmentRegistry)
// Fatals on that error, matching every other Start-failure path in this
// module.
func (s *envSubscriber) Start(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error {
	brokers := lookupBrokers()

	read := s.readEndOffsets
	if read == nil {
		read = consumer.ReadReplayableEndOffsets
	}
	offsets, err := resolveStartOffsets(ctx, brokers, s.topic, read, l)
	if err != nil {
		return err
	}
	s.caughtUp.SetEndOffsets(offsets)

	cmf := consumer.GetManager().AddConsumer(l, ctx, wg)
	cmf(consumer.NewConfig(brokers, "configuration_environment_status", s.topic, groupId),
		consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
		consumer.SetStartOffset(kafka.FirstOffset))
	if _, err := consumer.GetManager().RegisterHandler(s.topic, s.handle(l)); err != nil {
		return err
	}
	return nil
}

// handle decodes one environment-status message and applies it to the
// registry. registry.Observe is called before decoding so an unreadable
// message still counts as liveness for the FR-1.7 staleness bound.
func (s *envSubscriber) handle(l logrus.FieldLogger) handler.Handler {
	return func(_ logrus.FieldLogger, _ context.Context, msg kafka.Message) (bool, error) {
		s.registry.Observe(time.Now())
		s.caughtUp.Observe(msg.Partition, msg.Offset)

		if isEnvTombstone(msg.Value) {
			id, ok := envIdFromKey(msg.Key)
			if !ok {
				return true, nil
			}
			s.registry.ApplyTombstone(id)
			return true, nil
		}

		rec, err := decodeEnvEnvelope(msg.Value)
		if err != nil {
			if !errors.Is(err, errUnsupportedEnvSchema) {
				l.WithError(err).Warn("envregistry.decode_failed")
			}
			// Forward-compatible: don't retry on a schema we can't read.
			return true, nil
		}
		s.registry.Apply(rec)
		return true, nil
	}
}

// envEnvelope is the wire shape published by atlas-configurations'
// outbox.NewEnvironmentEnvelope (Task 19). Config carries the env.Record
// this projection ultimately applies.
type envEnvelope struct {
	SchemaVersion int             `json:"schema_version"`
	Id            string          `json:"id"`
	Config        json.RawMessage `json:"config"`
	EmittedAt     string          `json:"emitted_at"`
}

// supportedEnvSchemaVersion is the highest envelope schema this projection
// can decode. Bump in lockstep with atlas-configurations/outbox.CurrentSchemaVersion
// when the wire shape changes.
const supportedEnvSchemaVersion = 1

// errUnsupportedEnvSchema is returned when the envelope's schema_version is
// higher than this projection understands. Forward-compatible: log and
// skip rather than crash.
var errUnsupportedEnvSchema = errors.New("envregistry: unsupported envelope schema_version")

// envKeyPrefix is the outbox key prefix atlas-configurations' environments
// package writes (environmentOutboxKey): "environment:<name>".
const envKeyPrefix = "environment:"

func isEnvTombstone(value []byte) bool { return value == nil }

func envIdFromKey(key []byte) (env.Id, bool) {
	k := string(key)
	if !strings.HasPrefix(k, envKeyPrefix) || len(k) == len(envKeyPrefix) {
		return "", false
	}
	return env.Id(k[len(envKeyPrefix):]), true
}

func decodeEnvEnvelope(value []byte) (env.Record, error) {
	var e envEnvelope
	if err := json.Unmarshal(value, &e); err != nil {
		return env.Record{}, err
	}
	if e.SchemaVersion > supportedEnvSchemaVersion {
		return env.Record{}, errUnsupportedEnvSchema
	}
	var rec env.Record
	if err := json.Unmarshal(e.Config, &rec); err != nil {
		return env.Record{}, err
	}
	return rec, nil
}

func lookupBrokers() []string {
	return []string{os.Getenv("BOOTSTRAP_SERVERS")}
}

// envCaughtUp gates readiness on having consumed past the end-offset
// snapshot taken at boot for the single environment-status topic. Mirrors
// the semantics of services/atlas-channel/.../configuration/projection.CaughtUp
// (one-way flip, never reverts) narrowed to one topic instead of two.
type envCaughtUp struct {
	mu         sync.Mutex
	ends       map[int]int64
	consumed   map[int]int64
	set        bool
	caughtUp   atomic.Bool
	readyChans []chan struct{}
}

func newEnvCaughtUp() *envCaughtUp {
	return &envCaughtUp{consumed: map[int]int64{}}
}

// SetEndOffsets records the topic's boot end-offset snapshot. An empty
// offsets map (topic has no data yet) counts as trivially caught up.
func (c *envCaughtUp) SetEndOffsets(offsets map[int]int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if offsets == nil {
		offsets = map[int]int64{}
	}
	c.ends = offsets
	c.set = true
	c.evaluateLocked()
}

// Observe records that the subscriber has consumed up to (and including)
// offset on partition p. Idempotent: lower offsets are ignored.
func (c *envCaughtUp) Observe(partition int, offset int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, present := c.consumed[partition]; present && existing >= offset {
		return
	}
	c.consumed[partition] = offset
	c.evaluateLocked()
}

// CaughtUpNow is the cheap check Ready() calls between requests.
func (c *envCaughtUp) CaughtUpNow() bool { return c.caughtUp.Load() }

// WaitCaughtUp blocks until the gate flips or ctx is canceled.
func (c *envCaughtUp) WaitCaughtUp(ctx context.Context) error {
	if c.caughtUp.Load() {
		return nil
	}
	c.mu.Lock()
	if c.caughtUp.Load() {
		c.mu.Unlock()
		return nil
	}
	ch := make(chan struct{})
	c.readyChans = append(c.readyChans, ch)
	c.mu.Unlock()
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *envCaughtUp) evaluateLocked() {
	if !c.set {
		return
	}
	for p, end := range c.ends {
		// end == 0 means the partition is empty (Kafka end-offset is the
		// next-to-be-written offset); trivially caught up.
		if end == 0 {
			continue
		}
		// Distinguish "never observed" from "observed offset 0" — a
		// default int64 zero from a missing map key would otherwise
		// satisfy `observed >= end-1` when end == 1.
		observed, present := c.consumed[p]
		if !present || observed < end-1 {
			return
		}
	}
	if !c.caughtUp.Load() {
		c.caughtUp.Store(true)
		for _, ch := range c.readyChans {
			close(ch)
		}
		c.readyChans = nil
	}
}
