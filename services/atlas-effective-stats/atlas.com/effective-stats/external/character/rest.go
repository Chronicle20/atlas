package character

import (
	"strconv"

	"github.com/Chronicle20/atlas-constants/world"
)

// RestModel represents a character from atlas-character service
type RestModel struct {
	Id           uint32   `json:"-"`
	AccountId    uint32   `json:"accountId"`
	WorldId      world.Id `json:"worldId"`
	Name         string   `json:"name"`
	Level        byte     `json:"level"`
	Strength     uint16   `json:"strength"`
	Dexterity    uint16   `json:"dexterity"`
	Intelligence uint16   `json:"intelligence"`
	Luck         uint16   `json:"luck"`
	Hp           uint16   `json:"hp"`
	MaxHp        uint16   `json:"maxHp"`
	Mp           uint16   `json:"mp"`
	MaxMp        uint16   `json:"maxMp"`
}

func (r RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
