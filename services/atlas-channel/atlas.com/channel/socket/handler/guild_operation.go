package handler

import (
	"atlas-channel/character"
	"atlas-channel/guild"
	"atlas-channel/invite"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	invite2 "github.com/Chronicle20/atlas-constants/invite"
	guild2 "github.com/Chronicle20/atlas-packet/guild"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const (
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
		p := guild2.Operation{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		op := p.Op()
		if isGuildOperation(l)(readerOptions, op, GuildOperationRequestCreate) {
			sp := &guild2.RequestCreate{}
			sp.Decode(l, ctx)(r, readerOptions)
			_ = guild.NewProcessor(l, ctx).RequestCreate(s.Field(), s.CharacterId(), sp.Name())
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationAgreementResponse) {
			sp := &guild2.AgreementResponse{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] responded to the request to create a guild with [%t]. unk [%d].", s.CharacterId(), sp.Agreed(), sp.Unk())
			_ = guild.NewProcessor(l, ctx).CreationAgreement(s.CharacterId(), sp.Agreed())
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationSetEmblem) {
			g, _ := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if !g.IsLeader(s.CharacterId()) {
				l.Errorf("Character [%d] attempting to change guild emblem when they are not the guild leader.", s.CharacterId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			sp := &guild2.SetEmblem{}
			sp.Decode(l, ctx)(r, readerOptions)

			_ = guild.NewProcessor(l, ctx).RequestEmblemUpdate(g.Id(), s.CharacterId(), sp.LogoBackground(), sp.LogoBackgroundColor(), sp.Logo(), sp.LogoColor())
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationSetNotice) {
			sp := &guild2.SetNotice{}
			sp.Decode(l, ctx)(r, readerOptions)
			if len(sp.Notice()) > 100 {
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

			_ = guild.NewProcessor(l, ctx).RequestNoticeUpdate(g.Id(), s.CharacterId(), sp.Notice())
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationWithdraw) {
			sp := &guild2.Withdraw{}
			sp.Decode(l, ctx)(r, readerOptions)
			if sp.Cid() != s.CharacterId() {
				l.Errorf("Character [%d] attempting to have [%d] leave guild.", s.CharacterId(), sp.Cid())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			c, err := character.NewProcessor(l, ctx).GetById()(sp.Cid())
			if err != nil || c.Name() != sp.Name() {
				l.Errorf("Character [%d] attempting to have [%s] leave guild.", s.CharacterId(), sp.Name())
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
			sp := &guild2.Kick{}
			sp.Decode(l, ctx)(r, readerOptions)

			g, _ := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if !g.IsLeadership(s.CharacterId()) {
				l.Errorf("Character [%d] attempting to leave guild, while not in one.", s.CharacterId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			_ = guild.NewProcessor(l, ctx).Expel(g.Id(), s.CharacterId(), sp.Cid(), sp.Name())
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationInvite) {
			g, _ := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if !g.IsLeadership(s.CharacterId()) {
				l.Errorf("Character [%d] attempting to invite someone to the guild when they're not in leadership.", s.CharacterId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}
			sp := &guild2.Invite{}
			sp.Decode(l, ctx)(r, readerOptions)

			c, err := character.NewProcessor(l, ctx).GetByName(sp.Target())
			if err != nil {
				l.Errorf("Unable to locate character [%s] to invite.", sp.Target())
				// TODO announce error
				return
			}
			_ = guild.NewProcessor(l, ctx).RequestInvite(g.Id(), s.CharacterId(), c.Id())
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationJoin) {
			sp := &guild2.Join{}
			sp.Decode(l, ctx)(r, readerOptions)
			if s.CharacterId() != sp.CharacterId() {
				l.Errorf("Character [%d] attempting to have [%d] join guild.", s.CharacterId(), sp.CharacterId())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			err := invite.NewProcessor(l, ctx).Accept(s.CharacterId(), s.WorldId(), string(invite2.TypeGuild), sp.GuildId())
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
			sp := &guild2.SetTitleNames{}
			sp.Decode(l, ctx)(r, readerOptions)
			_ = guild.NewProcessor(l, ctx).RequestTitleChanges(g.Id(), s.CharacterId(), sp.Titles())
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationSetMemberTitle) {
			sp := &guild2.SetMemberTitle{}
			sp.Decode(l, ctx)(r, readerOptions)

			if sp.NewTitle() <= 1 || sp.NewTitle() > 5 {
				l.Errorf("Character [%d] attempting to change [%d] to a title [%d] outside of bounds.", s.CharacterId(), sp.TargetId(), sp.NewTitle())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}
			g, _ := guild.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if !g.TitlePossible(s.CharacterId(), sp.NewTitle()) {
				l.Errorf("Character [%d] attempting to change [%d] to a title [%d] outside of bounds.", s.CharacterId(), sp.TargetId(), sp.NewTitle())
				_ = session.NewProcessor(l, ctx).Destroy(s)
				return
			}

			_ = guild.NewProcessor(l, ctx).RequestMemberTitleUpdate(g.Id(), s.CharacterId(), sp.TargetId(), sp.NewTitle())
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
