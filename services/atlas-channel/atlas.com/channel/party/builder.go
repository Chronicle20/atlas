package party

type builder struct {
	id       uint32
	leaderId uint32
	members  []MemberModel
}

// NewBuilder returns a new party model builder. Used by tests and any
// code path that needs to construct a party.Model in-process (the
// production path uses Extract over the REST response).
func NewBuilder() *builder {
	return &builder{}
}

func (b *builder) SetId(v uint32) *builder             { b.id = v; return b }
func (b *builder) SetLeaderId(v uint32) *builder       { b.leaderId = v; return b }
func (b *builder) SetMembers(v []MemberModel) *builder { b.members = v; return b }

func (b *builder) Build() Model {
	return Model{
		id:       b.id,
		leaderId: b.leaderId,
		members:  b.members,
	}
}

// MustBuild returns a Model unconditionally. Kept symmetric with the
// character package's MustBuild.
func (b *builder) MustBuild() Model {
	return b.Build()
}
