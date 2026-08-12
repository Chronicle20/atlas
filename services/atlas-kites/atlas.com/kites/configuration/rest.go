package configuration

// RestModel is the JSON:API representation of the kite placement-policy
// configuration fetched from atlas-tenants. Fields default to the zero value
// when atlas-tenants has not yet provisioned the resource; Extract folds any
// zero knob back to its default so a partial config never yields a
// nonsensical zero.
type RestModel struct {
	Id                 string   `json:"-"`
	MaxPerMap          int      `json:"maxPerMap"`
	MaxMessageLength   int      `json:"maxMessageLength"`
	BlockedMapPrefixes []uint32 `json:"blockedMapPrefixes"`
}

func (r RestModel) GetName() string {
	return "kite-configs"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs are required by
// api2go/jsonapi's UnmarshalToOneRelations/UnmarshalToManyRelations
// interfaces even though kite-configs has no relationships of its own --
// without them, any response body carrying a `relationships` block fails
// decode with "does not implement UnmarshalToManyRelations" instead of a
// clean fetch error (libs/atlas-rest/CLAUDE.md; the task-037 bug class).
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// Extract converts the fetched RestModel into the immutable domain Model,
// substituting the default for any knob left at its zero value.
func Extract(r RestModel) Model {
	d := DefaultConfig()
	m := Model{
		maxPerMap:          r.MaxPerMap,
		maxMessageLength:   r.MaxMessageLength,
		blockedMapPrefixes: r.BlockedMapPrefixes,
	}
	if m.maxPerMap == 0 {
		m.maxPerMap = d.maxPerMap
	}
	if m.maxMessageLength == 0 {
		m.maxMessageLength = d.maxMessageLength
	}
	if len(m.blockedMapPrefixes) == 0 {
		m.blockedMapPrefixes = d.blockedMapPrefixes
	}
	return m
}
