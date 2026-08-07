package handler

type RestModel struct {
	OpCode    string `json:"opCode"`
	Validator string `json:"validator"`
	Handler   string `json:"handler"`
	// FName carries the client-side function name
	// ("CLogin::SendCheckPasswordPacket"). It is informational only: it never
	// participates in comparison, validation or ancestry classification
	// (PRD FR-10.4).
	FName string `json:"fname,omitempty"`
	// Options uses omitzero (not omitempty) so an entry that supplied none
	// does not round-trip to "options":null while an explicit {} still
	// survives as {}. encoding/json's "omitempty" treats maps as empty by
	// length (a non-nil empty map is dropped too), which would lose the
	// absent-vs-{} distinction; "omitzero" checks the map against its zero
	// value (nil), which is exactly the distinction wanted here.
	Options  map[string]interface{} `json:"options,omitzero"`
	Services []string               `json:"services,omitempty"`
}
