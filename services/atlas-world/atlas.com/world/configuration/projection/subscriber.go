package projection

import (
	"context"
	"errors"
	"sync"

	consumer2 "atlas-world/kafka/consumer"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Subscriber consumes the tenant config-status topic, snapshots end
// offsets at start (gating CaughtUp), then applies envelopes to State.
type Subscriber struct {
	State    *State
	CaughtUp *CaughtUp

	// TenantTopic is the env-var-resolved topic name for tenant config
	// events (EVENT_TOPIC_CONFIGURATION_TENANT_STATUS).
	TenantTopic string
}

// Start snapshots end offsets for the tenant topic and registers a single
// FirstOffset consumer that decodes envelopes into State. wg is the
// teardown manager's WaitGroup (must not be nil).
//
// When TenantTopic is empty the projection has nothing to consume; it
// registers an empty end-offset snapshot so CaughtUp flips trivially and
// the service runs degraded (FR-5) rather than wedging not-ready.
func (s *Subscriber) Start(ctx context.Context, l logrus.FieldLogger, wg *sync.WaitGroup, groupId string) error {
	if s.TenantTopic == "" {
		s.CaughtUp.SetEndOffsets("", map[int]int64{})
		return nil
	}

	brokers := consumer2.LookupBrokers()

	offsets, err := offsetsOrEmpty(ctx, brokers, s.TenantTopic, l)
	if err != nil {
		return err
	}
	s.CaughtUp.SetEndOffsets(s.TenantTopic, offsets)

	cmf := consumer.GetManager().AddConsumer(l, ctx, wg)
	cmf(consumer.NewConfig(brokers, "configuration_tenant_status", s.TenantTopic, groupId),
		consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
		consumer.SetStartOffset(kafka.FirstOffset))
	if _, err := consumer.GetManager().RegisterHandler(s.TenantTopic, s.handleTenant(l)); err != nil {
		return err
	}
	return nil
}

func (s *Subscriber) handleTenant(l logrus.FieldLogger) handler.Handler {
	return func(_ logrus.FieldLogger, _ context.Context, msg kafka.Message) (bool, error) {
		s.CaughtUp.Observe(msg.Topic, msg.Partition, msg.Offset)
		if IsTombstone(msg.Value) {
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
		l.WithError(err).WithField("topic", topic).Warn("projection.read_end_offsets_failed")
		return map[int]int64{}, nil
	}
	return off, nil
}
