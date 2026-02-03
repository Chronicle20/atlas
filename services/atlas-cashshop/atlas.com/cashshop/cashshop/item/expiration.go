package item

import (
	"time"
)

// CalculateExpiration determines the expiration time for a cash item based on:
// - period: the commodity's period value (in days, or 1 for hourly items)
// - templateId: the item template ID (used to look up hourly config)
// - hourlyConfig: a map of templateId -> hours for special hourly items
//
// Logic:
// - period == 0: permanent item (no expiration, returns zero time)
// - period != 1: standard day-based expiration (returns now + period days)
// - period == 1: check hourly config for special handling, otherwise 1 day
func CalculateExpiration(period uint32, templateId uint32, hourlyConfig map[uint32]uint32) time.Time {
	now := time.Now()

	// Period of 0 means permanent (no expiration)
	if period == 0 {
		return time.Time{} // Zero time = no expiration
	}

	// Period != 1: standard day-based expiration
	if period != 1 {
		return now.AddDate(0, 0, int(period))
	}

	// Period == 1: check for hourly override
	if hours, ok := hourlyConfig[templateId]; ok {
		return now.Add(time.Duration(hours) * time.Hour)
	}

	// Default: 1 day
	return now.AddDate(0, 0, 1)
}
