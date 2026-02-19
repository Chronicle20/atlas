package stage

import (
	"atlas-party-quests/condition"
	"atlas-party-quests/reward"
)

type RestModel struct {
	Index           uint32                 `json:"index"`
	Name            string                 `json:"name"`
	MapIds          []uint32               `json:"mapIds"`
	Type            string                 `json:"type"`
	Duration        uint64                 `json:"duration"`
	ClearConditions []condition.RestModel   `json:"clearConditions"`
	ClearActions    []string               `json:"clearActions,omitempty"`
	Rewards         []reward.RestModel      `json:"rewards"`
	WarpType        string                 `json:"warpType"`
	Properties      map[string]any         `json:"properties,omitempty"`
}

func Transform(m Model) (RestModel, error) {
	conditions := make([]condition.RestModel, 0, len(m.ClearConditions()))
	for _, c := range m.ClearConditions() {
		rc, err := condition.Transform(c)
		if err != nil {
			return RestModel{}, err
		}
		conditions = append(conditions, rc)
	}

	rewards := make([]reward.RestModel, 0, len(m.Rewards()))
	for _, r := range m.Rewards() {
		rr, err := reward.Transform(r)
		if err != nil {
			return RestModel{}, err
		}
		rewards = append(rewards, rr)
	}

	return RestModel{
		Index:           m.Index(),
		Name:            m.Name(),
		MapIds:          m.MapIds(),
		Type:            m.Type(),
		Duration:        m.Duration(),
		ClearConditions: conditions,
		ClearActions:    m.ClearActions(),
		Rewards:         rewards,
		WarpType:        m.WarpType(),
		Properties:      m.Properties(),
	}, nil
}

func Extract(r RestModel) (Model, error) {
	conditions := make([]condition.Model, 0, len(r.ClearConditions))
	for _, rc := range r.ClearConditions {
		c, err := condition.Extract(rc)
		if err != nil {
			return Model{}, err
		}
		conditions = append(conditions, c)
	}

	rewards := make([]reward.Model, 0, len(r.Rewards))
	for _, rr := range r.Rewards {
		rew, err := reward.Extract(rr)
		if err != nil {
			return Model{}, err
		}
		rewards = append(rewards, rew)
	}

	return NewBuilder().
		SetIndex(r.Index).
		SetName(r.Name).
		SetMapIds(r.MapIds).
		SetType(r.Type).
		SetDuration(r.Duration).
		SetClearConditions(conditions).
		SetClearActions(r.ClearActions).
		SetRewards(rewards).
		SetWarpType(r.WarpType).
		SetProperties(r.Properties).
		Build()
}
