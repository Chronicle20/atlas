package definition

import (
	"atlas-party-quests/condition"
	"atlas-party-quests/reward"
	"atlas-party-quests/stage"
	"time"

	"github.com/google/uuid"
)

type Registration struct {
	regType  string
	mode     string
	duration int64
	mapId    uint32
	affinity string
}

func (r Registration) Type() string    { return r.regType }
func (r Registration) Mode() string    { return r.mode }
func (r Registration) Duration() int64 { return r.duration }
func (r Registration) MapId() uint32   { return r.mapId }
func (r Registration) Affinity() string { return r.affinity }

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
	bonus             *Bonus
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
func (m Model) Bonus() *Bonus                          { return m.bonus }
func (m Model) Stages() []stage.Model                  { return m.stages }
func (m Model) Rewards() []reward.Model                { return m.rewards }
func (m Model) CreatedAt() time.Time                   { return m.createdAt }
func (m Model) UpdatedAt() time.Time                   { return m.updatedAt }

