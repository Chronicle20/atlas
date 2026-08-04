package seed

import (
	"atlas-tenants/configuration"
	"atlas-tenants/kafka/message"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// seededResourceId is the ResourceId carried by the single
// configuration-status event a seed run emits. atlas-transports switches
// on ResourceType and uses ResourceId only for logging, so a synthetic
// value is correct and reads clearly in a log line.
const seededResourceId = "*"

// Groups returns one seeder.Group per transport configuration resource.
// Separate groups (rather than one group with three subdomains) keep each
// resource's seed_state revision independent and let the Setup page seed
// them individually, as FR-2.1 requires.
func Groups(l logrus.FieldLogger) []seeder.Group {
	return []seeder.Group{
		newGroup(l, "routes", configuration.EventTypeRouteUpdated, configuration.CreateRouteStatusEventProvider),
		newGroup(l, "vessels", configuration.EventTypeVesselUpdated, configuration.CreateVesselStatusEventProvider),
		newGroup(l, "instance-routes", configuration.EventTypeInstanceRouteUpdated, configuration.CreateInstanceRouteStatusEventProvider),
	}
}

func newGroup(
	l logrus.FieldLogger,
	resource string,
	eventType string,
	provider func(tenantId uuid.UUID, eventType string, resourceId string) model.Provider[[]kafka.Message],
) seeder.Group {
	return seeder.Group{
		Name:      resource,
		URLPrefix: "/tenants/configurations/" + resource,
		Subdomains: []seeder.SubdomainAny{
			seeder.AdaptSubdomain[Entry, Entry](NewSubdomain(resource)),
		},
		AfterSeed: afterSeed(l, eventType, provider),
	}
}

// afterSeed enqueues exactly ONE configuration-status event per seed run.
// The atlas-transports handler for these events is ClearTenant + full
// reload, so one event per file would trigger N reloads and — worse — a
// reload could land mid-seed and load a partial set.
//
// The context here is libs/atlas-seeder's tenant-bearing background
// context, so outbox.EnqueueBuffer snapshots the four tenant headers by
// construction. That is the whole point of routing the emit through
// AfterSeed rather than through BulkCreate.
//
// libs/atlas-seeder's runSubdomain deletes the tenant's entire resource row
// set BEFORE walking the catalog, and Walk returns (nil, nil) for a missing
// directory — so a seed run against an absent/unmounted catalog mount scores
// Created=0, Deleted=N and classifyOutcome still calls that "success"
// (nothing failed). Emitting from that state would tell atlas-transports'
// ClearTenant + full-reload consumer to reload a NOW-EMPTY resource,
// deleting the live Redis registry (boats/planes stop) while the seed_state
// row and status endpoint claim the run succeeded. Guard on the result
// counts: Created==0 && Deleted>0 is that dangerous case and must not emit.
// Created==0 && Deleted==0 is a legitimately-empty first seed of a
// never-seeded tenant (nothing lost, nothing to protect) and must still
// emit normally.
func afterSeed(
	l logrus.FieldLogger,
	eventType string,
	provider func(tenantId uuid.UUID, eventType string, resourceId string) model.Provider[[]kafka.Message],
) func(context.Context, *gorm.DB, seeder.Result) error {
	return func(ctx context.Context, db *gorm.DB, res seeder.Result) error {
		t, err := tenant.FromContext(ctx)()
		if err != nil {
			return err
		}
		var created, deleted int64
		for _, c := range res.Subdomains {
			created += c.Created
			deleted += c.Deleted
		}
		if created == 0 && deleted > 0 {
			l.WithFields(logrus.Fields{
				"tenant_id":  t.Id(),
				"group_name": res.GroupName,
				"created":    created,
				"deleted":    deleted,
			}).Error("seed run deleted existing rows but created none; suspected missing/unhealthy catalog mount — skipping AfterSeed emit to protect the live atlas-transports registry; re-seed once the catalog mount is confirmed healthy")
			return fmt.Errorf("afterSeed: group %q deleted %d row(s) but created 0; refusing to emit configuration-status (suspected catalog mount failure)", res.GroupName, deleted)
		}
		return database.ExecuteTransaction(db.WithContext(ctx), func(tx *gorm.DB) error {
			return message.Emit(outbox.EmitProvider(l, ctx, tx))(func(mb *message.Buffer) error {
				return mb.Put(configuration.EventTopicConfigurationStatus, provider(t.Id(), eventType, seededResourceId))
			})
		})
	}
}
