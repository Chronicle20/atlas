package tenants

import (
	"os"
	"time"

	"atlas-configurations/outbox"

	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Backfill re-publishes every existing tenant row into the outbox.
// Same semantics as services.Backfill — idempotent on (topic, key),
// no-op when EnvTenantStatusTopic is unset.
func Backfill(db *gorm.DB) (int, error) {
	topic := os.Getenv(EnvTenantStatusTopic)
	if topic == "" {
		return 0, nil
	}

	type row struct {
		id uuid.UUID
		rm RestModel
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
		return tenantOutboxKey(v.(row).id), nil
	}
	valueFn := func(v any) ([]byte, error) {
		r := v.(row)
		return outbox.NewTenantEnvelope(r.id, r.rm, time.Now())
	}
	return outboxlib.Backfill(db, topic, loader, keyFn, valueFn)
}
