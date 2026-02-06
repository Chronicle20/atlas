package quest

import (
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel represents the full quest definition combining QuestInfo, Check, and Act data
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

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

// RequirementsRestModel represents quest requirements (for both start and completion)
type RequirementsRestModel struct {
	NpcId           uint32             `json:"npcId,omitempty"`
	LevelMin        uint16             `json:"levelMin,omitempty"`
	LevelMax        uint16             `json:"levelMax,omitempty"`
	FameMin         int16              `json:"fameMin,omitempty"`
	MesoMin         uint32             `json:"mesoMin,omitempty"`
	MesoMax         uint32             `json:"mesoMax,omitempty"`
	Jobs            []uint16           `json:"jobs,omitempty"`
	Quests          []QuestRequirement `json:"quests,omitempty"`
	Items           []ItemRequirement  `json:"items,omitempty"`
	Mobs            []MobRequirement   `json:"mobs,omitempty"`
	FieldEnter      []uint32           `json:"fieldEnter,omitempty"`
	Pet             []uint32           `json:"pet,omitempty"`
	PetTamenessMin  int16              `json:"petTamenessMin,omitempty"`
	DayOfWeek       string             `json:"dayOfWeek,omitempty"`
	Start           string             `json:"start,omitempty"`
	End             string             `json:"end,omitempty"`
	Interval        uint32             `json:"interval,omitempty"`
	StartScript     string             `json:"startScript,omitempty"`
	EndScript       string             `json:"endScript,omitempty"`
	InfoNumber      uint32             `json:"infoNumber,omitempty"`
	NormalAutoStart bool               `json:"normalAutoStart,omitempty"`
	CompletionCount uint32             `json:"completionCount,omitempty"`
}

// QuestRequirement represents a prerequisite quest requirement
type QuestRequirement struct {
	Id    uint32 `json:"id"`
	State uint8  `json:"state"` // 0 = not started, 1 = started, 2 = completed
}

// ItemRequirement represents an item requirement (for start/complete)
type ItemRequirement struct {
	Id    uint32 `json:"id"`
	Count int32  `json:"count"` // Can be negative for removal
}

// MobRequirement represents a mob kill requirement
type MobRequirement struct {
	Id    uint32 `json:"id"`
	Count uint32 `json:"count"`
}

// ActionsRestModel represents quest actions (rewards and effects)
type ActionsRestModel struct {
	NpcId      uint32        `json:"npcId,omitempty"`
	Exp        int32         `json:"exp,omitempty"`
	Money      int32         `json:"money,omitempty"`
	Fame       int16         `json:"fame,omitempty"`
	Items      []ItemReward  `json:"items,omitempty"`
	Skills     []SkillReward `json:"skills,omitempty"`
	NextQuest  uint32        `json:"nextQuest,omitempty"`
	BuffItemId uint32        `json:"buffItemId,omitempty"`
	Interval   uint32        `json:"interval,omitempty"`
	LevelMin   uint16        `json:"levelMin,omitempty"`
}

// ItemReward represents an item reward
type ItemReward struct {
	Id         uint32 `json:"id"`
	Count      int32  `json:"count"`                // Negative for removal
	Job        int32  `json:"job,omitempty"`        // Job requirement for this reward (bitmask)
	Gender     int8   `json:"gender,omitempty"`     // -1 = any, 0 = male, 1 = female
	Prop       int32  `json:"prop,omitempty"`       // Probability (-1 = guaranteed, 0+ = chance)
	Period     uint32 `json:"period,omitempty"`     // Duration in minutes (0 = permanent)
	DateExpire string `json:"dateExpire,omitempty"` // Expiration date string
	Var        uint32 `json:"var,omitempty"`        // Variable selection
}

// SkillReward represents a skill reward
type SkillReward struct {
	Id          uint32   `json:"id"`
	Level       int32    `json:"level,omitempty"` // -1 = remove skill
	MasterLevel uint32   `json:"masterLevel,omitempty"`
	Jobs        []uint16 `json:"jobs,omitempty"` // Jobs that can receive this skill
}
