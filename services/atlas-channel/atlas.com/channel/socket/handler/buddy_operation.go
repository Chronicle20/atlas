package handler

import (
	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/invite"
	invite2 "atlas-channel/kafka/message/invite"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const (
	BuddyOperationHandle = "BuddyOperationHandle"
	BuddyOperationReload = "RELOAD"
	BuddyOperationAdd    = "ADD"
	BuddyOperationAccept = "ACCEPT"
	BuddyOperationDelete = "DELETE"
)

func BuddyOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		op := r.ReadByte()
		if isBuddyOperation(l)(readerOptions, op, BuddyOperationReload) {
			l.Debugf("Character [%d] attempting to reload buddy list.", s.CharacterId())
			return
		}
		if isBuddyOperation(l)(readerOptions, op, BuddyOperationAdd) {
			name := r.ReadAsciiString()
			if len(name) < 4 || len(name) > 13 {
				l.Warnf("Character [%d] attempting to add a buddy and input name is out of range.", s.CharacterId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}
			group := r.ReadAsciiString()
			if len(group) > 16 {
				l.Warnf("Character [%d] attempting to add a buddy and input group is out of range.", s.CharacterId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			tc, err := character.NewProcessor(l, ctx).GetByName(name)
			if err != nil || s.WorldId() != tc.WorldId() {
				l.WithError(err).Errorf("Unable to locate buddy character [%d] is attempting to add.", s.CharacterId())
				// TODO send error to requester
				return
			}

			err = buddylist.NewProcessor(l, ctx).RequestAdd(s.CharacterId(), s.WorldId(), tc.Id(), group)
			if err != nil {
				l.WithError(err).Errorf("Unable to request buddy addition for character [%d].", s.CharacterId())
			}
			return
		}
		if isBuddyOperation(l)(readerOptions, op, BuddyOperationAccept) {
			fromCharacterId := r.ReadUint32()
			l.Debugf("Character [%d] attempting to accept buddy request from [%d].", s.CharacterId(), fromCharacterId)
			err := invite.NewProcessor(l, ctx).Accept(s.CharacterId(), s.WorldId(), invite2.InviteTypeBuddy, fromCharacterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to issue invite acceptance command for character [%d].", s.CharacterId())
			}
			return
		}
		if isBuddyOperation(l)(readerOptions, op, BuddyOperationDelete) {
			buddyCharacterId := r.ReadUint32()
			// This happens both when a character uses the UI to remove an existing buddy. Or when a character rejects another characters invitation.
			err := buddylist.NewProcessor(l, ctx).RequestDelete(s.CharacterId(), s.WorldId(), buddyCharacterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to request buddy addition for character [%d].", s.CharacterId())
			}
			return
		}
		l.Warnf("Character [%d] issued a unhandled buddy operation [%d].", s.CharacterId(), op)
	}
}

func isBuddyOperation(l logrus.FieldLogger) func(options map[string]interface{}, op byte, key string) bool {
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
