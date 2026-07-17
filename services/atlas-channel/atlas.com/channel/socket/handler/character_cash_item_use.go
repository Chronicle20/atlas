package handler

import (
	"atlas-channel/chalkboard"
	character2 "atlas-channel/character"
	"atlas-channel/consumable"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/shopscanner"
	"atlas-channel/socket/writer"
	"context"
	"math"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func CharacterCashItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := cashsb.ItemUse{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		// update_time is a leading header int32 (updateTimeFirst) from GMS v87
		// onward and on JMS v185; only the two oldest GMS builds (v83/v84) carry
		// it as a trailing int32 in the per-type sub-body. IDA-verified via
		// CWvsContext::SendConsumeCashItemUseRequest: gms_v87 @0xa9fef9 and
		// jms_v185 @0xaef2f5 both Encode4(update_time) in the header before the
		// sub-body switch (task-126). Must match ItemUse's header gate.
		updateTimeFirst := t.MajorVersion() >= 87
		updateTime := p.UpdateTime()
		source := slot.Position(p.Source())
		itemId := item.Id(p.ItemId())

		templateId, err := cashItemInSlotFunc(l, ctx, s.CharacterId(), int16(source))
		if err != nil || item.Id(templateId) != itemId {
			l.Warnf("Character [%d] attempted to use cash item [%d] in slot [%d], but item not found or mismatched.", s.CharacterId(), itemId, source)
			return
		}

		it := GetCashSlotItemType(t)(itemId)

		if it == CashSlotItemTypePetConsumable {
			sp := cashsb.NewItemUsePetConsumable(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			if !updateTimeFirst {
				updateTime = sp.UpdateTime()
			}
			_ = consumable.NewProcessor(l, ctx).RequestItemConsume(s.Field(), character.Id(s.CharacterId()), itemId, source, updateTime)
			return
		}
		if it == CashSlotItemTypeChalkboard {
			sp := cashsb.NewItemUseChalkboard(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			_ = chalkboard.NewProcessor(l, ctx).AttemptUse(s.Field(), s.CharacterId(), sp.Message())
			return
		}
		if it == CashSlotItemTypeFieldEffect {
			sp := cashsb.NewItemUseFieldEffect(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			message := sp.Message()

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
		if it == CashSlotItemTypeTeleportRock {
			// Enum 12 is shared: teleport rocks (classification 504) AND some
			// megaphones alias here (GetCashSlotItemType's ClassificationMegaphones
			// branch, otherCategory==1, above). Only the rocks route into the
			// use-flow here; aliased megaphones fall through to the warn-and-drop
			// below, unchanged.
			if item.GetClassification(itemId) == item.ClassificationTeleportRock {
				sp := cashsb.NewItemUseTeleportRock(updateTimeFirst)
				sp.Decode(l, ctx)(r, readerOptions)
				if !sp.Target().Valid() {
					l.Warnf("Character [%d] sent cash teleport-rock use without a target payload.", s.CharacterId())
					return
				}
				useRockFunc(l, ctx, wp)(s, itemId, sp.Target())
				return
			}
		}
		if it == CashSlotItemTypeVegasSpellPre95 || it == CashSlotItemTypeVegasSpell95 {
			sp := cashsb.ItemUseVegaScroll{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("[%s] read vega sub-body [%s]", p.Operation(), sp.String())
			enableActions := func() {
				_ = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
			}
			if !item.IsVegasSpell(itemId) {
				l.Warnf("Character [%d] attempted vega scroll with non-vega category-561 item [%d]. Rejecting.", s.CharacterId(), itemId)
				enableActions()
				return
			}
			if sp.EquipTab() != 1 || sp.ScrollTab() != 2 {
				l.Warnf("Character [%d] vega scroll with unexpected tab markers equip [%d] scroll [%d]. Impossible from a legit client. Rejecting.", s.CharacterId(), sp.EquipTab(), sp.ScrollTab())
				enableActions()
				return
			}
			_ = consumable.NewProcessor(l, ctx).RequestVegaScrollUse(s.Field(), character.Id(s.CharacterId()), itemId, source, slot.Position(sp.ScrollSlot()), slot.Position(sp.EquipSlot()))
			return
		}

		if it == CashSlotItemTypePointResetTier1 || it == CashSlotItemTypePointResetShared {
			sp := cashsb.NewItemUsePointReset(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			handlePointResetItemUse(l, ctx, wp)(s, itemId, *sp)
			return
		}

		if it == viciousHammerCashSlotItemType(t) {
			sp := cashsb.NewItemUseViciousHammer()
			sp.Decode(l, ctx)(r, readerOptions)
			handleViciousHammerOpen(l, ctx, wp)(s, source, *sp)
			return
		}

		if it == CashSlotItemTypeStoreSearch {
			sp := &cashsb.ItemUseStoreSearch{}
			sp.Decode(l, ctx)(r, readerOptions)
			_ = shopscanner.NewProcessor(l, ctx).Search(wp)(s, sp.SearchItemId(), sp.Descending(), itemId, source, sp.UpdateTime())
			return
		}

		l.Warnf("Character [%d] attempting to use cash item [%d] in slot [%d] of type [%d]. updateTime [%d].", s.CharacterId(), itemId, source, it, updateTime)
	}
}

type CashSlotItemType uint32

const (
	CashSlotItemTypeFieldEffect   = CashSlotItemType(16)
	CashSlotItemTypeStoreSearch   = CashSlotItemType(29)
	CashSlotItemTypePetConsumable = CashSlotItemType(30)
	CashSlotItemTypeChalkboard    = CashSlotItemType(32)
	// GetCashSlotItemType's ClassificationPointReset branch (above) routes by
	// itemId%10==1: AP Reset (5050000) and SP Reset tiers 2-4 (5050002-5050004)
	// collapse onto type 23, while SP Reset tier 1 (5050001) alone lands on
	// type 24. The type byte therefore CANNOT distinguish AP-vs-SP — the labels
	// below name only which numeric bucket each is. The arm matches on either
	// bucket and then dispatches by item id (design §2.4), never by this type.
	CashSlotItemTypePointResetShared = CashSlotItemType(23) // AP Reset + SP Reset tiers 2-4
	CashSlotItemTypePointResetTier1  = CashSlotItemType(24) // SP Reset tier 1 only
	CashSlotItemTypeVegasSpellPre95  = CashSlotItemType(68)
	CashSlotItemTypeVegasSpell95     = CashSlotItemType(71)
	CashSlotItemTypeViciousHammer    = CashSlotItemType(66) // GMS < 95
	CashSlotItemTypeViciousHammerV95 = CashSlotItemType(67) // GMS >= 95
	// CashSlotItemTypeTeleportRock (enum 12) is shared with some megaphones
	// (GetCashSlotItemType's ClassificationMegaphones branch, otherCategory==1)
	// — the handler gates on item.ClassificationTeleportRock (504) before
	// routing into the use-flow, so aliased megaphones are unaffected.
	CashSlotItemTypeTeleportRock = CashSlotItemType(12)
)

// cashItemInSlotFunc is a test seam for the cash-inventory ownership check
// (package-var injection precedent: itemInSlotFunc in teleport_rock_use.go).
// Returns the template id of the TypeValueCash item in the slot.
var cashItemInSlotFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32, slot int16) (uint32, error) {
	a, err := character2.NewProcessor(l, ctx).GetItemInSlot(characterId, inventory.TypeValueCash, slot)()
	if err != nil {
		return 0, err
	}
	return uint32(a.TemplateId()), nil
}

// viciousHammerCashSlotItemType returns the version-scoped CashSlotItemType
// for the Vicious Hammer item. Plain 66 also denotes CharacterCreation on
// GMS >= 95 (see the category == item.ClassificationCharacterCreation
// branch below), so this check must remain version-scoped.
func viciousHammerCashSlotItemType(t tenant.Model) CashSlotItemType {
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		return CashSlotItemTypeViciousHammerV95
	}
	return CashSlotItemTypeViciousHammer
}

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
			return CashSlotItemTypeStoreSearch
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
				return CashSlotItemTypeViciousHammerV95
			} else {
				return CashSlotItemTypeViciousHammer
			}
		}
		if category == item.ClassificationVegasSpell {
			if t.Region() == "GMS" && t.MajorVersion() >= 95 {
				return CashSlotItemTypeVegasSpell95
			}
			return CashSlotItemTypeVegasSpellPre95
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

// handleViciousHammerOpen performs the cheap pre-check (existence + cap) for
// the CUIItemUpgrade open-arm gauge. It never mutates state: it either arms
// the client gauge (mode OPEN) or rejects immediately (mode FAILURE, code 1
// or 2). WZ eligibility (codes 1/3 from equip data) is left to the
// authoritative re-validation in atlas-consumables on Packet B (design §4.1)
// — a gauge that later fails with mode 62 there is correct UX.
func handleViciousHammerOpen(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, hammerSlot slot.Position, sp cashsb.ItemUseViciousHammer) {
	return func(s session.Model, hammerSlot slot.Position, sp cashsb.ItemUseViciousHammer) {
		announce := func(body func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte) {
			err := session.Announce(l)(ctx)(wp)(fieldcb.ViciousHammerWriter)(body)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write vicious hammer response to character [%d].", s.CharacterId())
			}
		}

		equipSlot := int16(sp.SlotPosition())
		target, err := character2.NewProcessor(l, ctx).GetEquipableInSlot(s.CharacterId(), equipSlot)()
		if err != nil {
			l.Warnf("Character [%d] attempted vicious hammer on missing equip slot [%d].", s.CharacterId(), equipSlot)
			announce(fieldpkt.ViciousHammerFailureBody(fieldpkt.ViciousHammerReasonNotUpgradable))
			return
		}
		if target.HammersApplied() >= 2 {
			announce(fieldpkt.ViciousHammerFailureBody(fieldpkt.ViciousHammerReasonCapReached))
			return
		}

		token := packViciousHammerToken(int16(hammerSlot), equipSlot)
		// The client stores this open-arm count and renders the TERMINAL success
		// notice as "2 - count upgrades are left" (CUIItemUpgrade::OnItemUpgradeResult
		// success branch — the SUCCESS packet carries no count of its own). That
		// notice fires AFTER the reservation callback applies +1 to hammersApplied,
		// so we must send the post-apply count. HammersApplied() here is the
		// pre-apply value and the arm is only reached when it is < 2 (cap check
		// above), so +1 always yields the correct 1 or 2 (IDA-verified, task-129).
		announce(fieldpkt.ViciousHammerOpenBody(token, target.HammersApplied()+1))
	}
}
