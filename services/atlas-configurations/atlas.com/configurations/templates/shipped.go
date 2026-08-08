package templates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/sirupsen/logrus"
)

// CatalogEntry is one seed file as it ships in the running image.
type CatalogEntry struct {
	// FileName is the base name, e.g. "template_gms_83_1.json". Carried so
	// the NFR-3 re-seed log names the source.
	FileName string
	// Model is the parsed document, NOT normalized - Revision and
	// canonicalBytes both normalize, so storing a normalized copy would put a
	// second, silently-diverging normalization point in the tree.
	Model RestModel
	// Revision is Revision(Model), precomputed once at load.
	Revision string
}

type catalogKey struct {
	region string
	major  uint16
	minor  uint16
}

// Catalog is the set of templates baked into this image, keyed by
// (region, majorVersion, minorVersion). The zero value is a usable empty
// catalog: every Lookup misses, which is the FR-2.4 "no shipped file"
// behaviour. An un-wired processor therefore degrades safely.
type Catalog struct {
	byKey   map[catalogKey]CatalogEntry
	ordered []CatalogEntry
}

// Lookup returns the shipped entry for a region/version, if one ships.
func (c Catalog) Lookup(region string, majorVersion uint16, minorVersion uint16) (CatalogEntry, bool) {
	e, ok := c.byKey[catalogKey{region: region, major: majorVersion, minor: minorVersion}]
	return e, ok
}

// Entries returns every entry in file-name sort order. The boot seeder
// iterates this, so the order is the seeder's import order and must stay
// deterministic.
func (c Catalog) Entries() []CatalogEntry {
	out := make([]CatalogEntry, len(c.ordered))
	copy(out, c.ordered)
	return out
}

// Len is the number of entries.
func (c Catalog) Len() int {
	return len(c.ordered)
}

// LoadCatalog reads every *.json file under dir into a catalog. It is pure -
// no globals, no environment - so tests use it directly against a fixture
// directory. It never returns an error: a missing directory yields an empty
// catalog, and a file that fails to parse is logged at ERROR and omitted
// (FR-1.5) rather than failing the load or blocking startup.
func LoadCatalog(l logrus.FieldLogger, dir string) Catalog {
	c := Catalog{byKey: make(map[catalogKey]CatalogEntry)}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			l.WithField("directory", dir).Debug("Shipped template directory does not exist")
			return c
		}
		l.WithError(err).WithField("directory", dir).Error("Unable to read shipped template directory")
		return c
	}

	var names []string
	for _, de := range entries {
		if de.IsDir() {
			continue
		}
		if filepath.Ext(de.Name()) != ".json" {
			continue
		}
		names = append(names, de.Name())
	}
	// Deterministic ordering: this is what makes FR-1.6's "first wins" and the
	// seeder's import order reproducible.
	sort.Strings(names)

	for _, name := range names {
		ll := l.WithField("file", name)

		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			ll.WithError(err).Error("Unable to read shipped template file")
			continue
		}

		var rm RestModel
		if err := json.Unmarshal(b, &rm); err != nil {
			ll.WithError(err).Error("Unable to parse shipped template file")
			continue
		}
		if rm.Region == "" {
			ll.Error("Shipped template file is missing required field: region")
			continue
		}

		rev, err := Revision(rm)
		if err != nil {
			ll.WithError(err).Error("Unable to compute revision for shipped template file")
			continue
		}

		k := catalogKey{region: rm.Region, major: rm.MajorVersion, minor: rm.MinorVersion}
		if existing, dup := c.byKey[k]; dup {
			ll.WithFields(logrus.Fields{
				"region":       rm.Region,
				"majorVersion": rm.MajorVersion,
				"minorVersion": rm.MinorVersion,
				"keeping":      existing.FileName,
			}).Error("Duplicate shipped template key; keeping the first file in sort order")
			continue
		}

		e := CatalogEntry{FileName: name, Model: rm, Revision: rev}
		c.byKey[k] = e
		c.ordered = append(c.ordered, e)
	}

	return c
}

var (
	shippedOnce sync.Once
	shippedMu   sync.RWMutex
	shipped     Catalog
)

// InitShippedCatalog loads the singleton catalog from dir exactly once and
// returns it (FR-1.2). Called from main.go before the seeder runs and before
// routes are registered.
//
// It is deliberately NOT gated on SEED_ENABLED: that flag governs whether
// templates are IMPORTED, not whether the service knows what ships. An
// operator who has disabled seeding still needs the drift badge and the reset
// button.
func InitShippedCatalog(l logrus.FieldLogger, dir string) Catalog {
	shippedOnce.Do(func() {
		c := LoadCatalog(l, dir)

		shippedMu.Lock()
		shipped = c
		shippedMu.Unlock()

		ll := l.WithFields(logrus.Fields{"directory": dir, "count": c.Len()})
		if c.Len() == 0 {
			// Loud, because the symptom is silent: every template reports "no
			// shipped file", the badge never lights up, and re-seed 409s.
			ll.Warn("Shipped template catalog is empty; drift detection and re-seed are inert")
		} else {
			ll.Info("Shipped template catalog loaded")
		}
	})
	return ShippedCatalog()
}

// ShippedCatalog returns the singleton catalog. Before InitShippedCatalog runs
// it is the zero Catalog, which reports "no shipped file" for everything.
func ShippedCatalog() Catalog {
	shippedMu.RLock()
	defer shippedMu.RUnlock()
	return shipped
}
