package preset

import (
	"atlas-configurations/data"
	"context"
	"errors"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// ValidationError describes a single rule violation in a preset.
type ValidationError struct {
	PresetId string `json:"presetId"`
	Field    string `json:"field"`
	Message  string `json:"message"`
}

// Validator validates a slice of RestModel presets against the 12 domain rules.
type Validator struct {
	client data.Client
}

// NewValidator constructs a Validator backed by the given atlas-data client.
func NewValidator(client data.Client) *Validator {
	return &Validator{client: client}
}

// Validate inspects the supplied preset list and returns the (possibly mutated)
// list together with any validation errors. The list is mutated in place to
// assign UUIDs to entries with empty Id fields (R-1) — this happens BEFORE
// validation so every error row carries a stable id.
func (v *Validator) Validate(ctx context.Context, presets []RestModel) ([]RestModel, []ValidationError) {
	for i := range presets {
		if presets[i].Id == "" {
			presets[i].Id = uuid.New().String()
		}
	}

	var out []ValidationError
	for _, p := range presets {
		out = append(out, v.validateOne(ctx, p)...)
	}
	return presets, out
}

func (v *Validator) validateOne(ctx context.Context, p RestModel) []ValidationError {
	var errs []ValidationError
	add := func(field, msg string) {
		errs = append(errs, ValidationError{PresetId: p.Id, Field: field, Message: msg})
	}

	// atlas-data lookups are tenant-scoped. atlas-configurations is a
	// bootstrap-tier service that does not require tenant headers on incoming
	// requests, so the template PATCH path has no tenant in ctx — in that case
	// we skip atlas-data-dependent rules (equipment existence, skill MaxLevel)
	// and rely on apply-time validation in atlas-character-factory's
	// CreateFromPreset as the safety net. Structural rules below still run.
	_, tenantErr := tenant.FromContext(ctx)()
	hasTenant := tenantErr == nil

	// R-1: name length 1..64
	if l := len(p.Attributes.Name); l < 1 || l > 64 {
		add("name", "must be 1..64 characters")
	}

	// R-2: description length ≤ 512
	if len(p.Attributes.Description) > 512 {
		add("description", "must be ≤512 characters")
	}

	// R-3: jobId must exist in job.Jobs
	if _, ok := job.Jobs[job.Id(p.Attributes.JobId)]; !ok {
		add("jobId", "unknown job id")
	}

	// R-4: gender in {0, 1}
	if p.Attributes.Gender > 1 {
		add("gender", "must be 0 or 1")
	}

	// R-5: level in [1, 250]
	if p.Attributes.Level < 1 || p.Attributes.Level > 250 {
		add("level", "must be in [1,250]")
	}

	// R-6 + R-7: equipment templateId exists + equippable + slot uniqueness.
	// Slot bucket is templateId/10000 (coarse taxonomy; catches duplicate hats,
	// gloves, etc. — the apply-time validator is the safety net for edge cases).
	// Existence + equippable checks require atlas-data and are skipped without
	// a tenant context; slot uniqueness is purely structural and always runs.
	seenSlots := map[uint32]uint32{}
	for i, eq := range p.Attributes.Equipment {
		if hasTenant {
			info, err := v.client.GetItemById(ctx, eq.TemplateId)
			if err != nil {
				if errors.Is(err, data.ErrNotFound) {
					add(fieldPath("equipment", i, "templateId"), "item not found in atlas-data")
				} else {
					add(fieldPath("equipment", i, "templateId"), "atlas-data lookup failed: "+err.Error())
				}
				continue
			}
			if !info.Equipable {
				add(fieldPath("equipment", i, "templateId"), "item is not equippable")
				continue
			}
		}
		slotBucket := eq.TemplateId / 10000
		if other, exists := seenSlots[slotBucket]; exists {
			add(fieldPath("equipment", i, "templateId"), "equipment slot collision with "+strconv.FormatUint(uint64(other), 10))
		} else {
			seenSlots[slotBucket] = eq.TemplateId
		}
	}

	// R-8 + R-9: inventory templateId exists; quantity ≥ 1.
	// Existence requires atlas-data; quantity check is structural.
	for i, it := range p.Attributes.Inventory {
		if hasTenant {
			if _, err := v.client.GetItemById(ctx, it.TemplateId); err != nil {
				if errors.Is(err, data.ErrNotFound) {
					add(fieldPath("inventory", i, "templateId"), "item not found in atlas-data")
				} else {
					add(fieldPath("inventory", i, "templateId"), "atlas-data lookup failed: "+err.Error())
				}
			}
		}
		if it.Quantity < 1 {
			add(fieldPath("inventory", i, "quantity"), "must be ≥1")
		}
	}

	// R-10 + R-11 + R-12: skill ids exist; level in [1, maxLevel]; batch error.
	// Entirely atlas-data-dependent — skipped without a tenant context.
	if hasTenant && len(p.Attributes.Skills) > 0 {
		ids := make([]uint32, 0, len(p.Attributes.Skills))
		for _, s := range p.Attributes.Skills {
			ids = append(ids, s.SkillId)
		}
		got, err := v.client.GetSkillsByIds(ctx, ids)
		if err != nil {
			add("skills", "atlas-data lookup failed: "+err.Error())
		} else {
			byId := map[uint32]data.SkillInfo{}
			for _, sk := range got {
				byId[sk.Id] = sk
			}
			for i, s := range p.Attributes.Skills {
				sk, ok := byId[s.SkillId]
				if !ok {
					add(fieldPath("skills", i, "skillId"), "skill not found in atlas-data")
					continue
				}
				if s.Level < 1 || s.Level > sk.MaxLevel {
					add(fieldPath("skills", i, "level"), "must be in [1,maxLevel]")
				}
			}
		}
	}

	return errs
}

// fieldPath formats an array-indexed field path, e.g. "equipment[2].templateId".
func fieldPath(arr string, i int, name string) string {
	return arr + "[" + strconv.Itoa(i) + "]." + name
}
