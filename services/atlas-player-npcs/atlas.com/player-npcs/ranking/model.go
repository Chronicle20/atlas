package ranking

// Model is a character's computed ranking, read from atlas-rankings at
// deploy time (design §6.3, D-3). A character with no computed ranking
// yet is represented as the zero value — GetByCharacterId returns it
// rather than erroring, since a character with no ranking must still be
// able to deploy.
type Model struct {
	rank    uint32
	jobRank uint32
}

func (m Model) Rank() uint32    { return m.rank }
func (m Model) JobRank() uint32 { return m.jobRank }
