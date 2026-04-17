package handler

import (
	"atlas-channel/character"
	"atlas-channel/message"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	chatCB "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	chat "github.com/Chronicle20/atlas/libs/atlas-packet/chat/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
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
				_ = session.Announce(l)(ctx)(wp)(chatCB.WhisperWriter)(chatCB.NewWhisperSendResult(0x0A, p.TargetName(), false).Encode)(s)
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
					var resultMode byte
					if mode == chat.WhisperModeBuddyWindowFind {
						resultMode = 0x48
					} else {
						resultMode = 0x09
					}

					af := session.Announce(l)(ctx)(wp)(chatCB.WhisperWriter)

					tc, err := character.NewProcessor(l, ctx).GetByName(targetName)
					if err != nil {
						return af(chatCB.NewWhisperFindResultError(resultMode, targetName).Encode)(s)
					}
					// TODO query cash shop.
					cs := false
					if cs {
						return af(chatCB.NewWhisperFindResultCashShop(resultMode, targetName).Encode)(s)
					}

					_, err = session.NewProcessor(l, ctx).GetByCharacterId(s.Field().Channel())(tc.Id())
					if err == nil {
						if resultMode == 0x09 {
							return af(chatCB.NewWhisperFindResultMapWithXY(resultMode, tc.Name(), uint32(tc.MapId()), tc.X(), tc.Y()).Encode)(s)
						}
						return af(chatCB.NewWhisperFindResultMap(resultMode, tc.Name(), uint32(tc.MapId())).Encode)(s)
					}

					// TODO find a way to look up remote channel.
					return af(chatCB.NewWhisperFindResultChannel(resultMode, targetName, 0).Encode)(s)
				}
			}
		}
	}
}
