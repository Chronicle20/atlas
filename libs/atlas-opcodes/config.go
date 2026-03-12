package opcodes

// HandlerConfig represents a client-to-server handler opcode mapping.
type HandlerConfig struct {
	OpCode    string                 `json:"opCode"`
	Validator string                 `json:"validator"`
	Handler   string                 `json:"handler"`
	Options   map[string]interface{} `json:"options"`
}

// WriterConfig represents a server-to-client writer opcode mapping.
type WriterConfig struct {
	OpCode  string                 `json:"opCode"`
	Writer  string                 `json:"writer"`
	Options map[string]interface{} `json:"options"`
}
