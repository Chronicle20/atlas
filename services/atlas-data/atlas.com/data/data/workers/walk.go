package workers

import (
	"context"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/sirupsen/logrus"
)

// findSub locates a SubProperty by name among the given properties.
func findSub(props []property.Property, name string) *property.SubProperty {
	for _, p := range props {
		if sub, ok := p.(*property.SubProperty); ok && sub.Name() == name {
			return sub
		}
	}
	return nil
}

// RegisterFunc mirrors atlas-data/data.RegisterFunc (curried path consumer).
type RegisterFunc func(l logrus.FieldLogger) func(ctx context.Context) func(path string) error

// registerAllInDirectory walks dir and calls rf for every regular file. Errors
// from individual files are logged and do not abort the walk; only the directory
// walk itself can fail.
func registerAllInDirectory(l logrus.FieldLogger, ctx context.Context, dir string, rf RegisterFunc) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".img.xml") {
			return nil
		}
		if err := rf(l)(ctx)(path); err != nil {
			l.WithError(err).Warnf("register %s", filepath.Base(path))
		}
		return nil
	})
}

// imgID parses a WZ image name into a numeric id. wz.Directory.parseDirectory
// strips the ".img" suffix when constructing the tree (directory.go:127), so
// names reach this helper *without* the suffix — e.g. "0100100", not
// "0100100.img". For tolerance with raw WZ paths fed from XML-derived code,
// the suffix is also accepted. Returns (0, false) for non-numeric names like
// "MobSkill" or "BFSkill".
func imgID(name string) (uint32, bool) {
	name = strings.TrimSuffix(name, ".img")
	id, err := strconv.ParseUint(name, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint32(id), true
}
