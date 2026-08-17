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
//
// The tenant-status topic (EVENT_TOPIC_CONFIGURATION_TENANT_STATUS) feeds
// the tenant→environment derivation (FR-7.3). It is independently optional:
// an unset tenant topic warns and skips the second consumer, but does not
// prevent the environment topic from starting (R21-3) — mirroring this same
// function's own unset-topic path for the environment topic one branch up.
func (r *Runtime) startEnvironmentRegistry(c *envRegistryConfig) {
	topic := os.Getenv("EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS")
	reg := env.NewMapRegistry(env.Self(), time.Now)
	env.SetRegistry(reg)
	if topic == "" {
		r.logger.Warn("environment registry: EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS unset; running in legacy single-environment mode")
		return
	}
	tenantTopic := os.Getenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS")
	if tenantTopic == "" {
		r.logger.Warn("environment registry: EVENT_TOPIC_CONFIGURATION_TENANT_STATUS unset; tenant→environment derivation (FR-7.3) disabled")
	}
	s := &envSubscriber{registry: reg, caughtUp: newEnvCaughtUp(), topic: topic, tenantTopic: tenantTopic}
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
	ApplyTenant(tenantId string, e env.Id)
	RemoveTenant(tenantId string)
}

// envSubscriber is the consumer-side wiring for the environment-status and
// tenant-status topics. It mirrors services/atlas-channel/.../configuration/projection
// structurally: snapshot end offsets per topic, register one consumer per
// topic replaying from FirstOffset under a per-process group id, decode
// envelopes, and apply them to the registry.
type envSubscriber struct {
	registry envMutator
	caughtUp *envCaughtUp
	topic    string
	// tenantTopic is EVENT_TOPIC_CONFIGURATION_TENANT_STATUS. Empty means
	// the tenant→environment derivation (FR-7.3) is disabled (R21-3): no
	// second consumer is registered and no offsets are added to the gate.
	tenantTopic string

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
	s.caughtUp.SetEndOffsets(s.topic, offsets)

	cmf := consumer.GetManager().AddConsumer(l, ctx, wg)
	cmf(consumer.NewConfig(brokers, "configuration_environment_status", s.topic, groupId),
		consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser),
		consumer.SetStartOffset(kafka.FirstOffset))
	if _, err := consumer.GetManager().RegisterHandler(s.topic, s.handle(l)); err != nil {
		return err
	}

	// The tenant-status topic is independently optional (R21-3): when
	// unset, no second consumer is registered and no offsets are added to
	// the gate — the gate's readiness depends only on the environment
	// topic, exactly as before this topic existed.
	if s.tenantTopic != "" {
		tenantOffsets, err := resolveStartOffsets(ctx, brokers, s.tenantTopic, read, l)
		if err != nil {
			return err
		}
		s.caughtUp.SetEndOffsets(s.tenantTopic, tenantOffsets)

		cmf(consumer.NewConfig(brokers, "configuration_tenant_status", s.tenantTopic, groupId),
			consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser),
			consumer.SetStartOffset(kafka.FirstOffset))
		if _, err := consumer.GetManager().RegisterHandler(s.tenantTopic, s.handleTenant(l)); err != nil {
			return err
		}
	}
	return nil
}

