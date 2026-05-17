package outbox

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Publisher interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type PublisherFunc func(ctx context.Context, msgs ...kafka.Message) error

func (f PublisherFunc) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	return f(ctx, msgs...)
}

type drainerConfig struct {
	pollInterval    time.Duration
	batchSize       int
	sweeperInterval time.Duration
	retention       time.Duration
}

type DrainerOption func(*drainerConfig)

func WithPollInterval(d time.Duration) DrainerOption {
	return func(c *drainerConfig) { c.pollInterval = d }
}

func WithBatchSize(n int) DrainerOption {
	return func(c *drainerConfig) { c.batchSize = n }
}

func WithSweeperInterval(d time.Duration) DrainerOption {
	return func(c *drainerConfig) { c.sweeperInterval = d }
}

func WithRetention(d time.Duration) DrainerOption {
	return func(c *drainerConfig) { c.retention = d }
}

type Drainer struct {
	l    logrus.FieldLogger
	db   *gorm.DB
	pub  Publisher
	cfg  drainerConfig
	stop chan struct{}
}

func NewDrainer(l logrus.FieldLogger, db *gorm.DB, pub Publisher, opts ...DrainerOption) *Drainer {
	cfg := drainerConfig{
		pollInterval:    1 * time.Second,
		batchSize:       100,
		sweeperInterval: 1 * time.Hour,
		retention:       7 * 24 * time.Hour,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &Drainer{l: l, db: db, pub: pub, cfg: cfg, stop: make(chan struct{})}
}

func (d *Drainer) Run(ctx context.Context) {
	t := time.NewTicker(d.cfg.pollInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stop:
			return
		case <-t.C:
			if err := d.publishBatch(ctx); err != nil {
				d.l.WithError(err).Warn("outbox.publish_failed")
			}
		}
	}
}

func (d *Drainer) Stop() { close(d.stop) }

func (d *Drainer) publishBatch(ctx context.Context) error {
	var rows []Entity
	if err := d.db.WithContext(ctx).
		Where("sent_at IS NULL").
		Order("enqueued_at ASC").
		Limit(d.cfg.batchSize).
		Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	msgs := make([]kafka.Message, 0, len(rows))
	for _, r := range rows {
		msgs = append(msgs, kafka.Message{
			Topic: r.Topic,
			Key:   r.MessageKey,
			Value: r.MessageValue,
		})
	}
	if err := d.pub.WriteMessages(ctx, msgs...); err != nil {
		ids := make([]uint64, 0, len(rows))
		for _, r := range rows {
			ids = append(ids, r.ID)
		}
		d.db.WithContext(ctx).
			Model(&Entity{}).
			Where("id IN ?", ids).
			Updates(map[string]any{"attempts": gorm.Expr("attempts + 1"), "last_error": err.Error()})
		return err
	}

	now := time.Now()
	ids := make([]uint64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return d.db.WithContext(ctx).
		Model(&Entity{}).
		Where("id IN ?", ids).
		Updates(map[string]any{"sent_at": &now}).Error
}
