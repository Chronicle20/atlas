package progress

import "strconv"

// RestModel represents a single quest progress entry from atlas-quest's
// GET /characters/{characterId}/quests/{questId}/progress collection.
type RestModel struct {
	Id         uint32 `json:"-"`
	InfoNumber uint32 `json:"infoNumber"`
	Progress   string `json:"progress"`
}

// GetName returns the JSON:API type name
func (r RestModel) GetName() string {
	return "progress"
}

// GetID returns the JSON:API resource ID
func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

// SetID sets the JSON:API resource ID
func (r *RestModel) SetID(strId string) error {
	if strId == "" {
		r.Id = 0
		return nil
	}

	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// Model is the domain representation of a quest progress entry. Progress is
// stored and returned as a string, unparsed: parsing it when it happens to
// look numeric would make comparison against a stored context value depend
// on the content of the data (design.md §8).
type Model struct {
	infoNumber uint32
	progress   string
}

// InfoNumber returns the quest info number this progress entry tracks
func (m Model) InfoNumber() uint32 {
	return m.infoNumber
}

// Progress returns the raw, unparsed progress value
func (m Model) Progress() string {
	return m.progress
}

// Extract converts a RestModel into a Model
func Extract(rm RestModel) (Model, error) {
	return Model{
		infoNumber: rm.InfoNumber,
		progress:   rm.Progress,
	}, nil
}
