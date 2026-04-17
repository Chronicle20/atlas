package extraction

import (
	"fmt"
	"path/filepath"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TenantPath(t tenant.Model) string {
	version := fmt.Sprintf("%d.%d", t.MajorVersion(), t.MinorVersion())
	return filepath.Join(t.Id().String(), t.Region(), version)
}

func ResolveTenantInputDir(root string, t tenant.Model) string {
	return filepath.Join(root, TenantPath(t))
}

func ResolveTenantOutputDir(root string, t tenant.Model) string {
	return filepath.Join(root, TenantPath(t))
}

func TenantKey(t tenant.Model) string {
	return fmt.Sprintf("%s:%s:%d.%d", t.Id().String(), t.Region(), t.MajorVersion(), t.MinorVersion())
}
