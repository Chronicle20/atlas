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
// Id is cleared rather than trusted to its json:"-" tag - strictly stronger
// than FR-2.2, and it costs one assignment. Socket is normalized because both
// Make (read) and Create (write) normalize, so a revision that saw the
// nil-vs-empty distinction would report drift on every template whose stored
// document omits a socket collection.
func Revision(rm RestModel) (string, error) {
	rm.Id = ""
	rm.Socket = socket.Normalize(rm.Socket)

	b, err := json.Marshal(rm)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