// handle decodes one environment-status message and applies it to the
// registry. registry.Observe is called before decoding so an unreadable
// message still counts as liveness for the FR-1.7 staleness bound.
func (s *envSubscriber) handle(l logrus.FieldLogger) handler.Handler {
	return func(_ logrus.FieldLogger, _ context.Context, msg kafka.Message) (bool, error) {
		s.registry.Observe(time.Now())
		s.caughtUp.Observe(s.topic, msg.Partition, msg.Offset)

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

// handleTenant decodes one tenant-status message and projects its
// "environment" attribute (task-232 R21-1, FR-7.3) into the registry.
// Mirrors handle() above and services/atlas-channel/.../configuration/projection/subscriber.go's
// handleTenant for the tombstone key convention ("tenant:<uuid>").
func (s *envSubscriber) handleTenant(l logrus.FieldLogger) handler.Handler {
	return func(_ logrus.FieldLogger, _ context.Context, msg kafka.Message) (bool, error) {
		s.registry.Observe(time.Now())
		s.caughtUp.Observe(s.tenantTopic, msg.Partition, msg.Offset)

		if isEnvTombstone(msg.Value) {
			id, ok := tenantIdFromKey(msg.Key)
			if !ok {
				return true, nil
			}
			s.registry.RemoveTenant(id)
			return true, nil
		}

		id, environment, err := decodeTenantEnvelope(msg.Value)
		if err != nil {
			if !errors.Is(err, errUnsupportedEnvSchema) {
				l.WithError(err).Warn("envregistry.tenant_decode_failed")
			}
			// Forward-compatible: don't retry on a schema we can't read.
			return true, nil
		}
		s.registry.ApplyTenant(id, environment)
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

// tenantEnvelope is the wire shape published by atlas-configurations'
// outbox.NewTenantEnvelope for the tenant-status topic: the same
// {schema_version, id, config, emitted_at} shape as envEnvelope. Config
// carries the tenant's RestModel; this projection reads only the
// server-owned "environment" attribute (task-232 R21-1) out of it.
type tenantEnvelope struct {
	SchemaVersion int             `json:"schema_version"`
	Id            string          `json:"id"`
	Config        json.RawMessage `json:"config"`
	EmittedAt     string          `json:"emitted_at"`
}

// tenantConfig reads only the "environment" attribute out of a tenant's
// Config; every other tenant attribute is irrelevant to this projection.
type tenantConfig struct {
	Environment string `json:"environment"`
}

// tenantKeyPrefix is the outbox key prefix atlas-configurations' tenants
// package writes (tenantOutboxKey): "tenant:<uuid>". Matches
// services/atlas-channel/.../configuration/projection/subscriber.go's
// tombstone key convention.
const tenantKeyPrefix = "tenant:"

func tenantIdFromKey(key []byte) (string, bool) {
	k := string(key)
	if !strings.HasPrefix(k, tenantKeyPrefix) || len(k) == len(tenantKeyPrefix) {
		return "", false
	}
	return k[len(tenantKeyPrefix):], true
}

func decodeTenantEnvelope(value []byte) (id string, environment env.Id, err error) {
	var e tenantEnvelope
	if err := json.Unmarshal(value, &e); err != nil {
		return "", "", err
	}
	if e.SchemaVersion > supportedEnvSchemaVersion {
		return "", "", errUnsupportedEnvSchema
	}
	var cfg tenantConfig
	if err := json.Unmarshal(e.Config, &cfg); err != nil {
		return "", "", err
	}
	return e.Id, env.Id(cfg.Environment), nil
}

func lookupBrokers() []string {
	return []string{os.Getenv("BOOTSTRAP_SERVERS")}
}

// envCaughtUp gates readiness on having consumed past the end-offset
// snapshot taken at boot, for every registered topic. Keyed on
// (topic, partition) — never partition alone (task-232 R21-2): the
// environment and tenant-status topics are two independent partition
// spaces, and a partition-only key lets one topic's offsets satisfy the
// other's end-offset bar, flipping /readyz Ready before the second topic
// has actually been consumed. Mirrors
// services/atlas-channel/.../configuration/projection.CaughtUp, which
// already gets this right for its own two topics.
type envCaughtUp struct {
	mu         sync.Mutex
	snapshots  map[string]map[int]int64 // topic -> partition -> boot end offset
	consumed   map[string]map[int]int64 // topic -> partition -> highest consumed
	caughtUp   atomic.Bool
	readyChans []chan struct{}
}

func newEnvCaughtUp() *envCaughtUp {
	return &envCaughtUp{
		snapshots: map[string]map[int]int64{},
		consumed:  map[string]map[int]int64{},
	}
}

// SetEndOffsets records topic's boot end-offset snapshot. An empty offsets
// map (topic has no data yet) counts as trivially caught up for that topic.
func (c *envCaughtUp) SetEndOffsets(topic string, offsets map[int]int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if offsets == nil {
		offsets = map[int]int64{}
	}
	c.snapshots[topic] = offsets
	if c.consumed[topic] == nil {
		c.consumed[topic] = map[int]int64{}
	}
	c.evaluateLocked()
}

// Observe records that the subscriber has consumed up to (and including)
// offset on partition p of topic. Idempotent: lower offsets are ignored.
func (c *envCaughtUp) Observe(topic string, partition int, offset int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cur, ok := c.consumed[topic]
	if !ok {
		cur = map[int]int64{}
		c.consumed[topic] = cur
	}
	if existing, present := cur[partition]; present && existing >= offset {
		return
	}
	cur[partition] = offset
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
	if len(c.snapshots) == 0 {
		return
	}
	for topic, ends := range c.snapshots {
		got := c.consumed[topic]
		for p, end := range ends {
			// end == 0 means the partition is empty (Kafka end-offset is the
			// next-to-be-written offset); trivially caught up.
			if end == 0 {
				continue
			}
			// Distinguish "never observed" from "observed offset 0" — a
			// default int64 zero from a missing map key would otherwise
			// satisfy `observed >= end-1` when end == 1.
			observed, present := got[p]
			if !present || observed < end-1 {
				return
			}
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
