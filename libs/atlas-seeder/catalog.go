package seeder

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// CatalogSource abstracts the origin of seed data, allowing filesystem and
// in-memory (test) implementations to be swapped transparently.
type CatalogSource interface {
	Roots(t tenant.Model) ([]string, error)
	Revision(root string) (string, error)
	Open(root, relPath string) (io.ReadCloser, error)
	Walk(root, relPath string) ([]string, error)
}

type filesystemSource struct {
	envVar       string
	fallbackRoot string
}

// NewFilesystemCatalogSource returns a CatalogSource that resolves its base
// directory from envVar when set, otherwise from fallbackRoot.
func NewFilesystemCatalogSource(envVar, fallbackRoot string) CatalogSource {
	return &filesystemSource{envVar: envVar, fallbackRoot: fallbackRoot}
}

func (s *filesystemSource) base() string {
	if v := os.Getenv(s.envVar); v != "" {
		return v
	}
	abs, err := filepath.Abs(s.fallbackRoot)
	if err != nil {
		return s.fallbackRoot
	}
	return abs
}

// Roots returns the tenant-specific root directory path(s) under the base.
// Returns an error when the tenant has a zero major or minor version.
//
// The region segment is lower-cased to match the catalog directory layout
// emitted by tools/seed-splitters (e.g. "gms/83_1"). Tenants store their
// region in an unspecified casing — production tenants use uppercase
// ("GMS") while tests use lowercase ("gms") — and the catalog resolver
// normalizes both to a single canonical form so a Linux filesystem (which
// is case-sensitive) can find the directory either way.
func (s *filesystemSource) Roots(t tenant.Model) ([]string, error) {
	if t.MajorVersion() == 0 || t.MinorVersion() == 0 {
		return nil, fmt.Errorf("catalog: tenant has zero major/minor version (region=%q)", t.Region())
	}
	root := filepath.Join(s.base(), strings.ToLower(t.Region()), fmt.Sprintf("%d_%d", t.MajorVersion(), t.MinorVersion()))
	return []string{root}, nil
}

// Revision reads the CATALOG_REVISION file from root. Returns ("", nil) when
// the file does not exist.
func (s *filesystemSource) Revision(root string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, "CATALOG_REVISION"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// Open returns a ReadCloser for the file at root/relPath.
func (s *filesystemSource) Open(root, relPath string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(root, relPath))
}

// Walk returns the sorted names of non-underscore, non-dot .json files
// directly inside root/relPath. Subdirectories are not descended into.
func (s *filesystemSource) Walk(root, relPath string) ([]string, error) {
	dir := filepath.Join(root, relPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		out = append(out, name)
	}
	return out, nil
}
