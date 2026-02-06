package writer

import (
	"atlas-channel/character"
	"atlas-channel/party"
	"strconv"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	PartyOperation             = "PartyOperation"
	PartyOperationCreated      = "CREATED"
	PartyOperationDisband      = "DISBAND"
	PartyOperationExpel        = "EXPEL"
	PartyOperationLeave        = "LEAVE"
	PartyOperationJoin         = "JOIN"
	PartyOperationUpdate       = "UPDATE"
	PartyOperationChangeLeader = "CHANGE_LEADER"
	PartyOperationInvite       = "INVITE"
)

func PartyCreatedBody(l logrus.FieldLogger) func(partyId uint32) BodyProducer {
	return func(partyId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getPartyOperation(l)(options, PartyOperationCreated))
			w.WriteInt(partyId)

			// TODO write doors for party.
			w.WriteInt(uint32(_map.EmptyMapId))
			w.WriteInt(uint32(_map.EmptyMapId))
			w.WriteShort(0)
			w.WriteShort(0)
			return w.Bytes()
		}
	}
}

func PartyLeftBody(l logrus.FieldLogger) func(p party.Model, t character.Model, forChannel channel.Id) BodyProducer {
	return func(p party.Model, t character.Model, forChannel channel.Id) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getPartyOperation(l)(options, PartyOperationLeave))
			w.WriteInt(p.Id())
			w.WriteInt(t.Id())
			w.WriteByte(1)
			w.WriteBool(false) // forced
			w.WriteAsciiString(t.Name())
			return WriteParty(w, p, forChannel)
		}
	}
}

func PartyExpelBody(l logrus.FieldLogger) func(p party.Model, t character.Model, forChannel channel.Id) BodyProducer {
	return func(p party.Model, t character.Model, forChannel channel.Id) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getPartyOperation(l)(options, PartyOperationExpel))
			w.WriteInt(p.Id())
			w.WriteInt(t.Id())
			w.WriteByte(1)
			w.WriteBool(true) // forced
			w.WriteAsciiString(t.Name())
			return WriteParty(w, p, forChannel)
		}
	}
}

func PartyDisbandBody(l logrus.FieldLogger) func(partyId uint32, t character.Model, forChannel channel.Id) BodyProducer {
	return func(partyId uint32, t character.Model, forChannel channel.Id) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getPartyOperation(l)(options, PartyOperationDisband))
			w.WriteInt(partyId)
			w.WriteInt(t.Id())
			w.WriteByte(0)
			w.WriteInt(partyId)
			return w.Bytes()
		}
	}
}

func PartyJoinBody(l logrus.FieldLogger) func(p party.Model, t character.Model, forChannel channel.Id) BodyProducer {
	return func(p party.Model, t character.Model, forChannel channel.Id) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getPartyOperation(l)(options, PartyOperationJoin))
			w.WriteInt(p.Id())
			w.WriteAsciiString(t.Name())
			return WriteParty(w, p, forChannel)
		}
	}
}

func PartyUpdateBody(l logrus.FieldLogger) func(p party.Model, t character.Model, forChannel channel.Id) BodyProducer {
	return func(p party.Model, t character.Model, forChannel channel.Id) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getPartyOperation(l)(options, PartyOperationUpdate))
			w.WriteInt(p.Id())
			return WriteParty(w, p, forChannel)
		}
	}
}

func PartyChangeLeaderBody(l logrus.FieldLogger) func(targetCharacterId uint32, disconnected bool) BodyProducer {
	return func(targetCharacterId uint32, disconnected bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getPartyOperation(l)(options, PartyOperationChangeLeader))
			w.WriteInt(targetCharacterId)
			w.WriteBool(disconnected)
			return w.Bytes()
		}
	}
}

func PartyErrorBody(l logrus.FieldLogger) func(code string, name string) BodyProducer {
	return func(code string, name string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getPartyOperation(l)(options, code))
			w.WriteAsciiString(name)
			return w.Bytes()
		}
	}
}

func PartyInviteBody(l logrus.FieldLogger) func(partyId uint32, originatorName string) BodyProducer {
	return func(partyId uint32, originatorName string) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getPartyOperation(l)(options, PartyOperationInvite))
			w.WriteInt(partyId)
			w.WriteAsciiString(originatorName)
			w.WriteByte(0) // TODO this triggers op 3 for party operation if 1, feels like auto reject, blocked packet
			return w.Bytes()
		}
	}
}

func WriteParty(w *response.Writer, p party.Model, forChannel channel.Id) []byte {
	for _, m := range p.Members() {
		w.WriteInt(m.Id())
	}
	for range 6 - len(p.Members()) {
		w.WriteInt(0)
	}

	for _, m := range p.Members() {
		WritePaddedString(w, m.Name(), 13)
	}
	for range 6 - len(p.Members()) {
		WritePaddedString(w, "", 13)
	}

	for _, m := range p.Members() {
		w.WriteInt(uint32(m.JobId()))
	}
	for range 6 - len(p.Members()) {
		w.WriteInt(0)
	}

	for _, m := range p.Members() {
		w.WriteInt(uint32(m.Level()))
	}
	for range 6 - len(p.Members()) {
		w.WriteInt(0)
	}

	for _, m := range p.Members() {
		if m.Online() {
			w.WriteInt(uint32(m.ChannelId()))
		} else {
			w.WriteInt32(-2)
		}
	}
	for range 6 - len(p.Members()) {
		w.WriteInt(0)
	}

	w.WriteInt(p.LeaderId())

	for _, m := range p.Members() {
		if forChannel == m.ChannelId() {
			w.WriteInt(uint32(m.MapId()))
		} else {
			w.WriteInt(0)
		}
	}
	for range 6 - len(p.Members()) {
		w.WriteInt(0)
	}

	for range 6 {
		// TODO write doors for party.
		w.WriteInt(uint32(_map.EmptyMapId))
		w.WriteInt(uint32(_map.EmptyMapId))
		w.WriteInt(0)
		w.WriteInt(0)
	}
	return w.Bytes()
}

// TODO test with JMS before moving to library
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
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var code interface{}
		if code, ok = codes[key]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		op, err := strconv.ParseUint(code.(string), 0, 16)
		if err != nil {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}
		return byte(op)
	}
}
