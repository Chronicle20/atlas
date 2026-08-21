package handler

import (
	"atlas-channel/chalkboard"
	character2 "atlas-channel/character"
	"atlas-channel/consumable"
	cashData "atlas-channel/data/cash"
	equipmentData "atlas-channel/data/equipment"
	"atlas-channel/data/tradeability"
	"atlas-channel/incubator"
	"atlas-channel/kite"
	"atlas-channel/pendingchange"
	"atlas-channel/pet"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/shopscanner"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	af "github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	fieldpkt "github.com/Chronicle20/atlas/libs/atlas-packet/field"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	incubatorcb "github.com/Chronicle20/atlas/libs/atlas-packet/incubator/clientbound"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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
		updateTimeFirst := cashsb.UpdateTimeFirst(t)
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
			_ = consumable.NewProcessor(l, ctx).RequestItemConsume(s.Field(), character.Id(s.CharacterId()), itemId, source, 1, updateTime)
			return
		}
		if it == CashSlotItemTypePetSkill {
			// Case-28 sub-body carries a bare 8-byte petId and nothing else --
			// update_time is already decoded once from the common ItemUse header
			// above (jms_v185 IDA-verified, task-139 task-8/9; see
			// libs/atlas-packet/cash/serverbound/item_use_pet_skill.go). There is
			// no per-type trailing/leading updateTime to re-derive here.
			sp := cashsb.NewItemUsePetSkill()
			sp.Decode(l, ctx)(r, readerOptions)
			// The wire value is the pet's CLIENT serial
			// (GW_ItemSlotBase::liCashItemSN), not the Atlas pet id. Resolve it
			// here, at the socket boundary, exactly as the other serverbound pet
			// handlers do: atlas-consumables' ConsumePetSkillPouch keys on the
			// Atlas pet id (it calls pet.GetById and range-checks against
			// MaxUint32), so forwarding a 64-bit cash serial would fail every
			// pouch use on a cash-purchased pet with ErrPetCannotLearn.
			pm, perr := pet.NewProcessor(l, ctx).GetBySerialNumber(s.CharacterId(), sp.PetId())
			if perr != nil {
				l.WithError(perr).Debugf("Unable to resolve pet [%d] for character [%d] pet-skill pouch use.", sp.PetId(), s.CharacterId())
				return
			}
			_ = consumable.NewProcessor(l, ctx).RequestItemConsumeWithPet(s.Field(), character.Id(s.CharacterId()), itemId, source, updateTime, uint64(pm.Id()))
			return
		}
		if it == CashSlotItemTypeChalkboard {
			sp := cashsb.NewItemUseChalkboard(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			_ = chalkboard.NewProcessor(l, ctx).AttemptUse(s.Field(), s.CharacterId(), sp.Message())
			return
		}
		if it == CashSlotItemTypeKite {
			sp := cashsb.NewItemUseKite(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)

			// The sub-body is the message alone — the client sends no
			// coordinates for a kite (case-18 arm of
			// CWvsContext::SendConsumeCashItemUseRequest performs exactly one
			// EncodeStr). Position and owner name therefore come from
			// server-side character state, the same source
			// skill/handler/mysticdoor uses.
			c, err := character2.NewProcessor(l, ctx).GetById()(s.CharacterId())
			if err != nil {
				l.WithError(err).Debugf("Unable to resolve character [%d] for kite placement.", s.CharacterId())
				return
			}

			// No item is consumed (FR-4.1): no saga.DestroyAsset step and no
			// inventory mutation, so this is a direct command. Placement is
			// gated by the per-character cap in atlas-kites instead.
			//
			// No EnableActions either: the client's kite dialog is modal
			// (CDialog::DoModal @0x9ed0d9) and unlocks itself, and the sibling
			// chalkboard use arm sends none. Unlocking here would only widen
			// the client's duplicate-request gate.
			if err = kite.NewProcessor(l, ctx).AttemptUse(s.Field(), s.CharacterId(), c.Name(), uint32(itemId), sp.Message(), c.X(), c.Y()); err != nil {
				l.WithError(err).Debugf("Unable to request kite placement for character [%d].", s.CharacterId())
			}
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
		if it == CashSlotItemTypeSongPlayer {
			sp := cashsb.NewItemUseSongPlayer(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)

			// Unlike the pet-consumable/morph-coupon arms, this arm never
			// forwards updateTime anywhere: neither DestroyAssetPayload nor
			// PlayJukeboxPayload carries it, so sp.UpdateTime() is read only
			// to advance the reader past the trailing field on GMS <= v84 --
			// NewItemUseSongPlayer(updateTimeFirst) already does that inside
			// Decode regardless of whether the value is captured here.

			// The client sends the song's own length (IWzSound::length) and
			// resolves the BGM itself from the item's WZ info/path node
			// (CMapLoadable::PlayNextMusic). The server carries neither the BGM
			// name nor a duration constant -- only the client's value, which
			// atlas-maps caps.
			//
			// A zero length is a broken or spoofed client: the song would end
			// the instant it started. Reject before consuming anything.
			if sp.SoundLengthMs() == 0 {
				l.Warnf("Character [%d] sent song player [%d] use with a zero sound length; ignoring without consuming.", s.CharacterId(), itemId)
				return
			}

			// PlayJukebox carries the starting player's name, which is not on
			// the wire -- resolve it from server-side character state, the same
			// source the kite arm uses.
			c, cerr := character2.NewProcessor(l, ctx).GetById()(s.CharacterId())
			if cerr != nil {
				l.WithError(cerr).Debugf("Unable to resolve character [%d] for jukebox use.", s.CharacterId())
				return
			}

			// No EnableActions: the non-silent INVENTORY_OPERATION emitted by
			// the consume commit already clears the client's exclusive-request
			// lock -- the same reasoning recorded on the field-effect and
			// morph-coupon arms.
			transactionId := uuid.New()
			now := time.Now()
			f := s.Field()
			steps := []saga.Step{
				{
					StepId: "consume_song_player_item",
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
					StepId: "play_jukebox",
					Status: saga.Pending,
					Action: saga.PlayJukebox,
					Payload: saga.PlayJukeboxPayload{
						WorldId:    f.WorldId(),
						ChannelId:  f.ChannelId(),
						MapId:      f.MapId(),
						Instance:   f.Instance(),
						ItemId:     uint32(itemId),
						PlayerName: c.Name(),
						DurationMs: sp.SoundLengthMs(),
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
		if it == CashSlotItemTypeNote {
			sp := cashsb.NewItemUseNote(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			// Slot/template validation above proves ownership of the claimed
			// Note item; pre-flight receiver checks + the destroy-first saga
			// live in handleNoteSendRequest (note_send.go).
			handleNoteSendRequest(l, ctx, wp)(s, uint32(itemId), sp.ToName(), sp.Message())
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
		if it == CashSlotItemTypeItemTag {
			sp := cashsb.NewItemUseTargetSlot(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			targetSlot := sp.Slot()
			if targetSlot >= 0 {
				l.Warnf("Character [%d] attempted to use item tag [%d] on non-equipped slot [%d].", s.CharacterId(), itemId, targetSlot)
				return
			}
			target, err := character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), inventory.TypeValueEquip, targetSlot)()
			if err != nil {
				l.Warnf("Character [%d] attempted to use item tag [%d] on empty slot [%d].", s.CharacterId(), itemId, targetSlot)
				return
			}
			if tt, ok := inventory.TypeFromItemId(item.Id(target.TemplateId())); !ok || tt != inventory.TypeValueEquip {
				l.Warnf("Character [%d] attempted to use item tag [%d] on non-equip item [%d].", s.CharacterId(), itemId, target.TemplateId())
				return
			}
			c, err := character2.NewProcessor(l, ctx).GetById()(s.CharacterId())
			if err != nil {
				l.WithError(err).Warnf("Unable to resolve character [%d] name for item tag.", s.CharacterId())
				return
			}
			transactionId := uuid.New()
			now := time.Now()
			_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
				TransactionId: transactionId,
				SagaType:      saga.ItemTagUse,
				InitiatedBy:   "CASH_ITEM_USE",
				Steps: []saga.Step{
					{
						StepId: "consume_item_tag",
						Status: saga.Pending,
						Action: saga.DestroyAsset,
						Payload: saga.DestroyAssetPayload{
							CharacterId: s.CharacterId(),
							TemplateId:  uint32(itemId),
							Quantity:    1,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						StepId: "set_asset_owner",
						Status: saga.Pending,
						Action: saga.SetAssetOwner,
						Payload: saga.SetAssetOwnerPayload{
							CharacterId:   s.CharacterId(),
							InventoryType: byte(inventory.TypeValueEquip),
							Slot:          targetSlot,
							Owner:         c.Name(),
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			})
			return
		}
		sealTimed := sealTimedCashSlotItemType(t)
		if it == CashSlotItemTypeSeal || it == sealTimed {
			sp := cashsb.NewItemUseSeal(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			invType := inventory.Type(sp.InventoryType())
			targetSlot := int16(sp.Slot())
			if invType != inventory.TypeValueEquip {
				l.Warnf("Character [%d] attempted to use sealing lock [%d] on non-equip inventory [%d].", s.CharacterId(), itemId, invType)
				return
			}
			target, err := character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), invType, targetSlot)()
			if err != nil {
				l.Warnf("Character [%d] attempted to use sealing lock [%d] on empty slot [%d].", s.CharacterId(), itemId, targetSlot)
				return
			}
			if !target.Expiration().IsZero() && !target.Locked() {
				// A genuinely time-limited item must not be laundered into a permanent one.
				l.Warnf("Character [%d] attempted to seal time-limited item [%d] in slot [%d].", s.CharacterId(), target.TemplateId(), targetSlot)
				return
			}
			expiration := time.Time{}
			cd, err := cashData.NewProcessor(l, ctx).GetById(uint32(itemId))
			if err != nil {
				l.WithError(err).Warnf("Unable to resolve cash item data for sealing lock [%d].", itemId)
				return
			}
			if cd.ProtectTime > 0 {
				base := time.Now()
				if target.Locked() && !target.Expiration().IsZero() {
					base = target.Expiration()
				}
				expiration = base.AddDate(0, 0, int(cd.ProtectTime))
			}
			transactionId := uuid.New()
			now := time.Now()
			_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
				TransactionId: transactionId,
				SagaType:      saga.SealingLockUse,
				InitiatedBy:   "CASH_ITEM_USE",
				Steps: []saga.Step{
					{
						StepId: "consume_sealing_lock",
						Status: saga.Pending,
						Action: saga.DestroyAsset,
						Payload: saga.DestroyAssetPayload{
							CharacterId: s.CharacterId(),
							TemplateId:  uint32(itemId),
							Quantity:    1,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						StepId: "apply_asset_lock",
						Status: saga.Pending,
						Action: saga.ApplyAssetLock,
						Payload: saga.ApplyAssetLockPayload{
							CharacterId:   s.CharacterId(),
							InventoryType: byte(invType),
							Slot:          targetSlot,
							Expiration:    expiration,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			})
			return
		}
		if it == karmaScissorsCashSlotItemType(t) {
			sp := cashsb.NewItemUseKarmaScissors(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			invTypeRaw := sp.InventoryType()
			targetSlot := int16(sp.Slot())

			// The client takes an exclusive-request lock before sending
			// (gms_v83 @0x830FB5 gates on CanSendExclRequest(500, 0) and then
			// sets the lock), so EVERY outcome must unlock — a refusal that
			// returns silently wedges the client until the next unlocking
			// packet. The success path's non-silent INVENTORY_OPERATION, driven
			// by the UPDATED event, clears the lock on its own; only the
			// refusals need this.
			refuse := func(format string, args ...interface{}) {
				l.Warnf(format, args...)
				_ = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
			}

			// Gate 0b: the raw inventory-type int off the wire must be one of the
			// five known compartments. inventory.Type is a signed int8, so an
			// out-of-range value would otherwise address a nonexistent
			// compartment rather than fail.
			invType, ok := knownInventoryType(invTypeRaw)
			if !ok {
				refuse("Character [%d] attempted to use karma scissors [%d] against unknown inventory type [%d] slot [%d].", s.CharacterId(), itemId, invTypeRaw, targetSlot)
				return
			}
			// Gate 0d: a negative slot is an equipped item.
			if targetSlot < 0 {
				refuse("Character [%d] attempted to use karma scissors [%d] on equipped slot [%d] of inventory [%d].", s.CharacterId(), itemId, targetSlot, invType)
				return
			}
			// Gate 0e: the slot must be occupied.
			target, err := karmaCharacterProcessorFunc(l, ctx).GetItemInSlot(s.CharacterId(), invType, targetSlot)()
			if err != nil {
				refuse("Character [%d] attempted to use karma scissors [%d] on empty slot [%d] of inventory [%d].", s.CharacterId(), itemId, targetSlot, invType)
				return
			}
			// Gate 0c: pets carry karma on bit 0x01, which is FlagLock in
			// Atlas's shared flag column. See libs/atlas-constants/asset.KarmaFlagFor.
			karmaBit, ok := af.KarmaFlagFor(target.TemplateId())
			if !ok {
				refuse("Character [%d] attempted to use karma scissors [%d] on pet-class item [%d] in inventory [%d] slot [%d]; pets are not karma targets.", s.CharacterId(), itemId, target.TemplateId(), invType, targetSlot)
				return
			}
			// Gate 1: CUIKarmaDlg::PutItem's first refusal — IsProtectedItem.
			if target.Locked() {
				refuse("Character [%d] attempted to use karma scissors [%d] on sealing-locked item [%d] in inventory [%d] slot [%d].", s.CharacterId(), itemId, target.TemplateId(), invType, targetSlot)
				return
			}
			// Gate 2: the eligibility predicate. The scissors' own karma type
			// comes from ITS data, the target's from the target's — no literal
			// karma type appears anywhere, which is why 5520001 works the moment
			// a tenant's WZ carries it and is unusable when it does not.
			cd, err := karmaCashDataProcessorFunc(l, ctx).GetById(uint32(itemId))
			if err != nil {
				refuse("Character [%d] used karma scissors [%d] but its cash item data could not be read; refusing rather than assuming an untyped scissors. Target item [%d] in inventory [%d] slot [%d].", s.CharacterId(), itemId, target.TemplateId(), invType, targetSlot)
				return
			}
			td, err := karmaTradeabilityProcessorFunc(l, ctx).Get(invType, item.Id(target.TemplateId()))
			if err != nil {
				refuse("Character [%d] used karma scissors [%d] on item [%d] whose item data could not be read; refusing rather than assuming eligibility. Inventory [%d] slot [%d].", s.CharacterId(), itemId, target.TemplateId(), invType, targetSlot)
				return
			}
			if !af.KarmaEligible(cd.Karma, td.TradeAvailable()) {
				refuse("Character [%d] attempted to use karma scissors [%d] (karma type [%d]) on ineligible item [%d] (tradeAvailable [%d]) in inventory [%d] slot [%d].", s.CharacterId(), itemId, cd.Karma, target.TemplateId(), td.TradeAvailable(), invType, targetSlot)
				return
			}
			// Gate 3: IsPossibleTradingItem — the mark is already set.
			if af.HasFlag(target.Flag(), karmaBit) {
				refuse("Character [%d] attempted to use karma scissors [%d] on already-marked item [%d] in inventory [%d] slot [%d].", s.CharacterId(), itemId, target.TemplateId(), invType, targetSlot)
				return
			}
			// Gate 4: server-only. Karma exists to unlock an UNTRADEABLE item;
			// marking a tradeable one is a no-op that still consumes the
			// scissors. "Untradeable" is the same pair of conditions
			// atlas-trades enforces, so this gate and the trade-side override
			// are two readings of one definition and cannot disagree.
			if !af.HasFlag(target.Flag(), af.FlagUntradeable) && !af.HasFlag(target.Flag(), af.FlagMergeUntradeable) && !td.TradeBlock() {
				refuse("Character [%d] attempted to use karma scissors [%d] on already-tradeable item [%d] in inventory [%d] slot [%d].", s.CharacterId(), itemId, target.TemplateId(), invType, targetSlot)
				return
			}

			// Consume first, mark second: a failure to apply the mark then
			// compensates by restoring the scissors rather than leaving a free
			// trade behind.
			transactionId := uuid.New()
			now := time.Now()
			_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
				TransactionId: transactionId,
				SagaType:      saga.KarmaScissorsUse,
				InitiatedBy:   "CASH_ITEM_USE",
				Steps: []saga.Step{
					{
						StepId: "consume_karma_scissors",
						Status: saga.Pending,
						Action: saga.DestroyAsset,
						Payload: saga.DestroyAssetPayload{
							CharacterId: s.CharacterId(),
							TemplateId:  uint32(itemId),
							Quantity:    1,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						StepId: "apply_asset_karma",
						Status: saga.Pending,
						Action: saga.ApplyAssetKarma,
						Payload: saga.ApplyAssetKarmaPayload{
							CharacterId:   s.CharacterId(),
							InventoryType: byte(invType),
							Slot:          targetSlot,
							ScissorsKarma: cd.Karma,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			})
			return
		}
		if it == expirationExtenderCashSlotItemType(t) {
			// Sub-body: a bare int16 equip position, shared verbatim with the
			// Item Tag arm (the client uses one jump-table target for both --
			// gms_v83 SendConsumeCashItemUseRequest @0xA0CAE0, "cases 25,61").
			// No inventory type is on the wire: the client hard-codes EQUIP
			// (CharacterData::GetItem(charData, 1, -hitTestResult)), so the
			// compartment is EQUIP unconditionally and the slot is negative.
			sp := cashsb.NewItemUseTargetSlot(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			targetSlot := sp.Slot()

			// Every rejection below MUST unlock the client before returning.
			// CWvsContext::SendConsumeCashItemUseRequest is the sole caller of
			// SetExclRequestSent (gms_v83 @0xa0ea6f -> @0xa0ebbc), so the excl
			// lock is already armed by the time this arm runs; it mutates
			// nothing and does not warp, so only an explicit EnableActions
			// clears it, and the client has no timeout.
			target, err := character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), inventory.TypeValueEquip, targetSlot)()
			if err != nil {
				// Reachable only by double-clicking the extender in the CASH
				// tab: CDraggableItem::OnDoubleClicked (gms_v83 @0x4efd25)
				// falls through get_cashslot_item_type 61 into the
				// get_consume_cash_item_type allow-list (@0x4863d5) and sends
				// the request with a hard-coded nEPOS of 0 and an empty string
				// (@0x4f05a6), having asked the player for no target at all.
				// The supported flow is the drag-drop one --
				// CDraggableItem::ModifyEquipItem (@0x4f4bb7), which hit-tests
				// the target and runs the client's own confirm/reject dialogs
				// -- so tell the player that rather than leaving a dead click.
				l.Warnf("Character [%d] attempted to use expiration extender [%d] on empty equip slot [%d].", s.CharacterId(), itemId, targetSlot)
				_ = session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePopUpBody("Drag the item onto the equipment you want to extend."))(s)
				_ = enableActions(l)(ctx)(wp)(s)
				return
			}

			cd, err := cashData.NewProcessor(l, ctx).GetById(uint32(itemId))
			if err != nil {
				l.WithError(err).Warnf("Character [%d] unable to resolve cash item data for expiration extender [%d].", s.CharacterId(), itemId)
				_ = enableActions(l)(ctx)(wp)(s)
				return
			}

			ed, err := equipmentData.NewProcessor(l, ctx).GetById(target.TemplateId())
			if err != nil {
				l.WithError(err).Warnf("Character [%d] unable to resolve equipment data for extender target [%d] in slot [%d].", s.CharacterId(), target.TemplateId(), targetSlot)
				_ = enableActions(l)(ctx)(wp)(s)
				return
			}

			outcome := evaluateExpirationExtension(time.Now(), extensionTarget{
				Expiration: target.Expiration(),
				Locked:     target.Locked(),
				CashId:     target.CashId(),
				NotExtend:  ed.NotExtend(),
			}, cd.AddTime, cd.MaxDays)
			if outcome.Reason != "" {
				l.Warnf("Character [%d] expiration extender [%d] rejected on equip slot [%d] target [%d]: %s.", s.CharacterId(), itemId, targetSlot, target.TemplateId(), outcome.Reason)
				_ = enableActions(l)(ctx)(wp)(s)
				return
			}

			// No EnableActions on the accepted path: this arm mutates inventory
			// without warping, and the non-silent inventory change carries the
			// unlock itself, matching the sealing-lock and kite arms.
			transactionId := uuid.New()
			now := time.Now()
			_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
				TransactionId: transactionId,
				SagaType:      saga.ExpirationExtenderUse,
				InitiatedBy:   "CASH_ITEM_USE",
				Steps: []saga.Step{
					{
						StepId: "consume_expiration_extender",
						Status: saga.Pending,
						Action: saga.DestroyAsset,
						Payload: saga.DestroyAssetPayload{
							CharacterId: s.CharacterId(),
							TemplateId:  uint32(itemId),
							Quantity:    1,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						StepId: "extend_asset_expiration",
						Status: saga.Pending,
						Action: saga.ExtendAssetExpiration,
						Payload: saga.ExtendAssetExpirationPayload{
							CharacterId:        s.CharacterId(),
							InventoryType:      byte(inventory.TypeValueEquip),
							Slot:               targetSlot,
							Expiration:         outcome.Expiration,
							ExtenderTemplateId: uint32(itemId),
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			})
			return
		}
		if it == CashSlotItemTypeIncubator {
			sp := cashsb.NewItemUseIncubator(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			invType := inventory.Type(sp.InventoryType())
			targetSlot := int16(sp.Slot())
			announceFailure := func(egg uint32) {
				_ = session.Announce(l)(ctx)(wp)(incubatorcb.IncubatorResultWriter)(incubatorcb.NewIncubatorResult(0, 0, egg).Encode)(s)
			}
			target, err := character2.NewProcessor(l, ctx).GetItemInSlot(s.CharacterId(), invType, targetSlot)()
			if err != nil {
				l.Warnf("Character [%d] attempted to incubate empty slot [%d] of inventory [%d].", s.CharacterId(), targetSlot, invType)
				announceFailure(0)
				return
			}
			eggId := target.TemplateId()
			if !isPigmyEgg(item.Id(eggId)) {
				l.Warnf("Character [%d] attempted to incubate non-egg item [%d].", s.CharacterId(), eggId)
				announceFailure(0)
				return
			}
			// Gate before rolling/consuming: the client renders the incubation
			// result via a fixed NPC (incubator.SuccessNpcId) hard-coded in
			// OnIncubatorResult. GMS never shipped that NPC, so a successful
			// result would crash the client (CUtilDlgEx::SetNPC ->
			// STG_E_FILENOTFOUND). If the NPC is absent from the tenant's game
			// data, block gracefully — consume nothing, and tell the player with
			// an accurate popup rather than the client's misleading
			// "inventory is full" INCUBATOR_RESULT(0) message.
			if available, npcErr := incubator.NewProcessor(l, ctx).SuccessNpcAvailable(); npcErr != nil || !available {
				l.WithError(npcErr).Warnf("Character [%d] used incubator on egg [%d] but result NPC [%d] is absent from game data; blocking to avoid a client crash.", s.CharacterId(), eggId, incubator.SuccessNpcId)
				_ = session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePopUpBody("The incubator is currently unavailable."))(s)
				// The cash-item-use is an exclusive request; without a result the
				// client stays input-locked. Re-enable actions (empty StatChanged
				// with ExclRequestSent) as the Vega-scroll rejection arm does.
				_ = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
				return
			}
			reward, err := incubator.NewProcessor(l, ctx).SelectReward(eggId)
			if err != nil {
				l.WithError(err).Warnf("Character [%d] used incubator on egg [%d]; no reward selected.", s.CharacterId(), eggId)
				announceFailure(eggId)
				return
			}
			if _, ok := inventory.TypeFromItemId(item.Id(reward.ItemId())); !ok {
				l.Warnf("Incubator reward [%d] has no inventory type.", reward.ItemId())
				announceFailure(eggId)
				return
			}
			// Inventory capacity is enforced by the saga's award_reward (AwardAsset)
			// step: if the target inventory is full the award fails and the saga
			// compensates, re-creating the consumed egg + incubator (compensator
			// reverse-walk DestroyAsset*/->CreateItem). This mirrors the gachapon
			// reward flow, which likewise has no channel-side capacity pre-check.
			// The former pre-check here was broken — it fetched the compartment
			// without its assets (so len(Assets) was always 0 and it never detected
			// fullness) and reported any GetByType REST error as "inventory full".
			f := s.Field()
			transactionId := uuid.New()
			now := time.Now()
			_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
				TransactionId: transactionId,
				SagaType:      saga.IncubatorUse,
				InitiatedBy:   "CASH_ITEM_USE",
				Steps: []saga.Step{
					{
						StepId: "consume_sacrifice",
						Status: saga.Pending,
						Action: saga.DestroyAssetFromSlot,
						Payload: saga.DestroyAssetFromSlotPayload{
							CharacterId:   s.CharacterId(),
							InventoryType: byte(invType),
							Slot:          targetSlot,
							Quantity:      1,
							TemplateId:    eggId,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						StepId: "consume_incubator",
						Status: saga.Pending,
						Action: saga.DestroyAsset,
						Payload: saga.DestroyAssetPayload{
							CharacterId: s.CharacterId(),
							TemplateId:  uint32(itemId),
							Quantity:    1,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						StepId: "award_reward",
						Status: saga.Pending,
						Action: saga.AwardAsset,
						Payload: saga.AwardAssetPayload{
							CharacterId: s.CharacterId(),
							Item: saga.ItemPayload{
								TemplateId: reward.ItemId(),
								Quantity:   reward.Quantity(),
							},
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						StepId: "announce_result",
						Status: saga.Pending,
						Action: saga.IncubatorResult,
						Payload: saga.IncubatorResultPayload{
							CharacterId: s.CharacterId(),
							WorldId:     f.WorldId(),
							ChannelId:   f.ChannelId(),
							ItemId:      reward.ItemId(),
							Count:       reward.Quantity(),
							EggId:       eggId,
						},
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			})
			return
		}
		if it == CashSlotItemTypeCube {
			// 5062xxx (GMS >= 95) is the Miracle Cube / potential re-roll family — a
			// separate feature, deliberately not part of task-128 (design.md §11).
			l.Warnf("Character [%d] attempted to use cube-family item [%d]; not implemented.", s.CharacterId(), itemId)
			return
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

		if it == CashSlotItemTypePetNameTag {
			sp := cashsb.NewItemUsePetNameTag(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			handlePetNameTagUse(l, ctx, wp)(s, itemId, sp.Name())
			return
		}

		if it == CashSlotItemTypeCurrencySack {
			// No sub-body: the classification-520 arm of
			// CWvsContext::SendConsumeCashItemUseRequest encodes nothing beyond
			// the common header on all ten versions (design §3, per-version
			// addresses). Nothing to decode off r.
			handleMesoSackUse(l, ctx, wp)(s, itemId)
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

		// No sub-body: CWvsContext::SendConsumeCashItemUseRequest's cases 52/53
		// (gms_v83 @0xa0b1b4/@0xa0b294) and 53/54 (gms_v95 @0x9ec299/@0x9ec384)
		// carry no name and no target world -- only an optional, hard-coded
		// confirmation byte (Encode1), which is the LAST field on the wire and
		// carries no domain data, so there is nothing here worth decoding off
		// r. Two facts settle that, both derived by direct disassembly on v83
		// AND v95 (docs/tasks/task-227-cash-name-change-world-transfer/
		// cancel-confirm-semantics.md): (1) dismissing either of the two
		// chained CUICancelCharacterCouponRequests::DoModal dialogs sends NO
		// packet at all -- the byte and the SendPacket call are gated
		// together, so a received packet of this arm IS necessarily the
		// double-confirmed cancel; there is no tail-less variant on the wire
		// to distinguish. (2) the byte's value is always 1 on both versions
		// (v95 encodes a literal 1; v83 sets edi=1 once at function entry via
		// push 1/pop edi and never reassigns it before either call site) --
		// not "0 or 1 depending on which dialog," and not version-divergent.
		//
		// This arm is therefore the client's cancel entry point (task-227
		// client-cancel addendum, see cancel-entry-point.md): using a
		// name-change or world-transfer coupon that already has a pending
		// change outstanding is how the client asks to cancel it. The
		// character id comes from the session, never the client -- ownership
		// holds by construction on the atlas-character side too
		// (pending_change.CancelForCharacterAndType).
		if it == nameChangeCashSlotItemType(t) {
			handleCashCouponCancel(l, ctx, wp)(s, itemId, pendingchange.TypeNameChange)
			return
		}
		if it == worldTransferCashSlotItemType(t) {
			handleCashCouponCancel(l, ctx, wp)(s, itemId, pendingchange.TypeWorldTransfer)
			return
		}

		// Classification-FIRST dispatch (design §1.1): cash-slot type 12
		// collides with teleport rock (task-124), type 42 with pet evolution,
		// so megaphone/avatar-megaphone routing must branch on classification
		// before any cash-slot-type sub-switch, never the other way around.
		category := item.GetClassification(itemId)
		// Classification-FIRST, same reason as the megaphone branch below: the
		// cash-slot type byte collides (37 is also the wedding-ticket bucket,
		// 59/60 are also triple-megaphone buckets — GetCashSlotItemType).
		if category == item.ClassificationRemoteMerchant {
			handleRemoteMerchantUse(l, ctx, wp)(s, t, itemId, source, it)
			return
		}
		if category == item.ClassificationMegaphones || category == item.ClassificationAvatarMegaphone {
			// Legacy GMS (v48/61/72/79, MajorVersion < 83) item-loss guard.
			// task-123 legacy-phase-1 (.superpowers/sdd/legacy-megaphone-protocol.md)
			// IDA-verified the following per-tier matrix for these four builds:
			//   - basic (tier 1) / super (tier 2) megaphone: serverbound codec AND
			//     clientbound WorldMessage MEGAPHONE(2)/SUPER_MEGAPHONE(3) arms
			//     verified (spec §2/§3) — legacy-phase-2 wired the writer/handler
			//     opcodes into template_gms_{48,61,72,79}_1.json, so these two
			//     tiers now render on legacy clients. ALLOWED.
			//   - Cheap (tier 0) / Heart (tier 3): task-123 cheap-heart-skull-report
			//     found NEITHER v83 nor v95's get_cashslot_item_type has an arm for
			//     these tiers (both fall to the default -> the client's
			//     SendConsumeCashItemUseRequest dispatcher sends NO sub-body at all
			//     for them — verified with addresses on v83/v95, not independently
			//     re-verified per-build on v48/61/72/79). There is therefore no wire
			//     evidence a legacy client ever emits this op for these item ids.
			//     ALLOWED anyway per explicit user-confirmed scope (reuses the
			//     basic(0)/super(3) codec+scope exactly like v83+), since the
			//     alternative — dropping the item silently — is worse and the
			//     decode path is a no-op if no packet ever arrives.
			//   - Skull (tier 4): v83 also has no get_cashslot_item_type arm
			//     (falls to the same default as 0/3) — genuinely no send path <v95
			//     (GMS). ALLOWED here as the super-megaphone shape (same "no
			//     confirmed wire event on legacy" caveat as 0/3 above). Skull is
			//     NEVER Maple TV on any version — the cheap-heart-skull finalize
			//     pass (task-123, see character_cash_item_use_megaphone.go case 4)
			//     removed the earlier (incorrect) GMS>=95 -> handleMapleTVUse
			//     routing entirely; handleMegaphoneUse's case 4 now always decodes
			//     the super shape, so this legacy branch inherits that
			//     unconditionally.
			//   - avatar megaphone: no legacy build's serverbound send case could be
			//     reliably located (spec §5a — cash-slot type 42 does not match the
			//     known 4-line+whisper body on any of the four builds); consuming
			//     the item would destroy it with no verified broadcast to render.
			//     BLOCKED on legacy regardless of tier.
			//   - Item megaphone (tier 6) / triple megaphone (tier 7): legacy
			//     TV/item/triple gap-fill pass (task-123) IDA-verified the wire
			//     shape by SHAPE-MATCHING inside each legacy build's
			//     SendConsumeCashItemUseRequest (byte-fixtures in
			//     libs/atlas-packet/cash/serverbound/v{61,72,79}_test.go):
			//       - v61: Item megaphone (case 14 @0x832e37, dedicated dialog
			//         send fn sub_55DC01) ALLOWED. Triple megaphone (5077000)
			//         does not exist as an item on v61
			//         (megaphone-item-availability.md: v72+) — no case found
			//         either; BLOCKED (tier absent, not merely unverified).
			//       - v72/v79: Item megaphone (v72 case 14 @0x905609 send fn
			//         sub_5A7C42 @0x5a7c42; v79 case 14 @0x956975 send fn
			//         sub_5C2336 @0x5c2336) ALLOWED. Triple megaphone (case 60
			//         @0x905090/0x9563f8, self-contained: count+lines+whisper)
			//         ALLOWED.
			//     Both arms funnel into the SAME shared rate-check-and-send tail
			//     as basic/super Megaphone (a `call SetExclRequestSent; Encode4;
			//     SendPacket` epilogue) — update_time IS present (trailing
			//     uint32) on every arm, matching each codec's existing
			//     `if !updateTimeFirst { WriteInt(updateTime) }` unconditional
			//     write (no extra gate needed; this pass also found and fixed a
			//     bug where Megaphone/SuperMegaphone WRONGLY omitted this same
			//     trailing field on legacy — see item_use_megaphone.go). Both
			//     tiers broadcast through the WorldMessage writer, which already
			//     carries ITEM_MEGAPHONE(8)/MULTI_MEGAPHONE(10) operations
			//     entries on every legacy template — no new writer wiring
			//     needed.
			//   - Maple TV (tier 5): SERVERBOUND wire IS verified on v61/72/79
			//     (case 45/46 @0x834d1c/0x907702/0x958b2a, first of 6
			//     consecutive tvType cases, self-contained:
			//     pad+receiverName+5 lines, same shared send-tail as above) —
			//     but STAYS BLOCKED anyway: handleMapleTVUse's CLIENTBOUND acks
			//     (TvSendMessageResult/TvSetMessage/TvClearMessage) have ZERO
			//     writer entries in ANY of the four legacy templates (confirmed:
			//     `grep '"writer": "Tv'` template_gms_{48,61,72,79}_1.json is
			//     empty, vs v83+ which all carry them). Enabling tier 5 would
			//     still DESTROY the item (the saga's consume step runs before
			//     any ack) while every TV response packet fails to resolve an
			//     opcode — a real item-loss-equivalent regression the
			//     serverbound-only verification bar doesn't catch. Wiring the
			//     TV writer family into the legacy templates (its own IDA pass
			//     on the clientbound side) is a separate follow-up; NOT done
			//     this pass.
			//     v48: NOT reverified this pass (out of scope) — v48 keeps the
			//     tier>4 block below regardless.
			//
			// jms185 verification (task-123 cheap-heart-skull finalize pass): unlike
			// v83/v95, JMS's get_cashslot_item_type@0x49a1ee genuinely sends
			// Cheap(tier0->type12)/Heart(tier3->type47)/Skull(tier4->type48) —
			// confirmed via CWvsContext::SendConsumeCashItemUseRequest@0xaef2f5,
			// shared arm @0xaef5b9. Cheap encodes message-only (matches case 0's
			// basic/channel routing); Heart/Skull encode message+whisper (matches
			// case 3/4's super/world routing). Byte-fixtures pinned in
			// libs/atlas-packet/cash/serverbound/{item_use_megaphone,
			// item_use_super_megaphone}_test.go. jms185 is MajorVersion 185 (>=83)
			// so it never enters this legacy branch; noted here because it means the
			// "no confirmed send on GMS<95" reasoning above does NOT generalize to
			// JMS, where these tiers are real, verified sends.
			//
			// v83+/JMS behavior is otherwise unchanged: the MajorVersion < 83 branch
			// below is never entered for them, so every tier keeps dispatching
			// exactly as it did before this gate was refined.
			if t.MajorVersion() < 83 {
				if category == item.ClassificationAvatarMegaphone {
					l.Warnf("Character [%d] attempted avatar megaphone item [%d] on unsupported legacy version [major %d]; ignoring without consuming.", s.CharacterId(), itemId, t.MajorVersion())
					return
				}
				// ClassificationMegaphones per-(version,tier) legacy allow-list.
				// Verified matrix (task-123 legacy TV/item/triple gap-fill):
				//   v72/v79: tiers 0-4,6,7 (item+triple verified+wired; tier 5/TV
				//            serverbound-verified but blocked — see the TV
				//            writer-wiring gap documented above).
				//   v61:     tiers 0-4,6 (item verified+wired; triple item
				//            absent; tier 5/TV blocked for the same reason).
				//   v48/others: tiers 0-4 only (6/7 never reverified; 5 blocked).
				tier := (uint32(itemId) / 1000) % 10
				allowed := tier <= 4
				if t.Region() == "GMS" {
					switch t.MajorVersion() {
					case 72, 79:
						allowed = allowed || tier == 6 || tier == 7
					case 61:
						allowed = allowed || tier == 6
					}
				}
				if !allowed {
					l.Warnf("Character [%d] attempted megaphone item [%d] tier [%d] unsupported on legacy version [major %d]; ignoring without consuming.", s.CharacterId(), itemId, tier, t.MajorVersion())
					return
				}
			}
			if category == item.ClassificationMegaphones {
				handleMegaphoneUse(l, ctx, wp)(s, r, readerOptions, t, itemId, source, updateTimeFirst)
				return
			}
			handleAvatarMegaphoneUse(l, ctx, wp)(s, r, readerOptions, t, itemId, source, updateTimeFirst)
			return
		}

		// Transformation (morph) coupons, classification 530. Gated on
		// CLASSIFICATION, never on the cash-slot type byte `it`: those bytes
		// collide across versions (GetCashSlotItemType maps 530 -> 41 on
		// GMS >= 95 and 40 otherwise, while 522 gachapon takes 40 on GMS >= 95
		// and 538 pet evolution takes 41 on GMS < 95), so a type-keyed arm
		// would change meaning at a version bump.
		//
		// The sub-body is empty apart from the trailing updateTime on the
		// versions that trail it (IDA-verified: the case-40 arm of
		// CWvsContext::SendConsumeCashItemUseRequest @0xa0caf0-0xa0cb37 on GMS
		// v83 contains no Encode* call).
		//
		// No EnableActions: the effect does not warp, and the non-silent
		// INVENTORY_OPERATION emitted by the consume commit already clears the
		// client's exclusive-request lock — CWvsContext::OnInventoryOperation
		// @0xa1ead9 clears the same dword pair OnGameStageChanged does, gated
		// on the packet's leading bOnExclRequest byte, which
		// inventory/clientbound/change_batch.go writes as !silent.
		if category == item.ClassificationTransformationCoupon {
			sp := cashsb.NewItemUseMorphCoupon(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			if !updateTimeFirst {
				updateTime = sp.UpdateTime()
			}
			_ = requestItemConsumeFunc(l, ctx, s.Field(), character.Id(s.CharacterId()), itemId, source, 1, updateTime)
			return
		}

		l.Warnf("Character [%d] attempting to use cash item [%d] in slot [%d] of type [%d]. updateTime [%d].", s.CharacterId(), itemId, source, it, updateTime)
	}
}

type CashSlotItemType uint32

const (
	CashSlotItemTypeFieldEffect      = CashSlotItemType(16)
	CashSlotItemTypeNote             = CashSlotItemType(21)
	CashSlotItemTypeStoreSearch      = CashSlotItemType(29)
	CashSlotItemTypePetConsumable    = CashSlotItemType(30)
	CashSlotItemTypePetSkill         = CashSlotItemType(28)
	CashSlotItemTypeChalkboard       = CashSlotItemType(32)
	CashSlotItemTypeKite             = CashSlotItemType(18)
	CashSlotItemTypeItemTag          = CashSlotItemType(25)
	CashSlotItemTypeSeal             = CashSlotItemType(26)
	CashSlotItemTypeIncubator        = CashSlotItemType(27)
	CashSlotItemTypeSealTimed        = CashSlotItemType(64)
	CashSlotItemTypeSealTimedV95     = CashSlotItemType(65)
	CashSlotItemTypeKarmaScissors    = CashSlotItemType(63) // GMS < 95, and JMS
	CashSlotItemTypeKarmaScissorsV95 = CashSlotItemType(64) // GMS >= 95
	CashSlotItemTypeCube             = CashSlotItemType(74)
	// CashSlotItemTypeCurrencySack is classification 520 (meso sacks). Atlas
	// returns 19 on EVERY version even though the v48 client's own table says
	// 17 and v61's says 18: the type is derived from the server-resolved
	// template id and never rides the wire, and no other classification maps to
	// 19 here. Do NOT version-gate this (design §3.1(a)).
	CashSlotItemTypeCurrencySack = CashSlotItemType(19)

	// GetCashSlotItemType's ClassificationPointReset branch (above) routes by
	// itemId%10==1: AP Reset (5050000) and SP Reset tiers 2-4 (5050002-5050004)
	// collapse onto type 23, while SP Reset tier 1 (5050001) alone lands on
	// type 24. The type byte therefore CANNOT distinguish AP-vs-SP — the labels
	// below name only which numeric bucket each is. The arm matches on either
	// bucket and then dispatches by item id (design §2.4), never by this type.
	CashSlotItemTypePointResetShared      = CashSlotItemType(23) // AP Reset + SP Reset tiers 2-4
	CashSlotItemTypePointResetTier1       = CashSlotItemType(24) // SP Reset tier 1 only
	CashSlotItemTypeVegasSpellPre95       = CashSlotItemType(68)
	CashSlotItemTypeVegasSpell95          = CashSlotItemType(71)
	CashSlotItemTypeViciousHammer         = CashSlotItemType(66) // GMS < 95
	CashSlotItemTypeViciousHammerV95      = CashSlotItemType(67) // GMS >= 95
	CashSlotItemTypeExpirationExtender    = CashSlotItemType(61) // GMS < 95, JMS
	CashSlotItemTypeExpirationExtenderV95 = CashSlotItemType(62) // GMS >= 95
	// CashSlotItemTypeTeleportRock (enum 12) is shared with some megaphones
	// (GetCashSlotItemType's ClassificationMegaphones branch, otherCategory==1)
	// — the handler gates on item.ClassificationTeleportRock (504) before
	// routing into the use-flow, so aliased megaphones are unaffected.
	CashSlotItemTypeTeleportRock = CashSlotItemType(12)

	// CashSlotItemTypePetNameTag is classification 517 (Pet Name Tag, 5170000).
	// No other classification maps to 17 in GetCashSlotItemType — meso sacks
	// return 19 on every version by deliberate Atlas policy (see
	// CashSlotItemTypeCurrencySack above) even though the v48 client's own
	// table says 17 — so gating the arm on `it` alone is unambiguous.
	CashSlotItemTypePetNameTag = CashSlotItemType(17)

	// CashSlotItemTypeSongPlayer is classification 510 (jukebox). Unlike the
	// 530 morph coupon, which had to be classification-keyed because its type
	// byte moves across versions, 20 is stable: get_cashslot_item_type maps
	// 510 -> 20 on every version examined (GMS v95.0 @0x488c70, GMS v83) and
	// no other classification yields 20 in that function.
	CashSlotItemTypeSongPlayer = CashSlotItemType(20)
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

// requestItemConsumeFunc is a test seam over the atlas-consumables consume
// command emit (package-var injection precedent: cashItemInSlotFunc above,
// useRockFunc in teleport_rock_use.go). Handler tests must not require a live
// Kafka broker to assert which arm a request reached.
var requestItemConsumeFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32) error {
	return consumable.NewProcessor(l, ctx).RequestItemConsume(f, characterId, itemId, source, quantity, updateTime)
}

// karmaCharacterProcessorFunc is a test seam for the karma arm's target-item
// lookup (package-var injection precedent: cashItemInSlotFunc above). Unlike
// that seam, which resolves only a template id, the karma arm's gates
// 0c/1/3/4 need the full target asset (locked, flag, template id), so this
// seam exposes the whole character2.Processor — tests substitute
// character/mock.MockProcessor's GetItemInSlotFunc.
var karmaCharacterProcessorFunc = func(l logrus.FieldLogger, ctx context.Context) character2.Processor {
	return character2.NewProcessor(l, ctx)
}

// karmaCashDataProcessorFunc is a test seam for the karma arm's scissors
// cash-item-data lookup (Task 10). Tests substitute data/cash/mock's
// ProcessorMock.
var karmaCashDataProcessorFunc = func(l logrus.FieldLogger, ctx context.Context) cashData.Processor {
	return cashData.NewProcessor(l, ctx)
}

// karmaTradeabilityProcessorFunc is a test seam for the karma arm's target
// tradeability lookup (Task 10). Tests substitute data/tradeability/mock's
// ProcessorMock — see that package's doc comment on why GetFunc must always
// be set explicitly in a karma test.
var karmaTradeabilityProcessorFunc = func(l logrus.FieldLogger, ctx context.Context) tradeability.Processor {
	return tradeability.NewProcessor(l, ctx)
}

// knownInventoryType decodes the raw inventory-type int off the wire into a
// shared inventory.Type, reporting false for anything that is not one of the
// five compartments. inventory.Type is a SIGNED int8, so an out-of-range value
// would silently address a nonexistent compartment if merely converted — a
// crafted packet must be a refusal, not a panic or a wrong-compartment read.
// Mirrors atlas-trades' stageableInventoryType.
func knownInventoryType(raw int32) (inventory.Type, bool) {
	if raw < 0 || raw > math.MaxInt8 {
		return 0, false
	}
	t := inventory.Type(raw)
	for _, known := range inventory.Types {
		if t == known {
			return t, true
		}
	}
	return 0, false
}

const (
	pigmyEggMinId item.Id = 4170000
	pigmyEggMaxId item.Id = 4170009
)

// isPigmyEgg reports whether templateId is an incubatable Pigmy Egg (the client
// enforces this; the server re-checks so a crafted request can't sacrifice
// arbitrary items).
func isPigmyEgg(templateId item.Id) bool {
	return templateId >= pigmyEggMinId && templateId <= pigmyEggMaxId
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

// nameChangeCashSlotItemType returns the version-scoped CashSlotItemType for
// the name-change coupon (item id prefix 5400). task-227 derivation.md §3
// settles the prefix->flow assignment from the client's own ProcessBuy/
// get_cashslot_item_type arms: 5400000 is name change on every GMS version
// v48-v95 (v83 -> 52, v95 -> 53). Do not reorder against
// worldTransferCashSlotItemType without re-reading §3 -- jms_v185 has no
// 5400000 at all (§1.5), so this value is never produced there in practice,
// but the helper still returns a value distinct from the world-transfer one.
func nameChangeCashSlotItemType(t tenant.Model) CashSlotItemType {
	if t.IsRegion("GMS") && t.MajorAtLeast(95) {
		return CashSlotItemType(53)
	}
	return CashSlotItemType(52)
}

// worldTransferCashSlotItemType returns the version-scoped CashSlotItemType
// for the world-transfer coupon (item id prefix 5401). task-227
// derivation.md §3: 5401000 is world transfer on every GMS version v48-v95
// (v83 -> 53, v95 -> 54) and on jms_v185, which maps 5401000 to this flow
// despite lacking a name-change item at all (§1.5).
func worldTransferCashSlotItemType(t tenant.Model) CashSlotItemType {
	if t.IsRegion("GMS") && t.MajorAtLeast(95) {
		return CashSlotItemType(54)
	}
	return CashSlotItemType(53)
}

// handleCashCouponCancel is the server side of the coupon item-use cancel
// arm (task-227 client-cancel addendum, see the case-52/53 comment above).
// It cancels the calling character's own pending record of changeType via
// atlas-character's self-scoped cancel route -- the character id comes from
// the session, never the client. It unlocks the client (enableActions) on
// EVERY path: success, "nothing pending" (404, not an error for the
// player), and infrastructure failure alike. Dropping enableActions on any
// one of those leaves the client permanently locked (FR-5.3), which is the
// whole reason this arm existed before this task.
//
// No clientbound CANCEL_* is emitted from here: the server's reply is the
// PENDING_CHANGE_RESOLVED event atlas-character emits on a successful
// cancel, which a separate consumer (task-227 Task 27) turns into
// CancelNameChangeResult / CancelTransferWorldResult. Emitting from this
// handler too would double-send.
func handleCashCouponCancel(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id, changeType string) {
	return func(s session.Model, itemId item.Id, changeType string) {
		characterId := s.CharacterId()
		_, err := pendingchange.NewProcessor(l, ctx).CancelPendingChange(characterId, changeType)
		if err != nil {
			var re *pendingchange.RejectedError
			if errors.As(err, &re) && re.Status == http.StatusNotFound {
				// A normal race against the sweeper or an operator cancel,
				// not a failure -- there was simply nothing pending to cancel.
				l.Debugf("Character [%d] used coupon [%d] to cancel a pending [%s] change, but nothing was pending.", characterId, itemId, changeType)
			} else {
				l.WithError(err).Warnf("Character [%d] failed to cancel pending [%s] change via coupon [%d].", characterId, changeType, itemId)
			}
			_ = enableActions(l)(ctx)(wp)(s)
			return
		}
		_ = enableActions(l)(ctx)(wp)(s)
	}
}

// sealTimedCashSlotItemType returns the version-scoped CashSlotItemType for the
// Sealing Lock (timed). Extracted from the handler arm so that
// karma_slot_type_test.go's disjointness guard can assert against the SAME code
// the runtime executes — a test that re-derives this rule would keep passing
// against a stale copy if the real threshold ever moved.
func sealTimedCashSlotItemType(t tenant.Model) CashSlotItemType {
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		return CashSlotItemTypeSealTimedV95
	}
	return CashSlotItemTypeSealTimed
}

// karmaScissorsCashSlotItemType returns the version-scoped CashSlotItemType for
// the Scissors of Karma (classification 552).
//
// A bare constant compare is FORBIDDEN here: pre-95, CashSlotItemTypeSealTimed
// is also 64. The karma and seal arms are disjoint at runtime today only because
// the seal arm recomputes itself to 65 on GMS >= 95 (:261-265) — a coincidence
// that a version-scoped resolver on both sides turns into a structural property.
// karma_slot_type_test.go asserts the two never collide on any configured
// version.
func karmaScissorsCashSlotItemType(t tenant.Model) CashSlotItemType {
	if t.Region() == "GMS" && t.MajorVersion() >= 95 {
		return CashSlotItemTypeKarmaScissorsV95
	}
	return CashSlotItemTypeKarmaScissors
}

func GetCashSlotItemType(t tenant.Model) func(itemId item.Id) CashSlotItemType {
	return func(itemId item.Id) CashSlotItemType {
		category := item.GetClassification(itemId)
		if category == item.ClassificationPet {
			return CashSlotItemType(8)
		}
		if category == item.ClassificationCurrencySack {
			return CashSlotItemTypeCurrencySack
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
			return CashSlotItemTypeNote
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
			// get_cashslot_item_type @0x48645b, case 517:
			//   return a1 % 10000 != 0 ? 0 : 17;
			// The previous spelling of this was `10000*itemId/10000 != itemId`,
			// which OVERFLOWS: item.Id is uint32, so 10000 * 5170000 wraps to
			// 160,392,448 and the branch returned 0 for the one item id it was
			// supposed to admit. The Pet Name Tag never reached a handler.
			if itemId%10000 != 0 {
				return CashSlotItemType(0)
			}
			return CashSlotItemTypePetNameTag
		}
		if category == item.ClassificationWaterOfLife {
			return CashSlotItemType(5)
		}
		if category == item.ClassificationPetSkill {
			return CashSlotItemTypePetSkill
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
		if category == item.ClassificationExpirationExtender {
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
		if category == item.ClassificationKarmaScissors {
			return karmaScissorsCashSlotItemType(t)
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
