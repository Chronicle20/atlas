package templates

import (
	"atlas-configurations/templates/socket"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Revision is the ONE definition of a template's content hash. Both sides of
// the drift comparison call it: LoadCatalog hashes the parsed seed file
// (FR-1.4), and makeView hashes the RestModel that Make produced from the
// stored row (FR-2.1). Wording them as two definitions that happen to agree
// is how they eventually stop agreeing, so there is only one.
//
// Revision hashes only client-authored content. Any field that is
// server-owned or read-time-derived - set from something other than the
// caller's request body - must be cleared or normalized here before the
// next such field silently makes drift permanent instead of being caught at
// review.
//
// Id is cleared rather than trusted to its json:"-" tag - strictly stronger
// than FR-2.2, and it costs one assignment. Socket is normalized because both
// Make (read) and Create (write) normalize, so a revision that saw the
// nil-vs-empty distinction would report drift on every template whose stored
// document omits a socket collection. Environment is cleared because it is
// server-owned (see rest.go and Make): Make unconditionally overwrites it
// from Entity.Environment, no shipped seed file carries an "environment"
// key, and on any deployment with a non-empty ATLAS_ENVIRONMENT (e.g.
// atlas-main) the stored side would hash a value the shipped side never has,
// making SeedDrift permanently true for every template.
func Revision(rm RestModel) (string, error) {
	rm.Id = ""
	rm.Environment = ""
	rm.Socket = socket.Normalize(rm.Socket)

	b, err := json.Marshal(rm)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
