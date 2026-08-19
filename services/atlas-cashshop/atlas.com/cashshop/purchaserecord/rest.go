package purchaserecord

import (
	"strconv"
)

// RestModel answers "has this account ever bought this serial number", and
// how many times. A miss is 200 with Purchased=false -- never 404 -- because
// the client needs a definitive answer, not an error (see resource.go).
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
