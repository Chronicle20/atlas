package key

import "errors"

// Builder provides a fluent API for constructing key.Model instances.
type Builder struct {
	characterId uint32
	key         int32
	theType     int8
	action      int32
}

// NewBuilder creates a new Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// CloneBuilder creates a new Builder initialized from an existing Model.
func CloneBuilder(m Model) *Builder {
	return &Builder{
		characterId: m.CharacterId(),
		key:         m.Key(),
		theType:     m.Type(),
		action:      m.Action(),
	}
}

// SetCharacterId sets the character ID.
func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

// SetKey sets the key binding.
func (b *Builder) SetKey(key int32) *Builder {
	b.key = key
	return b
}

// SetType sets the key type.
func (b *Builder) SetType(theType int8) *Builder {
	b.theType = theType
	return b
}

// SetAction sets the action.
func (b *Builder) SetAction(action int32) *Builder {
	b.action = action
	return b
}

// Build validates and constructs the Model. Returns an error if validation fails.
func (b *Builder) Build() (Model, error) {
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}
	return Model{
		characterId: b.characterId,
		key:         b.key,
		theType:     b.theType,
		action:      b.action,
	}, nil
}

// MustBuild builds the model and panics if validation fails.
// Use this only when building from a known-valid source (e.g., cloning an existing model).
func (b *Builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic("MustBuild failed: " + err.Error())
	}
	return m
}

// CharacterId returns the characterId from the builder.
func (b *Builder) CharacterId() uint32 {
	return b.characterId
}

// Key returns the key from the builder.
func (b *Builder) Key() int32 {
	return b.key
}

// Type returns the type from the builder.
func (b *Builder) Type() int8 {
	return b.theType
}

// Action returns the action from the builder.
func (b *Builder) Action() int32 {
	return b.action
}
