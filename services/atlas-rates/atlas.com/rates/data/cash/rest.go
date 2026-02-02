package cash

import (
	"strconv"
	"time"
)

type SpecType string

const (
	// Rate coupon properties (EXP coupons in 0521.img, Drop coupons in 0536.img)
	SpecTypeRate = SpecType("rate") // Rate multiplier from info node (e.g., 2 for 2x)
	SpecTypeExpR = SpecType("expR") // EXP rate value from spec node
	SpecTypeDrpR = SpecType("drpR") // Drop rate value from spec node
	SpecTypeTime = SpecType("time") // Duration in minutes from spec node
)

// TimeWindow represents an active time window for a coupon (e.g., "MON:18-20")
type TimeWindow struct {
	Day       string `json:"day"`       // Day of week: MON, TUE, WED, THU, FRI, SAT, SUN, HOL
	StartHour int    `json:"startHour"` // Start hour (0-23)
	EndHour   int    `json:"endHour"`   // End hour (1-24, where 24 means midnight)
}

// RestModel represents cash item data from atlas-data
type RestModel struct {
	Id          uint32             `json:"-"`
	Spec        map[SpecType]int32 `json:"spec"`
	TimeWindows []TimeWindow       `json:"timeWindows,omitempty"` // Active time windows from info/time
}

func (r RestModel) GetName() string {
	return "cash_items"
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

// HasRateProperties returns true if this cash item has rate properties
func (r RestModel) HasRateProperties() bool {
	if r.Spec == nil {
		return false
	}
	_, hasRate := r.Spec[SpecTypeRate]
	return hasRate
}

// GetRate returns the rate multiplier from info/rate, or 0 if not set
func (r RestModel) GetRate() int32 {
	if r.Spec == nil {
		return 0
	}
	return r.Spec[SpecTypeRate]
}

// GetExpR returns the expR value from spec node, or 0 if not set
func (r RestModel) GetExpR() int32 {
	if r.Spec == nil {
		return 0
	}
	return r.Spec[SpecTypeExpR]
}

// GetDrpR returns the drpR value from spec node, or 0 if not set
func (r RestModel) GetDrpR() int32 {
	if r.Spec == nil {
		return 0
	}
	return r.Spec[SpecTypeDrpR]
}

// GetTime returns the active duration in minutes, or 0 if not set
func (r RestModel) GetTime() int32 {
	if r.Spec == nil {
		return 0
	}
	return r.Spec[SpecTypeTime]
}

// GetTimeWindows returns the active time windows for this coupon
func (r RestModel) GetTimeWindows() []TimeWindow {
	return r.TimeWindows
}

// HasTimeWindows returns true if the coupon has time window restrictions
func (r RestModel) HasTimeWindows() bool {
	return len(r.TimeWindows) > 0
}

// dayAbbreviations maps Go's time.Weekday to the WZ day abbreviations
var dayAbbreviations = map[time.Weekday]string{
	time.Monday:    "MON",
	time.Tuesday:   "TUE",
	time.Wednesday: "WED",
	time.Thursday:  "THU",
	time.Friday:    "FRI",
	time.Saturday:  "SAT",
	time.Sunday:    "SUN",
}

// IsActiveAt checks if the coupon is active at a given time based on time windows
// If no time windows are defined, returns true (always active)
// The isHoliday parameter should be true if the given time is a holiday
func (r RestModel) IsActiveAt(t time.Time, isHoliday bool) bool {
	if !r.HasTimeWindows() {
		return true
	}

	hour := t.Hour()
	dayAbbr := dayAbbreviations[t.Weekday()]

	for _, tw := range r.TimeWindows {
		// Check holiday window
		if isHoliday && tw.Day == "HOL" {
			if hour >= tw.StartHour && hour < tw.EndHour {
				return true
			}
			// EndHour of 24 means through midnight
			if tw.EndHour == 24 && hour >= tw.StartHour {
				return true
			}
		}

		// Check regular day window
		if tw.Day == dayAbbr {
			if hour >= tw.StartHour && hour < tw.EndHour {
				return true
			}
			// EndHour of 24 means through midnight
			if tw.EndHour == 24 && hour >= tw.StartHour {
				return true
			}
		}
	}

	return false
}

// IsActiveNow checks if the coupon is currently active based on time windows
// If no time windows are defined, returns true (always active)
func (r RestModel) IsActiveNow() bool {
	return r.IsActiveAt(time.Now(), false)
}
