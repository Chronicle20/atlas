package discrete

import (
	"atlas-npc-conversations/character"
	"atlas-npc-conversations/conversation/script"
	_map "atlas-npc-conversations/map"
	"atlas-npc-conversations/message"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
)

// Phil (1002000) is located in Victoria Road - Lith Harbor (104000000)
type Phil struct {
}

func (r Phil) Name() string {
	return "Phil"
}

func (r Phil) Initial(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			return r.WouldYouLikeTo(l)(ctx)(c)
		}
	}
}

func (r Phil) WouldYouLikeTo(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Do you wanna head over to some other town? With a little money involved, I can make it happen. It's a tad expensive, but I run a special 90% discount for beginners.")
			return script.SendNextExit(l)(ctx)(c, m.String(), r.Confused, r.MoreToDo)
		}
	}
}

func (r Phil) Confused(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("It's understandable that you may be confused about this place if this is your first time around. If you got any questions about this place, fire away.").
				NewLine().
				OpenItem(0).BlueText().AddText("What kind of towns are here in Victoria Island?").CloseItem().NewLine().
				OpenItem(1).NormalText().AddText("Please take me somewhere else.").BlackText().CloseItem()
			return script.SendListSelectionExit(l)(ctx)(c, m.String(), r.ConfusedResult, r.MoreToDo)
		}
	}
}

func (r Phil) MoreToDo(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("There's a lot to see in this town, too. Let me know if you want to go somewhere else.")
			return script.SendNext(l)(ctx)(c, m.String(), script.Exit())
		}
	}
}

func (r Phil) ConfusedResult(selection int32) script.StateProducer {
	switch selection {
	case 0:
		return r.KindOfTowns
	case 1:
		return r.TakeMeSomewhere
	}
	return nil
}

func (r Phil) KindOfTowns(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("There are 7 big towns here in Victoria Island. Which of those do you want to know more of?").NewLine().
				OpenItem(0).BlueText().ShowMap(_map.LithHarbor).CloseItem().NewLine().
				OpenItem(1).BlueText().ShowMap(_map.Perion).CloseItem().NewLine().
				OpenItem(2).BlueText().ShowMap(_map.Ellinia).CloseItem().NewLine().
				OpenItem(3).BlueText().ShowMap(_map.Henesys).CloseItem().NewLine().
				OpenItem(4).BlueText().ShowMap(_map.KerningCity).CloseItem().NewLine().
				OpenItem(5).BlueText().ShowMap(_map.Nautalis).CloseItem().NewLine().
				OpenItem(6).BlueText().ShowMap(_map.Sleepywood).CloseItem()
			return script.SendListSelectionExit(l)(ctx)(c, m.String(), r.SelectTownInfo, r.MoreToDo)
		}
	}
}

func (r Phil) Cost(index int32, beginner bool) uint32 {
	costDivisor := 1
	if beginner {
		costDivisor = 10
	}

	cost := uint32(0)
	switch index {
	case 0:
		cost = 1000
		break
	case 1:
		cost = 1000
		break
	case 2:
		cost = 800
		break
	case 3:
		cost = 1000
		break
	case 4:
		cost = 800
		break
	}
	return cost / uint32(costDivisor)
}

func (r Phil) TakeMeSomewhere(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			mb := message.NewBuilder()
			beginner := character.IsBeginnerTree(l)(ctx)(c.CharacterId)

			if beginner {
				mb = mb.AddText("There's a special 90% discount for all beginners. Alright, where would you want to go?").BlueText().NewLine()
			} else {
				mb = mb.AddText("Oh you aren't a beginner, huh? Then I'm afraid I may have to charge you full price. Where would you like to go?").BlueText().NewLine()
			}
			mb = mb.
				OpenItem(0).ShowMap(_map.Perion).AddText(fmt.Sprintf(" (%d mesos)", r.Cost(0, beginner))).CloseItem().NewLine().
				OpenItem(1).ShowMap(_map.Ellinia).AddText(fmt.Sprintf(" (%d mesos)", r.Cost(1, beginner))).CloseItem().NewLine().
				OpenItem(2).ShowMap(_map.Henesys).AddText(fmt.Sprintf(" (%d mesos)", r.Cost(2, beginner))).CloseItem().NewLine().
				OpenItem(3).ShowMap(_map.KerningCity).AddText(fmt.Sprintf(" (%d mesos)", r.Cost(3, beginner))).CloseItem().NewLine().
				OpenItem(4).ShowMap(_map.Nautalis).AddText(fmt.Sprintf(" (%d mesos)", r.Cost(4, beginner))).CloseItem()
			return script.SendListSelectionExit(l)(ctx)(c, mb.String(), r.SelectTownConfirm(beginner), r.MoreToDo)
		}
	}
}

