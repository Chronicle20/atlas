package progress

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
