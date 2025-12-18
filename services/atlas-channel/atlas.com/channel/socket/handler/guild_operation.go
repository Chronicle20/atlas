package handler

import (
	"atlas-channel/character"
	"atlas-channel/guild"
	"atlas-channel/invite"
	invite2 "atlas-channel/kafka/message/invite"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const (
	GuildOperationHandle            = "GuildOperationHandle"
	GuildOperationLoad              = "LOAD"
	GuildOperationInputName         = "INPUT_NAME"
	GuildOperationRequestCreate     = "REQUEST_CREATE"
	GuildOperationAgreementResponse = "AGREEMENT_RESPONSE"
	GuildOperationCreate            = "CREATE"
	GuildOperationInvite            = "INVITE"
	GuildOperationJoin              = "JOIN"
	GuildOperationWithdraw          = "WITHDRAW"
	GuildOperationKick              = "KICK"
	GuildOperationRemove            = "REMOVE"
	GuildOperationIncreaseCapacity  = "INCREASE_CAPACITY"
	GuildOperationChangeLevel       = "CHANGE_LEVEL"
	GuildOperationChangeJob         = "CHANGE_JOB"
	GuildOperationSetTitleNames     = "SET_TITLE_NAMES"
	GuildOperationSetMemberTitle    = "SET_MEMBER_TITLE"
	GuildOperationSetEmblem         = "SET_EMBLEM"
	GuildOperationSetNotice         = "SET_NOTICE"
)

func GuildOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		op := r.ReadByte()
		if isGuildOperation(l)(readerOptions, op, GuildOperationRequestCreate) {
			name := r.ReadAsciiString()
			_ = guild.NewProcessor(l, ctx).RequestCreate(s.Map(), s.CharacterId(), name)
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationAgreementResponse) {
			unk := r.ReadUint32()
			agreed := r.ReadBool()
			l.Debugf("Character [%d] responded to the request to create a guild with [%t]. unk [%d].", s.CharacterId(), agreed, unk)
			_ = guild.NewProcessor(l, ctx).CreationAgreement(s.CharacterId(), agreed)
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationSetEmblem) {
			g, _ := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if !g.IsLeader(s.CharacterId()) {
				l.Errorf("Character [%d] attempting to change guild emblem when they are not the guild leader.", s.CharacterId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			logoBackground := r.ReadUint16()
			logoBackgroundColor := r.ReadByte()
			logo := r.ReadUint16()
			logoColor := r.ReadByte()

			_ = guild.NewProcessor(l, ctx).RequestEmblemUpdate(g.Id(), s.CharacterId(), logoBackground, logoBackgroundColor, logo, logoColor)
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationSetNotice) {
			notice := r.ReadAsciiString()
			if len(notice) > 100 {
				l.Errorf("Character [%d] setting a guild notice longer than possible.", s.CharacterId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			g, _ := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if !g.IsLeadership(s.CharacterId()) {
				l.Errorf("Character [%d] setting a guild notice when they are not allowed.", s.CharacterId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			_ = guild.NewProcessor(l, ctx).RequestNoticeUpdate(g.Id(), s.CharacterId(), notice)
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationWithdraw) {
			cid := r.ReadUint32()
			name := r.ReadAsciiString()
			if cid != s.CharacterId() {
				l.Errorf("Character [%d] attempting to have [%d] leave guild.", s.CharacterId(), cid)
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			c, err := character.NewProcessor(l, ctx).GetById()(cid)
			if err != nil || c.Name() != name {
				l.Errorf("Character [%d] attempting to have [%s] leave guild.", s.CharacterId(), name)
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			g, _ := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if g.Id() == 0 {
				l.Errorf("Character [%d] attempting to leave guild, while not in one.", s.CharacterId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			_ = guild.NewProcessor(l, ctx).Leave(g.Id(), s.CharacterId())
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationKick) {
			cid := r.ReadUint32()
			name := r.ReadAsciiString()

			g, _ := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if !g.IsLeadership(s.CharacterId()) {
				l.Errorf("Character [%d] attempting to leave guild, while not in one.", s.CharacterId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			_ = guild.NewProcessor(l, ctx).Expel(g.Id(), s.CharacterId(), cid, name)
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationInvite) {
			g, _ := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if !g.IsLeadership(s.CharacterId()) {
				l.Errorf("Character [%d] attempting to invite someone to the guild when they're not in leadership.", s.CharacterId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}
			target := r.ReadAsciiString()

			c, err := character.NewProcessor(l, ctx).GetByName(target)
			if err != nil {
				l.Errorf("Unable to locate character [%s] to invite.", target)
				// TODO announce error
				return
			}
			_ = guild.NewProcessor(l, ctx).RequestInvite(g.Id(), s.CharacterId(), c.Id())
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationJoin) {
			guildId := r.ReadUint32()
			characterId := r.ReadUint32()
			if s.CharacterId() != characterId {
				l.Errorf("Character [%d] attempting to have [%d] join guild.", s.CharacterId(), characterId)
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			err := invite.NewProcessor(l, ctx).Accept(s.CharacterId(), s.WorldId(), invite2.InviteTypeGuild, guildId)
			if err != nil {
				l.WithError(err).Errorf("Unable to issue invite acceptance command for character [%d].", s.CharacterId())
			}
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationSetTitleNames) {
			g, _ := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if !g.IsLeader(s.CharacterId()) {
				l.Errorf("Character [%d] attempting to change title names when they are not the guild leader.", s.CharacterId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}
			titles := make([]string, 5)
			for i := range 5 {
				titles[i] = r.ReadAsciiString()
			}
			_ = guild.NewProcessor(l, ctx).RequestTitleChanges(g.Id(), s.CharacterId(), titles)
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationSetMemberTitle) {
			targetId := r.ReadUint32()
			newTitle := r.ReadByte()

			if newTitle <= 1 || newTitle > 5 {
				l.Errorf("Character [%d] attempting to change [%d] to a title [%d] outside of bounds.", s.CharacterId(), targetId, newTitle)
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}
			g, _ := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if !g.TitlePossible(s.CharacterId(), newTitle) {
				l.Errorf("Character [%d] attempting to change [%d] to a title [%d] outside of bounds.", s.CharacterId(), targetId, newTitle)
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			_ = guild.NewProcessor(l, ctx).RequestMemberTitleUpdate(g.Id(), s.CharacterId(), targetId, newTitle)
			return
		}
		l.Warnf("Character [%d] issued unhandled guild operation with operation [%d].", s.CharacterId(), op)
	}
}

func isGuildOperation(l logrus.FieldLogger) func(options map[string]interface{}, op byte, key string) bool {
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
