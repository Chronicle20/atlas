package teleportrock

import (
	character2 "atlas-channel/character"
	chartrock "atlas-channel/character/teleportrock"
	datamap "atlas-channel/data/map"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	trpkt "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	trcb "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock/clientbound"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// rockUseBarMask are the fieldLimit bits that bar teleport-rock use on a map
// (client checks 0x40 and 0x02 on the source; the server also applies them to
// the target — design §1 Q2).
const rockUseBarMask = _map.FieldLimitNoTeleportItem | _map.FieldLimitNoMysticDoor

// Injection points for table tests (package-var precedent:
// socket/handler/mystic_door_enter.go:25-51).
var listsFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (chartrock.Model, error) {
	return chartrock.NewProcessor(l, ctx).GetByCharacterId(characterId)
}

var mapLimitFunc = func(l logrus.FieldLogger, ctx context.Context, mapId _map.Id) (uint32, error) {
	m, err := datamap.NewProcessor(l, ctx).GetById(mapId)
	if err != nil {
		return 0, err
	}
	return m.FieldLimit(), nil
}

var characterByNameFunc = func(l logrus.FieldLogger, ctx context.Context, name string) (uint32, error) {
	c, err := character2.NewProcessor(l, ctx).GetByName(name)
	if err != nil {
		return 0, err
	}
	return c.Id(), nil
}

var sessionByCharacterIdFunc = func(l logrus.FieldLogger, ctx context.Context, s session.Model, characterId uint32) (field.Model, error) {
	target, err := session.NewProcessor(l, ctx).GetByCharacterId(s.Field().Channel())(characterId)
	if err != nil {
		return field.Model{}, err
	}
	return target.Field(), nil
}

var createSagaFunc = func(l logrus.FieldLogger, ctx context.Context, s saga.Saga) error {
	return saga.NewProcessor(l, ctx).Create(s)
}

var announceErrorFunc = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, key string, vip bool) {
	err := session.Announce(l)(ctx)(wp)(trcb.MapTransferResultWriter)(trpkt.MapTransferResultErrorBody(key, vip))(s)
	if err != nil {
		l.WithError(err).Errorf("Unable to announce teleport-rock rejection to character [%d].", s.CharacterId())
	}
}

func continent(mapId _map.Id) uint32 {
	return uint32(mapId) / 100000000
}

// UseRock validates and executes a teleport-rock warp for both entry ops
// (USE_TELEPORT_ROCK and the cash-item-use branch). The caller has already
// verified the item exists in the claimed slot. Validation failures announce
// the faithful MAP_TRANSFER_RESULT mode and consume nothing (FR-1); success
// launches a warp[-then-consume] saga (FR-2, design §4.3).
func UseRock(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id, target trpkt.Target) {
	return func(s session.Model, itemId item.Id, target trpkt.Target) {
		// 5041xxx is the only VIP-list rock; 2320000/5040000/5040001 use the
		// regular list (client: bCanTransferContinent = nItemID/1000 != 5040,
		// evaluated only for 504x — design §1 Q5).
		useVipList := uint32(itemId)/1000 == 5041

		fail := func(key string) {
			announceErrorFunc(l, ctx, wp, s, key, useVipList)
		}

		// 1. Source field bar.
		srcLimit, err := mapLimitFunc(l, ctx, s.Field().MapId())
		if err != nil || srcLimit&rockUseBarMask != 0 {
			l.Debugf("Teleport rock: source map [%d] barred (limit=0x%x err=%v) for character [%d].", s.Field().MapId(), srcLimit, err, s.CharacterId())
			fail(trpkt.MapTransferModeCannotGo)
			return
		}

		// 2. Resolve the target map.
		var targetMapId _map.Id
		if target.ByName() {
			targetId, err := characterByNameFunc(l, ctx, target.TargetName())
			if err != nil {
				l.Debugf("Teleport rock: target [%s] not found for character [%d].", target.TargetName(), s.CharacterId())
				fail(trpkt.MapTransferModeUnableToLocate)
				return
			}
			tf, err := sessionByCharacterIdFunc(l, ctx, s, targetId)
			if err != nil {
				// Offline, other channel, or cash shop: same rejection (design §1 Q6).
				l.Debugf("Teleport rock: target [%s] (id %d) has no session on this channel.", target.TargetName(), targetId)
				fail(trpkt.MapTransferModeUnableToLocate)
				return
			}
			targetMapId = tf.MapId()
		} else {
			targetMapId = _map.Id(target.TargetMap())
			lists, err := listsFunc(l, ctx, s.CharacterId())
			if err != nil || !lists.Contains(useVipList, targetMapId) {
				l.Debugf("Teleport rock: map [%d] not in list (vip=%v err=%v) for character [%d].", targetMapId, useVipList, err, s.CharacterId())
				fail(trpkt.MapTransferModeCannotGo)
				return
			}
		}

		// 3. Same map.
		if targetMapId == s.Field().MapId() {
			fail(trpkt.MapTransferModeCurrentMap)
			return
		}

		// 4. Target field bar (server-side policy half of Q2).
		dstLimit, err := mapLimitFunc(l, ctx, targetMapId)
		if err != nil || dstLimit&rockUseBarMask != 0 {
			l.Debugf("Teleport rock: target map [%d] barred (limit=0x%x err=%v) for character [%d].", targetMapId, dstLimit, err, s.CharacterId())
			fail(trpkt.MapTransferModeCannotGo)
			return
		}

		// 5. Continent restriction for non-VIP rocks (server policy, design §1 Q3).
		if !useVipList && continent(s.Field().MapId()) != continent(targetMapId) {
			fail(trpkt.MapTransferModeCannotGoContinent)
			return
		}

		// Success: warp via random spawn portal; consume only the regular rock,
		// and only after the warp (FR-2).
		targetField := field.NewBuilder(s.Field().WorldId(), s.Field().ChannelId(), targetMapId).Build()
		now := time.Now()
		steps := []saga.Step{
			{
				StepId: "warp_to_target",
				Status: saga.Pending,
				Action: saga.WarpToRandomPortal,
				Payload: saga.WarpToRandomPortalPayload{
					CharacterId: s.CharacterId(),
					FieldId:     targetField.Id(),
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		}
		if uint32(itemId)/10000 == 232 {
			steps = append(steps, saga.Step{
				StepId: "consume_rock",
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
			})
		}
		err = createSagaFunc(l, ctx, saga.Saga{
			TransactionId: uuid.New(),
			SagaType:      saga.TeleportRockUse,
			InitiatedBy:   "TELEPORT_ROCK_USE",
			Steps:         steps,
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to create teleport-rock saga for character [%d].", s.CharacterId())
		}
	}
}
