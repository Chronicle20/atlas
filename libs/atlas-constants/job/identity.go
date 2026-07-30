package job

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
