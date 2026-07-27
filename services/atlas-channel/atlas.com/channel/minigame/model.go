package minigame

// Model is atlas-channel's in-field view of one mini-game room, sourced from
// atlas-mini-games' rooms-in-field REST endpoint (task-16/task-19). Id is the
// room id (== OwnerId, design D2).
type Model struct {
	id          uint32
	ownerId     uint32
	roomType    byte
	title       string
	private     bool
	hasPassword bool
	pieceType   byte
	occupancy   byte
	inProgress  bool
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) OwnerId() uint32 {
	return m.ownerId
}

func (m Model) RoomType() byte {
	return m.roomType
}

func (m Model) Title() string {
	return m.title
}

func (m Model) Private() bool {
	return m.private
}

// HasPassword reports whether the room is private AND has a non-empty password
// (the balloon lock-icon predicate). Computed service-side and mirrored here so
// the map-entry balloon render matches the live BALLOON_UPDATED event path.
func (m Model) HasPassword() bool {
	return m.hasPassword
}

func (m Model) PieceType() byte {
	return m.pieceType
}

func (m Model) Occupancy() byte {
	return m.occupancy
}

func (m Model) InProgress() bool {
	return m.inProgress
}
