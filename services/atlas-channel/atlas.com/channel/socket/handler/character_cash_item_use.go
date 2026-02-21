package handler

import (
	"atlas-channel/chalkboard"
	character2 "atlas-channel/character"
	"atlas-channel/consumable"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"math"
	"time"

	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const CharacterCashItemUseHandle = "CharacterCashItemUseHandle"

func CharacterCashItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		updateTimeFirst := t.Region() == "GMS" && t.MajorVersion() >= 95

		updateTime := uint32(0)
		if updateTimeFirst {
			updateTime = r.ReadUint32()
		}
		source := slot.Position(r.ReadInt16())
		itemId := item.Id(r.ReadUint32())

		a, err := character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), inventory.TypeValueCash, int16(source))()
		if err != nil || item.Id(a.TemplateId()) != itemId {
			l.Warnf("Character [%d] attempted to use cash item [%d] in slot [%d], but item not found or mismatched.", s.CharacterId(), itemId, source)
			return
		}

		it := GetCashSlotItemType(t)(itemId)

		if it == CashSlotItemTypePetConsumable {
			if !updateTimeFirst {
				updateTime = r.ReadUint32()
			}
			_ = consumable.NewProcessor(l, ctx).RequestItemConsume(s.Field(), character.Id(s.CharacterId()), itemId, source, updateTime)
			return
		}
		if it == CashSlotItemTypeChalkboard {
			message := r.ReadAsciiString()
			if !updateTimeFirst {
				updateTime = r.ReadUint32()
			}
			_ = chalkboard.NewProcessor(l, ctx).AttemptUse(s.Field(), s.CharacterId(), message)
			return
		}
		if it == CashSlotItemTypeFieldEffect {
			message := r.ReadAsciiString()
			if !updateTimeFirst {
				updateTime = r.ReadUint32()
			}
			_ = updateTime

			transactionId := uuid.New()
			now := time.Now()
			f := s.Field()
			steps := []saga.Step{
				{
					StepId: "consume_field_effect_item",
					Status: saga.Pending,
					Action: saga.DestroyAsset,
					Payload: saga.DestroyAssetPayload{
						CharacterId: s.CharacterId(),
						TemplateId:  uint32(itemId),
						Quantity:    1,
						RemoveAll:   false,
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					StepId: "show_field_effect_weather",
					Status: saga.Pending,
					Action: saga.FieldEffectWeather,
					Payload: saga.FieldEffectWeatherPayload{
						WorldId:   f.WorldId(),
						ChannelId: f.ChannelId(),
						MapId:     f.MapId(),
						Instance:  f.Instance(),
						ItemId:    uint32(itemId),
						Message:   message,
						Duration:  20,
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
			}
			_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
				TransactionId: transactionId,
				SagaType:      saga.FieldEffectUse,
				InitiatedBy:   "CASH_ITEM_USE",
				Steps:         steps,
			})
			return
		}

		// TODO for v83 there is a trailing updateTime.

		l.Warnf("Character [%d] attempting to use cash item [%d] in slot [%d] of type [%d]. updateTime [%d].", s.CharacterId(), itemId, source, it, updateTime)
	}
}

type CashSlotItemType uint32

const (
	CashSlotItemTypeFieldEffect   = CashSlotItemType(16)
	CashSlotItemTypePetConsumable = CashSlotItemType(30)
	CashSlotItemTypeChalkboard    = CashSlotItemType(32)
)

