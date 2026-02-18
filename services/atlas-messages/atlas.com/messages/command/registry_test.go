package command

import (
	"atlas-messages/character"
	"context"
	"testing"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

// createTestCharacter creates a character model for testing using the builder
func createTestCharacter(id uint32, name string, isGm bool) character.Model {
	gm := 0
	if isGm {
		gm = 1
	}
	return character.NewModelBuilder().
		SetId(id).
		SetName(name).
		SetGm(gm).
		SetMapId(100000000).
		Build()
}

// mockCommandProducer creates a simple command producer for testing
func mockCommandProducer(pattern string) Producer {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (Executor, bool) {
		return func(ctx context.Context) func(f field.Model, c character.Model, m string) (Executor, bool) {
			return func(_ field.Model, c character.Model, m string) (Executor, bool) {
				if m == pattern {
					return func(l logrus.FieldLogger) func(ctx context.Context) error {
						return func(ctx context.Context) error {
							return nil
						}
					}, true
				}
				return nil, false
			}
		}
	}
}

func TestRegistry_Singleton(t *testing.T) {
	// Get registry twice and verify it's the same instance
	r1 := Registry()
	r2 := Registry()

	if r1 != r2 {
		t.Error("Registry() should return the same singleton instance")
	}
}

func TestRegistry_Add(t *testing.T) {
	// Note: Due to singleton pattern, we can't fully isolate this test
	// This test verifies that Add doesn't panic and commands are stored
	r := Registry()
	initialLen := len(r.commandRegistry)

	// Add a mock command
	r.Add(mockCommandProducer("@test_add_command"))

	if len(r.commandRegistry) != initialLen+1 {
		t.Errorf("Expected registry length to increase by 1, got %d (was %d)", len(r.commandRegistry), initialLen)
	}
}

func TestRegistry_Get(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()

	// Create a test character
	gmCharacter := createTestCharacter(12345, "TestGM", true)
	_ = createTestCharacter(12346, "TestPlayer", false) // Available for future permission tests

	f := field.NewBuilder(1, 1, 100000000).Build()

	testCases := []struct {
		name         string
		message      string
		character    character.Model
		expectFound  bool
		setupCommand string
	}{
		{
			name:         "Matching command found",
			message:      "@test_registry_command",
			character:    gmCharacter,
			expectFound:  true,
			setupCommand: "@test_registry_command",
		},
		{
			name:         "Non-matching command not found",
			message:      "@unknown_command",
			character:    gmCharacter,
			expectFound:  false,
			setupCommand: "",
		},
		{
			name:         "Empty message",
			message:      "",
			character:    gmCharacter,
			expectFound:  false,
			setupCommand: "",
		},
	}

	// Add the test command to registry
	Registry().Add(mockCommandProducer("@test_registry_command"))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			executor, found := Registry().Get(logger, ctx, f, tc.character, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v, got found=%v for message '%s'", tc.expectFound, found, tc.message)
			}

			if tc.expectFound && executor == nil {
				t.Error("Expected executor to be non-nil when found=true")
			}

			if !tc.expectFound && executor != nil {
				t.Error("Expected executor to be nil when found=false")
			}
		})
	}
}

func TestRegistry_Get_MultipleCommands(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()

	gmCharacter := createTestCharacter(12345, "TestGM", true)

	// Add multiple test commands
	Registry().Add(mockCommandProducer("@multi_test_1"))
	Registry().Add(mockCommandProducer("@multi_test_2"))
	Registry().Add(mockCommandProducer("@multi_test_3"))

	f := field.NewBuilder(1, 1, 100000000).Build()

	testCases := []struct {
		name        string
		message     string
		expectFound bool
	}{
		{
			name:        "First command matches",
			message:     "@multi_test_1",
			expectFound: true,
		},
		{
			name:        "Second command matches",
			message:     "@multi_test_2",
			expectFound: true,
		},
		{
			name:        "Third command matches",
			message:     "@multi_test_3",
			expectFound: true,
		},
		{
			name:        "No command matches",
			message:     "@multi_test_4",
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, found := Registry().Get(logger, ctx, f, gmCharacter, tc.message)

			if found != tc.expectFound {
				t.Errorf("Expected found=%v, got found=%v for message '%s'", tc.expectFound, found, tc.message)
			}
		})
	}
}

func TestRegistry_Get_ExecutorExecution(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := context.Background()

	gmCharacter := createTestCharacter(12345, "TestGM", true)

	f := field.NewBuilder(1, 1, 100000000).Build()

	// Add a command that we can verify execution
	executed := false
	executorCommand := func(l logrus.FieldLogger) func(ctx context.Context) func(f field.Model, c character.Model, m string) (Executor, bool) {
		return func(ctx context.Context) func(f field.Model, c character.Model, m string) (Executor, bool) {
			return func(_ field.Model, c character.Model, m string) (Executor, bool) {
				if m == "@executor_test" {
					return func(l logrus.FieldLogger) func(ctx context.Context) error {
						return func(ctx context.Context) error {
							executed = true
							return nil
						}
					}, true
				}
				return nil, false
			}
		}
	}

	Registry().Add(executorCommand)

	executor, found := Registry().Get(logger, ctx, f, gmCharacter, "@executor_test")
	if !found {
		t.Fatal("Expected command to be found")
	}

	// Execute the command
	err := executor(logger)(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !executed {
		t.Error("Expected executor to be executed")
	}
}
