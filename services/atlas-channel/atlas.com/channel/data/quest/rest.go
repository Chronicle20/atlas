package quest

import "strconv"

type RestModel struct {
	Id                uint32                `json:"-"`
	Name              string                `json:"name"`
	Parent            string                `json:"parent,omitempty"`
	Area              uint32                `json:"area"`
	Order             uint32                `json:"order,omitempty"`
	AutoStart         bool                  `json:"autoStart"`
	AutoPreComplete   bool                  `json:"autoPreComplete"`
	AutoComplete      bool                  `json:"autoComplete"`
	TimeLimit         uint32                `json:"timeLimit,omitempty"`
	TimeLimit2        uint32                `json:"timeLimit2,omitempty"`
	SelectedMob       bool                  `json:"selectedMob,omitempty"`
	Summary           string                `json:"summary,omitempty"`
	DemandSummary     string                `json:"demandSummary,omitempty"`
	RewardSummary     string                `json:"rewardSummary,omitempty"`
	StartRequirements RequirementsRestModel `json:"startRequirements"`
	EndRequirements   RequirementsRestModel `json:"endRequirements"`
	StartActions      ActionsRestModel      `json:"startActions"`
	EndActions        ActionsRestModel      `json:"endActions"`
}

func (r RestModel) GetName() string {
	return "quests"
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

type RequirementsRestModel struct {
	NpcId           uint32                 `json:"npcId,omitempty"`
	LevelMin        uint16                 `json:"levelMin,omitempty"`
	LevelMax        uint16                 `json:"levelMax,omitempty"`
	FameMin         int16                  `json:"fameMin,omitempty"`
	MesoMin         uint32                 `json:"mesoMin,omitempty"`
	MesoMax         uint32                 `json:"mesoMax,omitempty"`
	Jobs            []uint16               `json:"jobs,omitempty"`
	Quests          []QuestRequirementRest `json:"quests,omitempty"`
	Items           []ItemRequirementRest  `json:"items,omitempty"`
	Mobs            []MobRequirementRest   `json:"mobs,omitempty"`
	FieldEnter      []uint32               `json:"fieldEnter,omitempty"`
	Pet             []uint32               `json:"pet,omitempty"`
	PetTamenessMin  int16                  `json:"petTamenessMin,omitempty"`
	DayOfWeek       []string               `json:"dayOfWeek,omitempty"`
	Start           string                 `json:"start,omitempty"`
	End             string                 `json:"end,omitempty"`
	Interval        uint32                 `json:"interval,omitempty"`
	StartScript     string                 `json:"startScript,omitempty"`
	EndScript       string                 `json:"endScript,omitempty"`
	InfoNumber      uint32                 `json:"infoNumber,omitempty"`
	NormalAutoStart bool                   `json:"normalAutoStart,omitempty"`
	CompletionCount uint32                 `json:"completionCount,omitempty"`
}

type QuestRequirementRest struct {
	Id    uint32 `json:"id"`
	State uint8  `json:"state"`
}

type ItemRequirementRest struct {
	Id    uint32 `json:"id"`
	Count int32  `json:"count"`
}

type MobRequirementRest struct {
	Id    uint32 `json:"id"`
	Count uint32 `json:"count"`
}

type ActionsRestModel struct {
	NpcId      uint32            `json:"npcId,omitempty"`
	Exp        int32             `json:"exp,omitempty"`
	Money      int32             `json:"money,omitempty"`
	Fame       int16             `json:"fame,omitempty"`
	Items      []ItemRewardRest  `json:"items,omitempty"`
	Skills     []SkillRewardRest `json:"skills,omitempty"`
	NextQuest  uint32            `json:"nextQuest,omitempty"`
	BuffItemId uint32            `json:"buffItemId,omitempty"`
	Interval   uint32            `json:"interval,omitempty"`
	LevelMin   uint16            `json:"levelMin,omitempty"`
}

type ItemRewardRest struct {
	Id         uint32 `json:"id"`
	Count      int32  `json:"count"`
	Job        int32  `json:"job,omitempty"`
	Gender     int8   `json:"gender,omitempty"`
	Prop       int32  `json:"prop,omitempty"`
	Period     uint32 `json:"period,omitempty"`
	DateExpire string `json:"dateExpire,omitempty"`
	Var        uint32 `json:"var,omitempty"`
}

type SkillRewardRest struct {
	Id          uint32   `json:"id"`
	Level       int32    `json:"level,omitempty"`
	MasterLevel uint32   `json:"masterLevel,omitempty"`
	Jobs        []uint16 `json:"jobs,omitempty"`
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:                m.id,
		Name:              m.name,
		Parent:            m.parent,
		Area:              m.area,
		Order:             m.order,
		AutoStart:         m.autoStart,
		AutoPreComplete:   m.autoPreComplete,
		AutoComplete:      m.autoComplete,
		TimeLimit:         m.timeLimit,
		TimeLimit2:        m.timeLimit2,
		SelectedMob:       m.selectedMob,
		Summary:           m.summary,
		DemandSummary:     m.demandSummary,
		RewardSummary:     m.rewardSummary,
		StartRequirements: transformRequirements(m.startRequirements),
		EndRequirements:   transformRequirements(m.endRequirements),
		StartActions:      transformActions(m.startActions),
		EndActions:        transformActions(m.endActions),
	}, nil
}

