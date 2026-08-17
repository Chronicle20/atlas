// Package missingbootstrap is a reference fixture for env-bootstrap-guard.sh
// (not wired into analysistest — the bootstrap check is a source scan, not
// an analyzer, per the task-28 brief). It documents the exact shape the
// guard flags: a main.go that calls service.Bootstrap without also passing
// service.WithEnvironmentRegistry.
package missingbootstrap

// Bootstrap and WithEnvironmentRegistry stand in for
// github.com/Chronicle20/atlas/libs/atlas-service's real Bootstrap/Option
// surface, just enough for this fixture to read the way a real main.go
// does.
type Option func()

func Bootstrap(serviceName string, opts ...Option) {}

func WithEnvironmentRegistry(serviceName string) Option { return nil }

func main() {
	// Missing service.WithEnvironmentRegistry(serviceName) — this is the
	// shape tools/env-bootstrap-guard.sh flags across all ~64 services until
	// Phase C wires the registry into every one.
	Bootstrap("missingbootstrap")
}
