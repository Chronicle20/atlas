package validation

import (
	"strconv"

	sharedsaga "github.com/Chronicle20/atlas-saga"
)

const (
	ItemCondition = sharedsaga.ItemCondition
)

// ConditionInput is the wire format for validation condition inputs.
// Re-exported from atlas-saga shared library.
type ConditionInput = sharedsaga.ValidationConditionInput

// RequestModel represents the validation request to query-aggregator
type RequestModel struct {
	Id         uint32           `json:"-"`
	Conditions []ConditionInput `json:"conditions"`
}

func (r RequestModel) GetName() string {
	return "validations"
}

func (r RequestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

func (r *RequestModel) SetID(id string) error {
	parsed, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(parsed)
	return nil
}

// ResponseModel represents the validation response from query-aggregator
type ResponseModel struct {
	Id      uint32            `json:"-"`
	Passed  bool              `json:"passed"`
	Results []ConditionResult `json:"results"`
}

func (r ResponseModel) GetName() string {
	return "validations"
}

func (r ResponseModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

func (r *ResponseModel) SetID(id string) error {
	parsed, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(parsed)
	return nil
}

// ConditionResult represents the result of a single condition check
type ConditionResult struct {
	Type   string `json:"type"`
	Passed bool   `json:"passed"`
}

// AllPassed checks if all conditions passed
func (r ResponseModel) AllPassed() bool {
	return r.Passed
}
