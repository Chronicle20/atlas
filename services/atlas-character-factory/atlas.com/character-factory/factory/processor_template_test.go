package factory

import (
	"testing"

	"atlas-character-factory/configuration/tenant/characters/template"
)

// TestFindCreationTemplate pins the graceful-failure behavior: a job with no
// configured creation template (e.g. a newly-introduced class such as Evan)
// must report not-found, so Create returns ErrTemplateNotFound rather than a nil
// error. The nil error previously left the client hanging on the creation screen
// because the login handler's AddCharacterError path only fires on a real error.
func TestFindCreationTemplate(t *testing.T) {
	templates := []template.RestModel{
		{JobIndex: 0, SubJobIndex: 0, Gender: 0, MapId: 10000},
		{JobIndex: 0, SubJobIndex: 0, Gender: 1, MapId: 10000},
		{JobIndex: 2, SubJobIndex: 0, Gender: 0, MapId: 914000000},
	}

	// An existing (job, subjob, gender) resolves to its template.
	if got, ok := findCreationTemplate(templates, 0, 0, 1); !ok || got.MapId != 10000 {
		t.Fatalf("findCreationTemplate(0,0,1): ok=%v mapId=%d; want ok=true mapId=10000", ok, got.MapId)
	}

	// A job with no configured template (the Evan case) must report not-found.
	if _, ok := findCreationTemplate(templates, 3, 0, 0); ok {
		t.Errorf("findCreationTemplate(3,0,0): ok=true; want false for an unconfigured job")
	}

	// Right job, wrong gender must not match.
	if _, ok := findCreationTemplate(templates, 2, 0, 1); ok {
		t.Errorf("findCreationTemplate(2,0,1): ok=true; want false")
	}
}