func (r Phil) SelectTownInfo(selection int32) script.StateProducer {
	switch selection {
	case 0:
		return r.LithFirst
	case 1:
		return r.PerionFirst
	case 2:
		return r.ElliniaFirst
	case 3:
		return r.HenesysFirst
	case 4:
		return r.KerningCityFirst
	case 5:
		return r.NautalisFirst
	case 6:
		return r.SleepywoodFirst
	}
	return nil
}

func (r Phil) LithFirst(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("The town you are at is Lith Harbor! Alright I'll explain to you more about ").
				BlueText().AddText("Lith Harbor").
				BlackText().AddText(". It's the place you landed on Victoria Island by riding The Victoria. That's Lith Harbor. A lot of beginners who just got here from Maple Island start their journey here.")
			return script.SendNextExit(l)(ctx)(c, m.String(), r.LithSecond, r.MoreToDo)
		}
	}
}

func (r Phil) LithSecond(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("It's a quiet town with the wide body of water on the back of it, thanks to the fact that the harbor is located at the west end of the island. Most of the people here are, or used to be fisherman, so they may look intimidating, but if you strike up a conversation with them, they'll be friendly to you.")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.LithThird, r.LithFirst, r.MoreToDo)
		}
	}
}

func (r Phil) LithThird(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Around town lies a beautiful prairie. Most of the monsters there are small and gentle, perfect for beginners. If you haven't chosen your job yet, this is a good place to boost up your level.")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.Confused, r.LithSecond, r.MoreToDo)
		}
	}
}

func (r Phil) PerionFirst(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Alright I'll explain to you more about ").
				BlueText().AddText("Perion").
				BlackText().AddText(". It's a warrior-town located at the northern-most part of Victoria Island, surrounded by rocky mountains. With an unfriendly atmosphere, only the strong survives there.")
			return script.SendNextExit(l)(ctx)(c, m.String(), r.PerionSecond, r.MoreToDo)
		}
	}
}

func (r Phil) PerionSecond(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Around the highland you'll find a really skinny tree, a wild hog running around the place, and monkeys that live all over the island. There's also a deep valley, and when you go deep into it, you'll find a humongous dragon with the power to match his size. Better go in there very carefully, or don't go at all.")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.PerionThird, r.PerionFirst, r.MoreToDo)
		}
	}
}

func (r Phil) PerionThird(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("If you want to be a ").
				BlueText().AddText("Warrior").
				BlackText().AddText(" then find ").
				RedText().AddText("Dances with Balrog").
				BlackText().AddText(", the chief of Perion. If you're level 10 or higher, along with a good STR level, he may make you a warrior after all. If not, better keep training yourself until you reach that level.")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.Confused, r.PerionSecond, r.MoreToDo)
		}
	}
}

func (r Phil) ElliniaFirst(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Alright I'll explain to you more about ").
				BlueText().AddText("Ellinia").
				BlackText().AddText(". It's a magician-town located at the far east of Victoria Island, and covered in tall, mystic trees. You'll find some fairies there, too. They don't like humans in general so it'll be best for you to be on their good side and stay quiet.")
			return script.SendNextExit(l)(ctx)(c, m.String(), r.ElliniaSecond, r.MoreToDo)
		}
	}
}

func (r Phil) ElliniaSecond(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Near the forest you'll find green slimes, walking mushrooms, monkeys and zombie monkeys all residing there. Walk deeper into the forest and you'll find witches with the flying broomstick navigating the skies. A word of warning: Unless you are really strong, I recommend you don't go near them.")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.ElliniaThird, r.ElliniaFirst, r.MoreToDo)
		}
	}
}

func (r Phil) ElliniaThird(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("If you want to be a ").
				BlueText().AddText("Magician").
				BlackText().AddText(", search for ").
				RedText().AddText("Grendel the Really Old").
				BlackText().AddText(", the head wizard of Ellinia. He may make you a wizard if you're at or above level 8 with a decent amount of INT. If that's not the case, you may have to hunt more and train yourself to get there.")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.Confused, r.ElliniaSecond, r.MoreToDo)
		}
	}
}

