package job

import "sort"

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
