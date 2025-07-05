package messenger

import (
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic               = "COMMAND_TOPIC_MESSENGER"
	CommandMessengerCreate        = "CREATE"
	CommandMessengerJoin          = "JOIN"
	CommandMessengerLeave         = "LEAVE"
	CommandMessengerRequestInvite = "REQUEST_INVITE"

	EnvEventStatusTopic             = "EVENT_TOPIC_MESSENGER_STATUS"
	EventMessengerStatusTypeCreated = "CREATED"
	EventMessengerStatusTypeJoined  = "JOINED"
	EventMessengerStatusTypeLeft    = "LEFT"
	EventMessengerStatusTypeError   = "ERROR"

	EventMessengerStatusErrorUnexpected                 = "ERROR_UNEXPECTED"
	EventMessengerStatusErrorTypeAlreadyJoined1         = "ALREADY_HAVE_JOINED_A_MESSENGER_1"
	EventMessengerStatusErrorTypeBeginnerCannotCreate   = "A_BEGINNER_CANT_CREATE_A_MESSENGER"
	EventMessengerStatusErrorTypeDoNotYetHaveMessenger  = "YOU_HAVE_YET_TO_JOIN_A_MESSENGER"
	EventMessengerStatusErrorTypeAlreadyJoined2         = "ALREADY_HAVE_JOINED_A_MESSENGER_2"
	EventMessengerStatusErrorTypeAtCapacity             = "THE_MESSENGER_YOURE_TRYING_TO_JOIN_IS_ALREADY_IN_FULL_CAPACITY"
	EventMessengerStatusErrorTypeUnableToFindInChannel  = "UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL"
	EventMessengerStatusErrorTypeBlockingInvites        = "IS_CURRENTLY_BLOCKING_ANY_MESSENGER_INVITATIONS"
	EventMessengerStatusErrorTypeAnotherInvite          = "IS_TAKING_CARE_OF_ANOTHER_INVITATION"
	EventMessengerStatusErrorTypeInviteDenied           = "HAVE_DENIED_REQUEST_TO_THE_MESSENGER"
	EventMessengerStatusErrorTypeCannotKickInMap        = "CANNOT_KICK_ANOTHER_USER_IN_THIS_MAP"
	EventMessengerStatusErrorTypeNewLeaderNotInVicinity = "THIS_CAN_ONLY_BE_GIVEN_TO_A_MESSENGER_MEMBER_WITHIN_THE_VICINITY"
	EventMessengerStatusErrorTypeUnableToInVicinity     = "UNABLE_TO_HAND_OVER_THE_LEADERSHIP_POST_NO_MESSENGER_MEMBER_IS_CURRENTLY_WITHIN_THE"
	EventMessengerStatusErrorTypeNotInChannel           = "YOU_MAY_ONLY_CHANGE_WITH_THE_MESSENGER_MEMBER_THATS_ON_THE_SAME_CHANNEL"
	EventMessengerStatusErrorTypeGmCannotCreate         = "AS_A_GM_YOURE_FORBIDDEN_FROM_CREATING_A_MESSENGER"
	EventMessengerStatusErrorTypeCannotFindCharacter    = "UNABLE_TO_FIND_THE_CHARACTER"
)

type CommandEvent[E any] struct {
	TransactionID uuid.UUID `json:"transactionId"`
	ActorId       uint32    `json:"actorId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreateCommandBody struct {
}

type JoinCommandBody struct {
	MessengerId uint32 `json:"messengerId"`
}

type LeaveCommandBody struct {
	MessengerId uint32 `json:"messengerId"`
}

type RequestInviteBody struct {
	CharacterId uint32 `json:"characterId"`
}

type StatusEvent[E any] struct {
	TransactionID uuid.UUID `json:"transactionId"`
	ActorId       uint32    `json:"actorId"`
	WorldId       world.Id  `json:"worldId"`
	MessengerId   uint32    `json:"messengerId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedEventBody struct {
}

type JoinedEventBody struct {
	Slot byte `json:"slot"`
}

type LeftEventBody struct {
	Slot byte `json:"slot"`
}

type ErrorEventBody struct {
	Type          string `json:"type"`
	CharacterName string `json:"characterName"`
}
