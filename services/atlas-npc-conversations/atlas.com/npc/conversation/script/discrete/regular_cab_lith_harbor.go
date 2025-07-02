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

// RegularCabLithHarbor (1002007) is located in Victoria Road - Lith Harbor (104000000)
type RegularCabLithHarbor struct {
}

func (r RegularCabLithHarbor) Name() string {
	return "RegularCabLithHarbor"
}

func (r RegularCabLithHarbor) Initial(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			// TODO
			//if character.HasItem(l, span)(c.CharacterId, item.TruTaxiCoupon) {
			//	return r.TruTaxiCoupon(l)(ctx)(c)
			//}
			return r.Hello(l)(ctx)(c)
		}
	}
}

func (r RegularCabLithHarbor) Hello(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("Hello, I drive the Regular Cab. If you want to go from town to town safely and fast, then ride our cab. We'll gladly take you to your destination with an affordable price.")
			return script.SendNextExit(l)(ctx)(c, m.String(), r.WhereToGo, r.MoreToSee)
		}
	}
}

func (r RegularCabLithHarbor) MoreToSee(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
	return func(ctx context.Context) func(c script.Context) script.State {
		return func(c script.Context) script.State {
			m := message.NewBuilder().
				AddText("There's a lot to see in this town, too. Come back and find us when you need to go to a different town.")
			return script.SendNext(l)(ctx)(c, m.String(), script.Exit())
		}
	}
}

func (r RegularCabLithHarbor) WhereToGo(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
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
				OpenItem(0).BlueText().ShowMap(_map.Henesys).CloseItem().NewLine().
				OpenItem(1).BlueText().ShowMap(_map.Perion).CloseItem().NewLine().
				OpenItem(2).BlueText().ShowMap(_map.Ellinia).CloseItem().NewLine().
				OpenItem(3).BlueText().ShowMap(_map.KerningCity).CloseItem().NewLine().
				OpenItem(4).BlueText().ShowMap(_map.Nautalis).CloseItem()
			return script.SendListSelectionExit(l)(ctx)(c, m.String(), r.SelectTownConfirm(beginner), r.MoreToSee)
		}
	}
}

func (r RegularCabLithHarbor) SelectTownConfirm(beginner bool) script.ProcessSelection {
	return func(selection int32) script.StateProducer {
		switch selection {
		case 0:
			return r.ConfirmHenesys(r.Cost(selection, beginner))
		case 1:
			return r.ConfirmPerion(r.Cost(selection, beginner))
		case 2:
			return r.ConfirmEllinia(r.Cost(selection, beginner))
		case 3:
			return r.ConfirmKerningCity(r.Cost(selection, beginner))
		case 4:
			return r.ConfirmNautalis(r.Cost(selection, beginner))
		}
		return nil
	}
}

func (r RegularCabLithHarbor) Cost(index int32, beginner bool) uint32 {
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

func (r RegularCabLithHarbor) ConfirmKerningCity(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.KerningCity, cost)
}

func (r RegularCabLithHarbor) ConfirmEllinia(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.Ellinia, cost)
}

func (r RegularCabLithHarbor) ConfirmPerion(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.Perion, cost)
}

func (r RegularCabLithHarbor) ConfirmHenesys(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.Henesys, cost)
}

func (r RegularCabLithHarbor) ConfirmNautalis(cost uint32) script.StateProducer {
	return r.ConfirmMap(_map.Nautalis, cost)
}

func (r RegularCabLithHarbor) ConfirmMap(mapId uint32, cost uint32) script.StateProducer {
	m := message.NewBuilder().
		AddText("You don't have anything else to do here, huh? Do you really want to go to ").
		BlueText().ShowMap(mapId).
		BlackText().AddText("? It'll cost you ").
		BlueText().AddText(fmt.Sprintf("%d mesos", cost))
	return func(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
		return func(ctx context.Context) func(c script.Context) script.State {
			return func(c script.Context) script.State {
				return script.SendYesNoExit(l)(ctx)(c, m.String(), r.PerformTransaction(mapId, cost), r.MoreToSee, r.MoreToSee)
			}
		}
	}
}

func (r RegularCabLithHarbor) PerformTransaction(mapId uint32, cost uint32) script.StateProducer {
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

//func (r RegularCabLithHarbor) TruTaxiCoupon(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
//	m := message.NewBuilder().
//		AddText("I see that you have a coupon to go to Henesys. One moment, I'll bring you there right over!")
//	return script.SendNext(l)(ctx)(c, m.String(), r.PerformTruTaxiTransaction)
//}
//
//func (r RegularCabLithHarbor) PerformTruTaxiTransaction(l logrus.FieldLogger) func(ctx context.Context) func(c script.Context) script.State {
//	character.GainItem(l, span)(c.CharacterId, item.TruTaxiCoupon, -1)
//	return script.WarpById(_map.Henesys, 0)(l)(ctx)(c)
//}
