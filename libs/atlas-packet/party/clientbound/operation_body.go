package clientbound

import (
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	"github.com/Chronicle20/atlas/libs/atlas-packet/party"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
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
)

func PartyCreatedBody(partyId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationCreated, func(mode byte) packet.Encoder {
		return NewCreated(mode, partyId)
	})
}

// PartyCreatedBodyWithDoor is like PartyCreatedBody but populates the party
// minimap door indicator fields (FR-3.3).  townMapId and targetMapId follow
// the Cosmic partyCreated convention: townMapId is the town (portal exit) and
// targetMapId is the dungeon/area map that the door came from.  x and y are
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
