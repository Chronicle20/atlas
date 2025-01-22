package handler

import (
	"atlas-channel/guild"
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
	GuildOperationSetRankName       = "SET_RANK_NAME"
	GuildOperationSetMemberRank     = "SET_MEMBER_RANK"
	GuildOperationSetEmblem         = "SET_EMBLEM"
	GuildOperationSetNotice         = "SET_NOTICE"
)

func GuildOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		op := r.ReadByte()
		if isGuildOperation(l)(readerOptions, op, GuildOperationRequestCreate) {
			name := r.ReadAsciiString()
			_ = guild.RequestCreate(l)(ctx)(s.WorldId(), s.ChannelId(), s.MapId(), s.CharacterId(), name)
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationAgreementResponse) {
			unk := r.ReadUint32()
			agreed := r.ReadBool()
			l.Debugf("Character [%d] responded to the request to create a guild with [%t]. unk [%d].", s.CharacterId(), agreed, unk)
			_ = guild.CreationAgreement(l)(ctx)(s.CharacterId(), agreed)
			return
		}
		if isGuildOperation(l)(readerOptions, op, GuildOperationSetEmblem) {
			logoBackground := r.ReadUint16()
			logoBackgroundColor := r.ReadByte()
			logo := r.ReadUint16()
			logoColor := r.ReadByte()

			g, _ := guild.GetByMemberId(l)(ctx)(s.CharacterId())
			if g.Id() == 0 || g.LeaderId() != s.CharacterId() {
				l.Errorf("Character [%d] attempting to change guild emblem when they are not the guild leader.", s.CharacterId())
				_ = session.Destroy(l, ctx, session.GetRegistry())(s)
			}

			_ = guild.RequestEmblemUpdate(l)(ctx)(g.Id(), s.CharacterId(), logoBackground, logoBackgroundColor, logo, logoColor)
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
