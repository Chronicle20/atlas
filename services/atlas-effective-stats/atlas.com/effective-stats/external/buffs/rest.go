package buffs

import (
	"strconv"
	"time"
)

// BuffRestModel represents a buff from atlas-buffs service
type BuffRestModel struct {
	Id       string `json:"-"`
	SourceId int32  `json:"sourceId"`
	// Level is the source skill level. Needed to resolve level-dependent
	// payoffs from skill effect data (task-216: Energy Charge's `pad`).
	Level     byte            `json:"level"`
	Duration  int32           `json:"duration"`
	Changes   []StatRestModel `json:"changes"`
	CreatedAt time.Time       `json:"createdAt"`
	ExpiresAt time.Time       `json:"expiresAt"`
}

func (r BuffRestModel) GetName() string {
	return "buffs"
}

func (r BuffRestModel) GetID() string {
	return r.Id
}

func (r *BuffRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs satisfy the jsonapi
// UnmarshalToOneRelations / UnmarshalToManyRelations interfaces. BuffRestModel
// is the unmarshal target for GET /characters/{id}/buffs; should atlas-buffs
// ever add a relationship to that document, api2go's Unmarshal would otherwise
// fail with "struct does not implement UnmarshalToManyRelations" and the caller
// would surface it as a fetch error. Buff bonuses are computed entirely from
// attributes, so the methods are intentionally no-ops.
func (r *BuffRestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *BuffRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// StatRestModel represents a stat change from a buff
type StatRestModel struct {
	Id     string `json:"-"`
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func (r StatRestModel) GetName() string {
	return "stats"
}

func (r StatRestModel) GetID() string {
	return r.Id
}

func (r *StatRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs — see the BuffRestModel note.
// StatRestModel is only ever decoded as a nested attribute of BuffRestModel's
// changes array, so it carries no relationships of its own.
func (r *StatRestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *StatRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// BuffsArrayRestModel wraps an array of buffs for JSON:API compatibility
type BuffsArrayRestModel struct {
	Id    string          `json:"-"`
	Buffs []BuffRestModel `json:"buffs"`
}

func (r BuffsArrayRestModel) GetName() string {
	return "character-buffs"
}

func (r BuffsArrayRestModel) GetID() string {
	return r.Id
}

func (r *BuffsArrayRestModel) SetID(strId string) error {
	r.Id = strId
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs — see the BuffRestModel note.
func (r *BuffsArrayRestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *BuffsArrayRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// CharacterBuffsRestModel represents the character document with buffs
type CharacterBuffsRestModel struct {
	Id    uint32          `json:"-"`
	Buffs []BuffRestModel `json:"-"`
}

func (r CharacterBuffsRestModel) GetName() string {
	return "character-buffs"
}

func (r CharacterBuffsRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *CharacterBuffsRestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs — see the BuffRestModel note.
// The Buffs field is json:"-", so it would arrive as a toMany relationship
// rather than an attribute if this model were ever made an unmarshal target.
func (r *CharacterBuffsRestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *CharacterBuffsRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
