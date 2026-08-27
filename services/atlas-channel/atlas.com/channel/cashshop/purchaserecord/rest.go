package purchaserecord

import (
	"strconv"
)

type RestModel struct {
	SerialNumber uint32 `json:"-"`
	Purchased    bool   `json:"purchased"`
	Count        uint32 `json:"count"`
}

func (r RestModel) GetName() string {
	return "purchaseRecords"
}

func (r RestModel) GetID() string {
	return strconv.FormatUint(uint64(r.SerialNumber), 10)
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		return err
	}
	r.SerialNumber = uint32(id)
	return nil
}

// SetToOneReferenceID satisfies the api2go UnmarshalToOneRelations interface.
func (r *RestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

// SetToManyReferenceIDs satisfies the api2go UnmarshalToManyRelations interface.
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func Extract(rm RestModel) (Model, error) {
	return Model{
		serialNumber: rm.SerialNumber,
		purchased:    rm.Purchased,
		count:        rm.Count,
	}, nil
}
