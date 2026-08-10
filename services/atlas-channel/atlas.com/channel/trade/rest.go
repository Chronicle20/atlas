package trade

import (
	"github.com/google/uuid"
)

// RestModel is atlas-channel's projection of atlas-trades' room resource
// (services/atlas-trades/atlas.com/trades/trade/rest.go, resource type
// "rooms"). It is deliberately MINIMAL: atlas-channel reads this endpoint only
// to answer "is this character already seated in a trade room?" for the
// cross-family occupancy check (task-205 design §2.1), which needs nothing but
// the room's identity. JSON:API ignores the members that are not declared here.
type RestModel struct {
	Id string `json:"-"`
}

func (r RestModel) GetName() string {
	return "rooms"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(strId string) error {
	r.Id = strId
	return nil
}

func Extract(rm RestModel) (Model, error) {
	id, err := uuid.Parse(rm.Id)
	if err != nil {
		return Model{}, err
	}
	return Model{id: id}, nil
}
