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

// RegularCabEllinia (1032000) is located in Victoria Road - Ellinia (101000000)
type RegularCabEllinia struct {
}

func (r RegularCabEllinia) Name() string {
	return "RegularCabEllinia"
}

func (r RegularCabEllinia) Initial(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			return r.Hello(l)(ctx)(c)
		}
	}
}

func (r RegularCabEllinia) Hello(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Hello, I drive the Regular Cab. If you want to go from town to town safely and fast, then ride our cab. We'll gladly take you to your destination with an affordable price.")
			return script.SendNextExit(l)(ctx)(c, m.String(), r.WhereToGo, r.MoreToSee)
		}
	}
}

func (r RegularCabEllinia) MoreToSee(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("There's a lot to see in this town, too. Come back and find us when you need to go to a different town.")
			return script.SendNext(l)(ctx)(c, m.String(), script.Exit())
		}
	}
}

func (r RegularCabEllinia) WhereToGo(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder()
			beginner := character.IsBeginnerTree(l)(ctx)(c.CharacterId)

			if beginner {
				m = m.AddText("We have a special 90% discount for beginners. ")
			}
			m = m.
				AddText("Choose your destination, for fees will change from place to place.").
				BlueText().NewLine().
				OpenItem(0).BlueText().ShowMap(_map.LithHarbor).CloseItem().NewLine().
				OpenItem(1).BlueText().ShowMap(_map.Perion).CloseItem().NewLine().
				OpenItem(2).BlueText().ShowMap(_map.Henesys).CloseItem().NewLine().
				OpenItem(3).BlueText().ShowMap(_map.KerningCity).CloseItem().NewLine().
				OpenItem(4).BlueText().ShowMap(_map.Nautalis).CloseItem()
			return script.SendListSelectionExit(l)(ctx)(c, m.String(), r.SelectTownConfirm(beginner), r.MoreToSee)
		}
	}
}

func (r RegularCabEllinia) SelectTownConfirm(beginner bool) script.ProcessSelection {
	return func(selection int32) script.StateProducer {
		switch selection {
		case 0:
			return r.ConfirmLithHarbor(r.Cost(selection, beginner))
		case 1:
			return r.ConfirmPerion(r.Cost(selection, beginner))
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

func (r RegularCabEllinia) Cost(index int32, beginner bool) uint32 {
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

func (r RegularCabEllinia) ConfirmPerion(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.Perion, cost)
}

func (r RegularCabEllinia) ConfirmHenesys(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.Henesys, cost)
}

func (r RegularCabEllinia) ConfirmLithHarbor(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.LithHarbor, cost)
}

func (r RegularCabEllinia) ConfirmKerningCity(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.KerningCity, cost)
}

func (r RegularCabEllinia) ConfirmNautalis(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.Nautalis, cost)
}

func (r RegularCabEllinia) ConfirmMap(mapId uint32, cost uint32) script.StateProducer {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
		return func(ctx context.Context) func(c script.Context) script.State {
			return func(c script.Context) script.State {
				// TODO
				//if mapId == _map.Henesys && character.HasItem(l)(ctx)(c.CharacterId, item.NeinheartsTaxiCoupon) {
				//	return r.ConfirmNeinheart(l)(ctx)(c, mapId)
				//} else {
				return r.StandardConfirmMap(l)(ctx)(c, mapId, cost)
				//}
			}
		}
	}
}

func (r RegularCabEllinia) ConfirmNeinheart(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context, mapId uint32) script.State {
	return func(ctx context.Context) func(c script.Context, mapId uint32) script.State {
		return func(c script.Context, mapId uint32) script.State {
			m := message.NewBuilder().
				AddText("Hmm, I see you have been recommended by Neinheart to come to Victoria Island to improve your knightly skills. Well, just this time the ride will be free of charges. Will you take the ride?")
			return script.SendYesNoExit(l)(ctx)(c, m.String(), r.PerformNeinheartTransaction(mapId), r.MoreToSee, r.MoreToSee)
		}
	}
}

func (r RegularCabEllinia) StandardConfirmMap(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context, mapId uint32, cost uint32) script.State {
	return func(ctx context.Context) func(c script.Context, mapId uint32, cost uint32) script.State {
		return func(c script.Context, mapId uint32, cost uint32) script.State {
			m := message.NewBuilder().
				AddText("You don't have anything else to do here, huh? Do you really want to go to ").
				BlueText().ShowMap(mapId).
				BlackText().AddText("? It'll cost you ").
				BlueText().AddText(fmt.Sprintf("%d mesos", cost))
			return script.SendYesNoExit(l)(ctx)(c, m.String(), r.StartTransaction(mapId, cost), r.MoreToSee, r.MoreToSee)
		}
	}
}

func (r RegularCabEllinia) StartTransaction(mapId uint32, cost uint32) script.StateProducer {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
		return func(ctx context.Context) func(c script.Context) script.State {
			return func(c script.Context) script.State {
				if !character.HasMeso(l)(ctx)(c.CharacterId, cost) {
					m := message.NewBuilder().
						AddText("You don't have enough mesos. Sorry to say this, but without them, you won't be able to ride the cab.")
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

func (r RegularCabEllinia) PerformNeinheartTransaction(mapId uint32) script.StateProducer {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
		return func(ctx context.Context) func(c script.Context) script.State {
			return func(c script.Context) script.State {
				// TOOD
				//character.GainItem(l)(ctx)(c.CharacterId, item.NeinheartsTaxiCoupon, -1)
				//npc.WarpById(l)(ctx)(c.WorldId, c.ChannelId, c.CharacterId, mapId, 0)
				return nil
			}
		}
	}
}
