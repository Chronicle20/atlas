package handler

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// TestIsSuperGm_v48Wire510 pins the v0.48 half of the GM/SuperGM job
// remapping (task-187): at v0.48, SuperGM is wire 510 -- the canonical
// job.SuperGmId (910) does not exist as a wire value at that version.
func TestIsSuperGm_v48Wire510(t *testing.T) {
	set := constants.For("GMS", 48, 1)
	if !IsSuperGm(set, job.Id(510)) {
		t.Fatal("v48 wire 510 must resolve as SuperGM")
	}
	if IsSuperGm(set, job.SuperGmId) {
		t.Fatalf("v48 wire %d (canonical SuperGmId) must NOT resolve as SuperGM -- it is not a valid v48 wire job id", job.SuperGmId)
	}
}

// TestIsSuperGm_v83Wire910 pins the canonical (v83+) half: SuperGM is wire
// 910, i.e. job.SuperGmId.
func TestIsSuperGm_v83Wire910(t *testing.T) {
	set := constants.For("GMS", 83, 1)
	if !IsSuperGm(set, job.SuperGmId) {
		t.Fatal("v83 wire job.SuperGmId (910) must resolve as SuperGM")
	}
	if IsSuperGm(set, job.Id(510)) {
		t.Fatal("v83 wire 510 must NOT resolve as SuperGM -- 510 is not a valid v83 wire job id")
	}
}

// TestIsSuperGm_NonGmJobRejected guards the ordinary case: an unrelated job
// id is never SuperGM.
func TestIsSuperGm_NonGmJobRejected(t *testing.T) {
	if IsSuperGm(constants.For("GMS", 83, 1), job.Id(100)) {
		t.Fatal("job.Id(100) must not resolve as SuperGM")
	}
}
