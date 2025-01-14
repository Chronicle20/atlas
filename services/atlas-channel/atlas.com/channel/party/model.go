package party

type Model struct {
	id       uint32
	leaderId uint32
	members  []MemberModel
}

func (m Model) LeaderId() uint32 {
	return m.leaderId
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Members() []MemberModel {
	return m.members
}

type MemberModel struct {
	id        uint32
	name      string
	level     byte
	jobId     uint16
	worldId   byte
	channelId byte
	mapId     uint32
	online    bool
}

func (m MemberModel) Id() uint32 {
	return m.id
}

func (m MemberModel) Name() string {
	return m.name
}

func (m MemberModel) JobId() uint16 {
	return m.jobId
}

func (m MemberModel) Level() byte {
	return m.level
}

func (m MemberModel) Online() bool {
	return m.online
}

func (m MemberModel) ChannelId() byte {
	return m.channelId
}

func (m MemberModel) MapId() uint32 {
	return m.mapId
}
