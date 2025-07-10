package guild

// Model represents a guild domain object
type Model struct {
	id     uint32
	name   string
	rank   uint32
	member bool
}

// Id returns the guild ID
func (m Model) Id() uint32 {
	return m.id
}

// Name returns the guild name
func (m Model) Name() string {
	return m.name
}

// Rank returns the member's rank in the guild
func (m Model) Rank() uint32 {
	return m.rank
}

// IsMember returns true if the character is a member of a guild
func (m Model) IsMember() bool {
	return m.member
}

// NewModel creates a new guild model
func NewModel(id uint32, name string, rank uint32) Model {
	return Model{
		id:     id,
		name:   name,
		rank:   rank,
		member: id != 0, // If guild ID is 0, character is not in a guild
	}
}

// EmptyModel creates an empty guild model for characters not in a guild
func EmptyModel() Model {
	return Model{
		id:     0,
		name:   "",
		rank:   0,
		member: false,
	}
}