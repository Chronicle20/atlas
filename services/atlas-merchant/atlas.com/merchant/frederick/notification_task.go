package frederick

import (
	merchant "atlas-merchant/kafka/message/merchant"
	producer2 "atlas-merchant/kafka/producer"
	"context"
	"time"

	database "github.com/Chronicle20/atlas-database"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const DefaultNotificationInterval = 1 * time.Hour

var notificationTiers = []uint16{2, 5, 10, 15, 30, 60, 90}

type NotificationTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	db       *gorm.DB
	interval time.Duration
}

func NewNotificationTask(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, interval time.Duration) *NotificationTask {
	l.Infof("Initializing Frederick notification task to run every %dms.", interval.Milliseconds())
	return &NotificationTask{l: l, ctx: ctx, db: db, interval: interval}
}

func (t *NotificationTask) Run() {
	noTenantCtx := database.WithoutTenantFilter(t.ctx)

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

	for _, n := range notifications {
		ten, err := tenant.Create(n.TenantId, n.TenantRegion, n.TenantMajor, n.TenantMinor)
		if err != nil {
			t.l.WithError(err).Errorf("Error creating tenant context for notification [%s].", n.Id)
			continue
		}
		tctx := tenant.WithContext(t.ctx, ten)

		kp := producer2.ProviderImpl(t.l)(tctx)
		_ = kp(merchant.EnvStatusEventTopic)(notificationProvider(n.CharacterId, n.NextDay))

		next, hasNext := nextTier(n.NextDay)
		if hasNext {
			if _, err := advanceNotification(n.Id, next)(t.db.WithContext(noTenantCtx))(); err != nil {
				t.l.WithError(err).Errorf("Error advancing notification [%s] to tier %d.", n.Id, next)
			}
		} else {
			if _, err := deleteNotification(n.Id)(t.db.WithContext(noTenantCtx))(); err != nil {
				t.l.WithError(err).Errorf("Error deleting final notification [%s].", n.Id)
			}
		}
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
