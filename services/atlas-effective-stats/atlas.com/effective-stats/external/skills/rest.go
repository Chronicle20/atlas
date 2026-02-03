package skills

import (
	"strconv"
	"time"
)

// RestModel represents a character's skill from atlas-skills service
type RestModel struct {
	Id                uint32    `json:"-"`
	Level             byte      `json:"level"`
	MasterLevel       byte      `json:"masterLevel"`
	Expiration        time.Time `json:"expiration"`
	CooldownExpiresAt time.Time `json:"cooldownExpiresAt"`
}

func (r RestModel) GetName() string {
	return "skills"
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
