package handler

import (
	"atlas-channel/character"
	"atlas-channel/message"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterChatWhisperHandle = "CharacterChatWhisperHandle"

type WhisperMode byte

const (
	WhisperModeFind            = WhisperMode(5)
	WhisperModeChat            = WhisperMode(6)
	WhisperModeBuddyWindowFind = WhisperMode(68)
	WhisperModeMacroNotice     = WhisperMode(134)
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

		if mode == WhisperModeFind || mode == WhisperModeBuddyWindowFind {
			_ = produceFindResultBody(l)(ctx)(wp)(mode, targetName)(s)
			return
		}
		if mode == WhisperModeChat {
			msg = r.ReadAsciiString()
			err := message.NewProcessor(l, ctx).WhisperChat(s.Map(), s.CharacterId(), msg, targetName)
			if err != nil {
				_ = session.Announce(l)(ctx)(wp)(writer.CharacterChatWhisper)(writer.CharacterChatWhisperSendFailureResultBody(targetName, false))(s)
				return
			}
			return
		}
		l.Warnf("Character [%d] using unhandled whipser mode [%d]. Target [%s], Message [%s], UpdateTime [%d]", s.CharacterId(), mode, targetName, msg, updateTime)
	}
}

func produceFindResultBody(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(mode WhisperMode, targetName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(mode WhisperMode, targetName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(mode WhisperMode, targetName string) model.Operator[session.Model] {
			return func(mode WhisperMode, targetName string) model.Operator[session.Model] {
				return func(s session.Model) error {
					var resultMode writer.WhisperMode
					if mode == WhisperModeBuddyWindowFind {
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

					_, err = session.NewProcessor(l, ctx).GetByCharacterId(s.WorldId(), s.ChannelId())(tc.Id())
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
