package report

import (
	report2 "atlas-ban/kafka/message/report"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kafkago "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func statusEventProvider(reportId uuid.UUID, kind Kind, worldId world.Id, reporterId uint32, status string, errorCode string) model.Provider[[]kafka.Message] {
	key := kafkago.CreateKey(int(reporterId))
	value := &report2.StatusEvent{
		ReportId:   reportId,
		Kind:       string(kind),
		WorldId:    worldId,
		ReporterId: reporterId,
		Status:     status,
		ErrorCode:  errorCode,
	}
	return kafkago.SingleMessageProvider(key, value)
}
