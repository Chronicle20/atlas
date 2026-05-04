package character

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

func TestWearerProfile_GettersReturnConstructorArgs(t *testing.T) {
	wp := NewWearerProfile(35, job.Id(200))
	if wp.Level() != 35 {
		t.Errorf("Level() = %d, want 35", wp.Level())
	}
	if wp.JobId() != job.Id(200) {
		t.Errorf("JobId() = %d, want 200", wp.JobId())
	}
}

func TestWearerProfile_ZeroValueIsBeginnerLevelZero(t *testing.T) {
	var wp WearerProfile
	if wp.Level() != 0 {
		t.Errorf("zero Level() = %d, want 0", wp.Level())
	}
	if wp.JobId() != job.Id(0) {
		t.Errorf("zero JobId() = %d, want 0", wp.JobId())
	}
}
