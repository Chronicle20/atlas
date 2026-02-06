package factory

import (
	"strconv"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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
