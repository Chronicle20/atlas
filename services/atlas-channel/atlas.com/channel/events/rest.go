package events

// RestModel is the channel's projection of the atlas-events map-entry
// visuals resource (event/occurrence/visual_rest.go VisualRestModel in
// atlas-events). Field names and the resource type below must match that
// server-side model exactly.
type RestModel struct {
	Id           string `json:"-"`
	OccurrenceId string `json:"occurrenceId"`
	Visual       string `json:"visual"`
	State        byte   `json:"state"`
	SubState     byte   `json:"subState"`
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
