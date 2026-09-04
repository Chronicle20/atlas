package frederick

import (
	"atlas-merchant/kafka/message"
	merchant "atlas-merchant/kafka/message/merchant"
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const DefaultNotificationInterval = 1 * time.Hour

var notificationTiers = []uint16{2, 5, 10, 15, 30, 60, 90}

type NotificationTask struct {
	l          logrus.FieldLogger
	db         *gorm.DB
	interval   time.Duration
	envContext func(context.Context) context.Context
}

func NewNotificationTask(l logrus.FieldLogger, db *gorm.DB, interval time.Duration, envContext func(context.Context) context.Context) *NotificationTask {
	l.Infof("Initializing Frederick notification task to run every %dms.", interval.Milliseconds())
	return &NotificationTask{l: l, db: db, interval: interval, envContext: envContext}
}

func (t *NotificationTask) Run(ctx context.Context) {
	noTenantCtx := database.WithoutTenantFilter(ctx)

	var notifications []NotificationEntity
	err := t.db.WithContext(noTenantCtx).
		Where("stored_at + (next_day || ' days')::interval <= NOW()").
		Find(&notifications).Error
	if err != nil {
		t.l.WithError(err).Errorln("Error querying due Frederick notifications.")
		return
	}

	if len(notifications) == 0 {
		return
	}

	_, err = topic.EnvProvider(t.l)(merchant.EnvStatusEventTopic)()
	if err != nil {
		t.l.WithError(err).Warnln("Merchant status event topic not configured, skipping notifications.")
		return
	}

	t.l.Infof("Processing %d Frederick notifications.", len(notifications))

	processDueNotifications(t.l, ctx, notifications, notifyAndAdvance(t.l, t.db, noTenantCtx), t.envContext)
}

// notifyAndAdvance emits the tier notification for one due entry and advances
// it to the next tier (or deletes it if none remain), both inside a single
// transaction: the Kafka message is buffered into the outbox alongside the
// write and only flushes once the write commits, so a failed advance/delete
// never leaves an already-emitted notification behind. Injected into
// processDueNotifications so the pure sweep logic can be tested with a spy
// in place of the real producer/db side effects.
func notifyAndAdvance(l logrus.FieldLogger, db *gorm.DB, noTenantCtx context.Context) func(ctx context.Context, n NotificationEntity) {
	return func(ctx context.Context, n NotificationEntity) {
		err := database.ExecuteTransaction(db.WithContext(noTenantCtx), func(tx *gorm.DB) error {
			return message.Emit(outbox.EmitProvider(l, ctx, tx))(func(buf *message.Buffer) error {
				if err := buf.Put(merchant.EnvStatusEventTopic, notificationProvider(n.CharacterId, n.NextDay)); err != nil {
					return err
				}

				next, hasNext := nextTier(n.NextDay)
				if hasNext {
					if _, err := advanceNotification(n.Id, next)(tx)(); err != nil {
						return err
					}
				} else {
					if _, err := deleteNotification(n.Id)(tx)(); err != nil {
						return err
					}
				}
				return nil
			})
		})
		if err != nil {
			l.WithError(err).Errorf("Error notifying and advancing notification [%s].", n.Id)
		}
	}
}

// processDueNotifications originates this pod's own environment identity
// onto each due notification's per-tenant context before the emit -- an
// empty ENVIRONMENT header would make decide() fail open per FR-1.8 and
// every live deployment, not just this pod's, would notify.
func processDueNotifications(l logrus.FieldLogger, ctx context.Context, notifications []NotificationEntity, notify func(ctx context.Context, n NotificationEntity), envContext func(context.Context) context.Context) {
	for _, n := range notifications {
		ten, err := tenant.Create(n.TenantId, n.TenantRegion, n.TenantMajor, n.TenantMinor)
		if err != nil {
			l.WithError(err).Errorf("Error creating tenant context for notification [%s].", n.Id)
			continue
		}
		tctx := envContext(tenant.WithContext(ctx, ten))
		notify(tctx, n)
	}
}

func (t *NotificationTask) SleepTime() time.Duration {
	return t.interval
}

func nextTier(current uint16) (uint16, bool) {
	for _, tier := range notificationTiers {
		if tier > current {
			return tier, true
		}
	}
	return 0, false
}

func notificationProvider(characterId uint32, daysSinceStorage uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &merchant.StatusEvent[merchant.StatusEventFrederickNotificationBody]{
		CharacterId: characterId,
		Type:        merchant.StatusEventFrederickNotification,
		Body: merchant.StatusEventFrederickNotificationBody{
			DaysSinceStorage: daysSinceStorage,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
