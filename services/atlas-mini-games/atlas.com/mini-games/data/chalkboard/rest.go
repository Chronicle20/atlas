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

// SetToOneReferenceID / SetToManyReferenceIDs are defensive no-op stubs so a
// future relationships block on atlas-chalkboards' resource cannot break the
// decode (task-037 failure class, see libs/atlas-rest/CLAUDE.md). Only the
// presence of the resource matters to the validation ladder.
func (r *RestModel) SetToOneReferenceID(_ string, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}
