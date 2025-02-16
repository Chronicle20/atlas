package skill

import (
	"strconv"
	"time"
)

type RestModel struct {
	Id          uint32    `json:"-"`
	Level       byte      `json:"level"`
	MasterLevel byte      `json:"masterLevel"`
	Expiration  time.Time `json:"expiration"`
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

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:          rm.Id,
		level:       rm.Level,
		masterLevel: rm.MasterLevel,
		expiration:  rm.Expiration,
	}, nil
}
