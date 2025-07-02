package discrete

import (
	"atlas-npc-conversations/conversation/script"
	"atlas-npc-conversations/guild"
	"atlas-npc-conversations/message"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
)

// Heracle (2010007) is located in Orbis - Guild Headquarters <Hall of Fame> (200000301)
type Heracle struct {
}

func (r Heracle) Name() string {
	return "Heracle"
}

func (r Heracle) Initial(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			return r.Hello(l)(ctx)(c)
		}
	}
}

func (r Heracle) Hello(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("What would you like to do?").NewLine().
				OpenItem(0).BlueText().AddText("Create a Guild").CloseItem().NewLine().
				OpenItem(1).BlueText().AddText("Disband your Guild").CloseItem().NewLine().
				OpenItem(2).BlueText().AddText("Increase your Guild's capacity").CloseItem()
			return script.SendListSelection(l)(ctx)(c, m.String(), r.Selection)
		}
	}
}

func (r Heracle) Selection(selection int32) script.StateProducer {
	switch selection {
	case 0:
		return r.Create
	case 1:
		return r.Disband
	case 2:
		return r.IncreaseCapacity
	}
	return nil
}

func (r Heracle) Create(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			if guild.HasGuild(l)(ctx)(c.CharacterId) {
				return r.AlreadyHaveGuild(l)(ctx)(c)
			}
			return r.CreateConfirmation(l)(ctx)(c)

		}
	}
}

func (r Heracle) Disband(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			if !guild.IsLeader(l)(ctx)(c.CharacterId) {
				return r.MustBeLeaderToDisband(l)(ctx)(c)
			}
			return r.DisbandConfirmation(l)(ctx)(c)
		}
	}
}

func (r Heracle) IncreaseCapacity(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			if !guild.IsLeader(l)(ctx)(c.CharacterId) {
				return r.MustBeLeaderToIncrease(l)(ctx)(c)
			}
			return r.IncreaseConfirmation(l)(ctx)(c)
		}
	}
}

func (r Heracle) AlreadyHaveGuild(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().AddText("You may not create a new Guild while you are in one.")
			return script.SendOk(l)(ctx)(c, m.String())
		}
	}
}

func (r Heracle) CreateConfirmation(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Creating a Guild costs ").
				BlueText().AddText("1500000 mesos").
				BlackText().AddText(", are you sure you want to continue?")
			return script.SendYesNo(l)(ctx)(c, m.String(), r.ValidateCreate, script.Exit())
		}
	}
}

func (r Heracle) MustBeLeaderToDisband(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().AddText("You can only disband a Guild if you are the leader of that Guild.")
			return script.SendOk(l)(ctx)(c, m.String())
		}
	}
}

func (r Heracle) DisbandConfirmation(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Are you sure you want to disband your Guild? You will not be able to recover it afterward and all your GP will be gone.")
			return script.SendYesNo(l)(ctx)(c, m.String(), r.PerformDisband, script.Exit())
		}
	}
}

func (r Heracle) MustBeLeaderToIncrease(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("You can only increase your Guild's capacity if you are the leader.")
			return script.SendOk(l)(ctx)(c, m.String())
		}
	}
}

func (r Heracle) ValidateCreate(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			guild.RequestName(l)(ctx)(c.Field.WorldId(), c.Field.ChannelId(), c.CharacterId)
			return script.Exit()(l)(ctx)(c)
		}
	}
}

func (r Heracle) PerformDisband(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			guild.RequestDisband(l)(ctx)(c.Field.WorldId(), c.Field.ChannelId(), c.CharacterId)
			return script.Exit()(l)(ctx)(c)
		}
	}
}

func (r Heracle) ValidateIncrease(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			guild.RequestCapacityIncrease(l)(ctx)(c.Field.WorldId(), c.Field.ChannelId(), c.CharacterId)
			return script.Exit()(l)(ctx)(c)
		}
	}
}

func (r Heracle) IncreaseConfirmation(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Increasing your Guild capacity by ").
				BlueText().AddText(fmt.Sprintf("%d", 5)).
				BlackText().AddText(" costs ").
				BlueText().AddText(fmt.Sprintf("%d mesos", r.GetGuildCapacityIncreaseCost(l)(c.CharacterId))).
				BlackText().AddText(", are you sure you want to continue?")
			return script.SendYesNo(l)(ctx)(c, m.String(), r.ValidateIncrease, script.Exit())
		}
	}
}

func (r Heracle) GetGuildCapacityIncreaseCost(l logrus.FieldLogger) func(characterId uint32) uint32 {
	return func(characterId uint32) uint32 {
		//TODO query this
		return 1000
	}
}
