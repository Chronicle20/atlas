package factory

import (
	"strconv"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type RestModel struct {
	Id           uint32   `json:"-"`
	AccountId    uint32   `json:"accountId"`
	WorldId      world.Id `json:"worldId"`
	Name         string   `json:"name"`
	Gender       byte     `json:"gender"`
	JobIndex     uint32   `json:"jobIndex"`
	SubJobIndex  uint32   `json:"subJobIndex"`
	Face         uint32   `json:"face"`
	Hair         uint32   `json:"hair"`
	HairColor    uint32   `json:"hairColor"`
	SkinColor    byte     `json:"skinColor"`
	Top          uint32   `json:"top"`
	Bottom       uint32   `json:"bottom"`
	Shoes        uint32   `json:"shoes"`
	Weapon       uint32   `json:"weapon"`
	Level        byte     `json:"level"`
	Strength     uint16   `json:"strength"`
	Dexterity    uint16   `json:"dexterity"`
	Intelligence uint16   `json:"intelligence"`
	Luck         uint16   `json:"luck"`
	Hp           uint16   `json:"hp"`
	Mp           uint16   `json:"mp"`
	MapId        _map.Id  `json:"mapId"`
}

func (r RestModel) GetName() string {
	return "characters"
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

// MapleLifeCreateRestModel mirrors the factory's MapleLifeCreateRestModel
// (services/atlas-character-factory/atlas.com/character-factory/factory/maple_life.go)
// field-for-field: only what the player chose in the Maple Life dialog --
// the class ordinal, gender, the four look values, and the SP level -- is
// sent. The factory owns what a Maple Life character of that class actually
// is (design.md §11 A5).
type MapleLifeCreateRestModel struct {
	AccountId    uint32 `json:"accountId"`
	WorldId      byte   `json:"worldId"`
	Name         string `json:"name"`
	ClassOrdinal uint32 `json:"classOrdinal"`
	Gender       byte   `json:"gender"`
	Face         uint32 `json:"face"`
	Hair         uint32 `json:"hair"`
	HairColor    uint32 `json:"hairColor"`
	SkinColor    byte   `json:"skinColor"`
	SP           byte   `json:"sp"`
}

// GetName, GetID, and SetID satisfy jsonapi.MarshalIdentifier /
// jsonapi.UnmarshalIdentifier so MapleLifeCreateRestModel encodes as JSON:API
// type "maple-life-create", matching the factory's
// MapleLifeCreateRestModel.GetName() in factory/resource.go.
func (r MapleLifeCreateRestModel) GetName() string     { return "maple-life-create" }
func (r MapleLifeCreateRestModel) GetID() string       { return "" }
func (r *MapleLifeCreateRestModel) SetID(string) error { return nil }

// CreateCharacterResponse represents the response for character creation requests
type CreateCharacterResponse struct {
	TransactionId string `json:"transactionId"`
}

func (r CreateCharacterResponse) GetName() string {
	return "characters"
}

func (r CreateCharacterResponse) GetID() string {
	return r.TransactionId
}

func (r *CreateCharacterResponse) SetID(strId string) error {
	r.TransactionId = strId
	return nil
}

// SetToOneReferenceID / SetToManyReferenceIDs are required by api2go's unmarshal
// even though this client doesn't care about the character resource's
// relationships (see libs/atlas-rest gotcha): a target struct must implement
// them or unmarshal errors whenever the upstream response includes a
// relationships block.
func (r *CreateCharacterResponse) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *CreateCharacterResponse) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}
