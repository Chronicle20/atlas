package party

const (
	EnvCommandTopic          = "COMMAND_TOPIC_PARTY"
	CommandPartyCreate       = "CREATE"
	CommandPartyJoin         = "JOIN"
	CommandPartyLeave        = "LEAVE"
	CommandPartyChangeLeader = "CHANGE_LEADER"

	EnvEventStatusTopic              = "EVENT_TOPIC_PARTY_STATUS"
	EventPartyStatusTypeCreated      = "CREATED"
	EventPartyStatusTypeJoined       = "JOINED"
	EventPartyStatusTypeLeft         = "LEFT"
	EventPartyStatusTypeExpel        = "EXPEL"
	EventPartyStatusTypeDisband      = "DISBAND"
	EventPartyStatusTypeChangeLeader = "CHANGE_LEADER"
	EventPartyStatusTypeError        = "ERROR"

	EventPartyStatusErrorUnexpected                 = "ERROR_UNEXPECTED"
	EventPartyStatusErrorTypeAlreadyJoined1         = "ALREADY_HAVE_JOINED_A_PARTY_1"
	EventPartyStatusErrorTypeBeginnerCannotCreate   = "A_BEGINNER_CANT_CREATE_A_PARTY"
	EventPartyStatusErrorTypeDoNotYetHaveParty      = "YOU_HAVE_YET_TO_JOIN_A_PARTY"
	EventPartyStatusErrorTypeAlreadyJoined2         = "ALREADY_HAVE_JOINED_A_PARTY_2"
	EventPartyStatusErrorTypeAtCapacity             = "THE_PARTY_YOURE_TRYING_TO_JOIN_IS_ALREADY_IN_FULL_CAPACITY"
	EventPartyStatusErrorTypeUnableToFindInChannel  = "UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL"
	EventPartyStatusErrorTypeBlockingInvites        = "IS_CURRENTLY_BLOCKING_ANY_PARTY_INVITATIONS"
	EventPartyStatusErrorTypeAnotherInvite          = "IS_TAKING_CARE_OF_ANOTHER_INVITATION"
	EventPartyStatusErrorTypeInviteDenied           = "HAVE_DENIED_REQUEST_TO_THE_PARTY"
	EventPartyStatusErrorTypeCannotKickInMap        = "CANNOT_KICK_ANOTHER_USER_IN_THIS_MAP"
	EventPartyStatusErrorTypeNewLeaderNotInVicinity = "THIS_CAN_ONLY_BE_GIVEN_TO_A_PARTY_MEMBER_WITHIN_THE_VICINITY"
	EventPartyStatusErrorTypeUnableToInVicinity     = "UNABLE_TO_HAND_OVER_THE_LEADERSHIP_POST_NO_PARTY_MEMBER_IS_CURRENTLY_WITHIN_THE"
	EventPartyStatusErrorTypeNotInChannel           = "YOU_MAY_ONLY_CHANGE_WITH_THE_PARTY_MEMBER_THATS_ON_THE_SAME_CHANNEL"
	EventPartyStatusErrorTypeGmCannotCreate         = "AS_A_GM_YOURE_FORBIDDEN_FROM_CREATING_A_PARTY"
	EventPartyStatusErrorTypeCannotFindCharacter    = "UNABLE_TO_FIND_THE_CHARACTER"
)

type commandEvent[E any] struct {
	ActorId uint32 `json:"actorId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type createCommandBody struct {
}

type joinCommandBody struct {
	PartyId uint32 `json:"partyId"`
}

type leaveCommandBody struct {
	PartyId uint32 `json:"partyId"`
	Force   bool   `json:"force"`
}

type changeLeaderBody struct {
	PartyId  uint32 `json:"partyId"`
	LeaderId uint32 `json:"leaderId"`
}

type statusEvent[E any] struct {
	ActorId uint32 `json:"actorId"`
	WorldId byte   `json:"worldId"`
	PartyId uint32 `json:"partyId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type createdEventBody struct {
}

type joinedEventBody struct {
}

type leftEventBody struct {
}

type expelEventBody struct {
	CharacterId uint32 `json:"characterId"`
}

type disbandEventBody struct {
	Members []uint32 `json:"members"`
}

type changeLeaderEventBody struct {
	CharacterId  uint32 `json:"characterId"`
	Disconnected bool   `json:"disconnected"`
}

type errorEventBody struct {
	Type          string `json:"type"`
	CharacterName string `json:"characterName"`
}