func (r Phil) HenesysFirst(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Alright I'll explain to you more about ").
				BlueText().AddText("Henesys").
				BlackText().AddText(". It's a bowman-town located at the southernmost part of the island, made on a flatland in the midst of a deep forest and prairies. The weather's just right, and everything is plentiful around that town, perfect for living. Go check it out.")
			return script.SendNextExit(l)(ctx)(c, m.String(), r.HenesysSecond, r.MoreToDo)
		}
	}
}

func (r Phil) HenesysSecond(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Around the prairie you'll find weak monsters such as snails, mushrooms, and pigs. According to what I hear, though, in the deepest part of the Pig Park, which is connected to the town somewhere, you'll find a humongous, powerful mushroom called Mushmom every now and then.")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.HenesysThird, r.HenesysFirst, r.MoreToDo)
		}
	}
}

func (r Phil) HenesysThird(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("If you want to be a ").
				BlueText().AddText("Bowman").
				BlackText().AddText(", you need to go see ").
				RedText().AddText("Athena Pierce").
				BlackText().AddText(" at Henesys. With a level at or above 10 and a decent amount of DEX, she may make you be one after all. If not, go train yourself, make yourself stronger, then try again.")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.Confused, r.HenesysSecond, r.MoreToDo)
		}
	}
}

func (r Phil) KerningCityFirst(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Alright I'll explain to you more about ").
				BlueText().AddText("Kerning City").
				BlackText().AddText(". It's a thief-town located at the northwest part of Victoria Island, and there are buildings up there that have just this strange feeling around them. It's mostly covered in black clouds, but if you can go up to a really high place, you'll be able to see a very beautiful sunset there.")
			return script.SendNextExit(l)(ctx)(c, m.String(), r.KerningCitySecond, r.MoreToDo)
		}
	}
}

func (r Phil) KerningCitySecond(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("From Kerning City, you can go into several dungeons. You can go to a swamp where alligators and snakes are abound, or hit the subway full of ghosts and bats. At the deepest part of the underground, you'll find Lace, who is just as big and dangerous as a dragon.")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.KerningCityThird, r.KerningCityFirst, r.MoreToDo)
		}
	}
}

func (r Phil) KerningCityThird(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("If you want to be a ").
				BlueText().AddText("Thief").
				BlackText().AddText(", seek ").
				RedText().AddText("Dark Lord").
				BlackText().AddText(", the heart of darkness of Kerning City. He may well make you a thief if you're at or above level 10 with a good amount of DEX. If not, go hunt and train yourself to reach there.")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.Confused, r.KerningCitySecond, r.MoreToDo)
		}
	}
}

func (r Phil) NautalisFirst(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Here's a little information on ").
				BlueText().ShowMap(120000000).
				BlackText().AddText(". It's a submarine that's currently parked in between Ellinia and Henesys in Victoria Island. That submarine serves as home to numerous pirates. You can have just as beautiful a view of the ocean there as you do here in Lith Harbor.")
			return script.SendNextExit(l)(ctx)(c, m.String(), r.NautalisSecond, r.MoreToDo)
		}
	}
}

func (r Phil) NautalisSecond(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				ShowMap(120000000).
				AddText(" is parked in between Henesys and Ellinia, so if you step out just a bit, you'll be able to enjoy the view of both towns. All the pirates you'll meet in town are very gregarious and friendly as well.")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.NautalisThird, r.NautalisFirst, r.MoreToDo)
		}
	}
}

func (r Phil) NautalisThird(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("If you are serious about becoming a ").
				BlueText().AddText("Pirate").
				BlackText().AddText(", then you better meet the captain of ").
				ShowMap(120000000).AddText(", ").
				RedText().ShowNPC(1090000).
				BlackText().AddText(". If you are over Level 10 with 20 DEX, then she may let you become one. If you aren't up to that level, then you'll need to train harder to get there!")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.Confused, r.NautalisSecond, r.MoreToDo)
		}
	}
}

func (r Phil) SleepywoodFirst(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Alright I'll explain to you more about ").
				BlueText().AddText("Sleepywood").
				BlackText().AddText(". It's a forest town located at the southeast side of Victoria Island. It's pretty much in between Henesys and the ant-tunnel dungeon. There's a hotel there, so you can rest up after a long day at the dungeon ... it's a quiet town in general.")
			return script.SendNextExit(l)(ctx)(c, m.String(), r.SleepywoodSecond, r.MoreToDo)
		}
	}
}

