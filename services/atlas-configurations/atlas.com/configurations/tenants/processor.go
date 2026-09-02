package tenants

import (
	"atlas-configurations/drift"
	"atlas-configurations/outbox"
	configsocket "atlas-configurations/socket"
	"atlas-configurations/templates"
	"atlas-configurations/tenants/characters/preset"
	"atlas-configurations/tenants/socket"
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	env "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/Chronicle20/atlas/libs/atlas-rest/degrade"
)

// EnvTenantStatusTopic names the env var carrying the Kafka topic that
// tenant config CRUD events are enqueued onto. Unset = enqueue skipped
// (matches the EnvServiceStatusTopic convention in services/processor).
const EnvTenantStatusTopic topic.Token = "EVENT_TOPIC_CONFIGURATION_TENANT_STATUS"

func tenantOutboxKey(id uuid.UUID) []byte {
	return []byte("tenant:" + id.String())
}

func enqueueTenantStatus(tx *gorm.DB, id uuid.UUID, config any) error {
	topic := os.Getenv(string(EnvTenantStatusTopic))
	if topic == "" {
		return nil
	}
	var value []byte
	if config != nil {
		v, err := outbox.NewTenantEnvelope(id, config, time.Now())
		if err != nil {
			return err
		}
		value = v
	}
	return outboxlib.Enqueue(tx, outboxlib.Message{
		Topic: topic,
		Key:   tenantOutboxKey(id),
		Value: value,
	})
}

// Sentinel errors the reset handler maps to HTTP statuses.
// server.WriteErrorResponse maps everything to 500, so the handler
// switches on these and writes the JSON:API error document itself --
// the same arrangement templates/processor.go:22-31 uses.
var (
	// ErrTenantNotFound wraps gorm.ErrRecordNotFound for a tenant id -> 404.
	ErrTenantNotFound = errors.New("tenant not found")
	// ErrNoBaselineTemplate means no template resolves for the tenant's
	// region/version in the caller's environment or its baseline -> 409.
	// There is nothing to reset to.
	ErrNoBaselineTemplate = errors.New("no baseline template")
)

type Processor interface {
	WithValidator(v *preset.Validator) Processor
	WithTemplates(tp templates.Processor) Processor
	ByIdProvider(id uuid.UUID) model.Provider[RestModel]
	ByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[RestModel]
	AllProvider(page model.Page) model.Provider[model.Paged[RestModel]]
	ViewByIdProvider(id uuid.UUID) model.Provider[ViewRestModel]
	AllViewProvider(page model.Page) model.Provider[model.Paged[ViewRestModel]]
	GetById(id uuid.UUID) (RestModel, error)
	GetByRegionAndVersion(region string, majorVersion uint16, minorVersion uint16) (RestModel, error)
	UpdateById(tenantId uuid.UUID, input RestModel) error
	DeleteById(tenantId uuid.UUID) error
	Create(input RestModel) (uuid.UUID, error)
	ResetById(tenantId uuid.UUID, sections []string) (ViewRestModel, error)
}

type ProcessorImpl struct {
	l         logrus.FieldLogger
	ctx       context.Context
	db        *gorm.DB
	validator *preset.Validator
	templates templates.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) WithValidator(v *preset.Validator) Processor {
	p.validator = v
	return p
}

// WithTemplates injects the templates processor used to resolve a
// tenant's baseline. Unset means "no baseline for anything": every row
// reports the FR-1.3 unknown state, so an un-wired processor degrades
// safely rather than nil-panicking. Same contract as
// templates.WithCatalog.
//
// Direction is tenants -> templates and never the reverse; templates
// imports nothing from tenants, so there is no cycle.
func (p *ProcessorImpl) WithTemplates(tp templates.Processor) Processor {
	p.templates = tp
	return p
}

func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[RestModel] {
	return model.Map(Make)(byIdEntityProvider(p.ctx)(id)(p.db))
}

func (p *ProcessorImpl) ByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[RestModel] {
	return model.Map(Make)(byRegionVersionEntityProvider(p.ctx)(region, majorVersion, minorVersion)(p.db))
}

// AllProvider returns a paged provider for every configuration tenant.
func (p *ProcessorImpl) AllProvider(page model.Page) model.Provider[model.Paged[RestModel]] {
	return model.MapPaged(Make)(getAll(p.ctx, page)(p.db))(model.ParallelMap())
}

