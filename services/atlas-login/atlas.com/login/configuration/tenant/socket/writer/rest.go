package writer

type RestModel struct {
	OpCode  string                 `json:"opCode"`
	Writer  string                 `json:"writer"`
	Options map[string]interface{} `json:"options"`
}
