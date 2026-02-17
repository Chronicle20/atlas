package stage

import (
	"atlas-party-quests/condition"
	"atlas-party-quests/reward"
	"errors"
)

type Builder struct {
	index           uint32
	name            string
	mapIds          []uint32
	stageType       string
	duration        uint64
	clearConditions []condition.Model
	clearActions    []string
	rewards         []reward.Model
	warpType        string
	properties      map[string]any
}

func NewBuilder() *Builder {
	return &Builder{
		mapIds:          make([]uint32, 0),
		clearConditions: make([]condition.Model, 0),
		clearActions:    make([]string, 0),
		rewards:         make([]reward.Model, 0),
		properties:      make(map[string]any),
	}
}

func (b *Builder) SetIndex(i uint32) *Builder {
	b.index = i
	return b
}

func (b *Builder) SetName(n string) *Builder {
	b.name = n
	return b
}

func (b *Builder) SetMapIds(ids []uint32) *Builder {
	b.mapIds = ids
	return b
}

func (b *Builder) SetType(t string) *Builder {
	b.stageType = t
	return b
}

func (b *Builder) SetDuration(d uint64) *Builder {
	b.duration = d
	return b
}

func (b *Builder) SetClearConditions(conditions []condition.Model) *Builder {
	b.clearConditions = conditions
	return b
}

func (b *Builder) SetClearActions(actions []string) *Builder {
	b.clearActions = actions
	return b
}

func (b *Builder) SetRewards(rewards []reward.Model) *Builder {
	b.rewards = rewards
	return b
}

func (b *Builder) SetWarpType(wt string) *Builder {
	b.warpType = wt
	return b
}

func (b *Builder) SetProperties(props map[string]any) *Builder {
	b.properties = props
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.stageType == "" {
		return Model{}, errors.New("stage type is required")
	}
	return Model{
		index:           b.index,
		name:            b.name,
		mapIds:          b.mapIds,
		stageType:       b.stageType,
		duration:        b.duration,
		clearConditions: b.clearConditions,
		clearActions:    b.clearActions,
		rewards:         b.rewards,
		warpType:        b.warpType,
		properties:      b.properties,
	}, nil
}
