package validation

import (
	"fmt"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

const (
	Resource = "validations"
)

// RestModel represents the REST model for validation requests and responses
//
// Example request for level validation:
//   {
//     "conditions": [
//       {
//         "type": "level",
//         "operator": ">=",
//         "value": 30
//       }
//     ]
//   }
//
// Example request for quest status validation:
//   {
//     "conditions": [
//       {
//         "type": "questStatus",
//         "operator": "=",
//         "value": 2,
//         "referenceId": 1001
//       }
//     ]
//   }
//
// Example request for quest progress validation:
//   {
//     "conditions": [
//       {
//         "type": "questProgress",
//         "operator": ">=",
//         "value": 5,
//         "referenceId": 1001,
//         "step": "collect_items"
//       }
//     ]
//   }
//
// Example request for guild validation:
//   {
//     "conditions": [
//       {
//         "type": "guildId",
//         "operator": "=",
//         "value": 123
//       },
//       {
//         "type": "guildRank",
//         "operator": "<=",
//         "value": 2
//       }
//     ]
//   }
//
// Example request for marriage gifts validation:
//   {
//     "conditions": [
//       {
//         "type": "hasUnclaimedMarriageGifts",
//         "operator": "=",
//         "value": 1
//       }
//     ]
//   }
//
// Example request for character stats validation:
//   {
//     "conditions": [
//       {
//         "type": "reborns",
//         "operator": ">=",
//         "value": 3
//       },
//       {
//         "type": "dojoPoints",
//         "operator": ">",
//         "value": 1000
//       },
//       {
//         "type": "vanquisherKills",
//         "operator": ">=",
//         "value": 50
//       },
//       {
//         "type": "gmLevel",
//         "operator": ">=",
//         "value": 1
//       }
//     ]
//   }
type RestModel struct {
	Id         uint32            `json:"-"`
	Conditions []ConditionInput  `json:"conditions,omitempty"`
	Passed     bool              `json:"passed"`
	Results    []ConditionResult `json:"results,omitempty"`
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return Resource
}

// GetID returns the resource ID
// For validation results, the character ID is used as the resource ID
func (r RestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

// SetID sets the resource ID
// For validation requests, the ID is parsed and set as the character ID
func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid character ID: %w", err)
	}
	r.Id = uint32(id)
	return nil
}

// GetReferences returns the resource references
func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

// GetReferencedIDs returns the referenced IDs
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

// GetReferencedStructs returns the referenced structs
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

// SetToOneReferenceID sets a to-one reference ID
func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many reference IDs
func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

// SetReferencedStructs sets referenced structs
func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	return nil
}

// Transform converts a domain model to a REST model
func Transform(result ValidationResult) (RestModel, error) {
	return RestModel{
		Id:      result.CharacterId(),
		Passed:  result.Passed(),
		Results: result.Results(),
	}, nil
}

// Extract converts a REST model to domain model parameters for structured validation
func Extract(rm RestModel) (uint32, []ConditionInput, error) {
	// Validate that CharacterId is provided
	if rm.Id == 0 {
		return 0, nil, fmt.Errorf("Id is required")
	}

	// Validate that at least one condition is provided
	if len(rm.Conditions) == 0 {
		return 0, nil, fmt.Errorf("at least one condition is required")
	}

	// Validate each condition input
	for i, condition := range rm.Conditions {
		if err := validateConditionInput(condition); err != nil {
			return 0, nil, fmt.Errorf("condition %d: %w", i, err)
		}
	}

	return rm.Id, rm.Conditions, nil
}

// validateConditionInput validates a single condition input
func validateConditionInput(input ConditionInput) error {
	// Validate condition type
	if input.Type == "" {
		return fmt.Errorf("condition type is required")
	}

	// Validate operator
	if input.Operator == "" {
		return fmt.Errorf("operator is required")
	}

	// Validate supported operators
	switch input.Operator {
	case "=", ">", "<", ">=", "<=":
		// Valid operators
	default:
		return fmt.Errorf("unsupported operator: %s", input.Operator)
	}

	// Validate condition-specific requirements
	switch input.Type {
	case "item":
		// Item conditions require referenceId (or legacy itemId)
		if input.ReferenceId == 0 && input.ItemId == 0 {
			return fmt.Errorf("referenceId is required for item conditions")
		}
		if input.ItemId != 0 && input.ReferenceId != 0 {
			return fmt.Errorf("both itemId and referenceId specified - use referenceId only")
		}
	case "questStatus":
		// Quest status conditions require referenceId
		if input.ReferenceId == 0 {
			return fmt.Errorf("referenceId is required for quest status conditions")
		}
		// Quest status values should be valid enum values (0-3)
		if input.Value < 0 || input.Value > 3 {
			return fmt.Errorf("quest status value must be between 0 and 3 (UNDEFINED=0, NOT_STARTED=1, STARTED=2, COMPLETED=3)")
		}
	case "questProgress":
		// Quest progress conditions require referenceId and step
		if input.ReferenceId == 0 {
			return fmt.Errorf("referenceId is required for quest progress conditions")
		}
		if input.Step == "" {
			return fmt.Errorf("step is required for quest progress conditions")
		}
	case "guildId":
		// Guild ID conditions require a valid guild ID value
		if input.Value <= 0 {
			return fmt.Errorf("guild ID value must be greater than 0")
		}
	case "guildRank":
		// Guild rank conditions should have reasonable rank values
		if input.Value < 0 || input.Value > 5 {
			return fmt.Errorf("guild rank value must be between 0 and 5")
		}
	case "hasUnclaimedMarriageGifts":
		// Marriage gift conditions should be boolean (0 or 1)
		if input.Value != 0 && input.Value != 1 {
			return fmt.Errorf("marriage gift value must be 0 or 1")
		}
		// Only equals operator makes sense for boolean conditions
		if input.Operator != "=" {
			return fmt.Errorf("marriage gift conditions only support '=' operator")
		}
	case "level", "reborns", "dojoPoints", "vanquisherKills", "gmLevel":
		// Numeric conditions should have non-negative values
		if input.Value < 0 {
			return fmt.Errorf("%s value must be non-negative", input.Type)
		}
	case "jobId", "meso", "mapId", "fame", "gender", "strength", "dexterity", "intelligence", "luck":
		// Standard numeric conditions - basic validation
		break
	default:
		return fmt.Errorf("unsupported condition type: %s", input.Type)
	}

	return nil
}
