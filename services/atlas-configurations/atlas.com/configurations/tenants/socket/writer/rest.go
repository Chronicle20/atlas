package writer

type RestModel struct {
	OpCode string `json:"opCode"`
	Writer string `json:"writer"`
	// FName carries the client-side function name. Informational only; see
	// handler.RestModel.FName.
	FName string `json:"fname,omitempty"`
	// Options uses omitzero; see handler.RestModel.Options.
	Options  map[string]interface{} `json:"options,omitzero"`
	Services []string               `json:"services,omitempty"`
}
