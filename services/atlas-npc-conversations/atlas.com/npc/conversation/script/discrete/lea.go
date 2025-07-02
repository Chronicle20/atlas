package discrete

import (
	"atlas-npc-conversations/conversation/script"
	"atlas-npc-conversations/guild"
	"atlas-npc-conversations/message"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
)

// Lea (2010008) is located in Orbis - Guild Headquarters <Hall of Fame> (200000301)
type Lea struct {
}

func (r Lea) Name() string {
	return "Lea"
}

func (r Lea) Initial(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			return r.Hello(l)(ctx)(c)
		}
	}
}

func (r Lea) Hello(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("What would you like to do?").NewLine().
				OpenItem(0).BlueText().AddText("Create/Change your Guild Emblem").CloseItem()
			return script.SendListSelection(l)(ctx)(c, m.String(), r.Selection)
		}
	}
}

func (r Lea) Selection(selection int32) script.StateProducer {
	switch selection {
	case 0:
		return r.ChangeEmblem
	}
	return nil
}

func (r Lea) ChangeEmblem(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			if !guild.IsLeader(l)(ctx)(c.CharacterId) {
				return r.MustBeLeader(l)(ctx)(c)
			}
			return r.Confirmation(l)(ctx)(c)
		}
	}
}

func (r Lea) MustBeLeader(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().AddText("You must be the Guild Leader to change the Emblem. Please tell your leader to speak with me.")
			return script.SendOk(l)(ctx)(c, m.String())
		}
	}
}

func (r Lea) Confirmation(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().AddText("Creating or changing Guild Emblem costs ").
				BlueText().AddText(fmt.Sprintf("%d mesos", 5000000)).
				BlackText().AddText(", are you sure you want to continue?")
			return script.SendYesNo(l)(ctx)(c, m.String(), r.ValidateChange, script.Exit())
		}
	}
}

func (r Lea) ValidateChange(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			_ = guild.RequestEmblem(l)(ctx)(c.Field.WorldId(), c.Field.ChannelId(), c.CharacterId)
			return script.Exit()(l)(ctx)(c)
		}
	}
}
