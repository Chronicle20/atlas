package party

// Builder provides a builder pattern for creating party models
type Builder struct {
	id       uint32
	leaderId uint32
	members  []uint32
}

// NewBuilder creates a new party model builder
func NewBuilder() *Builder {
	return &Builder{}
}

// SetId sets the party ID
func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

// SetLeaderId sets the party leader's character ID
func (b *Builder) SetLeaderId(leaderId uint32) *Builder {
	b.leaderId = leaderId
	return b
}

// SetMembers sets the member character IDs
func (b *Builder) SetMembers(members []uint32) *Builder {
	b.members = members
	return b
}

// Build creates a party model from the builder
func (b *Builder) Build() Model {
	return Model{
		id:       b.id,
		leaderId: b.leaderId,
		members:  b.members,
	}
}
