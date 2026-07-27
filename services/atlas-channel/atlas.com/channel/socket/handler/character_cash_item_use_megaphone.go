package handler

import (
	character2 "atlas-channel/character"
	"atlas-channel/saga"
	"atlas-channel/session"
	socketmodel "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"atlas-channel/worldbroadcast"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	chatpkg "github.com/Chronicle20/atlas/libs/atlas-packet/chat"             // A1: body funcs (resolved codes)
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound" // writer name consts
	tvpkg "github.com/Chronicle20/atlas/libs/atlas-packet/tv"                 // A1: body funcs (resolved codes)
	tvpkt "github.com/Chronicle20/atlas/libs/atlas-packet/tv/clientbound"     // writer name consts
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	tierMegaphone = "MEGAPHONE"
	tierSuper     = "SUPER"
	tierItem      = "ITEM"
	tierTriple    = "TRIPLE"

	scopeChannel = "CHANNEL"
	scopeWorld   = "WORLD"

	tvWaitCapSeconds     = uint32(3600) // client string: "the waiting line is longer than an hour"
	avatarWaitCapSeconds = uint32(15)   // client string SP_3972 (design §1.2)
	avatarDurationSecs   = uint32(10)   // client auto-clear constant, IDA v83+v95
)

// tvDurationSeconds — Cosmic MapleTVEffect.java:56-61 (server policy values, design D8).
func tvDurationSeconds(tvType byte) uint32 {
	switch tvType {
	case 4:
		return 30
	case 5:
		return 60
	default:
		return 15
	}
}

// tvMessageType — A1 delta: returns the SEMANTIC key, not a wire byte. The
// client byte is resolved from the tenant messageTypes table by tvpkg.TvSetMessageBody.
// Selector rule: Cosmic PacketCreator.sendTV call site (type <= 2 ? type : type - 3).
func tvMessageType(tvType byte) tvpkg.TvMessageType {
	sel := tvType
	if sel > 2 {
		sel -= 3
	}
	switch sel {
	case 1:
		return tvpkg.TvMessageStar
	case 2:
		return tvpkg.TvMessageHeart
	default:
		return tvpkg.TvMessageNormal
	}
}

func handleMegaphoneUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}, t tenant.Model, itemId item.Id, source slot.Position, updateTimeFirst bool) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}, t tenant.Model, itemId item.Id, source slot.Position, updateTimeFirst bool) {
		// Fetched with the same decorators as the messenger consumer's spawn
		// look fetch (kafka/consumer/messenger/consumer.go:164) so
		// socketmodel.NewAvatarSnapshot(c) (TV sender look, case 4/5) sees a
		// populated Equipment()/Pets() rather than the zero-valued undecorated
		// model — GetById() alone (as the brief's literal snippet had it) would
		// silently render a naked avatar. Cheap tiers (1/2/6/7) only read
		// c.Name(), so the extra decoration is a minor, correctness-driven cost.
		cp := character2.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator)(s.CharacterId())
		if err != nil {
			l.WithError(err).Warnf("Character [%d] not found for megaphone use.", s.CharacterId())
			return
		}
		f := s.Field()

		// classification 507 sub-family, Cosmic UseCashItemHandler: (itemId / 1000) % 10
		switch (uint32(itemId) / 1000) % 10 {
		case 0: // 5070000 Cheap Megaphone — channel scope, same wire shape as basic (tier 1).
			// IDA (task-123 cheap-heart-skull recon, GMS v95 port 13341 + v83 port
			// 13342): get_cashslot_item_type's 507-family switch has NO arm for
			// tier 0 on EITHER build (v95 @0x488d08 falls to default; v83
			// @0x4864c2 falls to default) — get_consume_cash_item_type therefore
			// returns 0, and SendConsumeCashItemUseRequest's dispatch
			// (`lea eax,[type-0Ch]; ja def_9EB50A` @0x9eb4fa) sends type 0 straight
			// to the default arm, which RemoveAll()s the packet buffer and returns
			// WITHOUT any Encode call (@0x9eb65c-9eb66c) — the stock client never
			// emits a sub-body for this item id. No wire evidence exists to derive
			// a distinct codec from, so this reuses ItemUseMegaphone (message-only,
			// same as tier 1) per the confirmed CHANNEL-scope routing — mirrors
			// case 1 exactly.
			sp := cashsb.NewItemUseMegaphone(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			createMegaphoneSaga(l, ctx)(s, itemId, saga.EmitMegaphonePayload{
				Tier: tierMegaphone, Scope: scopeChannel,
				WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
				SenderName: c.Name(), SenderMedal: "",
				Messages: []string{sp.Message()},
			})
		case 1: // basic megaphone — channel scope
			sp := cashsb.NewItemUseMegaphone(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			createMegaphoneSaga(l, ctx)(s, itemId, saga.EmitMegaphonePayload{
				Tier: tierMegaphone, Scope: scopeChannel,
				WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
				SenderName: c.Name(), SenderMedal: "",
				Messages: []string{sp.Message()},
			})
		case 2: // super megaphone — world scope
			sp := cashsb.NewItemUseSuperMegaphone(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			createMegaphoneSaga(l, ctx)(s, itemId, saga.EmitMegaphonePayload{
				Tier: tierSuper, Scope: scopeWorld,
				WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
				SenderName: c.Name(), SenderMedal: "",
				Messages: []string{sp.Message()}, WhispersOn: sp.Whisper(),
			})
		case 3: // 5073000 Heart Megaphone — world scope, same wire shape as super (tier 2).
			// Same "no client send path" finding as tier 0 above (v95
			// get_cashslot_item_type has no tier-3 arm either @0x488d08, falls to
			// default -> type 0 -> the same no-Encode default arm). Reuses
			// ItemUseSuperMegaphone (message+whisper) per the confirmed WORLD-scope
			// routing — mirrors case 2 exactly.
			sp := cashsb.NewItemUseSuperMegaphone(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			createMegaphoneSaga(l, ctx)(s, itemId, saga.EmitMegaphonePayload{
				Tier: tierSuper, Scope: scopeWorld,
				WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
				SenderName: c.Name(), SenderMedal: "",
				Messages: []string{sp.Message()}, WhispersOn: sp.Whisper(),
			})
		case 4: // 5074000 Skull Megaphone — world scope, SUPER-shaped on every
			// version that sends it. Skull is NEVER Maple TV; only 5075xxx (case 5
			// below) is CUIMapleTV.
			//
			// IDA v83 (port 13342, get_cashslot_item_type@0x48645b): tier 4 has no
			// arm (falls through the 507-family if-chain to `return 0`) — no
			// client send path <v95.
			// IDA v95 (port 13341, get_cashslot_item_type@0x488d08): tier 4 DOES
			// have an arm — `case 4: result = 45;` — and
			// SendConsumeCashItemUseRequest's dispatch (@0x9eb4fa jumptable
			// 009EB50A) routes type 45 to the SAME arm as type 12 (basic) and 13
			// (super) at label $LN217_18 (0x9eb811, annotated "cases 12,13,15,45").
			// The final Encode sequence (@0x9ebc44-9ebc79) is EncodeStr(message)
			// unconditionally, then Encode1(whisper) gated on `type==13 ||
			// type==45` (@0x9ebc62-9ebc75) — Skull's v95 wire body is
			// BYTE-IDENTICAL to Super Megaphone's (message+whisper), built via
			// CSpeakerWorldDlg, NOT CUIMapleTV.
			// IDA jms185 (port 13344, MapleStory_dump_SCY.exe,
			// get_cashslot_item_type@0x49a1ee): tier 4 -> type 48, and
			// SendConsumeCashItemUseRequest's jump table (jpt_AEF3A8, base
			// 0xaf2b6a) routes type 48 to the SAME shared arm as types 12/13/15/47
			// (@0xaef5b9, comment "cases 12,13,15,47,48"). The arm's tail
			// (@0xaef98a EncodeStr(message); @0xaef98f-0xaef9a7: whisper Encode1
			// emitted when type==13(0x0D) OR type==47(0x2F) OR type==48(0x30))
			// confirms type 48 is message+whisper — SUPER shape, same conclusion
			// as v95.
			//
			// This task fixes the previously-mismatched >=95 routing (which sent
			// Skull to handleMapleTVUse — see task-123 cheap-heart-skull-report.md
			// for the original escalation): Skull is now routed as super/world
			// unconditionally, on every version, mirroring case 2/3 exactly —
			// Skull is never Maple TV.
			sp := cashsb.NewItemUseSuperMegaphone(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			createMegaphoneSaga(l, ctx)(s, itemId, saga.EmitMegaphonePayload{
				Tier: tierSuper, Scope: scopeWorld,
				WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
				SenderName: c.Name(), SenderMedal: "",
				Messages: []string{sp.Message()}, WhispersOn: sp.Whisper(),
			})
		case 5: // Maple TV / messenger group (5075xxx)
			handleMapleTVUse(l, ctx, wp)(s, r, readerOptions, itemId, c, updateTimeFirst)
		case 6: // item megaphone
			sp := cashsb.NewItemUseItemMegaphone(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			payload := saga.EmitMegaphonePayload{
				Tier: tierItem, Scope: scopeWorld,
				WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
				SenderName: c.Name(), SenderMedal: "",
				Messages: []string{sp.Message()}, WhispersOn: sp.Whisper(),
			}
			if sp.HasItem() {
				ref, err := cp.GetItemInSlot(s.CharacterId(), inventory.Type(sp.InvType()), int16(sp.Slot()))()
				if err != nil {
					// FR-1.4: empty/mismatched referenced slot rejects the use — no consume, no broadcast.
					l.WithError(err).Warnf("Character [%d] item megaphone referenced empty slot [%d/%d].", s.CharacterId(), sp.InvType(), sp.Slot())
					return
				}
				snap := socketmodel.NewAssetSnapshot(ref)
				payload.Item = &snap
			}
			createMegaphoneSaga(l, ctx)(s, itemId, payload)
		case 7: // triple megaphone
			sp := cashsb.NewItemUseTripleMegaphone(updateTimeFirst)
			sp.Decode(l, ctx)(r, readerOptions)
			if len(sp.Lines()) < 1 || len(sp.Lines()) > 3 {
				l.Warnf("Character [%d] triple megaphone with invalid line count [%d].", s.CharacterId(), len(sp.Lines()))
				return
			}
			createMegaphoneSaga(l, ctx)(s, itemId, saga.EmitMegaphonePayload{
				Tier: tierTriple, Scope: scopeWorld,
				WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
				SenderName: c.Name(), SenderMedal: "",
				Messages: sp.Lines(), WhispersOn: sp.Whisper(),
			})
		default:
			// tier 8 (507x8xxx / cash slot type 15) has no item in v83 WZ (design
			// D11); tier 9 is unassigned in the 507 family on every checked
			// version. Cheap(0)/basic(1)/super(2)/Heart(3)/Skull(4) are all
			// handled above; item(6)/TV(5)/triple(7) are handled elsewhere in this switch.
			l.Warnf("Character [%d] used unsupported megaphone item [%d].", s.CharacterId(), itemId)
		}
	}
}

func createMegaphoneSaga(l logrus.FieldLogger, ctx context.Context) func(s session.Model, itemId item.Id, payload saga.EmitMegaphonePayload) {
	return func(s session.Model, itemId item.Id, payload saga.EmitMegaphonePayload) {
		now := time.Now()
		steps := []saga.Step{
			{
				StepId: "consume_megaphone_item",
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
				StepId:    "emit_megaphone_broadcast",
				Status:    saga.Pending,
				Action:    saga.EmitMegaphone,
				Payload:   payload,
				CreatedAt: now,
				UpdatedAt: now,
			},
		}
		_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
			TransactionId: uuid.New(),
			SagaType:      saga.MegaphoneUse,
			InitiatedBy:   "CASH_ITEM_USE",
			Steps:         steps,
		})
	}
}

func handleMapleTVUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}, itemId item.Id, c character2.Model, updateTimeFirst bool) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}, itemId item.Id, c character2.Model, updateTimeFirst bool) {
		tvType := byte(uint32(itemId) % 10)
		sp := cashsb.NewItemUseMapleTV(updateTimeFirst, tvType)
		sp.Decode(l, ctx)(r, readerOptions)
		f := s.Field()

		// Wait-cap guard BEFORE consuming (design D3). REST failure rejects conservatively.
		wait, err := worldbroadcast.NewProcessor(l, ctx).GetWaitSeconds(f.WorldId(), worldbroadcast.FamilyTV)
		if err != nil || wait > tvWaitCapSeconds {
			if err != nil {
				l.WithError(err).Warnf("Unable to check TV queue for world [%d]; rejecting without consuming.", f.WorldId())
			}
			// A1 delta: config-resolved reason, not the literal 2.
			_ = session.Announce(l)(ctx)(wp)(tvpkt.TvSendMessageResultWriter)(tvpkg.TvSendMessageResultErrorBody(tvpkg.TvResultQueueTooLong))(s)
			return
		}

		// Partner lookup by name (design §5: absent/mismatch → self-message).
		// GetByName (character/processor.go:226) has no decorator parameter, so
		// the model it returns is undecorated (no Equipment()/Pets()); the
		// InventoryDecorator/PetAssetEnrichmentDecorator functions are applied
		// directly to it here rather than re-issuing a second GetById fetch.
		var receiverName string
		var receiverLook *saga.AvatarSnapshot
		if sp.ReceiverName() != "" {
			cp := character2.NewProcessor(l, ctx)
			if partner, perr := cp.GetByName(sp.ReceiverName()); perr == nil {
				partner = cp.PetAssetEnrichmentDecorator(cp.InventoryDecorator(partner))
				snap := socketmodel.NewAvatarSnapshot(partner)
				receiverName = partner.Name()
				receiverLook = &snap
			} else {
				l.Debugf("TV partner [%s] not found; broadcasting without partner.", sp.ReceiverName())
			}
		}

		lines := sp.Lines()
		enqueue := saga.EnqueueWorldBroadcastPayload{
			Family:  worldbroadcast.FamilyTV,
			WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
			SenderName: c.Name(), SenderMedal: "",
			Messages:        lines,
			TvMessageType:   string(tvMessageType(tvType)), // A1 delta: semantic key on the wire-free payload
			DurationSeconds: tvDurationSeconds(tvType),
			SenderLook:      socketmodel.NewAvatarSnapshot(c),
			ReceiverName:    receiverName,
			ReceiverLook:    receiverLook,
		}

		now := time.Now()
		steps := []saga.Step{
			{
				StepId: "consume_tv_item", Status: saga.Pending, Action: saga.DestroyAsset,
				Payload:   saga.DestroyAssetPayload{CharacterId: s.CharacterId(), TemplateId: uint32(itemId), Quantity: 1, RemoveAll: false},
				CreatedAt: now, UpdatedAt: now,
			},
			{
				StepId: "enqueue_tv_broadcast", Status: saga.Pending, Action: saga.EnqueueWorldBroadcast,
				Payload: enqueue, CreatedAt: now, UpdatedAt: now,
			},
		}
		if tvType >= 3 {
			// Megassenger tiers also fire a super megaphone with the concatenated
			// lines and ear-as-whisper (Cosmic UseCashItemHandler case 5 parity).
			combined := ""
			for _, ln := range lines {
				if ln != "" {
					if combined != "" {
						combined += " "
					}
					combined += ln
				}
			}
			steps = append(steps, saga.Step{
				StepId: "emit_megassenger_super", Status: saga.Pending, Action: saga.EmitMegaphone,
				Payload: saga.EmitMegaphonePayload{
					Tier: tierSuper, Scope: scopeWorld,
					WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
					SenderName: c.Name(), SenderMedal: "",
					Messages: []string{combined}, WhispersOn: sp.Ear(),
				},
				CreatedAt: now, UpdatedAt: now,
			})
		}
		_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
			TransactionId: uuid.New(), SagaType: saga.MegaphoneUse,
			InitiatedBy: "CASH_ITEM_USE", Steps: steps,
		})
	}
}

func handleAvatarMegaphoneUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}, t tenant.Model, itemId item.Id, source slot.Position, updateTimeFirst bool) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}, t tenant.Model, itemId item.Id, source slot.Position, updateTimeFirst bool) {
		sp := cashsb.NewItemUseAvatarMegaphone(updateTimeFirst)
		sp.Decode(l, ctx)(r, readerOptions)

		// Decorated the same way as handleMegaphoneUse (see comment there):
		// NewAvatarSnapshot(c) below needs a populated Equipment()/Pets().
		cp := character2.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator)(s.CharacterId())
		if err != nil {
			l.WithError(err).Warnf("Character [%d] not found for avatar megaphone use.", s.CharacterId())
			return
		}
		f := s.Field()

		reject := func() {
			if t.Region() == "JMS" {
				return // no AVATAR_MEGAPHONE_RESULT op in jms (STATUS.md line 143)
			}
			// A1 delta: config-resolved reason, not the literal 83.
			_ = session.Announce(l)(ctx)(wp)(chatpkt.AvatarMegaphoneResultWriter)(chatpkg.AvatarMegaphoneResultBody(chatpkg.AvatarMegaphoneWaitingLine))(s)
		}

		wait, err := worldbroadcast.NewProcessor(l, ctx).GetWaitSeconds(f.WorldId(), worldbroadcast.FamilyAvatar)
		if err != nil || wait > avatarWaitCapSeconds {
			if err != nil {
				l.WithError(err).Warnf("Unable to check avatar queue for world [%d]; rejecting without consuming.", f.WorldId())
			}
			reject()
			return
		}

		now := time.Now()
		_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
			TransactionId: uuid.New(), SagaType: saga.MegaphoneUse, InitiatedBy: "CASH_ITEM_USE",
			Steps: []saga.Step{
				{
					StepId: "consume_avatar_megaphone", Status: saga.Pending, Action: saga.DestroyAsset,
					Payload:   saga.DestroyAssetPayload{CharacterId: s.CharacterId(), TemplateId: uint32(itemId), Quantity: 1, RemoveAll: false},
					CreatedAt: now, UpdatedAt: now,
				},
				{
					StepId: "enqueue_avatar_broadcast", Status: saga.Pending, Action: saga.EnqueueWorldBroadcast,
					Payload: saga.EnqueueWorldBroadcastPayload{
						Family:  worldbroadcast.FamilyAvatar,
						WorldId: f.WorldId(), ChannelId: f.ChannelId(), CharacterId: s.CharacterId(),
						SenderName: c.Name(), SenderMedal: "",
						Messages: sp.Lines(), WhispersOn: sp.Whisper(),
						ItemId: uint32(itemId), DurationSeconds: avatarDurationSecs,
						SenderLook: socketmodel.NewAvatarSnapshot(c),
					},
					CreatedAt: now, UpdatedAt: now,
				},
			},
		})
	}
}
