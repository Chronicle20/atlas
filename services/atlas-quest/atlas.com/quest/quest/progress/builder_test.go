package progress_test

import (
	"atlas-quest/quest/progress"
	"testing"
)

func TestNewModelBuilder_CreatesEmptyBuilder(t *testing.T) {
	builder := progress.NewModelBuilder()
	model := builder.Build()

	if model.Id() != 0 {
		t.Errorf("Id() = %d, want 0", model.Id())
	}
	if model.InfoNumber() != 0 {
		t.Errorf("InfoNumber() = %d, want 0", model.InfoNumber())
	}
	if model.Progress() != "" {
		t.Errorf("Progress() = %s, want empty string", model.Progress())
	}
}

func TestBuilder_SettersAreChainable(t *testing.T) {
	model := progress.NewModelBuilder().
		SetId(123).
		SetInfoNumber(456).
		SetProgress("005").
		Build()

	if model.Id() != 123 {
		t.Errorf("Id() = %d, want 123", model.Id())
	}
	if model.InfoNumber() != 456 {
		t.Errorf("InfoNumber() = %d, want 456", model.InfoNumber())
	}
	if model.Progress() != "005" {
		t.Errorf("Progress() = %s, want \"005\"", model.Progress())
	}
}

func TestCloneModel_PreservesAllFields(t *testing.T) {
	original := progress.NewModelBuilder().
		SetId(123).
		SetInfoNumber(456).
		SetProgress("010").
		Build()

	cloned := progress.CloneModel(original).Build()

	if cloned.Id() != original.Id() {
		t.Errorf("cloned.Id() = %d, want %d", cloned.Id(), original.Id())
	}
	if cloned.InfoNumber() != original.InfoNumber() {
		t.Errorf("cloned.InfoNumber() = %d, want %d", cloned.InfoNumber(), original.InfoNumber())
	}
	if cloned.Progress() != original.Progress() {
		t.Errorf("cloned.Progress() = %s, want %s", cloned.Progress(), original.Progress())
	}
}

func TestCloneModel_ModificationsDoNotAffectOriginal(t *testing.T) {
	original := progress.NewModelBuilder().
		SetId(123).
		SetInfoNumber(456).
		SetProgress("005").
		Build()

	modified := progress.CloneModel(original).
		SetId(999).
		SetInfoNumber(888).
		SetProgress("100").
		Build()

	// Verify original is unchanged
	if original.Id() != 123 {
		t.Errorf("original.Id() changed to %d", original.Id())
	}
	if original.InfoNumber() != 456 {
		t.Errorf("original.InfoNumber() changed to %d", original.InfoNumber())
	}
	if original.Progress() != "005" {
		t.Errorf("original.Progress() changed to %s", original.Progress())
	}

	// Verify modified has new values
	if modified.Id() != 999 {
		t.Errorf("modified.Id() = %d, want 999", modified.Id())
	}
	if modified.InfoNumber() != 888 {
		t.Errorf("modified.InfoNumber() = %d, want 888", modified.InfoNumber())
	}
	if modified.Progress() != "100" {
		t.Errorf("modified.Progress() = %s, want \"100\"", modified.Progress())
	}
}

func TestBuilder_ProgressFormats(t *testing.T) {
	tests := []struct {
		name     string
		progress string
		desc     string
	}{
		{"mob_zero_kills", "000", "mob kill progress at 0"},
		{"mob_some_kills", "005", "mob kill progress at 5"},
		{"mob_many_kills", "100", "mob kill progress at 100"},
		{"map_not_visited", "0", "map not visited"},
		{"map_visited", "1", "map visited"},
		{"item_count", "15", "item count progress"},
		{"empty", "", "empty progress"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := progress.NewModelBuilder().
				SetProgress(tt.progress).
				Build()

			if model.Progress() != tt.progress {
				t.Errorf("Progress() = %s, want %s (%s)", model.Progress(), tt.progress, tt.desc)
			}
		})
	}
}

func TestBuilder_FluentInterface(t *testing.T) {
	// Test that builder can be used in a single chain
	model := progress.NewModelBuilder().
		SetId(1).
		SetInfoNumber(100100).
		SetProgress("000").
		Build()

	if model.Id() != 1 {
		t.Errorf("Id() = %d, want 1", model.Id())
	}
	if model.InfoNumber() != 100100 {
		t.Errorf("InfoNumber() = %d, want 100100", model.InfoNumber())
	}
	if model.Progress() != "000" {
		t.Errorf("Progress() = %s, want \"000\"", model.Progress())
	}
}
