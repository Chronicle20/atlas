package clientbound

import (
	"context"

	atlas_packet "github.com/Chronicle20/atlas-packet"
	"github.com/Chronicle20/atlas-packet/party"
	"github.com/Chronicle20/atlas-socket/packet"
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
)

func PartyCreatedBody(partyId uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationCreated, func(mode byte) packet.Encoder {
		return NewCreated(mode, partyId)
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

func PartyErrorBody(code string, name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", code, func(mode byte) packet.Encoder {
		return NewError(mode, name)
	})
}

func PartyInviteBody(partyId uint32, originatorName string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", PartyOperationInvite, func(mode byte) packet.Encoder {
		return NewInvite(mode, partyId, originatorName)
	})
}