func (r Phil) SleepywoodSecond(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("In front of the hotel there's an old buddhist monk by the name of ").
				RedText().AddText("Chrishrama").
				BlackText().AddText(". Nobody knows a thing about that monk. Apparently he collects materials from the travelers and create something, but I am not too sure about the details. If you have any business going around that area, please check that out for me.")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.SleepywoodThird, r.SleepywoodFirst, r.MoreToDo)
		}
	}
}

func (r Phil) SleepywoodThird(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("From Sleepywood, head east and you'll find the ant tunnel connected to the deepest part of the Victoria Island. Lots of nasty, powerful monsters abound so if you walk in thinking it's a walk in the park, you'll be coming out as a corpse. You need to fully prepare yourself for a rough ride before going in.")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.SleepywoodFourth, r.SleepywoodSecond, r.MoreToDo)
		}
	}
}

func (r Phil) SleepywoodFourth(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("And this is what I hear ... apparently, at Sleepywood there's a secret entrance leading you to an unknown place. Apparently, once you move in deep, you'll find a stack of black rocks that actually move around. I want to see that for myself in the near future ...")
			return script.SendNextPreviousExit(l)(ctx)(c, m.String(), r.Confused, r.SleepywoodThird, r.MoreToDo)
		}
	}
}

func (r Phil) SelectTownConfirm(beginner bool) func(selection int32) script.StateProducer {
	return func(selection int32) script.StateProducer {
		switch selection {
		case 0:
			return r.ConfirmPerion(r.Cost(selection, beginner))
		case 1:
			return r.ConfirmEllinia(r.Cost(selection, beginner))
		case 2:
			return r.ConfirmHenesys(r.Cost(selection, beginner))
		case 3:
			return r.ConfirmKerningCity(r.Cost(selection, beginner))
		case 4:
			return r.ConfirmNautalis(r.Cost(selection, beginner))
		}
		return nil
	}
}

func (r Phil) ConfirmMap(mapId uint32, cost uint32) script.StateProducer {
	m := message.NewBuilder().
		AddText("I guess you don't need to be here. Do you really want to move to ").
		BlueText().ShowMap(mapId).
		BlackText().AddText("? Well it'll cost you ").
		BlueText().AddText(fmt.Sprintf("%d mesos", cost)).
		BlackText().AddText(". What do you think?")
	return func(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
		return func(ctx context.Context) func(c script.Context) script.State {
			return func(c script.Context) script.State {
				return script.SendYesNoExit(l)(ctx)(c, m.String(), r.PerformTransaction(mapId, cost), r.MoreToDo, r.MoreToDo)
			}
		}
	}
}

func (r Phil) ConfirmPerion(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.Perion, cost)
}

func (r Phil) ConfirmEllinia(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.Ellinia, cost)
}

func (r Phil) ConfirmHenesys(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.Henesys, cost)
}

func (r Phil) ConfirmKerningCity(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.KerningCity, cost)
}

func (r Phil) ConfirmNautalis(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.Nautalis, cost)
}

func (r Phil) PerformTransaction(mapId uint32, cost uint32) script.StateProducer {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
		return func(ctx context.Context) func(c script.Context) script.State {
			return func(c script.Context) script.State {
				if !character.HasMeso(l)(ctx)(c.CharacterId, cost) {
					m := message.NewBuilder().
						AddText("You don't have enough mesos. With your abilities, you should have more than that!")
					return script.SendNextExit(l)(ctx)(c, m.String(), script.Exit(), script.Exit())
				}

				_ = character.RequestChangeMeso(l)(ctx)(c.CharacterId, c.Field.WorldId(), -int32(cost))

				return func(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context, mode byte, theType byte, selection int32) script.State {
					return func(ctx context.Context) func(c script.Context, mode byte, theType byte, selection int32) script.State {
						return func(c script.Context, mode byte, theType byte, selection int32) script.State {
							if mode == 0 {
								_ = character.WarpById(l)(ctx)(c.Field.WorldId(), c.Field.ChannelId(), c.CharacterId, mapId, 0)
							}
							return nil
						}
					}
				}
			}
		}
	}
}