func GetCashSlotItemType(t tenant.Model) func(itemId item.Id) CashSlotItemType {
	return func(itemId item.Id) CashSlotItemType {
		category := item.GetClassification(itemId)
		if category == item.ClassificationPet {
			return CashSlotItemType(8)
		}
		if category == 501 {
			return CashSlotItemType(9)
		}
		if category == 502 {
			return CashSlotItemType(10)
		}
		if category == 503 {
			return CashSlotItemType(11)
		}
		if category == item.ClassificationTeleportRock {
			return CashSlotItemType(12)
		}
		if category == item.ClassificationPointReset {
			if itemId%10 == 1 {
				if (itemId%10 - 1) > 8 {
					return CashSlotItemType(0)
				}
				return CashSlotItemType(24)
			}
			return CashSlotItemType(23)
		}
		if category == item.ClassificationItemImprints {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				if uint32(math.Floor(float64(itemId)/1000)) == 5061 {
					return CashSlotItemType(65)
				}
				if uint32(math.Floor(float64(itemId)/1000)) == 5062 {
					return CashSlotItemType(74)
				}
			} else {
				if uint32(math.Floor(float64(itemId)/1000)) == 5061 {
					return CashSlotItemType(64)
				}
			}
			if itemId%10 == 0 {
				return CashSlotItemType(25)
			}
			if itemId%10 == 1 {
				return CashSlotItemType(26)
			}
			if itemId%10 == 2 {
				return CashSlotItemType(27)
			}
			if t.Region() == "GMS" && t.MajorVersion() >= 95 && itemId%10 == 3 {
				return CashSlotItemType(27)
			}
			return CashSlotItemType(0)
		}
		if category == item.ClassificationMegaphones {
			otherCategory := uint32(math.Floor(float64(itemId%10000) / float64(1000)))
			if otherCategory == 1 {
				return CashSlotItemType(12)
			}
			if otherCategory == 2 {
				return CashSlotItemType(13)
			}
			if otherCategory == 4 {
				if t.Region() == "GMS" && t.MajorVersion() >= 95 {
					return CashSlotItemType(45)
				}
			}
			if otherCategory == 5 {
				val := itemId % 10
				if t.Region() == "GMS" && t.MajorVersion() >= 95 {
					if val == 0 {
						return CashSlotItemType(47)
					}
					if val == 1 {
						return CashSlotItemType(48)
					}
					if val == 2 {
						return CashSlotItemType(49)
					}
					if val == 3 {
						return CashSlotItemType(50)
					}
					if val == 4 {
						return CashSlotItemType(51)
					}
					if val == 5 {
						return CashSlotItemType(52)
					}
					return CashSlotItemType(14)
				} else {
					if val == 0 {
						return CashSlotItemType(46)
					}
					if val == 1 {
						return CashSlotItemType(47)
					}
					if val == 2 {
						return CashSlotItemType(48)
					}
					if val == 3 {
						return CashSlotItemType(49)
					}
					if val == 4 {
						return CashSlotItemType(50)
					}
					if val != 5 {
						return CashSlotItemType(14)
					}
					return CashSlotItemType(51)
				}
			}
			if otherCategory == 6 {
				return CashSlotItemType(14)
			}
			if otherCategory == 7 {
				if t.Region() == "GMS" && t.MajorVersion() >= 95 {
					return CashSlotItemType(61)
				} else {
					return CashSlotItemType(60)
				}
			}
			if otherCategory == 8 {
				return CashSlotItemType(15)
			}
			return CashSlotItemType(0)
		}
		if category == item.ClassificationMessageBanner {
			return CashSlotItemType(18)
		}
		if category == item.ClassificationNote {
			return CashSlotItemType(21)
		}
		if category == item.ClassificationSongPlayer {
			return CashSlotItemType(20)
		}
		if category == item.ClassificationFieldEffect {
			return CashSlotItemTypeFieldEffect
		}
		if category == 513 {
			return CashSlotItemType(7)
		}
		if category == item.ClassificationStorePermit {
			return CashSlotItemType(4)
		}
		if category == item.ClassificationCosmeticCoupon {
			otherCategory := uint32(math.Floor(float64(itemId) / float64(1000)))
			if otherCategory == 5150 || otherCategory == 5151 || otherCategory == 5154 {
				return CashSlotItemType(1)
			}
			if otherCategory == 5152 {
				if uint32(math.Floor(float64(itemId)/100)) == 51520 {
					return CashSlotItemType(2)
				}
				if uint32(math.Floor(float64(itemId)/100)) == 51521 {
					return CashSlotItemType(35)
				}
				return CashSlotItemType(0)
			}
			if otherCategory == 5153 {
				return CashSlotItemType(3)
			}
			return CashSlotItemType(0)
		}
		if category == item.ClassificationExpression {
			return CashSlotItemType(6)
		}
		if category == item.ClassificationPetImprints {
			if 10000*itemId/10000 != itemId {
				return CashSlotItemType(0)
			}
			return CashSlotItemType(17)
		}
		if category == 518 {
			return CashSlotItemType(5)
		}
		if category == 519 {
			return CashSlotItemType(28)
		}
		if category == item.ClassificationCurrencySack {
			return CashSlotItemType(19)
		}
		if category == item.ClassificationGachaponCoupon {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(40)
			} else {
				return CashSlotItemType(39)
			}
		}
		if category == item.ClassificationStoreSearch {
			return CashSlotItemType(29)
		}
		if category == item.ClassificationPetConsumable {
			return CashSlotItemTypePetConsumable
		}
		if category == item.ClassificationWeddingTicket {
			if itemId%525100 != 100 {
				return CashSlotItemType(36)
			}
			return CashSlotItemType(37)
		}
		if category == 528 {
			if itemId/1000 == 5280 {
				return CashSlotItemType(33)
			}
			if itemId/1000 == 5281 {
				return CashSlotItemType(34)
			}
			return CashSlotItemType(0)
		}
		if category == item.ClassificationTransformationCoupon {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(41)
			} else {
				return CashSlotItemType(40)
			}
		}
		if category == item.ClassificationDueyCoupon {
			return CashSlotItemType(31)
		}
		if category == item.ClassificationChalkboard {
			return CashSlotItemTypeChalkboard
		}
		if category == item.ClassificationPetEvolution {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(42)
			} else {
				return CashSlotItemType(41)
			}
		}
		if category == item.ClassificationAvatarMegaphone {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(43)
			} else {
				return CashSlotItemType(42)
			}
		}
		if category == item.ClassificationCharacterImprints {
			if itemId/1000 == 5400 {
				if t.Region() == "GMS" && t.MajorVersion() >= 95 {
					return CashSlotItemType(53)
				} else {
					return CashSlotItemType(52)
				}
			}
			if itemId/1000 == 5401 {
				if t.Region() == "GMS" && t.MajorVersion() >= 95 {
					return CashSlotItemType(54)
				} else {
					return CashSlotItemType(53)
				}
			}
			if itemId/1000 == 5401 {
				if t.Region() == "GMS" && t.MajorVersion() >= 95 {
					return CashSlotItemType(54)
				} else {
					return CashSlotItemType(53)
				}
			}
			return CashSlotItemType(0)
		}
		if category == item.ClassificationCosmeticMembershipCoupon {
			if itemId/1000 == 5420 {
				if t.Region() == "GMS" && t.MajorVersion() >= 95 {
					return CashSlotItemType(55)
				} else {
					return CashSlotItemType(54)
				}
			}
			return CashSlotItemType(0)
		}
		if category == item.ClassificationCharacterCreation {
			if itemId/1000-5431 > 1 {
				if t.Region() == "GMS" && t.MajorVersion() >= 95 {
					return CashSlotItemType(58)
				} else {
					return CashSlotItemType(57)
				}
			}
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(66)
			} else {
				return CashSlotItemType(65)
			}
		}
		if category == item.ClassificationRemoteMerchant {
			if itemId/1000 != 5451 {
				if t.Region() == "GMS" && t.MajorVersion() >= 95 {
					return CashSlotItemType(38)
				} else {
					return CashSlotItemType(37)
				}
			}
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(60)
			} else {
				return CashSlotItemType(59)
			}
		}
		if category == item.ClassificationPetMultiConsumable {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(58)
			} else {
				return CashSlotItemType(57)
			}
		}
		if category == item.ClassificationRemoteStore {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(39)
			} else {
				return CashSlotItemType(38)
			}
		}
		if category == 549 {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(59)
			} else {
				return CashSlotItemType(58)
			}
		}
		if category == 550 {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(62)
			} else {
				return CashSlotItemType(61)
			}
		}
		if category == 551 {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(63)
			} else {
				return CashSlotItemType(62)
			}
		}
		if category == 552 {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(64)
			} else {
				return CashSlotItemType(63)
			}
		}
		if category == 553 {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(72)
			} else {
				return CashSlotItemType(69)
			}
		}
		if category == 557 {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(67)
			} else {
				return CashSlotItemType(66)
			}
		}
		if category == 561 {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(71)
			} else {
				return CashSlotItemType(68)
			}
		}
		if category == 562 {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(73)
			}
		}
		if category == 564 {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(77)
			}
		}
		if category == 566 {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemType(78)
			}
		}
		return CashSlotItemType(0)
	}
}
