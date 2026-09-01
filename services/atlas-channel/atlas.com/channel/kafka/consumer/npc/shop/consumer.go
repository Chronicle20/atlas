package shop

import (
	"atlas-channel/character/skill"
	consumer2 "atlas-channel/kafka/consumer"
	shops2 "atlas-channel/kafka/message/npc/shop"
	"atlas-channel/listener"
	"atlas-channel/npc/shops"
	"atlas-channel/remotemerchant"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("npc_shop_status_event")(shops2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(ctx context.Context) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(ctx context.Context) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
					var t string
					var err error
					var handles []listener.HandlerHandle
					t, err = topic.EnvProvider(l)(shops2.EnvStatusEventTopic)()
					if err != nil {
						return nil, err
					}
					id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEnteredStatusEvent(sc, wp))))
					if err != nil {
						return nil, err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
					id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleErrorStatusEvent(sc, wp))))
					if err != nil {
						return nil, err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
					id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleEnterErrorStatusEvent(sc, wp))))
					if err != nil {
						return nil, err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})

					startRemoteMerchantSweep(l, ctx, wp)

					return handles, nil
				}
			}
		}
	}
}

// unlockPendingRemoteMerchant sends EnableActions if and only if this character
// reached the shop by using a classification-545 cash item. The client sets
// m_bExclRequestSent when it sends CASH_ITEM_USE and CShopDlg::SetShopDlg never
// clears it (task-221 design §1.2 OQ-2), so the server must unlock — but only
// for that path. Unlocking the ordinary NPC-talk path would change the bytes on
// versions whose OPEN_NPC_SHOP cells are already verified.
func unlockPendingRemoteMerchant(l logrus.FieldLogger, t tenant.Model, characterId uint32, unlock func()) {
	e, ok := remotemerchant.GetRegistry().Take(t, characterId)
	if !ok {
		return
	}
	l.WithFields(logrus.Fields{
		"character_id": characterId,
		"item_id":      uint32(e.ItemId),
		"slot":         int16(e.Slot),
	}).Debug("Unlocking client after a remote-merchant shop open.")
	unlock()
}

func handleEnteredStatusEvent(sc server.Model, wp writer.Producer) message.Handler[shops2.StatusEvent[shops2.StatusEventEnteredBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e shops2.StatusEvent[shops2.StatusEventEnteredBody]) {
		if e.Type != shops2.StatusEventTypeEntered {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.CharacterId)
		if err != nil {
			return
		}

		// EnableActions is the client's duplicate-request gate, so it goes
		// AFTER the shop packet on the success path — but it must still fire if
		// the announce below bails out, or the player is locked for nothing.
		defer unlockPendingRemoteMerchant(l, t, e.CharacterId, func() {
			_ = session.EnableActions(l)(ctx)(wp)(s)
		})

		sms, err := skill.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get skills for character [%d].", s.CharacterId())
			return
		}

		nsm, err := shops.NewProcessor(l, ctx).GetShop(e.Body.NpcTemplateId)
		if err != nil {
			l.WithError(err).Errorf("Unable to get shop for NPC [%d].", e.Body.NpcTemplateId)
			return
		}
		set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
		bp := writer.NPCShopBody(e.Body.NpcTemplateId, nsm.Commodities(), sms, set.Skill)
		_ = session.Announce(l)(ctx)(wp)(npcpkt.NPCShopWriter)(bp)(s)
	}
}

func handleErrorStatusEvent(sc server.Model, wp writer.Producer) message.Handler[shops2.StatusEvent[shops2.StatusEventErrorBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e shops2.StatusEvent[shops2.StatusEventErrorBody]) {
		if e.Type != shops2.StatusEventTypeError {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.CharacterId)
		if err != nil {
			return
		}

		var bp packet.Encode
		switch e.Body.Error {
		case npcpkt.NPCShopOperationOk:
			bp = npcpkt.NPCShopOperationOkBody()
		case npcpkt.NPCShopOperationOutOfStock:
			bp = npcpkt.NPCShopOperationOutOfStockBody()
		case npcpkt.NPCShopOperationNotEnoughMoney:
			bp = npcpkt.NPCShopOperationNotEnoughMoneyBody()
		case npcpkt.NPCShopOperationInventoryFull:
			bp = npcpkt.NPCShopOperationInventoryFullBody()
		case npcpkt.NPCShopOperationOutOfStock2:
			bp = npcpkt.NPCShopOperationOutOfStock2Body()
		case npcpkt.NPCShopOperationOutOfStock3:
			bp = npcpkt.NPCShopOperationOutOfStock3Body()
		case npcpkt.NPCShopOperationNotEnoughMoney2:
			bp = npcpkt.NPCShopOperationNotEnoughMoney2Body()
		case npcpkt.NPCShopOperationNeedMoreItems:
			bp = npcpkt.NPCShopOperationNeedMoreItemsBody()
		case npcpkt.NPCShopOperationTradeLimit:
			bp = npcpkt.NPCShopOperationTradeLimitBody()
		case npcpkt.NPCShopOperationOverLevelRequirement:
			bp = npcpkt.NPCShopOperationOverLevelRequirementBody(e.Body.LevelLimit)
		case npcpkt.NPCShopOperationUnderLevelRequirement:
			bp = npcpkt.NPCShopOperationUnderLevelRequirementBody(e.Body.LevelLimit)
		case npcpkt.NPCShopOperationGenericError:
			bp = npcpkt.NPCShopOperationGenericErrorBody()
		case npcpkt.NPCShopOperationGenericErrorWithReason:
			bp = npcpkt.NPCShopOperationGenericErrorWithReasonBody(e.Body.Reason)
		default:
			l.Warnf("Unhandled NPC shop operation error code [%s].", e.Body.Error)
			return
		}
		_ = session.Announce(l)(ctx)(wp)(npcpkt.NPCShopOperationWriter)(bp)(s)
	}
}

