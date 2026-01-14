package skill

import (
	"strconv"
	"time"
)

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

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:                m.Id(),
		Level:             m.Level(),
		MasterLevel:       m.MasterLevel(),
		Expiration:        m.Expiration(),
		CooldownExpiresAt: m.CooldownExpiresAt(),
	}, nil
}
