package element

import (
	"errors"
	"strings"
)

const (
	Neutral  string = "NEUTRAL"
	Physical string = "PHYSICAL"
	Fire     string = "FIRE"
	Ice      string = "ICE"
	Lighting string = "LIGHTING"
	Poison   string = "POISON"
	Holy     string = "HOLY"
	Darkness string = "DARKNESS"

	EffectivenessNormal  string = "NORMAL"
	EffectivenessImmune  string = "IMMUNE"
	EffectivenessStrong  string = "STRONG"
	EffectivenessWeak    string = "WEAK"
	EffectivenessNeutral string = "NEUTRAL"
)

func FromChar(char string) (string, error) {
	switch strings.ToUpper(char) {
	case "F":
		return Fire, nil
	case "I":
		return Ice, nil
	case "L":
		return Lighting, nil
	case "S":
		return Poison, nil
	case "H":
		return Holy, nil
	case "D":
		return Darkness, nil
	case "P":
		return Neutral, nil
	}
	return "", errors.New("unknown element")
}

func EffectivenessByNumber(i int) (string, error) {
	switch i {
	case 1:
		return EffectivenessImmune, nil
	case 2:
		return EffectivenessStrong, nil
	case 3:
		return EffectivenessWeak, nil
	case 4:
		return EffectivenessNeutral, nil
	default:
		return "", errors.New("unknown effectivenes")
	}
}