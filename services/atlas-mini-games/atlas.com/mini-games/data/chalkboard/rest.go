package chalkboard

import (
	"strconv"
)

// RestModel mirrors the atlas-chalkboards single-chalkboard resource. Only its
// presence matters to the mini-game validation ladder (an open chalkboard
// blocks opening a mini-room).
type RestModel struct {
	Id      uint32 `json:"-"`
	Message string `json:"message"`
}

func (r RestModel) GetName() string {
	return "chalkboards"
}

func (r RestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
