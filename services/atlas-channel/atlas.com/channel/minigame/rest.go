package minigame

import "strconv"

// RestModel mirrors atlas-mini-games' game.RestModel wire shape verbatim
// (services/atlas-mini-games/atlas.com/mini-games/game/rest.go). Id is the
// room id, rendered as a string per JSON:API convention.
type RestModel struct {
	Id          uint32 `json:"-"`
	OwnerId     uint32 `json:"ownerId"`
	RoomType    byte   `json:"roomType"`
	Title       string `json:"title"`
	Private     bool   `json:"private"`
	HasPassword bool   `json:"hasPassword"`
	PieceType   byte   `json:"pieceType"`
	Occupancy   byte   `json:"occupancy"`
	InProgress  bool   `json:"inProgress"`
}

func (r RestModel) GetName() string {
	return "games"
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

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:          rm.Id,
		ownerId:     rm.OwnerId,
		roomType:    rm.RoomType,
		title:       rm.Title,
		private:     rm.Private,
		hasPassword: rm.HasPassword,
		pieceType:   rm.PieceType,
		occupancy:   rm.Occupancy,
		inProgress:  rm.InProgress,
	}, nil
}