// baselineFor resolves the template a tenant's region/version derives
// from. Lookup goes through templates.Processor.GetByRegionAndVersion,
// which carries the overlay/baseline environment fallback for free --
// that IS FR-1.1, and re-implementing the query here would be a second
// definition of visibility.
//
// Any failure degrades to "no baseline" (ok == false), never an error to
// the caller: a read must not 404 or 500 because a template is missing or
// the templates table hiccupped (FR-1.4).
func (p *ProcessorImpl) baselineFor(region string, majorVersion uint16, minorVersion uint16) (templates.RestModel, bool) {
	if p.templates == nil {
		return templates.RestModel{}, false
	}
	rm, err := p.templates.GetByRegionAndVersion(region, majorVersion, minorVersion)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			p.l.WithError(err).WithFields(logrus.Fields{
				"region":       region,
				"majorVersion": majorVersion,
				"minorVersion": minorVersion,
			}).Warn("Unable to resolve baseline template; reporting no baseline")
			// baselineFor is a single-tenant helper with no baseline
			// template id to attribute the failure to (the lookup is what
			// failed); entityId=0 signals a component-wide degradation,
			// matching the login.character.rankings precedent.
			degrade.Observe(p.l, "configurations.tenants.templates", 0, err)
		}
		return templates.RestModel{}, false
	}
	return rm, true
}

// decorate builds the view for one tenant against an already-resolved
// baseline. baselineOk == false is the FR-1.3 unknown state: empty
// revisions, empty id, and every flag false -- an unknown, never a true.
func decorate(rm RestModel, baseline templates.RestModel, baselineOk bool) (ViewRestModel, error) {
	stored, err := drift.Canonicalize(rm)
	if err != nil {
		return ViewRestModel{}, err
	}
	storedRev, err := drift.Aggregate(stored)
	if err != nil {
		return ViewRestModel{}, err
	}

	v := ViewRestModel{
		RestModel:      rm,
		StoredRevision: storedRev,
		SectionDrift:   make(map[string]bool, len(drift.Named)+1),
	}
	for _, name := range drift.All() {
		v.SectionDrift[name] = false
	}
	if !baselineOk {
		return v, nil
	}

	base, err := drift.Canonicalize(baseline)
	if err != nil {
		return ViewRestModel{}, err
	}
	baseRev, err := drift.Aggregate(base)
	if err != nil {
		return ViewRestModel{}, err
	}
	agg, per, err := drift.Compare(base, stored)
	if err != nil {
		return ViewRestModel{}, err
	}

	v.BaselineTemplateId = baseline.Id
	v.BaselineRevision = baseRev
	v.TemplateDrift = agg
	v.SectionDrift = per
	return v, nil
}

func (p *ProcessorImpl) makeView(rm RestModel) (ViewRestModel, error) {
	baseline, ok := p.baselineFor(rm.Region, rm.MajorVersion, rm.MinorVersion)
	return decorate(rm, baseline, ok)
}

func (p *ProcessorImpl) ViewByIdProvider(id uuid.UUID) model.Provider[ViewRestModel] {
	return model.Map(p.makeView)(p.ByIdProvider(id))
}

// AllViewProvider builds the page in TWO EXPLICIT PHASES rather than
// resolving a baseline inside ParallelMap (FR-3.4, NFR-1):
//
//  1. read the page,
//  2. collect the distinct {region, major, minor} keys -- realistically
//     1-3 per page -- and resolve each baseline ONCE, serially,
//  3. decorate every row from that map.
//
// A cache consulted from inside ParallelMap would need a mutex and would
// still race two goroutines into the same query. Phase separation makes
// "once per distinct key per request" a property of the control flow
// instead of a property of a lock.
func (p *ProcessorImpl) AllViewProvider(page model.Page) model.Provider[model.Paged[ViewRestModel]] {
	return func() (model.Paged[ViewRestModel], error) {
		paged, err := p.AllProvider(page)()
		if err != nil {
			return model.Paged[ViewRestModel]{}, err
		}

		type key struct {
			region string
			major  uint16
			minor  uint16
		}
		type resolved struct {
			rm templates.RestModel
			ok bool
		}
		baselines := make(map[key]resolved)
		for _, rm := range paged.Items {
			k := key{rm.Region, rm.MajorVersion, rm.MinorVersion}
			if _, seen := baselines[k]; seen {
				continue
			}
			b, ok := p.baselineFor(k.region, k.major, k.minor)
			baselines[k] = resolved{rm: b, ok: ok}
		}

		items := make([]ViewRestModel, 0, len(paged.Items))
		for _, rm := range paged.Items {
			b := baselines[key{rm.Region, rm.MajorVersion, rm.MinorVersion}]
			v, err := decorate(rm, b.rm, b.ok)
			if err != nil {
				return model.Paged[ViewRestModel]{}, err
			}
			items = append(items, v)
		}

		return model.Paged[ViewRestModel]{
			Items: items,
			Total: paged.Total,
			Page:  paged.Page,
		}, nil
	}
}

func Make(e Entity) (RestModel, error) {
	var rm RestModel
	err := json.Unmarshal(e.Data, &rm)
	if err != nil {
		return RestModel{}, err
	}
	rm.Socket = socket.Normalize(rm.Socket)
	rm.Id = e.Id.String()
	// Environment is server-owned (task-232 FR-7.3 / D5): the Entity column
	// always wins over whatever e.Data's JSON blob happened to contain, so a
	// client-supplied "environment" in a past create/update body can never
	// surface as this tenant's environment.
	rm.Environment = e.Environment
	return rm, nil
}

