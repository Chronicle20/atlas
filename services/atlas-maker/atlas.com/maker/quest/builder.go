package quest

// Builder constructs a Model.
type Builder struct {
	id          uint32
	characterId uint32
	questId     uint32
	state       byte
}

func NewBuilder(id uint32) *Builder {
	return &Builder{id: id}
}

func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

func (b *Builder) SetQuestId(questId uint32) *Builder {
	b.questId = questId
	return b
}

func (b *Builder) SetState(state byte) *Builder {
	b.state = state
	return b
}

func (b *Builder) Build() (Model, error) {
	return Model{
		id:          b.id,
		characterId: b.characterId,
		questId:     b.questId,
		state:       b.state,
	}, nil
}
