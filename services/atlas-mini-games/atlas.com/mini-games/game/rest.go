package game

import "strconv"

// RestModel is the wire shape of one mini-game room, consumed verbatim by
// atlas-channel's rooms-in-field REST client (task-19). Id is the room id
// (== OwnerId, design D2) rendered as a string per JSON:API convention.
type RestModel struct {
	Id       uint32 `json:"-"`
	OwnerId  uint32 `json:"ownerId"`
	RoomType byte   `json:"roomType"`
	Title    string `json:"title"`
	Private  bool   `json:"private"`
	// HasPassword is the balloon lock-icon predicate (Private && a non-empty
	// password). It is computed here so the channel's map-entry balloon render
	// matches the live BALLOON_UPDATED event path, which computes the same
	// thing (game/producer.go balloonProvider). Raw Private alone would show a
	// lock for a private-but-passwordless room the VISIT gate treats as unlocked.
	HasPassword bool `json:"hasPassword"`
	PieceType   byte `json:"pieceType"`
	Occupancy   byte `json:"occupancy"`
	InProgress  bool `json:"inProgress"`
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
		Id:          r.Id(),
		OwnerId:     r.OwnerId(),
		RoomType:    r.RoomType(),
		Title:       r.Title(),
		Private:     r.Private(),
		HasPassword: r.Private() && r.Password() != "",
		PieceType:   r.PieceType(),
		Occupancy:   occupancy,
		InProgress:  r.InProgress(),
	}, nil
}
