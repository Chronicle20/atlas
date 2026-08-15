// Package atlasdatabase is a minimal stub of
// github.com/Chronicle20/atlas/libs/atlas-database for analysistest, so
// testdata fixtures can import the real package path and call the real
// function name without pulling in the actual module (and its gorm
// dependency) as a test-only transitive dependency of this tool.
package atlasdatabase

import "context"

// WithoutTenantFilter mirrors libs/atlas-database/tenant_scope.go's real
// signature.
func WithoutTenantFilter(ctx context.Context) context.Context {
	return ctx
}
