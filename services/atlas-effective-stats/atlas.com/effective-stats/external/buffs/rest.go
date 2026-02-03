package buffs

import (
	"strconv"
	"time"
)

// BuffRestModel represents a buff from atlas-buffs service
type BuffRestModel struct {
	Id        string          `json:"-"`
	SourceId  int32           `json:"sourceId"`
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
