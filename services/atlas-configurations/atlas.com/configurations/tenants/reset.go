package tenants

import (
	"atlas-configurations/drift"
	"atlas-configurations/tenants/characters/preset"
	"atlas-configurations/tenants/socket"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
)

// ResetById replaces the tenant's stored content, for the requested
// scope, with its baseline template's (FR-4.1). nil or empty sections
// means every comparable section.
//
// It deliberately does NOT go through UpdateById. UpdateById runs the
// preset validator, which assigns a fresh uuid to any preset with an
// empty Id (preset/validator.go:36-47); persisting that output would make
// the tenant differ from the template the instant it was reset, and
// FR-4.10 would fail intermittently -- on exactly the templates that were
// hand-authored rather than round-tripped through a PATCH. This is the
// same trap templates.ReseedById documents at processor.go:152-160,
// arriving from a different direction.
//
// Resolution: run the validator for DETECTION, discard its MUTATION. The
// merged document is persisted verbatim. Consequence, accepted: a
// baseline preset with an empty id is persisted with an empty id, and the
// next ordinary PATCH assigns one -- at which point the tenant genuinely
// has drifted and the flag correctly says so.
func (p *ProcessorImpl) ResetById(tenantId uuid.UUID, sections []string) (ViewRestModel, error) {
	// Validate section names FIRST, before any I/O: a 400 for a typo
	// must not depend on the tenant existing (FR-4.3).
	if err := drift.ValidateSections(sections); err != nil {
		return ViewRestModel{}, err
	}

	e, err := byIdEntityProvider(p.ctx)(tenantId)(p.db)()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Scoped, so a tenant in another environment is a 404, not
			// a 403: a caller who cannot read the row learns nothing
			// about it.
			return ViewRestModel{}, fmt.Errorf("%w: %s", ErrTenantNotFound, tenantId)
		}
		return ViewRestModel{}, err
	}

	// Resolve the baseline from the ENTITY's region/version columns, not
	// the document's: the lookup key must come from the row, so a
	// document/column mismatch can never rewrite the key (same reasoning
	// as templates.ReseedById).
	baseline, ok := p.baselineFor(e.Region, e.MajorVersion, e.MinorVersion)
	if !ok {
		return ViewRestModel{}, fmt.Errorf("%w for %s %d.%d", ErrNoBaselineTemplate, e.Region, e.MajorVersion, e.MinorVersion)
	}

	storedModel, err := Make(e)
	if err != nil {
		return ViewRestModel{}, err
	}

	storedDoc, err := drift.Canonicalize(storedModel)
	if err != nil {
		return ViewRestModel{}, err
	}
	baseDoc, err := drift.Canonicalize(baseline)
	if err != nil {
		return ViewRestModel{}, err
	}
	beforeRevision, err := drift.Aggregate(storedDoc)
	if err != nil {
		// Best-effort, as ReseedById is: the point of a reset is to
		// repair a row, so a row too broken to hash must still be
		// repairable.
		p.l.WithError(err).WithField("tenantId", tenantId.String()).Warn("Unable to compute pre-reset revision")
		beforeRevision = ""
	}

	mergedDoc, err := drift.Merge(storedDoc, baseDoc, sections)
	if err != nil {
		return ViewRestModel{}, err
	}
	mergedBytes, err := json.Marshal(mergedDoc)
	if err != nil {
		return ViewRestModel{}, err
	}

	// Unmarshaling into RestModel discards any key the template document
	// has and the tenant model does not. There are none today; if one is
	// ever added the tenant simply does not gain a field it has no code
	// to use.
	var merged RestModel
	if err := json.Unmarshal(mergedBytes, &merged); err != nil {
		return ViewRestModel{}, err
	}

	// Re-apply everything the merge never carried. FR-4.4 is enforced
	// twice: once because Canonicalize dropped these keys so Merge could
	// not see them, and once here from the stored row.
	merged.Id = storedModel.Id
	merged.Environment = storedModel.Environment
	merged.Region = e.Region
	merged.MajorVersion = e.MajorVersion
	merged.MinorVersion = e.MinorVersion
	merged.Worlds = storedModel.Worlds
	merged.Diagnostics = storedModel.Diagnostics

	merged.Socket = socket.Normalize(merged.Socket)
	if err := p.validateReset(merged); err != nil {
		return ViewRestModel{}, err
	}

	data, err := json.Marshal(merged)
	if err != nil {
		return ViewRestModel{}, err
	}

	err = database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		// update gives history-before-write (FR-4.7) and AuthorizeWrite
		// (FR-4.6) with no new code.
		if err := update(p.ctx, tenantId, e.Region, e.MajorVersion, e.MinorVersion, data)(db); err != nil {
			return err
		}
		// Environment is server-owned: re-read the persisted row rather
		// than trusting the merged document, byte-identical to what
		// UpdateById does (processor.go:162-175).
		persisted, err := byIdEntityProvider(p.ctx)(tenantId)(db)()
		if err != nil {
			return err
		}
		sanitized := merged
		sanitized.Environment = persisted.Environment
		return enqueueTenantStatus(db, tenantId, sanitized)
	})
	if err != nil {
		return ViewRestModel{}, err
	}

	afterRevision, aErr := drift.Aggregate(mergedDoc)
	if aErr != nil {
		afterRevision = ""
	}
	// NFR-6: the change must be reconstructable from logs alone.
	p.l.WithFields(logrus.Fields{
		"tenantId":           tenantId.String(),
		"baselineTemplateId": baseline.Id,
		"sections":           sections,
		"beforeRevision":     beforeRevision,
		"afterRevision":      afterRevision,
	}).Info("Tenant configuration reset to baseline template")

	return p.ViewByIdProvider(tenantId)()
}

// validateReset runs the same validators a PATCH runs (FR-4.9) but
// DISCARDS the preset validator's mutation. Validate assigns a fresh uuid
// to any preset with an empty Id and returns the mutated slice; we hand
// it a copy and keep only the errors, so the merged document is persisted
// verbatim.
func (p *ProcessorImpl) validateReset(merged RestModel) error {
	issues := socketValidate(merged.Socket)

	var presetErrs []preset.ValidationError
	if p.validator != nil {
		// Validate mutates in place, so copy the slice. RestModel
		// elements are values, and only Id is assigned, so a shallow
		// copy is sufficient.
		scratch := make([]preset.RestModel, len(merged.Characters.Presets))
		copy(scratch, merged.Characters.Presets)
		_, presetErrs = p.validator.Validate(p.ctx, scratch)
	}

	if len(issues) > 0 || len(presetErrs) > 0 {
		return &validationFailureError{errors: presetErrs, socketIssues: issues}
	}
	return nil
}