// handleEnterErrorStatusEvent unlocks the client after a failed remote-merchant
// shop open. It deliberately writes NO packet: NPCShopOperation with no
// outstanding buy/sell/recharge request throws CDisconnectException in
// CShopDlg::OnPacket (@0x756da7), which is exactly why ENTER_ERROR is a
// separate status type from ERROR (task-221 design delta D5).
func handleEnterErrorStatusEvent(sc server.Model, wp writer.Producer) message.Handler[shops2.StatusEvent[shops2.StatusEventEnterErrorBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e shops2.StatusEvent[shops2.StatusEventEnterErrorBody]) {
		if e.Type != shops2.StatusEventTypeEnterError {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.CharacterId)
		if err != nil {
			return
		}

		l.WithFields(logrus.Fields{
			"character_id":    e.CharacterId,
			"npc_template_id": e.Body.NpcTemplateId,
			"reason":          e.Body.Reason,
		}).Warn("NPC shop enter failed.")

		unlockPendingRemoteMerchant(l, t, e.CharacterId, func() {
			_ = session.EnableActions(l)(ctx)(wp)(s)
		})
	}
}

// remoteMerchantSweepStarted deduplicates the sweep goroutine per tenant.
// InitHandlers runs once per (tenant, world, channel) listener key
// (buildListener's AddBody, invoked by listener.Registry.Add — see
// listener/registry.go and configuration/projection/apply.go), so a tenant
// with W worlds x C channels would otherwise spin up W*C independent tickers
// all polling the same package-level remotemerchant.Registry: harmless once
// Sweep is tenant-scoped, but wasteful, and the same root gap that produced
// the round-2 code review's cross-tenant entry-theft finding. One sweep
// goroutine per tenant is sufficient — Sweep(t, now) already scopes eviction
// to that tenant, and findSessionByCharacterId below searches every session
// in the tenant rather than just the (world, channel) the goroutine happened
// to be started from, so it doesn't matter which listener key's InitHandlers
// call wins the race to start it.
//
// The claim is released, not permanent: each (tenant, world, channel) key
// gets its own ctx from listener.Registry.Add, and Drain cancels just that
// one key's ctx on an ordinary channel-scale-down — the tenant's other keys
// stay live (listener/registry.go Drain, configuration/projection/apply.go
// OpDrain). If the key that won the claim above is the one that drains, its
// goroutine's ctx.Done() fires and it must give up the claim so a still-live
// (or future) key for the same tenant can start a replacement sweeper.
// Without this release, a routine channel drain — not tenant teardown —
// would silently disable the sweep for the tenant's remaining listeners
// forever (task-221 code review, round 3).
var (
	remoteMerchantSweepStarted   = make(map[uuid.UUID]bool)
	remoteMerchantSweepStartedMu sync.Mutex
)

// startRemoteMerchantSweep evicts pending unlocks whose status event never
// arrived and unlocks those clients, so a dropped event cannot leave a
// character permanently locked (task-221 design §2.3). It is a no-op if a
// sweep is already running for this tenant (see remoteMerchantSweepStarted
// above); when the running sweep's own ctx is cancelled it releases its
// claim so a later call for the same tenant can start a replacement.
func startRemoteMerchantSweep(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) {
	t := tenant.MustFromContext(ctx)

	remoteMerchantSweepStartedMu.Lock()
	if remoteMerchantSweepStarted[t.Id()] {
		remoteMerchantSweepStartedMu.Unlock()
		return
	}
	remoteMerchantSweepStarted[t.Id()] = true
	remoteMerchantSweepStartedMu.Unlock()

	routine.Go(l, ctx, func(c context.Context) {
		// Release this tenant's claim on the way out, however the loop below
		// exits, so the next InitHandlers call for this tenant — on this key
		// restarting, or any of the tenant's other still-live keys — can
		// start a fresh sweeper instead of finding the guard permanently
		// latched by a goroutine that no longer exists.
		defer func() {
			remoteMerchantSweepStartedMu.Lock()
			delete(remoteMerchantSweepStarted, t.Id())
			remoteMerchantSweepStartedMu.Unlock()
		}()

		ticker := time.NewTicker(remotemerchant.TTL)
		defer ticker.Stop()
		for {
			select {
			case <-c.Done():
				return
			case now := <-ticker.C:
				for _, ex := range remotemerchant.GetRegistry().Sweep(t, now) {
					s, ok := findSessionByCharacterId(l, c, ex.CharacterId)
					if !ok {
						continue
					}
					l.WithFields(logrus.Fields{
						"character_id": ex.CharacterId,
						"item_id":      uint32(ex.Entry.ItemId),
					}).Warn("Remote-merchant shop open timed out with no status event; unlocking client.")
					_ = session.EnableActions(l)(c)(wp)(s)
				}
			}
		}
	})
}

// findSessionByCharacterId resolves a character's session anywhere in the
// tenant, not just on the (world, channel) the sweep goroutine happened to
// start from. A character's remote-merchant registry entry carries no
// world/channel — only tenant + characterId — so the goroutine that wins the
// per-tenant dedup race in startRemoteMerchantSweep must be able to find a
// session on any of the tenant's channels.
func findSessionByCharacterId(l logrus.FieldLogger, ctx context.Context, characterId uint32) (session.Model, bool) {
	all, err := session.NewProcessor(l, ctx).AllInTenantProvider()
	if err != nil {
		return session.Model{}, false
	}
	for _, s := range all {
		if s.CharacterId() == characterId {
			return s, true
		}
	}
	return session.Model{}, false
}
