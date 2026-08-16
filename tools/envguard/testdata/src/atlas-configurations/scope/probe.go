// Package scope's import path here ("atlas-configurations/scope") is
// deliberately chosen to collide with a real domainAllowlist key in
// analyzer.go, so this fixture pins the allowlist-hit branch — not the
// kafka/rest/main.go path exemption — the same way testdata/src/domainimport
// pins the violating branch.
package scope

import env "github.com/Chronicle20/atlas/libs/atlas-env"

var _ = env.Key
