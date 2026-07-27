package matrix

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"strings"
)

// UnimplementedRef is one _unimplemented.json entry as it pertains to
// sub-struct dispositions. A ref names a sub-struct EITHER by an explicit
// `packet` path OR by a suffix-qualified `fname`
// ("CScriptMan::OnAskPet#AskPet"). Entries that carry a numeric dispatcher
// `case` and a bare base fname disposition an unbuilt dispatcher ARM, not a
// sub-struct row, and are handled by the validate bijection check — they are
// intentionally NOT resolved here (see ResolveUnimplemented).
type UnimplementedRef struct {
	FName  string `json:"fname"`
	Packet string `json:"packet"`
}

// LoadUnimplemented reads a per-version _unimplemented.json. A missing file
// yields an empty slice with no error so the path can be passed
// unconditionally. Only the fields relevant to sub-struct dispositions
// (fname, packet) are decoded; `case`/`reason` are ignored here.
func LoadUnimplemented(path string) ([]UnimplementedRef, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var doc struct {
		Entries []UnimplementedRef `json:"entries"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	return doc.Entries, nil
}

// BuildIDANameIndex maps every report's full IDAName (including any #suffix) to
// its packet ID, across all versions. PacketID is version-independent
// (pkgdir/WriterName), so a name seen in multiple versions maps to the same
// value; first occurrence wins. Used to resolve suffix-qualified _unimplemented
// fnames to sub-struct packet IDs.
func BuildIDANameIndex(reports map[string]map[string]LoadedReport) map[string]string {
	idx := map[string]string{}
	for _, reps := range reports {
		for _, r := range reps {
			if r.IDAName == "" {
				continue
			}
			if _, ok := idx[r.IDAName]; !ok {
				idx[r.IDAName] = PacketID(r)
			}
		}
	}
	return idx
}

// ResolveUnimplemented maps a version's _unimplemented refs to the set of
// sub-struct packet IDs they disposition. A ref resolves to a packet ID when:
//
//   - it carries an explicit `packet` path (the disposition names the sub-struct
//     directly), or
//   - its `fname` is suffix-qualified (contains '#') and that full IDAName is
//     known in idaToPacket (the suffix disambiguates the sub-struct variant).
//
// A BARE base fname (no '#') is deliberately NOT resolved: such entries
// disposition a dispatcher ARM by (fname, case), and the base name collides
// with the implemented sibling struct's IDAName — matching it would wrongly
// downgrade a built cell to n-a.
func ResolveUnimplemented(refs []UnimplementedRef, idaToPacket map[string]string) map[string]bool {
	out := map[string]bool{}
	for _, r := range refs {
		switch {
		case r.Packet != "":
			out[r.Packet] = true
		case strings.Contains(r.FName, "#"):
			if pid, ok := idaToPacket[r.FName]; ok {
				out[pid] = true
			}
		}
	}
	return out
}
