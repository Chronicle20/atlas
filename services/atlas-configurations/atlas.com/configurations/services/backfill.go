package services

import (
	"os"
	"time"

	"atlas-configurations/outbox"

	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Backfill re-publishes every existing service row into the outbox so a
// cold-start subscriber (e.g. atlas-channel projection) can rebuild state
// without waiting for a CRUD event. Idempotent: outbox dedupes by
// (topic, key), so running it on every boot is safe.
//
// Returns the number of new outbox rows inserted. Steady-state restarts
// return 0. When EnvServiceStatusTopic is unset, returns 0 with no error.
func Backfill(db *gorm.DB) (int, error) {
	topic := os.Getenv(EnvServiceStatusTopic)
	if topic == "" {
		return 0, nil
	}

	type row struct {
		id uuid.UUID
		rm any
	}

	loader := func() ([]any, error) {
		var ents []Entity
		if err := db.Find(&ents).Error; err != nil {
			return nil, err
		}
		out := make([]any, 0, len(ents))
		for i := range ents {
			rm, err := Make(ents[i])
			if err != nil {
				return nil, err
			}
			out = append(out, row{id: ents[i].Id, rm: rm})
		}
		return out, nil
	}
	keyFn := func(v any) ([]byte, error) {
		return serviceOutboxKey(v.(row).id), nil
	}
	valueFn := func(v any) ([]byte, error) {
		r := v.(row)
		return outbox.NewServiceEnvelope(r.id, r.rm, time.Now())
	}
	return outboxlib.Backfill(db, topic, loader, keyFn, valueFn)
}
