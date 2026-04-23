package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/mod/modfile"
)

const atlasLibPrefix = "github.com/Chronicle20/atlas/libs/"

// parseAtlasRequires returns the set of atlas-lib short names (e.g. "atlas-kafka")
// that the go.mod at path requires — both direct and indirect.
func parseAtlasRequires(path string) (map[string]struct{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	f, err := modfile.Parse(path, data, nil)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	out := make(map[string]struct{})
	for _, r := range f.Require {
		if strings.HasPrefix(r.Mod.Path, atlasLibPrefix) {
			short := strings.TrimPrefix(r.Mod.Path, atlasLibPrefix)
			out[short] = struct{}{}
		}
	}
	return out, nil
}
