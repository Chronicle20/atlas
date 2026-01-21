package validation

import "strconv"

// ConditionInput represents a validation condition to check against character state
type ConditionInput struct {
	Type        string `json:"type"`
	Operator    string `json:"operator"`
	Value       int    `json:"value"`
	Values      []int  `json:"values,omitempty"`
	ReferenceId uint32 `json:"referenceId,omitempty"`
	Step        string `json:"step,omitempty"`
}

// Condition types supported by query-aggregator
const (
	LevelCondition       = "level"
	JobCondition         = "jobId"
	FameCondition        = "fame"
	MesoCondition        = "meso"
	ItemCondition        = "item"
	QuestStatusCondition = "questStatus"
)

// Quest states for QuestStatusCondition
const (
	QuestStateNotStarted = 0
	QuestStateStarted    = 1
	QuestStateCompleted  = 2
)

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
	Id      uint32           `json:"-"`
	Results []ConditionResult `json:"results"`
}

func (r ResponseModel) GetName() string {
	return "validations"
}

func (r ResponseModel) GetID() string {
	return ""
}

func (r *ResponseModel) SetID(id string) error {
	return nil
}

// ConditionResult represents the result of a single condition check
type ConditionResult struct {
	Type   string `json:"type"`
	Passed bool   `json:"passed"`
}

// AllPassed checks if all conditions passed
func (r ResponseModel) AllPassed() bool {
	for _, result := range r.Results {
		if !result.Passed {
			return false
		}
	}
	return true
}

// GetFailedConditions returns the types of conditions that failed
func (r ResponseModel) GetFailedConditions() []string {
	var failed []string
	for _, result := range r.Results {
		if !result.Passed {
			failed = append(failed, result.Type)
		}
	}
	return failed
}
