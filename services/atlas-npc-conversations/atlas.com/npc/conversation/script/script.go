package script

import (
	"atlas-npc-conversations/npc"
	"context"
	"github.com/sirupsen/logrus"
)

type Context struct {
	WorldId     byte
	ChannelId   byte
	CharacterId uint32
	MapId       uint32
	NPCId       uint32
}

type Script interface {
	Name() string

	Initial(l logrus.FieldLogger) func(ctx context.Context) func(c Context) State
}

type StateProducer func(l logrus.FieldLogger) func(ctx context.Context) func(c Context) State

type ProcessNumber func(selection int32) StateProducer

type ProcessText func(text string) StateProducer

type State func(l logrus.FieldLogger) func(ctx context.Context) func(c Context, mode byte, theType byte, selection int32) State

type ProcessSelection func(selection int32) StateProducer

func Exit() StateProducer {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(c Context) State {
		return func(ctx context.Context) func(c Context) State {
			return func(c Context) State {
				npc.Dispose(l)(ctx)(c.WorldId, c.ChannelId, c.CharacterId)
				return nil
			}
		}
	}
}

func SendListSelection(l logrus.FieldLogger) func(ctx context.Context) func(c Context, message string, s ProcessSelection) State {
	return func(ctx context.Context) func(c Context, message string, s ProcessSelection) State {
		return func(c Context, message string, s ProcessSelection) State {
			npc.SendSimple(l)(ctx)(c.WorldId, c.ChannelId, c.CharacterId, c.NPCId)(message)
			return doListSelectionExit(Exit(), s)

		}
	}
}

func doListSelectionExit(e StateProducer, s ProcessSelection) State {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(c Context, mode byte, theType byte, selection int32) State {
		return func(ctx context.Context) func(c Context, mode byte, theType byte, selection int32) State {
			return func(c Context, mode byte, theType byte, selection int32) State {
				if mode == 0 && theType == 4 {
					return e(l)(ctx)(c)
				}

				f := s(selection)
				if f == nil {
					l.Errorf("unhandled selection %d for npc %d.", selection, c.NPCId)
					return nil
				}
				return f(l)(ctx)(c)
			}
		}

	}
}

type SendTalkConfig struct {
	configurators []npc.TalkConfigurator
	exit          StateProducer
}

func (c SendTalkConfig) Exit() StateProducer {
	return c.exit
}

func (c SendTalkConfig) Configurators() []npc.TalkConfigurator {
	return c.configurators
}

type SendTalkConfigurator func(config *SendTalkConfig)

func SendOk(l logrus.FieldLogger) func(ctx context.Context) func(c Context, message string, configurations ...SendTalkConfigurator) State {
	return func(ctx context.Context) func(c Context, message string, configurations ...SendTalkConfigurator) State {
		return func(c Context, message string, configurations ...SendTalkConfigurator) State {
			return sendTalk(l, c, message, configurations, npc.SendOk(l)(ctx)(c.WorldId, c.ChannelId, c.CharacterId, c.NPCId), func(exit StateProducer) State { return exit(l)(ctx)(c) })
		}
	}
}

type ProcessStateFunc func(exit StateProducer) State

func sendTalk(l logrus.FieldLogger, c Context, message string, configurations []SendTalkConfigurator, talkFunc npc.TalkFunc, do ProcessStateFunc) State {
	baseConfig := &SendTalkConfig{configurators: make([]npc.TalkConfigurator, 0), exit: Exit()}
	for _, configuration := range configurations {
		configuration(baseConfig)
	}

	talkFunc(message, baseConfig.Configurators()...)
	return do(baseConfig.Exit())
}

func SendYesNo(l logrus.FieldLogger) func(ctx context.Context) func(c Context, message string, yes StateProducer, no StateProducer, configurations ...SendTalkConfigurator) State {
	return func(ctx context.Context) func(c Context, message string, yes StateProducer, no StateProducer, configurations ...SendTalkConfigurator) State {
		return func(c Context, message string, yes StateProducer, no StateProducer, configurations ...SendTalkConfigurator) State {
			return sendTalk(l, c, message, configurations, npc.SendYesNo(l)(ctx)(c.WorldId, c.ChannelId, c.CharacterId, c.NPCId), doYesNo(yes, no))
		}
	}
}

func doYesNo(yes StateProducer, no StateProducer) ProcessStateFunc {
	return func(exit StateProducer) State {
		return func(l logrus.FieldLogger) func(ctx context.Context) func(c Context, mode byte, theType byte, selection int32) State {
			return func(ctx context.Context) func(c Context, mode byte, theType byte, selection int32) State {
				return func(c Context, mode byte, theType byte, selection int32) State {
					if mode == 255 && theType == 0 {
						return exit(l)(ctx)(c)
					}
					if mode == 0 && no != nil {
						return no(l)(ctx)(c)
					} else if mode == 1 && yes != nil {
						return yes(l)(ctx)(c)
					}
					return nil
				}
			}
		}
	}
}
