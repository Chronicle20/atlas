package handler

import (
	"atlas-channel/guild"
	"atlas-channel/guild/thread"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	guildcb "github.com/Chronicle20/atlas/libs/atlas-packet/guild/clientbound"
	guildsb "github.com/Chronicle20/atlas/libs/atlas-packet/guild/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const (
	GuildBBSOperationCreateOrEditThread = "CREATE_OR_EDIT_THREAD"
	GuildBBSOperationDeleteThread       = "DELETE_THREAD"
	GuildBBSOperationListThreads        = "LIST_THREADS"
	GuildBBSOperationDisplayThread      = "DISPLAY_THREAD"
	GuildBBSOperationReplyThread        = "REPLY_THREAD"
	GuildBBSOperationDeleteReply        = "DELETE_REPLY"
)

func GuildBBSHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		g, err := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
		if err != nil {
			l.Errorf("Character [%d] attempting to manipulate guild thread without a guild.", s.CharacterId())
			_ = session.NewProcessor(l, ctx).Destroy(s)
			return
		}

		p := guildsb.BBS{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		op := p.Op()
		if isGuildBBSOperation(l)(readerOptions, op, GuildBBSOperationCreateOrEditThread) {
			sp := &guildsb.BBSCreateOrEditThread{}
			sp.Decode(l, ctx)(r, readerOptions)
			if sp.Modify() {
				_ = thread.NewProcessor(l, ctx).ModifyThread(g.Id(), s.CharacterId(), sp.ThreadId(), sp.Notice(), sp.Title(), sp.Message(), sp.EmoticonId())
				return
			} else {
				_ = thread.NewProcessor(l, ctx).CreateThread(g.Id(), s.CharacterId(), sp.Notice(), sp.Title(), sp.Message(), sp.EmoticonId())
				return
			}
		}
		if isGuildBBSOperation(l)(readerOptions, op, GuildBBSOperationDeleteThread) {
			sp := &guildsb.BBSDeleteThread{}
			sp.Decode(l, ctx)(r, readerOptions)
			_ = thread.NewProcessor(l, ctx).DeleteThread(g.Id(), s.CharacterId(), sp.ThreadId())
			return
		}
		if isGuildBBSOperation(l)(readerOptions, op, GuildBBSOperationListThreads) {
			sp := &guildsb.BBSListThreads{}
			sp.Decode(l, ctx)(r, readerOptions)
			ts, err := thread.NewProcessor(l, ctx).GetAll(g.Id())
			if err != nil {
				l.WithError(err).Errorf("Unable to display the guild threads to character [%d].", s.CharacterId())
				return
			}
			err = session.Announce(l)(ctx)(wp)(guildcb.GuildBBSWriter)(writer.GuildBBSThreadsBody(ts, sp.StartIndex()*10))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to display the guild threads to character [%d].", s.CharacterId())
				return
			}

			return
		}
		if isGuildBBSOperation(l)(readerOptions, op, GuildBBSOperationDisplayThread) {
			sp := &guildsb.BBSDisplayThread{}
			sp.Decode(l, ctx)(r, readerOptions)
			t, err := thread.NewProcessor(l, ctx).GetById(g.Id(), sp.ThreadId())
			if err != nil {
				l.WithError(err).Errorf("Unable to display the requested thread [%d] to character [%d].", t.Id(), s.CharacterId())
				return
			}
			err = session.Announce(l)(ctx)(wp)(guildcb.GuildBBSWriter)(writer.GuildBBSThreadBody(t))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to display the requested thread [%d] to character [%d].", t.Id(), s.CharacterId())
				return
			}
			return
		}
		if isGuildBBSOperation(l)(readerOptions, op, GuildBBSOperationReplyThread) {
			sp := &guildsb.BBSReplyThread{}
			sp.Decode(l, ctx)(r, readerOptions)
			_ = thread.NewProcessor(l, ctx).ReplyToThread(g.Id(), s.CharacterId(), sp.ThreadId(), sp.Message())
			return
		}
		if isGuildBBSOperation(l)(readerOptions, op, GuildBBSOperationDeleteReply) {
			sp := &guildsb.BBSDeleteReply{}
			sp.Decode(l, ctx)(r, readerOptions)
			_ = thread.NewProcessor(l, ctx).DeleteReply(g.Id(), s.CharacterId(), sp.ThreadId(), sp.ReplyId())
			return
		}
		l.Warnf("Character [%d] issued unhandled guild bbs operation with operation [%d].", s.CharacterId(), op)
	}
}

func isGuildBBSOperation(l logrus.FieldLogger) func(options map[string]interface{}, op byte, key string) bool {
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
