// Package env is a minimal stub of github.com/Chronicle20/atlas/libs/atlas-env
// for analysistest — just enough surface (a single exported value) for the
// domainimport/ok fixtures to reference so the import type-checks.
package env

// Key stands in for whatever atlas-env symbol a caller happens to reference;
// the guard bans the import itself, not any particular symbol.
var Key string
