package charparts

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

// ErrSmapMissing is returned by ExtractSmap when the supplied Base.wz file
// does not contain an "smap.img" entry at its root. Callers may log and
// continue; the absence of smap.json downstream disables vslot-based
// occlusion (equipment claims do not suppress hair/face parts).
var ErrSmapMissing = errors.New("charparts: smap.img not found")

// ExtractSmap reads Base.wz/smap.img and returns the layer-name → slot-codes
// map needed for vslot-based occlusion. The smap is a flat property tree
// where each child's name is a layer-string (matching a part canvas's `z`
// value, e.g. "cap", "hair", "hairOverHead") and the StringProperty value
// is the concatenated two-character slot codes that layer occupies
// (e.g. "CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe").
//
// Returns ErrSmapMissing if smap.img cannot be located in the file's root.
// Donor: services/atlas-wz-extractor/atlas.com/wz-extractor/image/zmap.go
// writeSmap / writeSmapFromProps (zmap.go:58-74).
func ExtractSmap(base *wz.File) (map[string]string, error) {
	if base == nil {
		return nil, fmt.Errorf("charparts.ExtractSmap: nil wz.File")
	}
	root := base.Root()
	if root == nil {
		return nil, ErrSmapMissing
	}
	smapImg := findSmapImage(root.Images())
	if smapImg == nil {
		return nil, ErrSmapMissing
	}
	return smapFromProps(smapImg.Properties()), nil
}

// smapFromProps is the pure transformation: each StringProperty child of the
// smap.img becomes one entry in the output map. Non-string children (rare /
// none in shipped data) are ignored. Donor: writeSmapFromProps (zmap.go:66-74).
func smapFromProps(props []property.Property) map[string]string {
	out := make(map[string]string, len(props))
	for _, p := range props {
		if sp, ok := p.(*property.StringProperty); ok {
			out[sp.Name()] = sp.Value()
		}
	}
	return out
}

// findSmapImage returns the root-level image named "smap" (case-insensitive),
// or nil if no such entry exists. Donor: findImage (zmap.go:76-83) used a
// case-insensitive name compare; we reproduce that so synthetic test fixtures
// can use either capitalization.
func findSmapImage(images []*wz.Image) *wz.Image {
	for _, img := range images {
		// match donor case-insensitive on both raw and ".img"-stripped form.
		if eqFoldStripImg(img.Name(), "smap") {
			return img
		}
	}
	return nil
}

func eqFoldStripImg(a, target string) bool {
	a = stripImgSuffix(a)
	return equalFold(a, target)
}

func stripImgSuffix(s string) string {
	if len(s) >= 4 && (s[len(s)-4:] == ".img" || s[len(s)-4:] == ".IMG") {
		return s[:len(s)-4]
	}
	return s
}

// equalFold avoids pulling strings.EqualFold for two short literals — and
// keeps this file dependency-free of the unicode case tables.
func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// MarshalSmap serializes a smap map[string]string with deterministic key
// ordering. Go's encoding/json sorts map[string]string keys lexically since
// 1.12, so json.Marshal is sufficient — but we wrap it to make the contract
// explicit and to give a single hook point if the schema ever grows beyond
// flat string→string.
func MarshalSmap(m map[string]string) ([]byte, error) {
	if m == nil {
		m = map[string]string{}
	}
	return json.Marshal(m)
}
