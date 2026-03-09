package handler

import (
	"atlas-channel/character"
	"atlas-channel/message"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-packet/chat"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func CharacterChatWhisperHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := chat.Whisper{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		if p.Mode() == chat.WhisperModeFind || p.Mode() == chat.WhisperModeBuddyWindowFind {
			_ = produceFindResultBody(l)(ctx)(wp)(p.Mode(), p.TargetName())(s)
			return
		}
		if p.Mode() == chat.WhisperModeChat {
			err := message.NewProcessor(l, ctx).WhisperChat(s.Field(), s.CharacterId(), p.Msg(), p.TargetName())
			if err != nil {
				_ = session.Announce(l)(ctx)(wp)(writer.CharacterChatWhisper)(writer.CharacterChatWhisperSendFailureResultBody(p.TargetName(), false))(s)
				return
			}
			return
		}
		l.Warnf("Character [%d] using unhandled whisper mode [%d]. Target [%s], Message [%s], UpdateTime [%d]", s.CharacterId(), p.Mode(), p.TargetName(), p.Msg(), p.UpdateTime())
	}
}

func produceFindResultBody(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(mode chat.WhisperMode, targetName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(mode chat.WhisperMode, targetName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(mode chat.WhisperMode, targetName string) model.Operator[session.Model] {
			return func(mode chat.WhisperMode, targetName string) model.Operator[session.Model] {
				return func(s session.Model) error {
					var resultMode writer.WhisperMode
					if mode == chat.WhisperModeBuddyWindowFind {
						resultMode = writer.WhisperModeBuddyWindowFindResult
					} else {
						resultMode = writer.WhisperModeFindResult
					}

					af := session.Announce(l)(ctx)(wp)(writer.CharacterChatWhisper)

					tc, err := character.NewProcessor(l, ctx).GetByName(targetName)
					if err != nil {
						return af(writer.CharacterChatWhisperFindResultErrorBody(resultMode, targetName))(s)
					}
					// TODO query cash shop.
					cs := false
					if cs {
						return af(writer.CharacterChatWhisperFindResultInCashShopBody(resultMode, targetName))(s)
					}

					_, err = session.NewProcessor(l, ctx).GetByCharacterId(s.Field().Channel())(tc.Id())
					if err == nil {
						return af(writer.CharacterChatWhisperFindResultInMapBody(resultMode, tc, tc.MapId()))(s)
					}

					// TODO find a way to look up remote channel.
					return af(writer.CharacterChatWhisperFindResultInOtherChannelBody(resultMode, targetName, 0))(s)
				}
			}
		}
	}
}
