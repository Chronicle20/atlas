package key

import "errors"

// ModelBuilder provides a fluent API for constructing key.Model instances.
type ModelBuilder struct {
	characterId uint32
	key         int32
	theType     int8
	action      int32
}

// NewModelBuilder creates a new ModelBuilder.
func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

// CloneModelBuilder creates a new ModelBuilder initialized from an existing Model.
func CloneModelBuilder(m Model) *ModelBuilder {
	return &ModelBuilder{
		characterId: m.CharacterId(),
		key:         m.Key(),
		theType:     m.Type(),
		action:      m.Action(),
	}
}

// SetCharacterId sets the character ID.
func (b *ModelBuilder) SetCharacterId(characterId uint32) *ModelBuilder {
	b.characterId = characterId
	return b
}

// SetKey sets the key binding.
func (b *ModelBuilder) SetKey(key int32) *ModelBuilder {
	b.key = key
	return b
}

// SetType sets the key type.
func (b *ModelBuilder) SetType(theType int8) *ModelBuilder {
	b.theType = theType
	return b
}

// SetAction sets the action.
func (b *ModelBuilder) SetAction(action int32) *ModelBuilder {
	b.action = action
	return b
}

// Build validates and constructs the Model. Returns an error if validation fails.
func (b *ModelBuilder) Build() (Model, error) {
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
func (b *ModelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic("MustBuild failed: " + err.Error())
	}
	return m
}

// CharacterId returns the characterId from the builder.
func (b *ModelBuilder) CharacterId() uint32 {
	return b.characterId
}

// Key returns the key from the builder.
func (b *ModelBuilder) Key() int32 {
	return b.key
}

// Type returns the type from the builder.
func (b *ModelBuilder) Type() int8 {
	return b.theType
}

// Action returns the action from the builder.
func (b *ModelBuilder) Action() int32 {
	return b.action
}
