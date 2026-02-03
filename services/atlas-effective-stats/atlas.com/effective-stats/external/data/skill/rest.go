package skill

import (
	"strconv"
)

// RestModel represents skill data from atlas-data service
type RestModel struct {
	Id            uint32        `json:"-"`
	Name          string        `json:"name"`
	Action        bool          `json:"action"`
	Element       string        `json:"element"`
	AnimationTime uint32        `json:"animationTime"`
	Effects       []EffectModel `json:"effects"`
}

func (r RestModel) GetName() string {
	return "skills"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// IsPassive returns true if this skill is a passive skill (not an action skill)
func (r RestModel) IsPassive() bool {
	return !r.Action
}

// GetEffectForLevel returns the effect for a given skill level (1-indexed)
// Returns nil if the level is out of range
func (r RestModel) GetEffectForLevel(level byte) *EffectModel {
	if level == 0 || int(level) > len(r.Effects) {
		return nil
	}
	return &r.Effects[level-1]
}

// EffectModel represents skill effect data with stat bonuses
type EffectModel struct {
	WeaponAttack   int16         `json:"weaponAttack"`
	MagicAttack    int16         `json:"magicAttack"`
	WeaponDefense  int16         `json:"weaponDefense"`
	MagicDefense   int16         `json:"magicDefense"`
	Accuracy       int16         `json:"accuracy"`
	Avoidability   int16         `json:"avoidability"`
	Speed          int16         `json:"speed"`
	Jump           int16         `json:"jump"`
	HP             uint16        `json:"hp"`
	MP             uint16        `json:"mp"`
	HPR            float64       `json:"hpR"`
	MPR            float64       `json:"mpR"`
	Duration       int32         `json:"duration"`
	X              int16         `json:"x"`
	Y              int16         `json:"y"`
	Statups        []StatupModel `json:"statups"`
}

// StatupModel represents a single stat bonus from a skill effect
type StatupModel struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}
