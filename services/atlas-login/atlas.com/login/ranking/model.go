package ranking

// Model is the login-side view of a character's computed ranking. Zero
// values represent "not yet computed" or "not present in the bulk response"
// — callers that fail open never construct a Model, they simply leave the
// character's rank fields at their zero defaults.
type Model struct {
	characterId uint32
	rank        uint32
	rankMove    int32
	jobRank     uint32
	jobRankMove int32
}

func (m Model) CharacterId() uint32 { return m.characterId }
func (m Model) Rank() uint32        { return m.rank }
func (m Model) RankMove() int32     { return m.rankMove }
func (m Model) JobRank() uint32     { return m.jobRank }
func (m Model) JobRankMove() int32  { return m.jobRankMove }
