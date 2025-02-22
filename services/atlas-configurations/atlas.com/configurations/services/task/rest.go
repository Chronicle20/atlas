package task

type RestModel struct {
	Type     string `json:"type"`
	Interval int64  `json:"interval"`
	Duration int64  `json:"duration"`
}
