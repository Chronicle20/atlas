package extraction

const (
	EnvCommandTopic            = "COMMAND_TOPIC_WZ_EXTRACTION"
	CommandStartExtractionUnit = "START_EXTRACTION_UNIT"
)

type command[E any] struct {
	Type string `json:"type"`
	Body E      `json:"body"`
}

type startExtractionUnitBody struct {
	JobId      string `json:"jobId"`
	WzFile     string `json:"wzFile"`
	XmlOnly    bool   `json:"xmlOnly"`
	ImagesOnly bool   `json:"imagesOnly"`
}
