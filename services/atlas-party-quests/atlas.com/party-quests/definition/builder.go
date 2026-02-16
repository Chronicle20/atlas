package definition

import (
	"atlas-party-quests/condition"
	"atlas-party-quests/reward"
	"atlas-party-quests/stage"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Builder struct {
	id                uuid.UUID
	questId           string
	name              string
	fieldLock         string
	duration          uint64
	registration      Registration
	startRequirements []condition.Model
	startEvents       []EventTrigger
	failRequirements  []condition.Model
	exit              uint32
	stages            []stage.Model
	rewards           []reward.Model
	createdAt         time.Time
	updatedAt         time.Time
}

func NewBuilder() *Builder {
	return &Builder{
		id:                uuid.Nil,
		startRequirements: make([]condition.Model, 0),
		startEvents:       make([]EventTrigger, 0),
		failRequirements:  make([]condition.Model, 0),
		stages:            make([]stage.Model, 0),
		rewards:           make([]reward.Model, 0),
		createdAt:         time.Now(),
		updatedAt:         time.Now(),
	}
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetQuestId(qid string) *Builder {
	b.questId = qid
	return b
}

func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

func (b *Builder) SetFieldLock(fl string) *Builder {
	b.fieldLock = fl
	return b
}

func (b *Builder) SetDuration(d uint64) *Builder {
	b.duration = d
	return b
}

func (b *Builder) SetRegistration(r Registration) *Builder {
	b.registration = r
	return b
}

func (b *Builder) SetStartRequirements(reqs []condition.Model) *Builder {
	b.startRequirements = reqs
	return b
}

func (b *Builder) SetStartEvents(events []EventTrigger) *Builder {
	b.startEvents = events
	return b
}

func (b *Builder) SetFailRequirements(reqs []condition.Model) *Builder {
	b.failRequirements = reqs
	return b
}

func (b *Builder) SetExit(exit uint32) *Builder {
	b.exit = exit
	return b
}

func (b *Builder) SetStages(stages []stage.Model) *Builder {
	b.stages = stages
	return b
}

func (b *Builder) SetRewards(rewards []reward.Model) *Builder {
	b.rewards = rewards
	return b
}

func (b *Builder) SetCreatedAt(t time.Time) *Builder {
	b.createdAt = t
	return b
}

func (b *Builder) SetUpdatedAt(t time.Time) *Builder {
	b.updatedAt = t
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.questId == "" {
		return Model{}, errors.New("questId is required")
	}
	if b.name == "" {
		return Model{}, errors.New("name is required")
	}
	return Model{
		id:                b.id,
		questId:           b.questId,
		name:              b.name,
		fieldLock:         b.fieldLock,
		duration:          b.duration,
		registration:      b.registration,
		startRequirements: b.startRequirements,
		startEvents:       b.startEvents,
		failRequirements:  b.failRequirements,
		exit:              b.exit,
		stages:            b.stages,
		rewards:           b.rewards,
		createdAt:         b.createdAt,
		updatedAt:         b.updatedAt,
	}, nil
}
