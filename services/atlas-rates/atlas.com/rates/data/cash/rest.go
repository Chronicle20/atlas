package cash

import (
	"strconv"
)

type SpecType string

const (
	SpecTypeRate = SpecType("rate") // Rate multiplier (e.g., 2 for 2x)
	SpecTypeTime = SpecType("time") // Active duration in minutes
)

// RestModel represents cash item data from atlas-data
type RestModel struct {
	Id   uint32             `json:"-"`
	Spec map[SpecType]int32 `json:"spec"`
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

// HasRateProperties returns true if this cash item has rate/time properties
func (r RestModel) HasRateProperties() bool {
	if r.Spec == nil {
		return false
	}
	_, hasRate := r.Spec[SpecTypeRate]
	return hasRate
}

// GetRate returns the rate multiplier, or 0 if not set
func (r RestModel) GetRate() int32 {
	if r.Spec == nil {
		return 0
	}
	return r.Spec[SpecTypeRate]
}

// GetTime returns the active duration in minutes, or 0 if not set
func (r RestModel) GetTime() int32 {
	if r.Spec == nil {
		return 0
	}
	return r.Spec[SpecTypeTime]
}
