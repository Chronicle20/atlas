package ranking

import (
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// LeaderboardRestModel is the list projection for the per-world leaderboard.
// It intentionally differs from the single-character RestModel (which the
// login decoration depends on) by carrying the display fields.
type LeaderboardRestModel struct {
	Id          uint32    `json:"-"`
	CharacterId uint32    `json:"characterId"`
	Name        string    `json:"name"`
	WorldId     world.Id  `json:"worldId"`
	JobId       job.Id    `json:"jobId"`
	JobCategory uint16    `json:"jobCategory"`
	Level       byte      `json:"level"`
	Rank        uint32    `json:"rank"`
	RankMove    int32     `json:"rankMove"`
	JobRank     uint32    `json:"jobRank"`
	JobRankMove int32     `json:"jobRankMove"`
	ComputedAt  time.Time `json:"computedAt"`
}

func (r LeaderboardRestModel) GetName() string { return "rankings" }
func (r LeaderboardRestModel) GetID() string   { return strconv.Itoa(int(r.Id)) }

func (r *LeaderboardRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func TransformLeaderboard(m Model) (LeaderboardRestModel, error) {
	return LeaderboardRestModel{
		Id:          m.CharacterId(),
		CharacterId: m.CharacterId(),
		Name:        m.Name(),
		WorldId:     m.WorldId(),
		JobId:       m.JobId(),
		JobCategory: m.JobCategory(),
		Level:       m.Level(),
		Rank:        m.OverallRank(),
		RankMove:    m.OverallRankMove(),
		JobRank:     m.JobRank(),
		JobRankMove: m.JobRankMove(),
		ComputedAt:  m.ComputedAt(),
	}, nil
}
