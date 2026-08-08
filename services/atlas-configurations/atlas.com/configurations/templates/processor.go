package templates

import (
	configsocket "atlas-configurations/socket"
	"atlas-configurations/templates/characters/preset"
	"atlas-configurations/templates/socket"
	"context"
	"encoding/json"

	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type Processor interface {
	WithValidator(v *preset.Validator) Processor
	WithCatalog(c Catalog) Processor
	ByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[RestModel]
	ByIdProvider(templateId uuid.UUID) model.Provider[RestModel]
	AllProvider(page model.Page) model.Provider[model.Paged[RestModel]]
	ViewByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[ViewRestModel]
	ViewByIdProvider(templateId uuid.UUID) model.Provider[ViewRestModel]
	AllViewProvider(page model.Page) model.Provider[model.Paged[ViewRestModel]]
	GetByRegionAndVersion(region string, majorVersion uint16, minorVersion uint16) (RestModel, error)
	GetById(templateId uuid.UUID) (RestModel, error)
	Create(input RestModel) (uuid.UUID, error)
	UpdateById(templateId uuid.UUID, input RestModel) error
	DeleteById(templateId uuid.UUID) error
}

type ProcessorImpl struct {
	l         logrus.FieldLogger
	ctx       context.Context
	db        *gorm.DB
	validator *preset.Validator
	catalog   Catalog
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

// WithCatalog injects the shipped-template catalog used to compute drift.
// Unset means the zero Catalog, which reports "no shipped file" for every
// template - the FR-2.4 behaviour - so an un-wired processor degrades safely
// rather than panicking.
func (p *ProcessorImpl) WithCatalog(c Catalog) Processor {
	p.catalog = c
	return p
}

func (p *ProcessorImpl) ByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[RestModel] {
	return model.Map(Make)(byRegionVersionEntityProvider(p.ctx)(region, majorVersion, minorVersion)(p.db))
}

func (p *ProcessorImpl) ByIdProvider(templateId uuid.UUID) model.Provider[RestModel] {
	return model.Map(Make)(byIdEntityProvider(p.ctx)(templateId)(p.db))
}

// AllProvider returns a paged provider for every configuration template.
func (p *ProcessorImpl) AllProvider(page model.Page) model.Provider[model.Paged[RestModel]] {
	return model.MapPaged(Make)(getAll(p.ctx, page)(p.db))(model.ParallelMap())
}

// makeView decorates a RestModel with its revisions and drift flag (FR-2).
// Drift is computed on read; nothing is persisted, so there is no cache to
// invalidate and no state that can itself go stale.
func (p *ProcessorImpl) makeView(rm RestModel) (ViewRestModel, error) {
	stored, err := Revision(rm)
	if err != nil {
		return ViewRestModel{}, err
	}

	v := ViewRestModel{RestModel: rm, StoredRevision: stored}
	if entry, ok := p.catalog.Lookup(rm.Region, rm.MajorVersion, rm.MinorVersion); ok {
		v.ShippedRevision = entry.Revision
		v.SeedDrift = entry.Revision != stored
	}
	// No catalog entry: shippedRevision stays empty and seedDrift stays false.
	// Absence of a shipped file is not drift (FR-2.4).
	return v, nil
}

func (p *ProcessorImpl) ViewByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[ViewRestModel] {
	return model.Map(p.makeView)(p.ByRegionAndVersionProvider(region, majorVersion, minorVersion))
}

func (p *ProcessorImpl) ViewByIdProvider(templateId uuid.UUID) model.Provider[ViewRestModel] {
	return model.Map(p.makeView)(p.ByIdProvider(templateId))
}

// AllViewProvider maps over AllProvider, which already runs ParallelMap, so
// the per-row SHA-256 is parallel across the page (NFR-2) without a cache.
func (p *ProcessorImpl) AllViewProvider(page model.Page) model.Provider[model.Paged[ViewRestModel]] {
	return model.MapPaged(p.makeView)(p.AllProvider(page))(model.ParallelMap())
}

func Make(e Entity) (RestModel, error) {
	var rm RestModel
	err := json.Unmarshal(e.Data, &rm)
	if err != nil {
		return RestModel{}, err
	}
	rm.Socket = socket.Normalize(rm.Socket)
	rm.Id = e.Id.String()
	return rm, nil
}

func (p *ProcessorImpl) GetByRegionAndVersion(region string, majorVersion uint16, minorVersion uint16) (RestModel, error) {
	return p.ByRegionAndVersionProvider(region, majorVersion, minorVersion)()
}

func (p *ProcessorImpl) GetById(templateId uuid.UUID) (RestModel, error) {
	return p.ByIdProvider(templateId)()
}

// canonicalBytes applies the write-path normalization and validation and
// returns the EXACT bytes Create persists. Both Create and ReseedById call it,
// which is what makes a re-seeded row byte-identical to a freshly seeded one.
//
// UpdateById deliberately does NOT call it: UpdateById additionally runs the
// preset validator, which reassigns input.Characters.Presets before
// marshalling. Re-seeding through that path would persist bytes differing from
// the shipped file, and the row would report drift again the instant it was
// reset (FR-3.4).
func canonicalBytes(input RestModel) (json.RawMessage, error) {
	input.Socket = socket.Normalize(input.Socket)
	if issues := socketValidate(input.Socket); len(issues) > 0 {
		return nil, &validationFailureError{socketIssues: issues}
	}

	res, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(res), nil
}

func (p *ProcessorImpl) Create(input RestModel) (uuid.UUID, error) {
	data, err := canonicalBytes(input)
	if err != nil {
		return uuid.Nil, err
	}

	// Generate UUID in Go for database portability
	templateId := uuid.New()
	err = database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		e := &Entity{
			Id:           templateId,
			Region:       input.Region,
			MajorVersion: input.MajorVersion,
			MinorVersion: input.MinorVersion,
			Data:         data,
		}
		return db.Create(e).Error
	})
	if err != nil {
		return uuid.Nil, err
	}
	return templateId, nil
}

func (p *ProcessorImpl) UpdateById(templateId uuid.UUID, input RestModel) error {
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

	return database.ExecuteTransaction(p.db, update(p.ctx, templateId, input.Region, input.MajorVersion, input.MinorVersion, *rm))
}

func (p *ProcessorImpl) DeleteById(templateId uuid.UUID) error {
	return database.ExecuteTransaction(p.db, delete(p.ctx, templateId))
}

// socketValidate runs the shared, dependency-free socket rules. Unlike preset
// validation it is not routed through WithValidator, because it needs no
// atlas-data client and must therefore never be skippable.
func socketValidate(rm socket.RestModel) []configsocket.Issue {
	return configsocket.Validate(socket.ToValidationInput(rm))
}