func (p *ProcessorImpl) GetById(id uuid.UUID) (RestModel, error) {
	return p.ByIdProvider(id)()
}

func (p *ProcessorImpl) GetByRegionAndVersion(region string, majorVersion uint16, minorVersion uint16) (RestModel, error) {
	return p.ByRegionAndVersionProvider(region, majorVersion, minorVersion)()
}

func (p *ProcessorImpl) UpdateById(tenantId uuid.UUID, input RestModel) error {
	input.Socket = socket.Normalize(input.Socket)
	issues := socketValidate(input.Socket)

	// The preset validator always runs, even when socket issues already
	// exist, so a single 400 can report every problem the request has -
	// socket and preset failures must never mask each other. This means an
	// atlas-data round trip (p.validator.Validate) now happens on every
	// invalid-socket request too, not just clean ones; that I/O was already
	// paid on every valid-socket request and is not gated behind
	// WithValidator being unset, so the added cost is one otherwise-skippable
	// call on the socket-invalid path.
	var presetErrs []preset.ValidationError
	if p.validator != nil {
		assigned, errs := p.validator.Validate(p.ctx, input.Characters.Presets)
		input.Characters.Presets = assigned
		presetErrs = errs
	}

	if len(issues) > 0 || len(presetErrs) > 0 {
		return &validationFailureError{errors: presetErrs, socketIssues: issues}
	}

	res, err := json.Marshal(input)
	if err != nil {
		return err
	}
	rm := &json.RawMessage{}
	err = rm.UnmarshalJSON(res)
	if err != nil {
		return err
	}

	return database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		if err := update(p.ctx, tenantId, input.Region, input.MajorVersion, input.MinorVersion, *rm)(db); err != nil {
			return err
		}
		// Environment is server-owned (task-232 R21-1): the outbox message
		// published to EVENT_TOPIC_CONFIGURATION_TENANT_STATUS must carry
		// the persisted row's Environment, never whatever the client's
		// request body set. Re-read the row (rather than trusting
		// input.Environment) so libs/atlas-service's tenant->environment
		// projection — and therefore FR-7.7's Reconcile — can never be fed
		// an attacker-controlled value.
		persisted, err := byIdEntityProvider(p.ctx)(tenantId)(db)()
		if err != nil {
			return err
		}
		sanitized := input
		sanitized.Environment = persisted.Environment
		return enqueueTenantStatus(db, tenantId, sanitized)
	})
}

func (p *ProcessorImpl) DeleteById(tenantId uuid.UUID) error {
	return database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		if err := delete(p.ctx, tenantId)(db); err != nil {
			return err
		}
		return enqueueTenantStatus(db, tenantId, nil)
	})
}

func (p *ProcessorImpl) Create(input RestModel) (uuid.UUID, error) {
	input.Socket = socket.Normalize(input.Socket)
	if issues := socketValidate(input.Socket); len(issues) > 0 {
		return uuid.Nil, &validationFailureError{socketIssues: issues}
	}

	res, err := json.Marshal(input)
	if err != nil {
		return uuid.Nil, err
	}
	rm := &json.RawMessage{}
	err = rm.UnmarshalJSON(res)
	if err != nil {
		return uuid.Nil, err
	}

	// Use ID from input if provided, otherwise generate a new one
	var tenantId uuid.UUID
	if input.Id != "" {
		tenantId, err = uuid.Parse(input.Id)
		if err != nil {
			return uuid.Nil, err
		}
	} else {
		tenantId = uuid.New()
	}

	err = database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		e := &Entity{
			Id:           tenantId,
			Region:       input.Region,
			MajorVersion: input.MajorVersion,
			MinorVersion: input.MinorVersion,
			Data:         *rm,
			Environment:  string(env.MustFromContext(p.ctx)),
		}
		if err := db.Create(e).Error; err != nil {
			return err
		}
		// Environment is server-owned (task-232 R21-1): sanitize before
		// publishing so the outbox message never carries whatever the
		// client's request body set for "environment" — see the matching
		// comment in UpdateById.
		sanitized := input
		sanitized.Environment = e.Environment
		return enqueueTenantStatus(db, tenantId, sanitized)
	})
	if err != nil {
		return uuid.Nil, err
	}
	return tenantId, nil
}

// socketValidate runs the shared, dependency-free socket rules. Unlike preset
// validation it is not routed through WithValidator, because it needs no
// atlas-data client and must therefore never be skippable.
func socketValidate(rm socket.RestModel) []configsocket.Issue {
	return configsocket.Validate(socket.ToValidationInput(rm))
}
