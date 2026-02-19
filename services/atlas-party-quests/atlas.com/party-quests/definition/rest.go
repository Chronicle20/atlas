package definition

import (
	"atlas-party-quests/condition"
	"atlas-party-quests/reward"
	"atlas-party-quests/stage"
	"fmt"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

const Resource = "definitions"

type RegistrationRestModel struct {
	Type     string `json:"type"`
	Mode     string `json:"mode"`
	Duration int64  `json:"duration"`
	MapId    uint32 `json:"mapId"`
	Affinity string `json:"affinity,omitempty"`
}

type BonusRestModel struct {
	MapId           uint32         `json:"mapId"`
	Duration        uint64         `json:"duration"`
	Entry           string         `json:"entry"`
	CompletionMapId uint32         `json:"completionMapId,omitempty"`
	Properties      map[string]any `json:"properties,omitempty"`
}

type EventTriggerRestModel struct {
	Type   string `json:"type"`
	Target string `json:"target"`
	Value  string `json:"value"`
}

type RestModel struct {
	Id                uuid.UUID               `json:"-"`
	QuestId           string                  `json:"questId"`
	Name              string                  `json:"name"`
	FieldLock         string                  `json:"fieldLock"`
	Duration          uint64                  `json:"duration"`
	Registration      RegistrationRestModel   `json:"registration"`
	StartRequirements []condition.RestModel    `json:"startRequirements"`
	StartEvents       []EventTriggerRestModel  `json:"startEvents"`
	FailRequirements  []condition.RestModel    `json:"failRequirements"`
	Exit              uint32                  `json:"exit"`
	Bonus             *BonusRestModel          `json:"bonus,omitempty"`
	Stages            []stage.RestModel        `json:"stages"`
	Rewards           []reward.RestModel       `json:"rewards"`
}

func (r RestModel) GetName() string {
	return Resource
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

func (r *RestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid definition ID: %w", err)
	}
	r.Id = id
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

func Transform(m Model) (RestModel, error) {
	startReqs := make([]condition.RestModel, 0, len(m.StartRequirements()))
	for _, c := range m.StartRequirements() {
		rc, err := condition.Transform(c)
		if err != nil {
			return RestModel{}, err
		}
		startReqs = append(startReqs, rc)
	}

	startEvents := make([]EventTriggerRestModel, 0, len(m.StartEvents()))
	for _, e := range m.StartEvents() {
		startEvents = append(startEvents, EventTriggerRestModel{
			Type:   e.Type(),
			Target: e.Target(),
			Value:  e.Value(),
		})
	}

	failReqs := make([]condition.RestModel, 0, len(m.FailRequirements()))
	for _, c := range m.FailRequirements() {
		rc, err := condition.Transform(c)
		if err != nil {
			return RestModel{}, err
		}
		failReqs = append(failReqs, rc)
	}

	stages := make([]stage.RestModel, 0, len(m.Stages()))
	for _, s := range m.Stages() {
		rs, err := stage.Transform(s)
		if err != nil {
			return RestModel{}, err
		}
		stages = append(stages, rs)
	}

	rewards := make([]reward.RestModel, 0, len(m.Rewards()))
	for _, r := range m.Rewards() {
		rr, err := reward.Transform(r)
		if err != nil {
			return RestModel{}, err
		}
		rewards = append(rewards, rr)
	}

	var bonusRest *BonusRestModel
	if m.Bonus() != nil {
		bonusRest = &BonusRestModel{
			MapId:           m.Bonus().MapId(),
			Duration:        m.Bonus().Duration(),
			Entry:           string(m.Bonus().Entry()),
			CompletionMapId: m.Bonus().CompletionMapId(),
			Properties:      m.Bonus().Properties(),
		}
	}

	reg := m.Registration()
	return RestModel{
		Id:      m.Id(),
		QuestId: m.QuestId(),
		Name:    m.Name(),
		FieldLock: m.FieldLock(),
		Duration:  m.Duration(),
		Registration: RegistrationRestModel{
			Type:     reg.Type(),
			Mode:     reg.Mode(),
			Duration: reg.Duration(),
			MapId:    reg.MapId(),
			Affinity: reg.Affinity(),
		},
		StartRequirements: startReqs,
		StartEvents:       startEvents,
		FailRequirements:  failReqs,
		Exit:              m.Exit(),
		Bonus:             bonusRest,
		Stages:            stages,
		Rewards:           rewards,
	}, nil
}

func Extract(r RestModel) (Model, error) {
	if r.QuestId == "" {
		return Model{}, fmt.Errorf("questId is required")
	}
	if r.Name == "" {
		return Model{}, fmt.Errorf("name is required")
	}

	startReqs := make([]condition.Model, 0, len(r.StartRequirements))
	for _, rc := range r.StartRequirements {
		c, err := condition.Extract(rc)
		if err != nil {
			return Model{}, err
		}
		startReqs = append(startReqs, c)
	}

	startEvents := make([]EventTrigger, 0, len(r.StartEvents))
	for _, re := range r.StartEvents {
		startEvents = append(startEvents, EventTrigger{
			triggerType: re.Type,
			target:      re.Target,
			value:       re.Value,
		})
	}

	failReqs := make([]condition.Model, 0, len(r.FailRequirements))
	for _, rc := range r.FailRequirements {
		c, err := condition.Extract(rc)
		if err != nil {
			return Model{}, err
		}
		failReqs = append(failReqs, c)
	}

	stages := make([]stage.Model, 0, len(r.Stages))
	for _, rs := range r.Stages {
		s, err := stage.Extract(rs)
		if err != nil {
			return Model{}, err
		}
		stages = append(stages, s)
	}

	rewards := make([]reward.Model, 0, len(r.Rewards))
	for _, rr := range r.Rewards {
		rew, err := reward.Extract(rr)
		if err != nil {
			return Model{}, err
		}
		rewards = append(rewards, rew)
	}

	var bonus *Bonus
	if r.Bonus != nil {
		bonus = &Bonus{
			mapId:           r.Bonus.MapId,
			duration:        r.Bonus.Duration,
			entry:           BonusEntry(r.Bonus.Entry),
			completionMapId: r.Bonus.CompletionMapId,
			properties:      r.Bonus.Properties,
		}
	}

	builder := NewBuilder()
	if r.Id != uuid.Nil {
		builder.SetId(r.Id)
	}

	return builder.
		SetQuestId(r.QuestId).
		SetName(r.Name).
		SetFieldLock(r.FieldLock).
		SetDuration(r.Duration).
		SetRegistration(Registration{
			regType:  r.Registration.Type,
			mode:     r.Registration.Mode,
			duration: r.Registration.Duration,
			mapId:    r.Registration.MapId,
			affinity: r.Registration.Affinity,
		}).
		SetStartRequirements(startReqs).
		SetStartEvents(startEvents).
		SetFailRequirements(failReqs).
		SetExit(r.Exit).
		SetBonus(bonus).
		SetStages(stages).
		SetRewards(rewards).
		Build()
}
