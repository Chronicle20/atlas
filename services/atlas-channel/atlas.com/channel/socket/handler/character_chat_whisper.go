package handler

import (
	"atlas-channel/character"
	"atlas-channel/message"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterChatWhisperHandle = "CharacterChatWhisperHandle"

type WhisperMode byte

const (
	WhisperModeFind       = WhisperMode(5)
	WhisperModeChat       = WhisperMode(6)
	WhisperModeFindFriend = WhisperMode(68)
)

func CharacterChatWhisperHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		mode := WhisperMode(r.ReadByte())
		updateTime := uint32(0)
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			updateTime = r.ReadUint32()
		}
		targetName := r.ReadAsciiString()
		msg := ""

		if mode == WhisperModeFind || mode == WhisperModeFindFriend {
			tc, err := character.GetByName(l, ctx)(targetName)
			if err != nil {
				_ = session.Announce(l)(ctx)(wp)(writer.CharacterChatWhisper)(writer.CharacterChatWhisperFindResultErrorBody(targetName))(s)
				return
			}
			// TODO query cash shop.
			cs := false
			if cs {
				_ = session.Announce(l)(ctx)(wp)(writer.CharacterChatWhisper)(writer.CharacterChatWhisperFindResultInCashShopBody(targetName))(s)
				return
			}

			_, err = session.GetByCharacterId(t)(tc.Id())
			if err == nil {
				_ = session.Announce(l)(ctx)(wp)(writer.CharacterChatWhisper)(writer.CharacterChatWhisperFindResultInMapBody(tc, tc.MapId()))(s)
				return
			}

			// TODO find a way to look up remote channel.
			_ = session.Announce(l)(ctx)(wp)(writer.CharacterChatWhisper)(writer.CharacterChatWhisperFindResultInOtherChannelBody(targetName, 0))(s)
		}
		if mode == WhisperModeChat {
			msg = r.ReadAsciiString()
			err := message.WhisperChat(l)(ctx)(s.Map(), s.CharacterId(), msg, targetName)
			if err != nil {
				_ = session.Announce(l)(ctx)(wp)(writer.CharacterChatWhisper)(writer.CharacterChatWhisperSendFailureResultBody(targetName, false))(s)
				return
			}
			return
		}
		l.Warnf("Character [%d] using unhandled whipser mode [%d]. Target [%s], Message [%s], UpdateTime [%d]", s.CharacterId(), mode, targetName, msg, updateTime)
	}
}