func transformRequirements(m RequirementsModel) RequirementsRestModel {
	var quests []QuestRequirementRest
	if m.quests != nil {
		quests = make([]QuestRequirementRest, len(m.quests))
		for i, q := range m.quests {
			quests[i] = QuestRequirementRest{Id: q.id, State: q.state}
		}
	}

	var items []ItemRequirementRest
	if m.items != nil {
		items = make([]ItemRequirementRest, len(m.items))
		for i, item := range m.items {
			items[i] = ItemRequirementRest{Id: item.id, Count: item.count}
		}
	}

	var mobs []MobRequirementRest
	if m.mobs != nil {
		mobs = make([]MobRequirementRest, len(m.mobs))
		for i, mob := range m.mobs {
			mobs[i] = MobRequirementRest{Id: mob.id, Count: mob.count}
		}
	}

	jobs := copyUint16Slice(m.jobs)
	fieldEnter := copyUint32Slice(m.fieldEnter)
	pet := copyUint32Slice(m.pet)
	dayOfWeek := copyStringSlice(m.dayOfWeek)

	return RequirementsRestModel{
		NpcId:           m.npcId,
		LevelMin:        m.levelMin,
		LevelMax:        m.levelMax,
		FameMin:         m.fameMin,
		MesoMin:         m.mesoMin,
		MesoMax:         m.mesoMax,
		Jobs:            jobs,
		Quests:          quests,
		Items:           items,
		Mobs:            mobs,
		FieldEnter:      fieldEnter,
		Pet:             pet,
		PetTamenessMin:  m.petTamenessMin,
		DayOfWeek:       dayOfWeek,
		Start:           m.start,
		End:             m.end,
		Interval:        m.interval,
		StartScript:     m.startScript,
		EndScript:       m.endScript,
		InfoNumber:      m.infoNumber,
		NormalAutoStart: m.normalAutoStart,
		CompletionCount: m.completionCount,
	}
}

func transformActions(m ActionsModel) ActionsRestModel {
	var items []ItemRewardRest
	if m.items != nil {
		items = make([]ItemRewardRest, len(m.items))
		for i, item := range m.items {
			items[i] = ItemRewardRest{
				Id:         item.id,
				Count:      item.count,
				Job:        item.job,
				Gender:     item.gender,
				Prop:       item.prop,
				Period:     item.period,
				DateExpire: item.dateExpire,
				Var:        item.variable,
			}
		}
	}

	var skills []SkillRewardRest
	if m.skills != nil {
		skills = make([]SkillRewardRest, len(m.skills))
		for i, skill := range m.skills {
			skills[i] = SkillRewardRest{
				Id:          skill.id,
				Level:       skill.level,
				MasterLevel: skill.masterLevel,
				Jobs:        copyUint16Slice(skill.jobs),
			}
		}
	}

	return ActionsRestModel{
		NpcId:      m.npcId,
		Exp:        m.exp,
		Money:      m.money,
		Fame:       m.fame,
		Items:      items,
		Skills:     skills,
		NextQuest:  m.nextQuest,
		BuffItemId: m.buffItemId,
		Interval:   m.interval,
		LevelMin:   m.levelMin,
	}
}

