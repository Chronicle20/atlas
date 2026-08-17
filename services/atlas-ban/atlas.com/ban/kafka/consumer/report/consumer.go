package report

import (
	consumer2 "atlas-ban/kafka/consumer"
	report2 "atlas-ban/kafka/message/report"
	report3 "atlas-ban/report"
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("report_command")(report2.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(report2.EnvCommandTopic)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreateReportCommand(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleCreateReportCommand(db *gorm.DB) message.Handler[report2.Command[report2.CreateCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c report2.Command[report2.CreateCommandBody]) {
		if c.Type != report2.CommandTypeCreate {
			return
		}
		l.Debugf("Received create report command kind [%s] reporter [%d].", c.Body.Kind, c.Body.ReporterId)
		if err := report3.NewProcessor(l, ctx, db).CreateFromCommandAndEmit(c.Body); err != nil {
			l.WithError(err).Errorf("Error processing create report command from reporter [%d].", c.Body.ReporterId)
		}
	}
}
