package projection

import (
	"context"
	"errors"
	"sync"

	consumer2 "atlas-channel/kafka/consumer"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Subscriber is the consumer-side wiring for the two config-status
// topics. It snapshots end offsets at start (gating CaughtUp), then
// registers two kafka consumers that decode envelopes and apply them to
// State.
//
// The Subscriber does NOT call listener.Registry.Add/Drain on its own —
// that's the apply loop's job (see loop.go). Separation lets tests
// exercise the consumer wiring without spinning up the listener stack.
type Subscriber struct {
	State    *State
	CaughtUp *CaughtUp

	// ServiceTopic is the env-var-resolved topic name for service config
	// events (EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS).
	ServiceTopic string
	// TenantTopic is the env-var-resolved topic name for tenant config
	// events (EVENT_TOPIC_CONFIGURATION_TENANT_STATUS).
	TenantTopic string

	// ServiceId is this atlas-channel instance's service id; service
	// envelopes with a different id are skipped.
	ServiceId uuid.UUID
}

// Start snapshots end offsets for both topics, then registers a
// consumer + handler for each on the shared atlas-kafka manager. Returns
// the first error encountered; partial registration is tolerated by the
// caller (the gate will not flip and the apply loop will see no ops).
//
// wg is the teardown manager's WaitGroup — the consumer manager calls
// Add/Done on it to track in-flight consumers during shutdown. Must not
// be nil; the atlas-kafka library calls Add unconditionally.
func (s *Subscriber) Start(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error {
	brokers := consumer2.LookupBrokers()

	// Snapshot end offsets so CaughtUp has bars to clear.
	if s.ServiceTopic != "" {
		offsets, err := offsetsOrEmpty(ctx, brokers, s.ServiceTopic, l)
		if err != nil {
			return err
		}
		s.CaughtUp.SetEndOffsets(s.ServiceTopic, offsets)
	}
	if s.TenantTopic != "" {
		offsets, err := offsetsOrEmpty(ctx, brokers, s.TenantTopic, l)
		if err != nil {
			return err
		}
		s.CaughtUp.SetEndOffsets(s.TenantTopic, offsets)
	}

	// Register the two consumers on the shared manager. Start from the
	// earliest offset so log-compacted history is replayed.
	cmf := consumer.GetManager().AddConsumer(l, ctx, wg)
	if s.ServiceTopic != "" {
		cmf(consumer.NewConfig(brokers, "configuration_service_status", s.ServiceTopic, groupId),
			consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
			consumer.SetStartOffset(kafka.FirstOffset))
		if _, err := consumer.GetManager().RegisterHandler(s.ServiceTopic, s.handleService(l)); err != nil {
			return err
		}
	}
	if s.TenantTopic != "" {
		cmf(consumer.NewConfig(brokers, "configuration_tenant_status", s.TenantTopic, groupId),
			consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
			consumer.SetStartOffset(kafka.FirstOffset))
		if _, err := consumer.GetManager().RegisterHandler(s.TenantTopic, s.handleTenant(l)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Subscriber) handleService(l logrus.FieldLogger) handler.Handler {
	return func(_ logrus.FieldLogger, _ context.Context, msg kafka.Message) (bool, error) {
		s.CaughtUp.Observe(msg.Topic, msg.Partition, msg.Offset)
		if IsTombstone(msg.Value) {
			// A service tombstone matching our id means atlas-channel
			// should drop the entire config. Non-matching tombstones
			// are ignored.
			key := string(msg.Key)
			if key == "service:"+s.ServiceId.String() {
				s.State.ApplyServiceTombstone()
			}
			return true, nil
		}
		env, err := DecodeServiceEnvelope(msg.Value)
		if err != nil {
			if !errors.Is(err, ErrUnsupportedSchema) {
				l.WithError(err).Warn("projection.service.decode_failed")
			}
			// Forward-compatible: don't retry on a schema we can't read.
			return true, nil
		}
		if env.Id != s.ServiceId.String() {
			return true, nil
		}
		if err := s.State.ApplyService(env); err != nil {
			l.WithError(err).Warn("projection.service.apply_failed")
			return true, nil
		}
		return true, nil
	}
}

func (s *Subscriber) handleTenant(l logrus.FieldLogger) handler.Handler {
	return func(_ logrus.FieldLogger, _ context.Context, msg kafka.Message) (bool, error) {
		s.CaughtUp.Observe(msg.Topic, msg.Partition, msg.Offset)
		if IsTombstone(msg.Value) {
			// Tenant tombstone: key is "tenant:<uuid>". Strip the prefix.
			k := string(msg.Key)
			const prefix = "tenant:"
			if len(k) <= len(prefix) || k[:len(prefix)] != prefix {
				return true, nil
			}
			id, err := uuid.Parse(k[len(prefix):])
			if err != nil {
				return true, nil
			}
			s.State.ApplyTenantTombstone(id)
			return true, nil
		}
		env, err := DecodeTenantEnvelope(msg.Value)
		if err != nil {
			if !errors.Is(err, ErrUnsupportedSchema) {
				l.WithError(err).Warn("projection.tenant.decode_failed")
			}
			return true, nil
		}
		if err := s.State.ApplyTenant(env); err != nil {
			l.WithError(err).Warn("projection.tenant.apply_failed")
			return true, nil
		}
		return true, nil
	}
}

func offsetsOrEmpty(ctx context.Context, brokers []string, topic string, l logrus.FieldLogger) (map[int]int64, error) {
	off, err := consumer.ReadEndOffsets(ctx, brokers, topic)
	if err != nil {
		// A missing topic shouldn't kill startup; the gate will simply
		// stay at "needs end offsets" and the operator can debug. Log
		// at warn level so the operator notices.
		l.WithError(err).WithField("topic", topic).Warn("projection.read_end_offsets_failed")
		return map[int]int64{}, nil
	}
	return off, nil
}
