package frederick

import (
	merchant "atlas-merchant/kafka/message/merchant"
	producer2 "atlas-merchant/kafka/producer"
	"context"
	"sync"
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

const NotificationInterval = 1 * time.Hour

var notificationTiers = []uint16{2, 5, 10, 15, 30, 60, 90}

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

func StartNotificationScheduler(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup, db *gorm.DB) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(NotificationInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				l.Infoln("Frederick notification scheduler shutting down.")
				return
			case <-ticker.C:
				processNotifications(l, ctx, db)
			}
		}
	}()
	l.Infoln("Frederick notification scheduler started.")
}

func processNotifications(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) {
	noTenantCtx := database.WithoutTenantFilter(ctx)

	var notifications []NotificationEntity
	err := db.WithContext(noTenantCtx).
		Where("stored_at + (next_day || ' days')::interval <= NOW()").
		Find(&notifications).Error
	if err != nil {
		l.WithError(err).Errorln("Error querying due Frederick notifications.")
		return
	}

	if len(notifications) == 0 {
		return
	}

	// Verify the merchant status topic is configured.
	_, err = topic.EnvProvider(l)(merchant.EnvStatusEventTopic)()
	if err != nil {
		l.WithError(err).Warnln("Merchant status event topic not configured, skipping notifications.")
		return
	}

	l.Infof("Processing %d Frederick notifications.", len(notifications))

	for _, n := range notifications {
		t, err := tenant.Create(n.TenantId, n.TenantRegion, n.TenantMajor, n.TenantMinor)
		if err != nil {
			l.WithError(err).Errorf("Error creating tenant context for notification [%s].", n.Id)
			continue
		}
		tctx := tenant.WithContext(ctx, t)

		kp := producer2.ProviderImpl(l)(tctx)
		_ = kp(merchant.EnvStatusEventTopic)(notificationProvider(n.CharacterId, n.NextDay))

		// Advance to next notification tier or delete if final.
		next, hasNext := nextTier(n.NextDay)
		if hasNext {
			db.WithContext(noTenantCtx).Model(&NotificationEntity{}).Where("id = ?", n.Id).Update("next_day", next)
		} else {
			db.WithContext(noTenantCtx).Where("id = ?", n.Id).Delete(&NotificationEntity{})
		}
	}
}
