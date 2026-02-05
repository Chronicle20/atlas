package invite

type Id uint32

type Type string

const (
	TypeBuddy        Type = "BUDDY"
	TypeFamily       Type = "FAMILY"
	TypeFamilySummon Type = "FAMILY_SUMMON"
	TypeMessenger    Type = "MESSENGER"
	TypeTrade        Type = "TRADE"
	TypeParty        Type = "PARTY"
	TypeGuild        Type = "GUILD"
	TypeAlliance     Type = "ALLIANCE"
)

type CommandType string

const (
	CommandTypeCreate CommandType = "CREATE"
	CommandTypeAccept CommandType = "ACCEPT"
	CommandTypeReject CommandType = "REJECT"
)

type StatusType string

const (
	StatusTypeCreated  StatusType = "CREATED"
	StatusTypeAccepted StatusType = "ACCEPTED"
	StatusTypeRejected StatusType = "REJECTED"
)
