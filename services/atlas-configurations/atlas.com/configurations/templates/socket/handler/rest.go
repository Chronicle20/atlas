package handler

type RestModel struct {
	OpCode    string                 `json:"opCode"`
	Validator string                 `json:"validator"`
	Handler   string                 `json:"handler"`
	Options   map[string]interface{} `json:"options"`
	Services  []string               `json:"services,omitempty"`
}
