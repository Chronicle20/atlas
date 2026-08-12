// Package registrations exists solely to drive init() registration of
// per-skill handler subpackages. main.go blank-imports this package;
// each new handler subpackage is added below as a blank import.
package registrations

import (
	_ "atlas-channel/skill/handler/dispel"       // Priest Dispel party cure — task-163
	_ "atlas-channel/skill/handler/echoofhero"   // Echo of Hero map-wide — task-162
	_ "atlas-channel/skill/handler/flamegear"    // Blaze Wizard Flame Gear — task-218
	_ "atlas-channel/skill/handler/heal"         // Cleric Heal — task 045
	_ "atlas-channel/skill/handler/healdispel"   // SuperGM Heal + Dispel — task-156
	_ "atlas-channel/skill/handler/hide"         // SuperGM Hide — task-156
	_ "atlas-channel/skill/handler/mprecovery"   // Brawler MP Recovery — task-151
	_ "atlas-channel/skill/handler/mysticdoor"   // Priest Mystic Door — task-093
	_ "atlas-channel/skill/handler/poisonbomb"   // Night Walker Poison Bomb — task-218
	_ "atlas-channel/skill/handler/poisonmist"   // Fire/Poison Mage Poison Mist — task-200
	_ "atlas-channel/skill/handler/resurrection" // Bishop/GM/SuperGM Resurrection — task-111
	_ "atlas-channel/skill/handler/timeleap"     // Buccaneer Time Leap — task-155
)
