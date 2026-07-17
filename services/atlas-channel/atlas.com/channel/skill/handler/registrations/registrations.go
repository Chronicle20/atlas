// Package registrations exists solely to drive init() registration of
// per-skill handler subpackages. main.go blank-imports this package;
// each new handler subpackage is added below as a blank import.
package registrations

import (
	_ "atlas-channel/skill/handler/heal"         // Cleric Heal — task 045
	_ "atlas-channel/skill/handler/healdispel"   // SuperGM Heal + Dispel — task-156
	_ "atlas-channel/skill/handler/mysticdoor"   // Priest Mystic Door — task-093
	_ "atlas-channel/skill/handler/resurrection" // Bishop/GM/SuperGM Resurrection — task-111
)
