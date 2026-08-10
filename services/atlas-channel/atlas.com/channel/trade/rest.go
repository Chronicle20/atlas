package trade

import (
	"github.com/google/uuid"
)

// RestModel is atlas-channel's projection of atlas-trades' room resource
// (services/atlas-trades/atlas.com/trades/trade/rest.go, resource type
// "rooms"). It is deliberately PARTIAL: atlas-channel reads this endpoint only
// to answer "is this character already seated in a trade room?" for the
// cross-family occupancy check (task-205 design §2.1), so the participants
// array and its staged items are not mirrored. JSON:API ignores the members
// that are not declared here.
type RestModel struct {
	Id       string `json:"-"`
	RoomType byte   `json:"roomType"`
	State    string `json:"state"`
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
	return Model{
		id:       id,
		roomType: rm.RoomType,
		state:    rm.State,
	}, nil
}
