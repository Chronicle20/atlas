package party

// Model represents party data for a character
type Model struct {
	id       uint32
	leaderId uint32
	members  []uint32
}

// Id returns the party ID (0 = not in party)
func (m Model) Id() uint32 {
	return m.id
}

// LeaderId returns the party leader's character ID
func (m Model) LeaderId() uint32 {
	return m.leaderId
}

// Members returns the list of member character IDs
func (m Model) Members() []uint32 {
	return m.members
}

// MemberCount returns the number of members in the party
func (m Model) MemberCount() int {
	return len(m.members)
}

// ModelBuilder provides a builder pattern for creating party models
type ModelBuilder struct {
	id       uint32
	leaderId uint32
	members  []uint32
}

// NewModelBuilder creates a new party model builder
func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

// SetId sets the party ID
func (b *ModelBuilder) SetId(id uint32) *ModelBuilder {
	b.id = id
	return b
}

// SetLeaderId sets the party leader's character ID
func (b *ModelBuilder) SetLeaderId(leaderId uint32) *ModelBuilder {
	b.leaderId = leaderId
	return b
}

// SetMembers sets the member character IDs
func (b *ModelBuilder) SetMembers(members []uint32) *ModelBuilder {
	b.members = members
	return b
}

// Build creates a party model from the builder
func (b *ModelBuilder) Build() Model {
	return Model{
		id:       b.id,
		leaderId: b.leaderId,
		members:  b.members,
	}
}

// RestModel represents the REST representation of party data
type RestModel struct {
	Id       uint32            `json:"-"`
	LeaderId uint32            `json:"leaderId"`
	Members  []MemberRestModel `json:"-"`
}

// MemberRestModel represents a party member in REST responses
type MemberRestModel struct {
	Id uint32 `json:"-"`
}

func (r RestModel) GetName() string {
	return "parties"
}

func (r MemberRestModel) GetName() string {
	return "members"
}

// Extract transforms a RestModel into a domain Model
func Extract(r RestModel) (Model, error) {
	memberIds := make([]uint32, len(r.Members))
	for i, m := range r.Members {
		memberIds[i] = m.Id
	}

	return NewModelBuilder().
		SetId(r.Id).
		SetLeaderId(r.LeaderId).
		SetMembers(memberIds).
		Build(), nil
}
