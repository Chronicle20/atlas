package character

// Gender selectors. GenderUnspecified is the RenderQuery sentinel meaning the
// optional `gender` query param was absent and gender must be inferred.
const (
	GenderMale        = 0
	GenderFemale      = 1
	GenderUnspecified = -1
)

// Default beginner clothing item ids injected into an empty clothing slot so a
// character is never rendered bare (PRD FR-1). These are the single source of
// truth; if asset verification (plan Task 8) shows an id is not ingested, swap
// it here (and only here) — the UI never names item ids.
const (
	DefaultCoatMale    = 1040036
	DefaultPantsMale   = 1060026
	DefaultCoatFemale  = 1041046
	DefaultPantsFemale = 1061039
)

// ResolveGender maps an optional gender selector plus a face id to a concrete
// 0 (male) / 1 (female) value. Precedence: an explicit 0/1 wins; otherwise the
// v83 face convention (faceId/1000)%10 == 1 ⇒ female; anything else ⇒ male. A
// non-positive / unknown face id resolves to male.
//
// Idempotent: ResolveGender(0, face) == 0 and ResolveGender(1, face) == 1 for
// any face, so the handler can resolve once for the hash and Composite can
// resolve again for injection and always agree.
func ResolveGender(genderParam, face int) int {
	if genderParam == GenderMale || genderParam == GenderFemale {
		return genderParam
	}
	if face > 0 && (face/1000)%10 == 1 {
		return GenderFemale
	}
	return GenderMale
}

func defaultCoat(gender int) int {
	if gender == GenderFemale {
		return DefaultCoatFemale
	}
	return DefaultCoatMale
}

func defaultPants(gender int) int {
	if gender == GenderFemale {
		return DefaultPantsFemale
	}
	return DefaultPantsMale
}
