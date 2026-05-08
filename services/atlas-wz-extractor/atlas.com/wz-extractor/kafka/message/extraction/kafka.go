package extraction

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

const (
	EnvCommandTopic            = "COMMAND_TOPIC_WZ_EXTRACTION"
	CommandStartExtractionUnit = "START_EXTRACTION_UNIT"
)

type Command[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

type StartExtractionUnitBody struct {
	JobId      string `json:"jobId"`
	WzFile     string `json:"wzFile"`
	XmlOnly    bool   `json:"xmlOnly"`
	ImagesOnly bool   `json:"imagesOnly"`
}

// StartExtractionUnitProvider builds one kafka.Message keyed by jobId so all
// of one job's units land in the same partition (when partition count permits)
// — but partition count >= 16 means cross-job parallelism still works.
func StartExtractionUnitProvider(jobId, wzFile string, xmlOnly, imagesOnly bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(djb2(jobId)))
	value := &Command[StartExtractionUnitBody]{
		Type: CommandStartExtractionUnit,
		Body: StartExtractionUnitBody{
			JobId:      jobId,
			WzFile:     wzFile,
			XmlOnly:    xmlOnly,
			ImagesOnly: imagesOnly,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// djb2 hashes a string into an int suitable for producer.CreateKey, which
// expects an int. The exact hash is unimportant — we just want consistent
// keying so per-job units are not all glued to the same partition.
func djb2(s string) uint32 {
	var h uint32 = 5381
	for i := 0; i < len(s); i++ {
		h = h*33 + uint32(s[i])
	}
	return h
}
