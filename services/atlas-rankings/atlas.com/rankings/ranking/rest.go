package ranking

import (
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:          m.CharacterId(),
		WorldId:     m.WorldId(),
		Rank:        m.OverallRank(),
		RankMove:    m.OverallRankMove(),
		JobRank:     m.JobRank(),
		JobRankMove: m.JobRankMove(),
		ComputedAt:  m.ComputedAt(),
	}, nil
}

// TransformSlice maps a slice of domain Models to their REST projections.
// Returns the first transform error encountered, if any.
func TransformSlice(ms []Model) ([]RestModel, error) {
	out := make([]RestModel, 0, len(ms))
	for _, m := range ms {
		rm, err := Transform(m)
		if err != nil {
			return nil, err
		}
		out = append(out, rm)
	}
	return out, nil
}

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
