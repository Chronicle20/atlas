package character

import (
	"atlas-buffs/buff/stat"
)

var diseaseStatTypes = map[string]bool{
	"STUN": true, "POISON": true, "SEAL": true, "DARKNESS": true,
	"WEAKEN": true, "CURSE": true, "SEDUCE": true, "CONFUSE": true,
	"UNDEAD": true, "SLOW": true, "STOP_PORTION": true,
}

func isDiseaseChange(changes []stat.Model) bool {
	for _, c := range changes {
		if diseaseStatTypes[c.Type()] {
			return true
		}
	}
	return false
}

func hasImmunityBuff(m Model) bool {
	for _, b := range m.buffs {
		for _, c := range b.Changes() {
			if c.Type() == "HOLY_SHIELD" {
				return true
			}
		}
	}
	return false
}
