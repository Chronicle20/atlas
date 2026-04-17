package handler

import (
	"atlas-channel/character"
	"atlas-channel/invite"
	"atlas-channel/party"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	invite2 "github.com/Chronicle20/atlas/libs/atlas-constants/invite"
	partycb "github.com/Chronicle20/atlas/libs/atlas-packet/party/clientbound"
	partysb "github.com/Chronicle20/atlas/libs/atlas-packet/party/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

type PartyOperation byte

const (
	PartyOperationCreate       = "CREATE"
	PartyOperationLeave        = "LEAVE"
	PartyOperationExpel        = "EXPEL"
	PartyOperationChangeLeader = "CHANGE_LEADER"
	PartyOperationInvite       = "INVITE"
	PartyOperationJoin         = "JOIN"
)

func PartyOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := partysb.Operation{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		op := PartyOperation(p.Op())
		if isPartyOperation(l)(readerOptions, op, PartyOperationCreate) {
			err := party.NewProcessor(l, ctx).Create(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Character [%d] unable to attempt party creation.", s.CharacterId())
			}
			return
		}
		if isPartyOperation(l)(readerOptions, op, PartyOperationJoin) {
			sp := &partysb.OperationJoin{}
			sp.Decode(l, ctx)(r, readerOptions)
			err := invite.NewProcessor(l, ctx).Accept(s.CharacterId(), s.WorldId(), string(invite2.TypeParty), sp.PartyId())
			if err != nil {
				l.WithError(err).Errorf("Unable to issue invite acceptance command for character [%d].", s.CharacterId())
			}
			return
		}
		if isPartyOperation(l)(readerOptions, op, PartyOperationLeave) {
			p, err := party.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to locate party for character [%d] to leave.", s.CharacterId())
				return
			}
			err = party.NewProcessor(l, ctx).Leave(p.Id(), s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Character [%d] unable to attempt leaving party.", s.CharacterId())
			}
			return
		}
		if isPartyOperation(l)(readerOptions, op, PartyOperationExpel) {
			sp := &partysb.OperationExpel{}
			sp.Decode(l, ctx)(r, readerOptions)
			p, err := party.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to locate party for character [%d] to leave.", s.CharacterId())
				return
			}
			err = party.NewProcessor(l, ctx).Expel(p.Id(), s.CharacterId(), sp.TargetCharacterId())
			if err != nil {
				l.WithError(err).Errorf("Character [%d] unable to attempt expelling [%d] from party.", s.CharacterId(), sp.TargetCharacterId())
			}
			return
		}
		if isPartyOperation(l)(readerOptions, op, PartyOperationChangeLeader) {
			sp := &partysb.OperationChangeLeader{}
			sp.Decode(l, ctx)(r, readerOptions)
			p, err := party.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to locate party for character [%d] to leave.", s.CharacterId())
				return
			}
			err = party.NewProcessor(l, ctx).ChangeLeader(p.Id(), s.CharacterId(), sp.TargetCharacterId())
			if err != nil {
				l.WithError(err).Errorf("Character [%d] unable to pass leadership to [%d] in party.", s.CharacterId(), sp.TargetCharacterId())
			}
			return
		}
		if isPartyOperation(l)(readerOptions, op, PartyOperationInvite) {
			sp := &partysb.OperationInvite{}
			sp.Decode(l, ctx)(r, readerOptions)
			cs, err := character.NewProcessor(l, ctx).GetByName(sp.Name())
			if err != nil {
				l.WithError(err).Errorf("Unable to locate character by name [%s] to invite to party.", sp.Name())
				err := session.Announce(l)(ctx)(wp)(partycb.PartyOperationWriter)(partycb.PartyErrorBody("UNABLE_TO_FIND_THE_CHARACTER", sp.Name()))(s)
				if err != nil {
					return
				}
			}

			os, err := session.NewProcessor(l, ctx).GetByCharacterId(s.Field().Channel())(cs.Id())
			if err != nil || s.WorldId() != os.WorldId() || s.ChannelId() != os.ChannelId() {
				l.WithError(err).Errorf("Character [%d] not in channel. Cannot invite to party.", cs.Id())
				err = session.Announce(l)(ctx)(wp)(partycb.PartyOperationWriter)(partycb.PartyErrorBody("UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL", sp.Name()))(s)
				if err != nil {
				}
				return
			}

			err = party.NewProcessor(l, ctx).RequestInvite(s.CharacterId(), cs.Id())
			if err != nil {
				l.WithError(err).Errorf("Character [%d] was unable to request [%d] to join party.", s.CharacterId(), cs.Id())
			}
			return
		}
		l.Warnf("Character [%d] issued a unhandled party operation [%d].", s.CharacterId(), op)
	}
}

func isPartyOperation(l logrus.FieldLogger) func(options map[string]interface{}, op PartyOperation, key string) bool {
	return func(options map[string]interface{}, op PartyOperation, key string) bool {
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
		return PartyOperation(res) == op
	}
}
