package outbox

import (
	"gorm.io/gorm"
)

// Loader returns the source rows to be considered for backfill. The returned
// slice is iterated once; the caller controls whether the slice is bounded
// (e.g. paginated) or eagerly enumerates an entire table.
type Loader func() ([]any, error)

// ToBytes converts a source row into either the Kafka message key or value.
// Splitting into two functions lets callers reuse a single Loader for both.
type ToBytes func(any) ([]byte, error)

// Backfill enqueues outbox rows for each loader row whose (topic, key) is not
// already present in outbox_entries. Returns the number of rows actually
// inserted. Safe to invoke repeatedly — running it again after a successful
// pass is a no-op, which makes it appropriate for a fresh-cluster bootstrap
// path that runs on every service startup.
func Backfill(db *gorm.DB, topic string, loader Loader, keyFn, valueFn ToBytes) (int, error) {
	rows, err := loader()
	if err != nil {
		return 0, err
	}

	added := 0
	for _, r := range rows {
		k, err := keyFn(r)
		if err != nil {
			return added, err
		}
		var count int64
		if err := db.Model(&Entity{}).Where("topic = ? AND message_key = ?", topic, k).Count(&count).Error; err != nil {
			return added, err
		}
		if count > 0 {
			continue
		}

		v, err := valueFn(r)
		if err != nil {
			return added, err
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			return Enqueue(tx, Message{Topic: topic, Key: k, Value: v})
		})
		if err != nil {
			return added, err
		}
		added++
	}
	return added, nil
}
