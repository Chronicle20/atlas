package dragon

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Model is one Evan's dragon. It is 1:1 with its owning character: the client
// addresses all three clientbound dragon ops by owner character id
// (CUserPool::OnUserCommonPacket consumes the id before the family dispatch),
// so the owner IS the identity and there is no separate dragon id.
//
// x/y are int32 because SPAWN_DRAGON encodes 4-byte coordinates — unlike every
// other entity in the protocol. Keeping the wide type end to end stops a
// narrowing conversion from ever entering the pipeline.
type Model struct {
	ownerCharacterId uint32
	fld              field.Model
	x                int32
	y                int32
	stance           byte
	jobId            job.Id
}

func (m Model) OwnerCharacterId() uint32 { return m.ownerCharacterId }
func (m Model) Field() field.Model       { return m.fld }
func (m Model) X() int32                 { return m.x }
func (m Model) Y() int32                 { return m.y }
func (m Model) Stance() byte             { return m.stance }
func (m Model) JobId() job.Id            { return m.jobId }

// Move returns a copy at the new position. Stance is deliberately untouched:
// see the doc comment on ProcessorImpl.Move for why the caller must not pass
// a stance derived from a MOVE command through to this method.
func (m Model) Move(x int32, y int32) Model {
	return Clone(m).SetX(x).SetY(y).Build()
}

// HasDragon reports whether wireJobId resolves, on this tenant's client version,
// to an Evan growth stage (EvanStage1..EvanStage10). The Evan beginner (2001) is
// excluded: CDragon is created at the first growth stage.
//
// Expressed through the version-aware resolver rather than a numeric range on
// wire ids, which buys three things a `2200 <= id <= 2218` check would not:
//
//  1. v83 falls out for free — the v83 job table has no Evan entry, so Resolve
//     fails and no lifecycle path needs a version special-case.
//  2. tools/skill-job-id-guard.sh compliance — the comparison is over resolved
//     job.Identity values, never over a banned wire constant.
//  3. a future version that remaps 22xx cannot silently break it.
//
// The identity block 2200..2218 is exclusively Evan
// (libs/atlas-constants/job/identities_gen.go:83-92), so the closed range over
// Identity values is exact.
func HasDragon(t tenant.Model, wireJobId job.Id) bool {
	id, ok := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion()).Job.Resolve(wireJobId)
	if !ok {
		return false
	}
	return id >= job.EvanStage1 && id <= job.EvanStage10
}