// copyUint16Slice, copyUint32Slice, and copyStringSlice preserve the
// nil-vs-empty distinction of src: they return nil for a nil src rather than
// an allocated empty slice, so a Transform -> Extract round trip is exact.
func copyUint16Slice(src []uint16) []uint16 {
	if src == nil {
		return nil
	}
	dst := make([]uint16, len(src))
	copy(dst, src)
	return dst
}

func copyUint32Slice(src []uint32) []uint32 {
	if src == nil {
		return nil
	}
	dst := make([]uint32, len(src))
	copy(dst, src)
	return dst
}

func copyStringSlice(src []string) []string {
	if src == nil {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:                rm.Id,
		name:              rm.Name,
		parent:            rm.Parent,
		area:              rm.Area,
		order:             rm.Order,
		autoStart:         rm.AutoStart,
		autoPreComplete:   rm.AutoPreComplete,
		autoComplete:      rm.AutoComplete,
		timeLimit:         rm.TimeLimit,
		timeLimit2:        rm.TimeLimit2,
		selectedMob:       rm.SelectedMob,
		summary:           rm.Summary,
		demandSummary:     rm.DemandSummary,
		rewardSummary:     rm.RewardSummary,
		startRequirements: extractRequirements(rm.StartRequirements),
		endRequirements:   extractRequirements(rm.EndRequirements),
		startActions:      extractActions(rm.StartActions),
		endActions:        extractActions(rm.EndActions),
	}, nil
}

func extractRequirements(rm RequirementsRestModel) RequirementsModel {
	quests := make([]QuestRequirementModel, len(rm.Quests))
	for i, q := range rm.Quests {
		quests[i] = QuestRequirementModel{id: q.Id, state: q.State}
	}

	items := make([]ItemRequirementModel, len(rm.Items))
	for i, item := range rm.Items {
		items[i] = ItemRequirementModel{id: item.Id, count: item.Count}
	}

	mobs := make([]MobRequirementModel, len(rm.Mobs))
	for i, mob := range rm.Mobs {
		mobs[i] = MobRequirementModel{id: mob.Id, count: mob.Count}
	}

	return RequirementsModel{
		npcId:           rm.NpcId,
		levelMin:        rm.LevelMin,
		levelMax:        rm.LevelMax,
		fameMin:         rm.FameMin,
		mesoMin:         rm.MesoMin,
		mesoMax:         rm.MesoMax,
		jobs:            rm.Jobs,
		quests:          quests,
		items:           items,
		mobs:            mobs,
		fieldEnter:      rm.FieldEnter,
		pet:             rm.Pet,
		petTamenessMin:  rm.PetTamenessMin,
		dayOfWeek:       rm.DayOfWeek,
		start:           rm.Start,
		end:             rm.End,
		interval:        rm.Interval,
		startScript:     rm.StartScript,
		endScript:       rm.EndScript,
		infoNumber:      rm.InfoNumber,
		normalAutoStart: rm.NormalAutoStart,
		completionCount: rm.CompletionCount,
	}
}

func extractActions(rm ActionsRestModel) ActionsModel {
	items := make([]ItemRewardModel, len(rm.Items))
	for i, item := range rm.Items {
		items[i] = ItemRewardModel{
			id:         item.Id,
			count:      item.Count,
			job:        item.Job,
			gender:     item.Gender,
			prop:       item.Prop,
			period:     item.Period,
			dateExpire: item.DateExpire,
			variable:   item.Var,
		}
	}

	skills := make([]SkillRewardModel, len(rm.Skills))
	for i, skill := range rm.Skills {
		skills[i] = SkillRewardModel{
			id:          skill.Id,
			level:       skill.Level,
			masterLevel: skill.MasterLevel,
			jobs:        skill.Jobs,
		}
	}

	return ActionsModel{
		npcId:      rm.NpcId,
		exp:        rm.Exp,
		money:      rm.Money,
		fame:       rm.Fame,
		items:      items,
		skills:     skills,
		nextQuest:  rm.NextQuest,
		buffItemId: rm.BuffItemId,
		interval:   rm.Interval,
		levelMin:   rm.LevelMin,
	}
}
