package allocation

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func TestBranchFor(t *testing.T) {
	tests := []struct {
		name  string
		jobId job.Id
		mapId _map.Id
		want  uint32
	}{
		{"warrior, hall of warriors", job.Id(100), _map.VictoriaRoadHallOfWarriors1Id, 10},
		{"magician, hall of magicians", job.Id(200), _map.VictoriaRoadHallOfMagicians1Id, 11},
		{"bowman", job.Id(300), _map.VictoriaRoadHallOfBowmen1Id, 12},
		{"thief", job.Id(400), _map.VictoriaRoadHallOfThieves1Id, 13},
		{"pirate", job.Id(500), _map.TheNautilusTrainingRoom2Id, 14},
		{"dawn warrior", job.Id(1100), _map.EmpressRoadKnightsChamber1Id, 15},
		{"blaze wizard", job.Id(1200), _map.EmpressRoadKnightsChamber1Id, 16},
		{"wind archer", job.Id(1300), _map.EmpressRoadKnightsChamber1Id, 17},
		{"night walker", job.Id(1400), _map.EmpressRoadKnightsChamber1Id, 18},
		{"thunder breaker", job.Id(1500), _map.EmpressRoadKnightsChamber1Id, 19},
		{"aran", job.Id(2100), _map.SnowIslandPalaceOfTheMaster1Id, 20},
		{"evan", job.Id(2001), _map.EmpressRoadKnightsChamber1Id, 21},
		{"beginner", job.Id(0), _map.EmpressRoadKnightsChamber1Id, 22},
		{"noblesse", job.Id(1000), _map.EmpressRoadKnightsChamber1Id, 23},
		{"legend", job.Id(2000), _map.EmpressRoadKnightsChamber1Id, 24},
		// FR-3.3 GM-deploy formula: 26 + 4*(mapId/100000000). jobId is
		// irrelevant here — a non-Hall-of-Fame map always uses the GM
		// formula regardless of the deploying character's job.
		{"GM deploy, non-HoF map, continent 1", job.Id(100), _map.Id(100000000), 30},
		{"GM deploy, non-HoF map, continent 2", job.Id(100), _map.Id(200000000), 34},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BranchFor(tt.jobId, tt.mapId)
			if got != tt.want {
				t.Errorf("BranchFor(%v, %v) = %v, want %v", tt.jobId, tt.mapId, got, tt.want)
			}
		})
	}
}

func TestBranchRange(t *testing.T) {
	tests := []struct {
		branch  uint32
		wantMin uint32
		wantMax uint32
	}{
		{10, 9901000, 9901099},
		{14, 9901400, 9901499},
		{27, 9902700, 9902799},
	}

	for _, tt := range tests {
		gotMin, gotMax := BranchRange(tt.branch)
		if gotMin != tt.wantMin || gotMax != tt.wantMax {
			t.Errorf("BranchRange(%d) = (%d, %d), want (%d, %d)", tt.branch, gotMin, gotMax, tt.wantMin, tt.wantMax)
		}
	}
}

func TestAllocate(t *testing.T) {
	tests := []struct {
		name    string
		usable  map[uint32]bool
		inUse   map[uint32]bool
		branch  uint32
		want    uint32
		wantErr error
	}{
		{
			name:   "in-branch hit, lowest first",
			usable: map[uint32]bool{9901000: true, 9901001: true},
			inUse:  map[uint32]bool{},
			branch: 10,
			want:   9901000,
		},
		{
			name:   "in-branch, lowest free",
			usable: map[uint32]bool{9901000: true, 9901001: true},
			inUse:  map[uint32]bool{9901000: true},
			branch: 10,
			want:   9901001,
		},
		{
			name:   "branch empty -> global fallback",
			usable: map[uint32]bool{9901500: true},
			inUse:  map[uint32]bool{},
			branch: 14,
			want:   9901500,
		},
		{
			name:   "branch exhausted -> global fallback",
			usable: map[uint32]bool{9901000: true, 9901500: true},
			inUse:  map[uint32]bool{9901000: true},
			branch: 10,
			want:   9901500,
		},
		{
			name:    "whole pool exhausted",
			usable:  map[uint32]bool{9901000: true},
			inUse:   map[uint32]bool{9901000: true},
			branch:  10,
			wantErr: ErrPoolExhausted,
		},
		{
			name:    "empty usable set",
			usable:  map[uint32]bool{},
			inUse:   map[uint32]bool{},
			branch:  10,
			wantErr: ErrPoolExhausted,
		},
		{
			name:   "GM branch, nothing in branch, fallback wins",
			usable: map[uint32]bool{9901000: true},
			inUse:  map[uint32]bool{},
			branch: 27,
			want:   9901000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Allocate(tt.usable, tt.inUse, tt.branch)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Allocate() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Allocate() unexpected err = %v", err)
			}
			if got != tt.want {
				t.Errorf("Allocate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildUsablePool(t *testing.T) {
	errLookup := errors.New("lookup failed")

	tests := []struct {
		name    string
		exists  bool
		imitate bool
		err     error
		wantIn  bool
		wantErr error
	}{
		{"exists and imitate", true, true, nil, true, nil},
		{"exists, imitate 0", true, false, nil, false, nil},
		{"template missing", false, false, nil, false, nil},
		{"lookup error", false, false, errLookup, false, errLookup},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := BuildUsablePool(func(id uint32) (bool, bool, error) {
				return tt.exists, tt.imitate, tt.err
			})

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("BuildUsablePool() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildUsablePool() unexpected err = %v", err)
			}
			if got := pool[PoolMin]; got != tt.wantIn {
				t.Errorf("pool[%d] = %v, want %v", PoolMin, got, tt.wantIn)
			}
		})
	}
}

func TestUsablePoolForCaches(t *testing.T) {
	tenantId := uuid.New()
	calls := 0
	lookup := func(id uint32) (bool, bool, error) {
		calls++
		return id == PoolMin, true, nil
	}

	first, err := UsablePoolFor(tenantId, lookup)
	if err != nil {
		t.Fatalf("UsablePoolFor() unexpected err = %v", err)
	}
	callsAfterFirst := calls

	second, err := UsablePoolFor(tenantId, lookup)
	if err != nil {
		t.Fatalf("UsablePoolFor() unexpected err = %v", err)
	}

	if calls != callsAfterFirst {
		t.Errorf("UsablePoolFor() called lookup again on cache hit: %d calls after first build, %d after second", callsAfterFirst, calls)
	}
	if len(first) != len(second) || !first[PoolMin] || !second[PoolMin] {
		t.Errorf("UsablePoolFor() cached pool mismatch: first=%v second=%v", first, second)
	}
}
