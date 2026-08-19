package events

// RestModel is the channel's projection of the atlas-events map-entry
// visuals resource (event/occurrence/visual_rest.go VisualRestModel in
// atlas-events). Field names and the resource type below must match that
// server-side model exactly.
//
// No State/SubState: since B6 (commit d19237c1e), ContiMove's wire
// state/subState bytes are resolved on this side from the tenant
// writer-options table (socket/writer.ContiMoveBody), keyed off the visual
// Type (SHOW/HIDE) — announceActiveVisuals only ever resolves SHOW here, so
// there was never a per-visual value to carry.
type RestModel struct {
	Id           string `json:"-"`
	OccurrenceId string `json:"occurrenceId"`
	Visual       string `json:"visual"`
	Bgm          string `json:"bgm"`
}

// GetName returns the JSON:API resource type. Must match atlas-events'
// VisualRestModel.GetName ("event-visuals").
func (m RestModel) GetName() string {
	return "event-visuals"
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs are required by api2go
// (jsonapi.Unmarshal) if the upstream response ever carries a
// `relationships` block, even when this client doesn't care about the
// relationship payload. See libs/atlas-rest/CLAUDE.md.
func (m *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (m *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
