package writer

import (
	"atlas-channel/character"
	"atlas-channel/party"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	atlas_packet "github.com/Chronicle20/atlas-packet"
	partypkt "github.com/Chronicle20/atlas-packet/party"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
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

func toPartyMembers(p party.Model, forChannel channel.Id) []partypkt.PartyMember {
	members := make([]partypkt.PartyMember, 0, len(p.Members()))
	for _, m := range p.Members() {
		chId := int32(m.ChannelId())
		if !m.Online() {
			chId = -2
		}
		mapId := uint32(0)
		if forChannel == m.ChannelId() {
			mapId = uint32(m.MapId())
		}
		members = append(members, partypkt.PartyMember{
			Id:        m.Id(),
			Name:      m.Name(),
			JobId:     uint16(m.JobId()),
			Level:     uint16(m.Level()), // byte -> uint16
			ChannelId: chId,
			MapId:     mapId,
		})
	}
	return members
}

func PartyCreatedBody(partyId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getPartyOperation(l)(options, PartyOperationCreated)
			return partypkt.NewCreated(mode, partyId).Encode(l, ctx)(options)
		}
	}
}

func PartyLeftBody(p party.Model, t character.Model, forChannel channel.Id) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getPartyOperation(l)(options, PartyOperationLeave)
			members := toPartyMembers(p, forChannel)
			return partypkt.NewLeft(mode, p.Id(), t.Id(), t.Name(), false, members, p.LeaderId()).Encode(l, ctx)(options)
		}
	}
}

func PartyExpelBody(p party.Model, t character.Model, forChannel channel.Id) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getPartyOperation(l)(options, PartyOperationExpel)
			members := toPartyMembers(p, forChannel)
			return partypkt.NewLeft(mode, p.Id(), t.Id(), t.Name(), true, members, p.LeaderId()).Encode(l, ctx)(options)
		}
	}
}

func PartyDisbandBody(partyId uint32, t character.Model, forChannel channel.Id) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getPartyOperation(l)(options, PartyOperationDisband)
			return partypkt.NewDisbandW(mode, partyId, t.Id()).Encode(l, ctx)(options)
		}
	}
}

func PartyJoinBody(p party.Model, t character.Model, forChannel channel.Id) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getPartyOperation(l)(options, PartyOperationJoin)
			members := toPartyMembers(p, forChannel)
			return partypkt.NewJoinW(mode, p.Id(), t.Name(), members, p.LeaderId()).Encode(l, ctx)(options)
		}
	}
}

func PartyUpdateBody(p party.Model, t character.Model, forChannel channel.Id) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getPartyOperation(l)(options, PartyOperationUpdate)
			members := toPartyMembers(p, forChannel)
			return partypkt.NewUpdateW(mode, p.Id(), members, p.LeaderId()).Encode(l, ctx)(options)
		}
	}
}

func PartyChangeLeaderBody(targetCharacterId uint32, disconnected bool) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getPartyOperation(l)(options, PartyOperationChangeLeader)
			return partypkt.NewChangeLeaderW(mode, targetCharacterId, disconnected).Encode(l, ctx)(options)
		}
	}
}

func PartyErrorBody(code string, name string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getPartyOperation(l)(options, code)
			return partypkt.NewErrorW(mode, name).Encode(l, ctx)(options)
		}
	}
}

func PartyInviteBody(partyId uint32, originatorName string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := getPartyOperation(l)(options, PartyOperationInvite)
			return partypkt.NewInviteW(mode, partyId, originatorName).Encode(l, ctx)(options)
		}
	}
}

// WritePaddedString writes a string padded or truncated to the given length.
func WritePaddedString(w *response.Writer, str string, number int) {
	if len(str) > number {
		w.WriteByteArray([]byte(str)[:number])
	} else {
		w.WriteByteArray([]byte(str))
		w.WriteByteArray(make([]byte, number-len(str)))
	}
}

func getPartyOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
	return func(options map[string]interface{}, key string) byte {
		return atlas_packet.ResolveCode(l, options, "operations", key)
	}
}
