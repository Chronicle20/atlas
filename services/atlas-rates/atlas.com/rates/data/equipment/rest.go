package equipment

import (
	"strconv"
)

// BonusExpTier represents a time-based EXP bonus tier from equipment
type BonusExpTier struct {
	IncExpR   int32 `json:"incExpR"`   // EXP bonus percentage (e.g., 10 = +10%)
	TermStart int32 `json:"termStart"` // Hours equipped before this tier activates
}

// RestModel represents equipment data from atlas-data
type RestModel struct {
	Id       uint32         `json:"-"`
	BonusExp []BonusExpTier `json:"bonusExp,omitempty"`
	// Other fields omitted - we only need bonusExp for rate calculation
}

func (r RestModel) GetName() string {
	return "statistics"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// HasBonusExp returns true if this equipment has time-based EXP bonuses
func (r RestModel) HasBonusExp() bool {
	return len(r.BonusExp) > 0
}

// GetBonusExpForHours returns the applicable EXP bonus percentage for the given hours equipped
func (r RestModel) GetBonusExpForHours(hoursEquipped int32) int32 {
	if len(r.BonusExp) == 0 {
		return 0
	}

	// Find the highest tier that applies (highest termStart that is <= hoursEquipped)
	var applicableBonus int32 = 0
	for _, tier := range r.BonusExp {
		if tier.TermStart <= hoursEquipped {
			if tier.IncExpR > applicableBonus {
				applicableBonus = tier.IncExpR
			}
		}
	}
	return applicableBonus
}
