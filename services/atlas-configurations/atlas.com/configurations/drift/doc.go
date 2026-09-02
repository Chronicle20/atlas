// Package drift computes the comparable revision of a configuration
// document. It is a third package that neither `templates` nor `tenants`
// owns, and it imports neither: making one domain package reach into the
// other's hashing internals is exactly the cross-boundary call the repo
// convention warns about, and the direction chosen would decide by
// accident which package owns a policy that belongs to both.
//
// drift operates on MARSHALED JSON, never on either package's Go types.
// templates.maplelife.RestModel and tenants.maplelife.RestModel are
// distinct Go types with identical JSON tags; the hash never learns the
// difference, which is what makes one definition genuinely serve both
// sides (FR-2.1, FR-2.6).
//
// A drift hash is NOT comparable with templates.Revision. Revision hashes
// the STRUCT, in field-declaration order; Aggregate hashes a MAP, in
// key-sorted order. Same document, different bytes, different SHA-256.
// Each is only ever compared against itself:
//
//	template vs shipped seed file  -> templates.Revision, both sides
//	tenant vs baseline template    -> drift.Aggregate,   both sides
//
// Crossing them produces a flag that is permanently true for every row.
package drift

import "encoding/json"

// Doc is a canonical, package-neutral view of a configuration document:
// the comparable top-level keys, each holding its already-marshaled value.
// json.Marshal of a Doc emits keys in sorted order, so a Doc has exactly
// one serialization regardless of how it was built.
type Doc map[string]json.RawMessage

// Excluded names the keys that never participate in drift or reset
// (FR-2.4). "id" is deliberately absent: it carries json:"-" on both
// models, so it cannot be produced, and listing it would imply the
// marshal is untrusted.
var Excluded = []string{
	"environment",
	"region",
	"majorVersion",
	"minorVersion",
	"worlds",
	"diagnostics",
}

// Named lists the sections that get their own key in the drift report.
// Everything comparable and NOT named falls into Properties by
// subtraction, so a new top-level field participates in drift with no
// edit here (FR-2.7).
//
// The cost of that default: adding a new top-level SECTION (an object)
// without adding it here silently folds it into Properties. It still
// drifts and still resets -- it just does not get its own flag. If you
// add a section, add it here.
var Named = []string{
	"socket",
	"characters",
	"npcs",
	"cashShop",
	"mapleLife",
}

// Properties is the residual section: every comparable key not in Named.
// Today that is exactly "usesPin". It supersedes the PRD's literal
// "usesPin" section name (design OQ-1): an enumerated scalar section
// would leave a future top-level scalar in the aggregate hash but in no
// named section -- an indicator that says "something is wrong, and I will
// not tell you what".
const Properties = "properties"
