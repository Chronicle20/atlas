package definition

import (
	"atlas-party-quests/condition"
	"atlas-party-quests/reward"
	"atlas-party-quests/stage"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Registration struct {
	regType  string
	mode     string
	duration int64
	mapId    uint32
}

func (r Registration) Type() string  { return r.regType }
func (r Registration) Mode() string  { return r.mode }
func (r Registration) Duration() int64 { return r.duration }
func (r Registration) MapId() uint32 { return r.mapId }

type EventTrigger struct {
	triggerType string
	target      string
	value       string
}

func (e EventTrigger) Type() string   { return e.triggerType }
func (e EventTrigger) Target() string { return e.target }
func (e EventTrigger) Value() string  { return e.value }

type Model struct {
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

func (m Model) Id() uuid.UUID                          { return m.id }
func (m Model) QuestId() string                        { return m.questId }
func (m Model) Name() string                           { return m.name }
func (m Model) FieldLock() string                      { return m.fieldLock }
func (m Model) Duration() uint64                       { return m.duration }
func (m Model) Registration() Registration             { return m.registration }
func (m Model) StartRequirements() []condition.Model   { return m.startRequirements }
func (m Model) StartEvents() []EventTrigger            { return m.startEvents }
func (m Model) FailRequirements() []condition.Model    { return m.failRequirements }
func (m Model) Exit() uint32                           { return m.exit }
func (m Model) Stages() []stage.Model                  { return m.stages }
func (m Model) Rewards() []reward.Model                { return m.rewards }
func (m Model) CreatedAt() time.Time                   { return m.createdAt }
func (m Model) UpdatedAt() time.Time                   { return m.updatedAt }

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
