package clientbound

import (
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/party"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

const (
	PartyOperationCreated      = "CREATED"
	PartyOperationDisband      = "DISBAND"
	PartyOperationExpel        = "EXPEL"
	PartyOperationLeave        = "LEAVE"
	PartyOperationJoin         = "JOIN"
	PartyOperationUpdate       = "UPDATE"
	PartyOperationChangeLeader = "CHANGE_LEADER"
	PartyOperationInvite       = "INVITE"
	PartyOperationTownPortal   = "TOWN_PORTAL"

	// OnPartyResult error/notice arms (Task-2 discrete-struct split). Each key
	// matches docs/packets/dispatchers/party.yaml + the v83/v84 seed templates
	// verbatim — a typo resolves to the 99 fallback and crashes the client.
	PartyOperationAlreadyJoined1         = "ALREADY_HAVE_JOINED_A_PARTY_1"
	PartyOperationBeginnerCannotCreate   = "A_BEGINNER_CANT_CREATE_A_PARTY"
	PartyOperationNotInParty             = "YOU_HAVE_YET_TO_JOIN_A_PARTY"
	PartyOperationAlreadyJoined2         = "ALREADY_HAVE_JOINED_A_PARTY_2"
	PartyOperationPartyFull              = "THE_PARTY_YOURE_TRYING_TO_JOIN_IS_ALREADY_IN_FULL_CAPACITY"
	PartyOperationUnableToFindInChannel  = "UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL"
	PartyOperationBlockingInvitations    = "IS_CURRENTLY_BLOCKING_ANY_PARTY_INVITATIONS"
	PartyOperationTakingCareOfInvitation = "IS_TAKING_CARE_OF_ANOTHER_INVITATION"
	PartyOperationRequestDenied          = "HAVE_DENIED_REQUEST_TO_THE_PARTY"
	PartyOperationCannotKick             = "CANNOT_KICK_ANOTHER_USER_IN_THIS_MAP"
	PartyOperationOnlyWithinVicinity     = "THIS_CAN_ONLY_BE_GIVEN_TO_A_PARTY_MEMBER_WITHIN_THE_VICINITY"
	PartyOperationUnableToHandOver       = "UNABLE_TO_HAND_OVER_THE_LEADERSHIP_POST_NO_PARTY_MEMBER_IS_CURRENTLY_WITHIN_THE"
	PartyOperationOnlySameChannel        = "YOU_MAY_ONLY_CHANGE_WITH_THE_PARTY_MEMBER_THATS_ON_THE_SAME_CHANNEL"
	PartyOperationGmCannotCreate         = "AS_A_GM_YOURE_FORBIDDEN_FROM_CREATING_A_PARTY"
	PartyOperationUnableToFindCharacter  = "UNABLE_TO_FIND_THE_CHARACTER"
)

func PartyCreatedBody(partyId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationCreated, func(mode byte) packet.Encoder {
		return NewCreated(mode, partyId)
	})
}

// PartyCreatedBodyWithDoor is like PartyCreatedBody but populates the party
// minimap door indicator fields (FR-3.3).  townMapId is the town (portal
// exit) and targetMapId is the dungeon/area map that the door came from.  x and y are
// the area-side door minimap coordinates (door.AreaX/AreaY).
func PartyCreatedBodyWithDoor(partyId uint32, townMapId _map.Id, targetMapId _map.Id, x int16, y int16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationCreated, func(mode byte) packet.Encoder {
		return NewCreated(mode, partyId).WithDoor(townMapId, targetMapId, x, y)
	})
}

func PartyLeftBody(partyId uint32, targetId uint32, targetName string, members []party.PartyMember, leaderId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationLeave, func(mode byte) packet.Encoder {
		return NewLeft(mode, partyId, targetId, targetName, false, members, leaderId)
	})
}

func PartyExpelBody(partyId uint32, targetId uint32, targetName string, members []party.PartyMember, leaderId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationExpel, func(mode byte) packet.Encoder {
		return NewLeft(mode, partyId, targetId, targetName, true, members, leaderId)
	})
}

func PartyDisbandBody(partyId uint32, targetId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationDisband, func(mode byte) packet.Encoder {
		return NewDisband(mode, partyId, targetId)
	})
}

func PartyJoinBody(partyId uint32, targetName string, members []party.PartyMember, leaderId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationJoin, func(mode byte) packet.Encoder {
		return NewJoin(mode, partyId, targetName, members, leaderId)
	})
}

