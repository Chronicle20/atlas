package handler

import (
	"atlas-channel/character"
	"atlas-channel/party"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const (
	PartyOperationHandle       = "PartyOperationHandle"
	PartyOperationCreate       = "CREATE"
	PartyOperationLeave        = "LEAVE"
	PartyOperationExpel        = "EXPEL"
	PartyOperationChangeLeader = "CHANGE_LEADER"
	PartyOperationInvite       = "INVITE"
)

func PartyOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	partyOperationFunc := session.Announce(l)(ctx)(wp)(writer.PartyOperation)

	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		op := r.ReadByte()
		if isOperation(l)(readerOptions, op, PartyOperationCreate) {
			err := party.Create(l)(ctx)(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Character [%d] unable to attempt party creation.", s.CharacterId())
			}
			return
		}
		if isOperation(l)(readerOptions, op, PartyOperationLeave) {
			p, err := party.GetByMemberId(l)(ctx)(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to locate party for character [%d] to leave.", s.CharacterId())
				return
			}
			err = party.Leave(l)(ctx)(p.Id(), s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Character [%d] unable to attempt leaving party.", s.CharacterId())
			}
			return
		}
		if isOperation(l)(readerOptions, op, PartyOperationExpel) {
			targetCharacterId := r.ReadUint32()
			p, err := party.GetByMemberId(l)(ctx)(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to locate party for character [%d] to leave.", s.CharacterId())
				return
			}
			err = party.Expel(l)(ctx)(p.Id(), s.CharacterId(), targetCharacterId)
			if err != nil {
				l.WithError(err).Errorf("Character [%d] unable to attempt expelling [%d] from party.", s.CharacterId(), targetCharacterId)
			}
			return
		}
		if isOperation(l)(readerOptions, op, PartyOperationChangeLeader) {
			targetCharacterId := r.ReadUint32()
			p, err := party.GetByMemberId(l)(ctx)(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to locate party for character [%d] to leave.", s.CharacterId())
				return
			}
			err = party.ChangeLeader(l)(ctx)(p.Id(), s.CharacterId(), targetCharacterId)
			if err != nil {
				l.WithError(err).Errorf("Character [%d] unable to pass leadership to [%d] in party.", s.CharacterId(), targetCharacterId)
			}
			return
		}
		if isOperation(l)(readerOptions, op, PartyOperationInvite) {
			name := r.ReadAsciiString()
			cs, err := character.GetByName(l, ctx)(name)
			if err != nil || len(cs) < 1 {
				l.WithError(err).Errorf("Unable to locate character by name [%s] to invite to party.", name)
				err := partyOperationFunc(s, writer.PartyErrorBody(l)("UNABLE_TO_FIND_THE_CHARACTER", name))
				if err != nil {
					return
				}
			}
			err = party.RequestInvite(l)(ctx)(s.CharacterId(), cs[0].Id())
			if err != nil {
				l.WithError(err).Errorf("Character [%d] was unable to request [%d] to join party.", s.CharacterId(), cs[0].Id())
			}
			return
		}
		l.Warnf("Character [%d] issued a unhandled party operation [%d].", s.CharacterId(), op)
	}
}

func isOperation(l logrus.FieldLogger) func(options map[string]interface{}, op byte, key string) bool {
	return func(options map[string]interface{}, op byte, key string) bool {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		res, ok := codes[key].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}
		return byte(res) == op
	}
}
