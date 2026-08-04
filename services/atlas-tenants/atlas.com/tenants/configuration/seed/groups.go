package seed

import (
	"atlas-tenants/configuration"
	"atlas-tenants/kafka/message"
	"context"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// seededResourceId is the ResourceId carried by the single
// configuration-status event a seed run emits. atlas-transports switches
// on ResourceType and uses ResourceId only for logging, so a synthetic
// value is correct and reads clearly in a log line.
const seededResourceId = "*"

// InitResource registers POST /<prefix>/seed and GET /<prefix>/seed/status
// for each transport configuration resource. Register this BEFORE
// configuration.RegisterRoutes so the literal seed paths are matched ahead
// of the /tenants/{tenantId}/... patterns.
func InitResource(db *gorm.DB) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		// The three transport resources are version-agnostic: one shared
		// set of files applies to every tenant regardless of region or
		// client version, so the source resolves deploy/seed/shared/all
		// in addition to the tenant's version root.
		src := seeder.NewFilesystemCatalogSourceWithShared("SEED_CATALOG_ROOT", "./deploy/seed", "shared/all")
		for _, g := range Groups(l) {
			seeder.RegisterRoutes(router, db, l, src, g)
		}
	}
}

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
func afterSeed(
	l logrus.FieldLogger,
	eventType string,
	provider func(tenantId uuid.UUID, eventType string, resourceId string) model.Provider[[]kafka.Message],
) func(context.Context, *gorm.DB, seeder.Result) error {
	return func(ctx context.Context, db *gorm.DB, _ seeder.Result) error {
		t, err := tenant.FromContext(ctx)()
		if err != nil {
			return err
		}
		return database.ExecuteTransaction(db.WithContext(ctx), func(tx *gorm.DB) error {
			return message.Emit(outbox.EmitProvider(l, ctx, tx))(func(mb *message.Buffer) error {
				return mb.Put(configuration.EventTopicConfigurationStatus, provider(t.Id(), eventType, seededResourceId))
			})
		})
	}
}
