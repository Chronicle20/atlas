package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/extraction/lock"
	"atlas-wz-extractor/extraction/parallelism"
	consumer2 "atlas-wz-extractor/kafka/consumer"
	mext "atlas-wz-extractor/kafka/message/extraction"
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// processor is the slim subset of extraction.Processor that this consumer
// needs. Defined here so the test can inject a fake without pulling
// extraction.Processor.
type processor interface {
	ExtractUnit(l logrus.FieldLogger, ctx context.Context, wzFile string, xmlOnly, imagesOnly bool) error
}

func InitConsumers(l logrus.FieldLogger) func(rf func(consumer.Config, ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(consumer.Config, ...model.Decorator[consumer.Config])) func(string) {
		return func(consumerGroupId string) {
			rf(
				consumer2.NewConfig(l)("wz_extraction_command")(mext.EnvCommandTopic)(consumerGroupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
				consumer.SetStartOffset(kafka.FirstOffset),
				consumer.SetMaxInFlight(parallelism.FromEnv(l)),
			)
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(p processor, store job.Store, tl *lock.TenantLock) func(rf func(string, handler.Handler) (string, error)) error {
	return func(p processor, store job.Store, tl *lock.TenantLock) func(rf func(string, handler.Handler) (string, error)) error {
		return func(rf func(string, handler.Handler) (string, error)) error {
			t, _ := topic.EnvProvider(l)(mext.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStartExtractionUnit(p, store, tl)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleStartExtractionUnit(p processor, store job.Store, tl *lock.TenantLock) message.Handler[command[startExtractionUnitBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c command[startExtractionUnitBody]) {
		if c.Type != mext.CommandStartExtractionUnit {
			return
		}
		ll := l.WithFields(logrus.Fields{"jobId": c.Body.JobId, "wzFile": c.Body.WzFile})

		claimed, err := store.MarkUnitRunning(ctx, c.Body.JobId, c.Body.WzFile)
		if errors.Is(err, job.ErrNotFound) {
			ll.Warn("orphan unit message; job hash expired or never existed — skipping (offset will commit)")
			return
		}
		if err != nil {
			ll.WithError(err).Error("MarkUnitRunning failed; will retry via Kafka redelivery")
			return
		}
		if !claimed {
			ll.Info("unit already terminal; skipping (redelivery)")
			return
		}

		runErr := p.ExtractUnit(ll, ctx, c.Body.WzFile, c.Body.XmlOnly, c.Body.ImagesOnly)
		terminal := job.UnitSucceeded
		if runErr != nil {
			terminal = job.UnitFailed
		}

		cnt, err := store.FinalizeUnit(ctx, c.Body.JobId, c.Body.WzFile, terminal, runErr)
		if errors.Is(err, job.ErrNotFound) {
			ll.Warn("orphan unit on FinalizeUnit; job hash expired mid-processing — skipping")
			return
		}
		if err != nil {
			ll.WithError(err).Error("FinalizeUnit failed; will retry via Kafka redelivery")
			return
		}

		if !cnt.AllDone {
			return
		}

		jobTerminal := job.JobCompleted
		switch {
		case cnt.UnitsFailed == cnt.UnitsTotal:
			jobTerminal = job.JobFailed
		case cnt.UnitsFailed > 0:
			jobTerminal = job.JobCompletedWithErrors
		}
		claimedTerminal, err := store.MarkJobTerminal(ctx, c.Body.JobId, jobTerminal)
		if err != nil {
			ll.WithError(err).Error("MarkJobTerminal failed")
			return
		}
		if claimedTerminal {
			if err := tl.Release(ctx, cnt.LockKey, c.Body.JobId); err != nil {
				ll.WithError(err).Warn("Release tenant lock failed")
			}
			ll.WithField("status", jobTerminal).Info("job finalized")
		}
	}
}