func PartyUpdateBody(partyId uint32, members []party.PartyMember, leaderId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationUpdate, func(mode byte) packet.Encoder {
		return NewUpdate(mode, partyId, members, leaderId)
	})
}

func PartyChangeLeaderBody(targetCharacterId uint32, disconnected bool) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationChangeLeader, func(mode byte) packet.Encoder {
		return NewChangeLeader(mode, targetCharacterId, disconnected)
	})
}

func PartyInviteBody(partyId uint32, originatorName string, originatorJobId uint32, originatorLevel uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationInvite, func(mode byte) packet.Encoder {
		return NewInvite(mode, partyId, originatorName, originatorJobId, originatorLevel)
	})
}

// PartyTownPortalBody sets party-member slot's Mystic Door town portal in the
// client party town-portal array (the in-party town-door render source).
// townMapId/targetMapId are the door's town (exit) and area (origin) maps;
// x/y are the AREA-side door position. Mode is version-resolved (TOWN_PORTAL).
func PartyTownPortalBody(slot byte, townMapId _map.Id, targetMapId _map.Id, x int16, y int16) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationTownPortal, func(mode byte) packet.Encoder {
		return NewTownPortal(mode, slot, townMapId, targetMapId, x, y)
	})
}

// PartyTownPortalClearBody clears party-member slot's town portal (door removed).
func PartyTownPortalClearBody(slot byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationTownPortal, func(mode byte) packet.Encoder {
		return NewTownPortalClear(mode, slot)
	})
}

// --- OnPartyResult error/notice arm body funcs (Task-2 discrete-struct split) --
//
// One fixed-key body func per enumerated error/notice arm. Twelve are mode-only;
// three (Blocking/TakingCare/RequestDenied) carry a trailing target name. The two
// D8 arms — UnableToFindCharacter and UnableToFindInChannel — are PARAMETERLESS
// (mode-only per IDA; the legacy name arg is dropped). The name param on the three
// invite-target arms flows into the struct ctor, never into the resolved key.

func PartyAlreadyJoined1Body() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationAlreadyJoined1, func(mode byte) packet.Encoder {
		return NewAlreadyJoined1(mode)
	})
}

func PartyBeginnerCannotCreateBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationBeginnerCannotCreate, func(mode byte) packet.Encoder {
		return NewBeginnerCannotCreate(mode)
	})
}

func PartyNotInPartyBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationNotInParty, func(mode byte) packet.Encoder {
		return NewNotInParty(mode)
	})
}

func PartyAlreadyJoined2Body() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationAlreadyJoined2, func(mode byte) packet.Encoder {
		return NewAlreadyJoined2(mode)
	})
}

func PartyFullBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationPartyFull, func(mode byte) packet.Encoder {
		return NewPartyFull(mode)
	})
}

func PartyUnableToFindInChannelBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationUnableToFindInChannel, func(mode byte) packet.Encoder {
		return NewUnableToFindInChannel(mode)
	})
}

func PartyCannotKickBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationCannotKick, func(mode byte) packet.Encoder {
		return NewCannotKick(mode)
	})
}

func PartyOnlyWithinVicinityBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationOnlyWithinVicinity, func(mode byte) packet.Encoder {
		return NewOnlyWithinVicinity(mode)
	})
}

func PartyUnableToHandOverBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationUnableToHandOver, func(mode byte) packet.Encoder {
		return NewUnableToHandOver(mode)
	})
}

func PartyOnlySameChannelBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationOnlySameChannel, func(mode byte) packet.Encoder {
		return NewOnlySameChannel(mode)
	})
}

func PartyGmCannotCreateBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationGmCannotCreate, func(mode byte) packet.Encoder {
		return NewGmCannotCreate(mode)
	})
}

func PartyUnableToFindCharacterBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationUnableToFindCharacter, func(mode byte) packet.Encoder {
		return NewUnableToFindCharacter(mode)
	})
}

func PartyBlockingInvitationsBody(name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationBlockingInvitations, func(mode byte) packet.Encoder {
		return NewBlockingInvitations(mode, name)
	})
}

func PartyTakingCareOfInvitationBody(name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationTakingCareOfInvitation, func(mode byte) packet.Encoder {
		return NewTakingCareOfInvitation(mode, name)
	})
}

func PartyRequestDeniedBody(name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationRequestDenied, func(mode byte) packet.Encoder {
		return NewRequestDenied(mode, name)
	})
}
