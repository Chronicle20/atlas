package handler

import (
	"atlas-channel/guild"
	"atlas-channel/guild/thread"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const (
	GuildBBSHandle                      = "GuildBBSHandle"
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

		op := r.ReadByte()
		if isGuildBBSOperation(l)(readerOptions, op, GuildBBSOperationCreateOrEditThread) {
			modify := r.ReadBool()
			if modify {
				threadId := r.ReadUint32()
				notice := r.ReadBool()
				title := r.ReadAsciiString()
				message := r.ReadAsciiString()
				emoticonId := r.ReadUint32()
				_ = thread.NewProcessor(l, ctx).ModifyThread(g.Id(), s.CharacterId(), threadId, notice, title, message, emoticonId)
				return
			} else {
				notice := r.ReadBool()
				title := r.ReadAsciiString()
				message := r.ReadAsciiString()
				emoticonId := r.ReadUint32()
				_ = thread.NewProcessor(l, ctx).CreateThread(g.Id(), s.CharacterId(), notice, title, message, emoticonId)
				return
			}
		}
		if isGuildBBSOperation(l)(readerOptions, op, GuildBBSOperationDeleteThread) {
			threadId := r.ReadUint32()
			_ = thread.NewProcessor(l, ctx).DeleteThread(g.Id(), s.CharacterId(), threadId)
			return
		}
		if isGuildBBSOperation(l)(readerOptions, op, GuildBBSOperationListThreads) {
			startIndex := r.ReadUint32()
			ts, err := thread.NewProcessor(l, ctx).GetAll(g.Id())
			if err != nil {
				l.WithError(err).Errorf("Unable to display the guild threads to character [%d].", s.CharacterId())
				return
			}
			err = session.Announce(l)(ctx)(wp)(writer.GuildBBS)(writer.GuildBBSThreadsBody(l)(ts, startIndex*10))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to display the guild threads to character [%d].", s.CharacterId())
				return
			}

			return
		}
		if isGuildBBSOperation(l)(readerOptions, op, GuildBBSOperationDisplayThread) {
			threadId := r.ReadUint32()
			t, err := thread.NewProcessor(l, ctx).GetById(g.Id(), threadId)
			if err != nil {
				l.WithError(err).Errorf("Unable to display the requested thread [%d] to character [%d].", t.Id(), s.CharacterId())
				return
			}
			err = session.Announce(l)(ctx)(wp)(writer.GuildBBS)(writer.GuildBBSThreadBody(l)(t))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to display the requested thread [%d] to character [%d].", t.Id(), s.CharacterId())
				return
			}
			return
		}
		if isGuildBBSOperation(l)(readerOptions, op, GuildBBSOperationReplyThread) {
			threadId := r.ReadUint32()
			message := r.ReadAsciiString()
			_ = thread.NewProcessor(l, ctx).ReplyToThread(g.Id(), s.CharacterId(), threadId, message)
			return
		}
		if isGuildBBSOperation(l)(readerOptions, op, GuildBBSOperationDeleteReply) {
			threadId := r.ReadUint32()
			replyId := r.ReadUint32()
			_ = thread.NewProcessor(l, ctx).DeleteReply(g.Id(), s.CharacterId(), threadId, replyId)
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
