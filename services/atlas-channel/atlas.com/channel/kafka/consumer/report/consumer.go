// Package report translates atlas-ban's EVENT_TOPIC_REPORT_STATUS events
// (report2.StatusEvent, kafka/message/report/kafka.go) into the result
// packet the reporter's client expects (design.md §4.5) and delivers it to
// the reporting character's session. Delivery is by character id - there is
// no correlation token on the wire, and the reporter is looked up by
// characterId the same way the buddylist consumer does
// (kafka/consumer/buddylist/consumer.go).
package report

import (
	consumer2 "atlas-channel/kafka/consumer"
	report2 "atlas-channel/kafka/message/report"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	reportcb "github.com/Chronicle20/atlas/libs/atlas-packet/report/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("report_status_event")(report2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var handles []listener.HandlerHandle
				t, _ := topic.EnvProvider(l)(report2.EnvEventTopicStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// reportAnnouncer is the channel-side seam that resolves the reporter's
// session and announces the given report-result writer body to it. Package-
// level var so tests can swap in a recording stub without a live net.Conn or
// a real writer registry (mirrors the rps/mount consumers' seam pattern).
//
// A tenant whose config has no opcode mapped for writerName is NOT an error:
// writer.ProducerGetter (libs/atlas-socket/writer/writer.go) returns "writer
// not found" whenever BuildWriterProducer (libs/atlas-opcodes/producer.go)
// omitted the writer at boot because the tenant's socket-config template
// doesn't declare it. Config presence IS the feature gate - e.g. a v61
// tenant wired for sue-only, or a jms/gms-92 tenant with reporting disabled
// entirely. Checking wp(writerName) here, before ever touching
// session.Announce, means those tenants silently skip delivery (debug log)
// instead of spamming an error on every report.
var reportAnnouncer = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, sc server.Model, characterId uint32, writerName string, body packet.Encode) {
	if _, err := wp(writerName); err != nil {
		l.Debugf("Tenant configuration has no writer [%s] mapped; skipping report result delivery to character [%d].", writerName, characterId)
		return
	}
	err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId,
		session.Announce(l)(ctx)(wp)(writerName)(body))
	if err != nil {
		l.WithError(err).Errorf("Unable to deliver report result to character [%d].", characterId)
	}
}

func handleStatusEvent(sc server.Model, wp writer.Producer) message.Handler[report2.StatusEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e report2.StatusEvent) {
		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		writerName, body, ok := resultPacket(e)
		if !ok {
			l.Warnf("Dropping unmapped report status event kind [%s] status [%s] errorCode [%s].", e.Kind, e.Status, e.ErrorCode)
			return
		}

		reportAnnouncer(l, ctx, wp, sc, e.ReporterId, writerName, body)
	}
}

// resultPacket maps a report status event to the result packet the reporter
// sees (design.md §4.5, v1 result policy):
//
//	sue,   CREATED                    -> SueCharacterResult   SUCCESS
//	sue,   ERROR, NOT_FOUND           -> SueCharacterResult   UNABLE_TO_LOCATE
//	sue,   ERROR, anything else       -> SueCharacterResult   GENERIC_FAILURE
//	claim, CREATED                    -> ClaimResultSuccess   hasRemaining=1, remaining=ClaimResultSuccessRemaining
//	claim, ERROR, NOT_FOUND           -> ClaimResultNotice    RECHECK_NAME
//	claim, ERROR, anything else       -> ClaimResultNotice    TRY_AGAIN
//
// Unknown kind/status combinations are dropped by the caller, never sent as
// a guessed mode.
func resultPacket(e report2.StatusEvent) (string, packet.Encode, bool) {
	switch e.Kind {
	case report2.KindSue:
		switch {
		case e.Status == report2.EventStatusCreated:
			return reportcb.SueCharacterResultWriter, writer.SueCharacterResultBody(writer.SueResultSuccess), true
		case e.Status == report2.EventStatusError && e.ErrorCode == report2.ErrorCodeNotFound:
			return reportcb.SueCharacterResultWriter, writer.SueCharacterResultBody(writer.SueResultUnableToLocate), true
		case e.Status == report2.EventStatusError:
			return reportcb.SueCharacterResultWriter, writer.SueCharacterResultBody(writer.SueResultGenericFailure), true
		}
	case report2.KindClaim:
		switch {
		case e.Status == report2.EventStatusCreated:
			return reportcb.ClaimResultWriter, writer.ClaimResultSuccessBody(true, writer.ClaimResultSuccessRemaining), true
		case e.Status == report2.EventStatusError && e.ErrorCode == report2.ErrorCodeNotFound:
			return reportcb.ClaimResultWriter, writer.ClaimResultNoticeBody(writer.ClaimResultRecheckName), true
		case e.Status == report2.EventStatusError:
			return reportcb.ClaimResultWriter, writer.ClaimResultNoticeBody(writer.ClaimResultTryAgain), true
		}
	}
	return "", nil, false
}
