package cash

import (
	"strconv"
)

// RestModel is the subset of atlas-data's cash resource this service needs:
// the item-expiration-extender grant and ceiling, used to re-derive the cap
// server-side rather than trusting the channel-computed expiration.
type RestModel struct {
	Id      uint32 `json:"-"`
	AddTime uint32 `json:"addTime,omitempty"`
	MaxDays uint32 `json:"maxDays,omitempty"`
}

func (r RestModel) GetName() string {
	return "cash_items"
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

// SetToOneReferenceID / SetToManyReferenceIDs are required by api2go's
// unmarshal whenever the upstream response carries a relationships block.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{id: rm.Id, addTime: rm.AddTime, maxDays: rm.MaxDays}, nil
}
