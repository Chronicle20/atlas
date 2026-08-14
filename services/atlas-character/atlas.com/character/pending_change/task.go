package pending_change

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ExpiryTask names the otel span and the tasks.Register log line for the
// pending-change expiry sweep.
const ExpiryTask = "pending-change-expiry"

// tenantFetcher resolves a tenant's real Model (region + version) from its
// id. The default fetcher hits atlas-tenants; tests inject a stub via
// newExpiryWithFetcher so the sweep can be exercised without a live HTTP
// call, mirroring configuration.newRegistryWithFetcher.
type tenantFetcher func(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) (tenant.Model, error)

// Expiry periodically resolves every PENDING pending-change request whose
// ExpiresAt has passed to EXPIRED, via Processor.Sweep -- which routes the
// transition through ResolveAndEmit, the same guard the operator-cancel path
// uses, so a tick that re-observes an already-resolved row refunds nothing a
// second time.
//
// Unlike session.Timeout, there is no in-memory registry of live pending
// changes to walk: the candidate set is whichever tenants currently have an
// expired PENDING row, discovered directly from the table on every tick.
// Sweep asserts tenant.MustFromContext, so Run never queries untenanted and
// never reuses one tenant's context for another tenant's rows -- each tenant
// id found gets its own real tenant.Model (fetched from atlas-tenants) and
// its own tenant.WithContext before Sweep is called.
type Expiry struct {
	l           logrus.FieldLogger
	db          *gorm.DB
	interval    time.Duration
	fetchTenant tenantFetcher
}

// NewExpiry constructs the expiry sweep ticker. interval is the tasks.Task
// tick period (SleepTime); 15 minutes is chosen against a 7-day default
// expiry -- the sweep's latency budget is hours, and a tighter interval buys
// nothing but load.
func NewExpiry(l logrus.FieldLogger, db *gorm.DB, interval time.Duration) *Expiry {
	return newExpiryWithFetcher(l, db, interval, fetchTenant)
}

// newExpiryWithFetcher is the test seam: it constructs an Expiry with an
// explicit tenantFetcher so Run can be exercised across multiple tenants
// without a live atlas-tenants call.
func newExpiryWithFetcher(l logrus.FieldLogger, db *gorm.DB, interval time.Duration, f tenantFetcher) *Expiry {
	l.Infof("Initializing pending-change expiry sweep task to run every %s.", interval)
	return &Expiry{l: l, db: db, interval: interval, fetchTenant: f}
}

// Run executes one sweep tick: it enumerates the distinct tenants with an
// expired PENDING row, resolves each tenant's real Model, and sweeps each
// tenant under its own context. A tenant whose Model cannot be resolved (or
// whose sweep errors) is logged and skipped -- one bad tenant must not stall
// the rest of the tick.
func (e *Expiry) Run() {
	sctx, span := otel.GetTracerProvider().Tracer("atlas-character").Start(context.Background(), ExpiryTask)
	defer span.End()

	now := time.Now()

	tenantIds, err := expiredTenantIds(e.db.WithContext(sctx), now)
	if err != nil {
		e.l.WithError(err).Errorf("Unable to enumerate tenants with expired pending changes.")
		return
	}

	for _, tenantId := range tenantIds {
		t, err := e.fetchTenant(e.l, sctx, tenantId)
		if err != nil {
			e.l.WithError(err).Errorf("Unable to resolve tenant [%s] for pending-change expiry sweep; skipping.", tenantId)
			continue
		}

		tctx := tenant.WithContext(sctx, t)
		p := NewProcessor(e.l, tctx, e.db)
		if err := p.Sweep(now); err != nil {
			e.l.WithError(err).Errorf("Pending-change expiry sweep failed for tenant [%s].", tenantId)
		}
	}
}

func (e *Expiry) SleepTime() time.Duration {
	return e.interval
}

// expiredTenantIds returns the distinct tenants with a PENDING pending-change
// row whose deadline has passed. This is deliberately a raw, cross-tenant
// query rather than a provider.go helper -- every other query in this
// package is tenant-scoped because Processor.Sweep itself is
// tenant.MustFromContext-gated and must never be called untenanted; this is
// the one place that legitimately looks across tenants, precisely so the
// per-tenant Sweep calls that follow never are.
func expiredTenantIds(db *gorm.DB, now time.Time) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := db.Model(&entity{}).
		Where("status = ? AND expires_at < ?", StatusPending, now).
		Distinct("tenant_id").
		Pluck("tenant_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// tenantRestModel is the minimal JSON:API shape needed to unmarshal
// GET /tenants/{tenantId} from atlas-tenants -- just region and version, the
// two fields tenant.Create needs beyond the id already known from the query
// above. Downstream Kafka consumers of the refund/notification events
// Processor.Sweep emits rely on the real tenant region/version reaching them
// via the envelope headers, so a placeholder tenant.Model here would corrupt
// those headers for every other service reading the topic.
type tenantRestModel struct {
	Id           string `json:"-"`
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
}

func (r tenantRestModel) GetName() string { return "tenants" }

func (r tenantRestModel) GetID() string { return r.Id }

func (r *tenantRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// fetchTenant is the default tenantFetcher: GET /tenants/{tenantId} against
// atlas-tenants.
func fetchTenant(l logrus.FieldLogger, ctx context.Context, tenantId uuid.UUID) (tenant.Model, error) {
	url := fmt.Sprintf("%stenants/%s", requests.RootUrl("TENANTS"), tenantId.String())
	rm, err := requests.GetRequest[tenantRestModel](url)(l, ctx)
	if err != nil {
		return tenant.Model{}, err
	}
	return tenant.Create(tenantId, rm.Region, rm.MajorVersion, rm.MinorVersion)
}
