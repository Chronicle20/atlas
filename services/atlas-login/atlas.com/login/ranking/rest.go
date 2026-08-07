package ranking

import (
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel mirrors atlas-rankings' wire shape exactly — see
// services/atlas-rankings/atlas.com/rankings/ranking/rest.go: resource type
// "rankings", id = characterId, and the rank/rankMove/jobRank/jobRankMove/
// computedAt attribute names and types below all match that struct
// field-for-field.
type RestModel struct {
	Id          uint32    `json:"-"`
	WorldId     world.Id  `json:"worldId"`
	Rank        uint32    `json:"rank"`
	RankMove    int32     `json:"rankMove"`
	JobRank     uint32    `json:"jobRank"`
	JobRankMove int32     `json:"jobRankMove"`
	ComputedAt  time.Time `json:"computedAt"`
}

func (r RestModel) GetName() string {
	return "rankings"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs are required by api2go's
// jsonapi.Unmarshal whenever the response carries a relationships block,
// even when the caller has no relationship data to store (see
// libs/atlas-rest/CLAUDE.md). The rankings resource does not currently emit
// relationships, but omitting these stubs is a silent-failure trap the next
// time it does.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// Extract carries the RankMove/JobRankMove sign straight through: the
// atlas-rankings REST attribute is JSON-encoded as a signed int32 (not a
// two's-complement uint32 like the client packet wire), so no sign
// conversion happens here. character.MergeRankings hands these signed
// int32 values directly to Builder.SetRankMove/SetJobRankMove, which also
// take int32; the two's-complement conversion for the game client packet
// happens downstream in character.Model's own getters.
func Extract(r RestModel) (Model, error) {
	return Model{
		characterId: r.Id,
		rank:        r.Rank,
		rankMove:    r.RankMove,
		jobRank:     r.JobRank,
		jobRankMove: r.JobRankMove,
	}, nil
}
