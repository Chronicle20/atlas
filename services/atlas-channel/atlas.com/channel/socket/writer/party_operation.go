package writer

import (
	"atlas-channel/character"
	"atlas-channel/party"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
	"strconv"
)

const (
	PartyOperation        = "PartyOperation"
	PartyOperationCreated = "CREATED"
	PartyOperationDisband = "DISBAND"
	PartyOperationExpel   = "EXPEL"
	PartyOperationLeave   = "LEAVE"
	PartyOperationJoin    = "JOIN"
	PartyOperationUpdate  = "UPDATE"
)

func PartyCreatedBody(l logrus.FieldLogger) func(partyId uint32) BodyProducer {
	return func(partyId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getOperation(l)(options, PartyOperationCreated))
			w.WriteInt(partyId)

			// TODO write doors for party.
			w.WriteInt(999999999)
			w.WriteInt(999999999)
			w.WriteInt(0)
			w.WriteInt(0)
			return w.Bytes()
		}
	}
}

func PartyLeftBody(l logrus.FieldLogger) func(p party.Model, t character.Model, forChannel byte) BodyProducer {
	return func(p party.Model, t character.Model, forChannel byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getOperation(l)(options, PartyOperationLeave))
			w.WriteInt(p.Id())
			w.WriteInt(t.Id())
			w.WriteByte(1)
			w.WriteBool(false) // forced
			w.WriteAsciiString(t.Name())
			return WriteParty(w, p, forChannel)
		}
	}
}

func PartyDisbandBody(l logrus.FieldLogger) func(partyId uint32, t character.Model, forChannel byte) BodyProducer {
	return func(partyId uint32, t character.Model, forChannel byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getOperation(l)(options, PartyOperationDisband))
			w.WriteInt(partyId)
			w.WriteInt(t.Id())
			w.WriteByte(0)
			w.WriteInt(partyId)
			return w.Bytes()
		}
	}
}

func PartyJoinBody(l logrus.FieldLogger) func(p party.Model, t character.Model, forChannel byte) BodyProducer {
	return func(p party.Model, t character.Model, forChannel byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getOperation(l)(options, PartyOperationJoin))
			w.WriteInt(p.Id())
			w.WriteAsciiString(t.Name())
			return WriteParty(w, p, forChannel)
		}
	}
}

func PartyUpdateBody(l logrus.FieldLogger) func(p party.Model, t character.Model, forChannel byte) BodyProducer {
	return func(p party.Model, t character.Model, forChannel byte) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getOperation(l)(options, PartyOperationUpdate))
			w.WriteInt(p.Id())
			return WriteParty(w, p, forChannel)
		}
	}
}

func WriteParty(w *response.Writer, p party.Model, forChannel byte) []byte {
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
			w.WriteInt(m.MapId())
		} else {
			w.WriteInt(0)
		}
	}
	for range 6 - len(p.Members()) {
		w.WriteInt(0)
	}

	for range 6 {
		// TODO write doors for party.
		w.WriteInt(999999999)
		w.WriteInt(999999999)
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

func getOperation(l logrus.FieldLogger) func(options map[string]interface{}, key string) byte {
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

		op, err := strconv.ParseUint(codes[key].(string), 0, 16)
		if err != nil {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}
		return byte(op)
	}
}
