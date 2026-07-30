package job

import (
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// Identity is a version-blind job identity: a stable name for "this job
// concept" independent of the wire id any particular client version binds
// it to. It is distinct from Id, which is a version's raw wire id -- the
// same Identity can bind to different Id values across versions (and the
// same Id value can mean different things in different versions; see
// docs/tasks/task-187-version-aware-id-semantics).
//
// The identity constants in identities_gen.go are keyed by their canonical
// (v83-era) wire id token, which is opaque here: it is only a stable,
// collision-free numeric key for the identity, not necessarily this
// version's wire id. Per-version wireId<->Identity binding tables are
// built on top of this namespace by later tasks.
type Identity uint16

// IdentityName returns the checked-in display name for id, or "" if id has
// no known identity.
func IdentityName(id Identity) string {
	return identityNames[id]
}

// Set is one version's immutable wireId<->Identity binding table, built by
// the generator (see version_<r>_<maj>_<min>_gen.go) from
// docs/tasks/task-187-version-aware-id-semantics's per-version semantics +
// availability manifests. Zero value is a valid, empty Set.
type Set struct {
	byWire     map[Id]Identity
	byIdentity map[Identity]Id
	available  map[Identity]struct{} // this version's release-available identities (task-187 Task 5)
	names      map[Identity]string   // this version's identity -> display name (task-187 Task 5)
}

// Resolve returns the Identity this version's wireId is bound to, or
// (0, false) if wireId is not present in this version's semantics.
func (s Set) Resolve(wireId Id) (Identity, bool) {
	id, ok := s.byWire[wireId]
	return id, ok
}

// Wire returns the wireId this version binds id to, or (0, false) if id has
// no binding in this version's semantics.
func (s Set) Wire(id Identity) (Id, bool) {
	w, ok := s.byIdentity[id]
	return w, ok
}

// Available reports whether id was actually released/playable at this
// version -- a SUBSET of presence (Resolve/Wire): an identity can be
// present in the WZ data as an unreleased stub (e.g. the v61 Pirate job)
// well before its class actually shipped. See task-187 Task 5 and
// docs/tasks/task-187-version-aware-id-semantics/audit/availability.csv.
func (s Set) Available(id Identity) bool {
	_, ok := s.available[id]
	return ok
}

// Name returns the version-independent display name for id, or "" if id
// has no binding in this version's semantics.
func (s Set) Name(id Identity) string {
	return s.names[id]
}

// AvailableIdentities returns every Identity available at this version,
// sorted ascending by this version's wire id.
func (s Set) AvailableIdentities() []Identity {
	out := make([]Identity, 0, len(s.available))
	for id := range s.available {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return s.byIdentity[out[i]] < s.byIdentity[out[j]] })
	return out
}

// ---- Identity-keyed semantic predicates (task-187 Task 7) ----
//
// Every predicate below is a version-INDEPENDENT property of an Identity,
// never "what does wire id N mean in version V" (that question is
// Set.Resolve/Set.Wire above). Bodies are ported verbatim from their
// Id-typed originals in model.go/advancement.go: only the types change
// (Id -> Identity, the *Id reference constants -> the identities_gen.go
// Identity constants), because Identity tokens are keyed by the exact same
// canonical (v83-era) wire numbering the Id-typed arithmetic already
// assumes -- see this file's package doc and
// docs/tasks/task-187-version-aware-id-semantics.
//
// Names are suffixed "Identity" (IsAIdentity, not IsA) because the
// Id-typed originals in model.go/advancement.go already own the bare
// names and stay untouched (additive-only, per the task-187 Task 7
// brief) -- Go has no function overloading.

// IsAIdentity is the Identity form of IsA: reports whether id is
// classified under any of refs, per the same branch/root matching rule
// Is uses for the Id-typed jobs (same-branch descendant, or root-branch
// ancestor).
func IsAIdentity(id Identity, refs ...Identity) bool {
	is := false
	for _, ref := range refs {
		if isIdentity(id, ref) {
			is = true
		}
	}
	return is
}

// isIdentity is the Identity form of Is. Unexported: IsAIdentity is the
// only entry point the task-187 Task 7 brief asks for (mirroring IsA being
// the exported job-family check built on the unexported single-ref Is).
func isIdentity(id Identity, referenceId Identity) bool {
	characterBranch := id / 10
	referenceBranch := referenceId / 10
	return characterBranch == referenceBranch && id >= referenceId || referenceBranch%10 == 0 && id/100 == referenceId/100
}

// IsBeginnerIdentity is the Identity form of IsBeginner (model.go).
func IsBeginnerIdentity(id Identity) bool {
	return IsAIdentity(id, Beginner, Noblesse, Legend, Evan)
}

// GetTypeIdentity is the Identity form of GetType (model.go).
func GetTypeIdentity(id Identity) Type {
	return Type(uint16(id) / 1000)
}

// IsCygnusIdentity is the Identity form of IsCygnus (model.go).
func IsCygnusIdentity(id Identity) bool {
	return GetTypeIdentity(id) == TypeCygnus
}

// GetSkillBookIdentity is the Identity form of GetSkillBook (model.go).
func GetSkillBookIdentity(id Identity) int {
	if id >= EvanStage2 && id <= EvanStage10 {
		return int(uint16(id) - 2209)
	}
	return 0
}

// AdvancementIdentity is the Identity form of Advancement
// (advancement.go): returns the job-advancement tier (0-4) for a job
// identity. 0 for beginners (Beginner/Noblesse/Legend/Evan-beginner), 1
// for a branch root (id%100 == 0), else 2 + id%10. Evan stage identities
// (2200-2218) do not map onto the 4-tier scheme and return -1, as does any
// identity whose derived tier falls outside 0-4.
func AdvancementIdentity(id Identity) int {
	if id >= EvanStage1 && id <= EvanStage10 {
		return -1
	}
	if IsBeginnerIdentity(id) {
		return 0
	}
	if id%100 == 0 {
		return 1
	}
	tier := 2 + int(id%10)
	if tier > 4 {
		return -1
	}
	return tier
}

// IsFourthJobIdentity is the Identity form of IsFourthJob (model.go). It
// delegates to the Id-typed original via a direct numeric conversion
// rather than re-deriving the answer: Identity tokens are keyed by the
// exact same canonical numbering as the Jobs registry's Id keys (see this
// file's package doc), so Id(id) is a like-for-like lookup, not an
// approximation. IsFourthJob's answer comes from constants.go's curated
// per-job fourthJob field (e.g. Evan's non-arithmetic 4th-job band,
// EvanStage6..10), not a formula -- duplicating that table here under
// Identity keys would just invite drift between two copies of the same
// data.
func IsFourthJobIdentity(id Identity) bool {
	return IsFourthJob(Id(id))
}

// FromSkillIdentity is the Identity form of IdFromSkillId (model.go):
// derives a job identity from a skill identity via the shared
// job-identity-from-skill-identity convention (skill identity / 10000).
func FromSkillIdentity(sid skill.Identity) Identity {
	return Identity(uint32(sid) / 10000)
}
