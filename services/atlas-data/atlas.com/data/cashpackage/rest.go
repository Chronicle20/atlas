package cashpackage

import (
	"strconv"
)

type RestModel struct {
	Id            uint32   `json:"-"`
	SerialNumbers []uint32 `json:"serialNumbers"`
}

func (r RestModel) GetName() string {
	return "cashPackages"
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

// SetToOneReferenceID and SetToManyReferenceIDs satisfy jsonapi's
// EditToOneRelations/EditToManyRelations interfaces. RestModel carries no
// relationships, so both are no-ops.
func (r *RestModel) SetToOneReferenceID(_ string, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}
