package item

import (
	"net/url"
	"sort"
	"strings"
)

type filterSpec struct {
	Compartment *Compartment
	Subcategory string
	Class       string
	ClassIsAny  bool
	JobMaskBits uint8
}

var compartmentByToken = map[string]Compartment{
	"equipment": CompartmentEquipment,
	"use":       CompartmentUse,
	"setup":     CompartmentSetup,
	"etc":       CompartmentEtc,
	"cash":      CompartmentCash,
}

var classBitByToken = map[string]uint8{
	"warrior":  1,
	"magician": 2,
	"bowman":   4,
	"thief":    8,
	"pirate":   16,
}

var subcategoryCompartments = buildSubcategoryCompartments()

func buildSubcategoryCompartments() map[string][]Compartment {
	add := func(m map[string][]Compartment, compartment Compartment, tokens ...string) {
		for _, tok := range tokens {
			m[tok] = appendUnique(m[tok], compartment)
		}
	}
	m := map[string][]Compartment{}

	for _, sub := range equipmentArmorByClassification {
		add(m, CompartmentEquipment, sub)
	}
	add(m, CompartmentEquipment,
		"top", "earring",
		"one-handed-sword", "one-handed-axe", "one-handed-mace", "dagger",
		"wand", "staff",
		"two-handed-sword", "two-handed-axe", "two-handed-mace",
		"spear", "polearm",
		"bow", "crossbow",
		"claw", "knuckle", "gun",
		"pet-equip",
	)

	for _, sub := range useByClassification {
		add(m, CompartmentUse, sub)
	}
	for _, sub := range setupByClassification {
		add(m, CompartmentSetup, sub)
	}
	add(m, CompartmentSetup, "other-setup")
	for _, sub := range etcByClassification {
		add(m, CompartmentEtc, sub)
	}
	add(m, CompartmentEtc, "other-etc")
	for _, sub := range cashByClassification {
		add(m, CompartmentCash, sub)
	}
	add(m, CompartmentCash, "other-cash")

	// Plain "other" is a fallback Classify only ever returns for Equipment and Use.
	// Setup/Etc/Cash use "other-setup"/"other-etc"/"other-cash" instead, so accepting
	// bare "other" under those compartments would surface zero rows on every query.
	add(m, CompartmentEquipment, "other")
	add(m, CompartmentUse, "other")

	return m
}

func appendUnique(xs []Compartment, c Compartment) []Compartment {
	for _, x := range xs {
		if x == c {
			return xs
		}
	}
	return append(xs, c)
}

// parseFilters validates query params and returns the spec. errCode is 0 on
// success or 400 on validation failure.
func parseFilters(query url.Values) (filterSpec, int) {
	var spec filterSpec

	if raw := query.Get("filter[compartment]"); raw != "" {
		c, ok := compartmentByToken[strings.ToLower(raw)]
		if !ok {
			return spec, 400
		}
		spec.Compartment = &c
	}

	if raw := query.Get("filter[subcategory]"); raw != "" {
		sub := strings.ToLower(raw)
		comps, known := subcategoryCompartments[sub]
		if !known {
			return spec, 400
		}
		if spec.Compartment != nil {
			matched := false
			for _, c := range comps {
				if c == *spec.Compartment {
					matched = true
					break
				}
			}
			if !matched {
				return spec, 400
			}
		}
		spec.Subcategory = sub
	}

	if raw := query.Get("filter[class]"); raw != "" {
		if spec.Compartment == nil || *spec.Compartment != CompartmentEquipment {
			return spec, 400
		}
		raw = strings.ToLower(raw)
		if raw == "any" {
			spec.Class = "any"
			spec.ClassIsAny = true
		} else {
			tokens := strings.Split(raw, ",")
			sort.Strings(tokens)
			var bits uint8
			for _, tok := range tokens {
				bit, ok := classBitByToken[tok]
				if !ok {
					return spec, 400
				}
				bits |= bit
			}
			spec.Class = strings.Join(tokens, ",")
			spec.JobMaskBits = bits
		}
	}

	return spec, 0
}
