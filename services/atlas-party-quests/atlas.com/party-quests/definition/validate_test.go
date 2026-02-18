package definition

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func definitionsPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get caller info")
	// validate_test.go is in definition/, definitions are at atlas-party-quests/party-quests/
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "party-quests")
}

func TestValidate_AllDefinitions(t *testing.T) {
	t.Setenv("PARTY_QUEST_DEFINITIONS_PATH", definitionsPath(t))

	models, errs := LoadDefinitionFiles()
	for _, err := range errs {
		t.Errorf("failed to load definition file: %v", err)
	}
	require.NotEmpty(t, models, "no definition files loaded")

	for _, rm := range models {
		t.Run(rm.QuestId, func(t *testing.T) {
			result := Validate(rm)

			if !result.Valid {
				t.Errorf("definition %q (%s) failed validation:\n  %s",
					rm.Name, rm.QuestId, strings.Join(result.Errors, "\n  "))
			}

			for _, w := range result.Warnings {
				t.Logf("WARNING [%s]: %s", rm.QuestId, w)
			}
		})
	}
}

func TestValidate_AllDefinitions_CanExtract(t *testing.T) {
	t.Setenv("PARTY_QUEST_DEFINITIONS_PATH", definitionsPath(t))

	models, errs := LoadDefinitionFiles()
	for _, err := range errs {
		t.Errorf("failed to load definition file: %v", err)
	}
	require.NotEmpty(t, models)

	for _, rm := range models {
		t.Run(rm.QuestId, func(t *testing.T) {
			m, err := Extract(rm)
			require.NoError(t, err, "Extract failed for %q", rm.QuestId)

			assert.Equal(t, rm.QuestId, m.QuestId())
			assert.Equal(t, rm.Name, m.Name())
			assert.Equal(t, len(rm.Stages), len(m.Stages()))

			for i, s := range m.Stages() {
				assert.Equal(t, uint32(i), s.Index(), "stage %d index mismatch", i)
				assert.NotEmpty(t, s.Type(), "stage %d has empty type", i)
			}

			if rm.Bonus != nil {
				require.NotNil(t, m.Bonus(), "bonus should not be nil when rest model has bonus")
				assert.Equal(t, rm.Bonus.MapId, m.Bonus().MapId())
				assert.Equal(t, rm.Bonus.Duration, m.Bonus().Duration())
				assert.Equal(t, rm.Bonus.Entry, string(m.Bonus().Entry()))
			} else {
				assert.Nil(t, m.Bonus(), "bonus should be nil when rest model has no bonus")
			}
		})
	}
}

func TestValidate_StageTypesCoverage(t *testing.T) {
	t.Setenv("PARTY_QUEST_DEFINITIONS_PATH", definitionsPath(t))

	models, _ := LoadDefinitionFiles()
	require.NotEmpty(t, models)

	usedTypes := make(map[string][]string)
	for _, rm := range models {
		for _, s := range rm.Stages {
			usedTypes[s.Type] = append(usedTypes[s.Type], fmt.Sprintf("%s/%s", rm.QuestId, s.Name))
		}
	}

	for stageType := range validStageTypes {
		t.Run(stageType, func(t *testing.T) {
			pqs, ok := usedTypes[stageType]
			if !ok {
				t.Logf("WARNING: stage type %q is defined but not used by any PQ definition", stageType)
			} else {
				t.Logf("used by: %s", strings.Join(pqs, ", "))
			}
		})
	}

	for stageType := range usedTypes {
		assert.True(t, validStageTypes[stageType], "definitions use stage type %q which is not in validStageTypes", stageType)
	}
}

func TestValidate_ClearConditionTypesCoverage(t *testing.T) {
	t.Setenv("PARTY_QUEST_DEFINITIONS_PATH", definitionsPath(t))

	models, _ := LoadDefinitionFiles()
	require.NotEmpty(t, models)

	usedTypes := make(map[string]bool)
	usedOperators := make(map[string]bool)
	for _, rm := range models {
		for _, s := range rm.Stages {
			for _, c := range s.ClearConditions {
				usedTypes[c.Type] = true
				usedOperators[c.Operator] = true
			}
		}
	}

	for ct := range usedTypes {
		assert.True(t, validClearConditionTypes[ct], "definitions use clear condition type %q which is not in validClearConditionTypes", ct)
	}

	for op := range usedOperators {
		assert.True(t, validClearOperators[op], "definitions use clear operator %q which is not in validClearOperators", op)
	}
}

func TestValidate_NoStubDefinitionsHaveInvalidData(t *testing.T) {
	t.Setenv("PARTY_QUEST_DEFINITIONS_PATH", definitionsPath(t))

	models, _ := LoadDefinitionFiles()
	require.NotEmpty(t, models)

	for _, rm := range models {
		t.Run(rm.QuestId, func(t *testing.T) {
			if len(rm.Stages) == 0 {
				t.Logf("stub definition (no stages)")
				assert.NotEmpty(t, rm.QuestId, "stub must have questId")
				assert.NotEmpty(t, rm.Name, "stub must have name")
				return
			}

			assert.NotEmpty(t, rm.StartEvents, "non-stub definition should have startEvents")
			assert.NotZero(t, rm.Exit, "non-stub definition should have a non-zero exit map")
		})
	}
}
