package game

import "strconv"

// RestModel is the wire shape of one mini-game room, consumed verbatim by
// atlas-channel's rooms-in-field REST client (task-19). Id is the room id
// (== OwnerId, design D2) rendered as a string per JSON:API convention.
type RestModel struct {
	Id         uint32 `json:"-"`
	OwnerId    uint32 `json:"ownerId"`
	RoomType   byte   `json:"roomType"`
	Title      string `json:"title"`
	Private    bool   `json:"private"`
	PieceType  byte   `json:"pieceType"`
	Occupancy  byte   `json:"occupancy"`
	InProgress bool   `json:"inProgress"`
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

// Transform converts a Room snapshot to its REST shape. Occupancy is 2 when
// a visitor is seated, 1 otherwise (owner-only).
func Transform(r Room) (RestModel, error) {
	occupancy := byte(1)
	if r.VisitorId() != 0 {
		occupancy = 2
	}
	return RestModel{
		Id:         r.Id(),
		OwnerId:    r.OwnerId(),
		RoomType:   r.RoomType(),
		Title:      r.Title(),
		Private:    r.Private(),
		PieceType:  r.PieceType(),
		Occupancy:  occupancy,
		InProgress: r.InProgress(),
	}, nil
}
