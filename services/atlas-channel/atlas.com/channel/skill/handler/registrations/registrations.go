// Package registrations exists solely to drive init() registration of
// per-skill handler subpackages. main.go blank-imports this package;
// each new handler subpackage is added below as a blank import.
package registrations

import (
	_ "atlas-channel/skill/handler/heal" // Cleric Heal — task 045
)
