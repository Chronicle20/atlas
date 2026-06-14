package mount

// Model is the channel-side view of a character's mount progression, fetched
// from atlas-mounts to populate the CharacterInfo window's tamed-mob block.
type Model struct {
	characterId uint32
	level       int
	exp         int
	tiredness   int
}

func (m Model) CharacterId() uint32 { return m.characterId }
func (m Model) Level() int          { return m.level }
func (m Model) Exp() int            { return m.exp }
func (m Model) Tiredness() int      { return m.tiredness }
