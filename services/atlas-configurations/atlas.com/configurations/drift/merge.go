package drift

import (
	"errors"
	"fmt"
)

// ErrUnknownSection is returned by ValidateSections for a name that is
// not a comparable section. The reset handler maps it to 400.
//
// There is deliberately no alias: "usesPin" is rejected like any other
// unknown name. An alias would be a permanent second name for a section,
// existing only to paper over an earlier draft.
var ErrUnknownSection = errors.New("unknown section")

// ValidateSections rejects any name that is not comparable. nil and an
// empty slice mean "the whole document" and are always valid.
func ValidateSections(sections []string) error {
	for _, s := range sections {
		if s != Properties && !isNamed(s) {
			return fmt.Errorf("%w: %q", ErrUnknownSection, s)
		}
	}
	return nil
}

// Merge returns stored with the requested sections replaced WHOLESALE by
// base's. A section is replaced key-for-key at the top level, never
// field-merged: "restore to baseline" means the section looks like the
// baseline, not like a union.
//
// nil or empty sections means every comparable section.
//
// Properties is computed over base UNION stored, so a key present on only
// one side is still handled: present in base and not stored -> added;
// present in stored and not base -> removed. Both are correct "restore to
// baseline" outcomes.
func Merge(stored, base Doc, sections []string) (Doc, error) {
	if err := ValidateSections(sections); err != nil {
		return nil, err
	}
	if len(sections) == 0 {
		sections = All()
	}

	out := make(Doc, len(stored))
	for k, v := range stored {
		out[k] = v
	}

	for _, s := range sections {
		if s == Properties {
			for k := range stored {
				if !isNamed(k) {
					delete(out, k)
				}
			}
			for k, v := range base {
				if !isNamed(k) {
					out[k] = v
				}
			}
			continue
		}
		if v, ok := base[s]; ok {
			out[s] = v
		} else {
			delete(out, s)
		}
	}
	return out, nil
}
