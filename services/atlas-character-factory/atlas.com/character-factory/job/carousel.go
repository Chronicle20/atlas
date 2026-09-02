package job

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Slot is a (raceIndex, subJobIndex) pair exactly as the client sends it in the
// character-creation request. raceIndex is an ordinal into the race carousel the login
// screen drew -- it is NOT a job id, and its meaning changes between client versions.
type Slot struct {
	RaceIndex   uint32
	SubJobIndex uint32
}

// Carousel is one client version's race-selection screen: the exact set of slots that
// client can send, and the beginner job each one creates. Absence from the map means the
// client could not have sent that slot.
//
// Every entry traces to a row in docs/tasks/task-283-race-index-job-mapping/findings.md
// with a cited IDA function and address. Do not add an entry without one.
type Carousel map[Slot]job.Id

// One table per distinct carousel findings.md established (findings.md, "Carousels
// required (Task 5)"). These are read-only after init; nothing mutates them, which is
// what makes FromIndex safe for concurrent multi-tenant use.
var (
	// unverifiedCarousel: gms_12. No binary, no IDB, no export. Its lone (1,0) slot is
	// present in every candidate mapping the client could plausibly have used, so it is
	// insensitive to the ambiguity -- but it is not confirmed against a binary.
	unverifiedCarousel = Carousel{
		{RaceIndex: 1, SubJobIndex: 0}: job.BeginnerId, // findings.md gms_12
	}

	// noRaceCarousel: gms_v48, gms_v61, gms_v72. CLogin::SendNewCharPacket encodes no race
	// member and no sub-job member on any of these three versions, so no ordinal other than
	// Explorer can ever be selected or transmitted. FR-8 / findings.md "Seed rows to
	// correct": the (0,0) and (2,0) rows some seed templates carry for these versions are
	// unreachable and are deliberately NOT carried into this carousel.
	noRaceCarousel = Carousel{
		{RaceIndex: 1, SubJobIndex: 0}: job.BeginnerId, // findings.md gms_v48 0x500545, gms_v61 0x5653e9, gms_v72 0x5b219a
	}

	// race3Carousel: gms_v79, gms_v83. Three-arm CLogin::Update switch, no sub-job field.
	race3Carousel = Carousel{
		{RaceIndex: 0, SubJobIndex: 0}: job.NoblesseId, // findings.md gms_v83 CLogin::Update 0x5f4f26 case 0 (Cygnus); gms_v79 0x5ca641 case 0
		{RaceIndex: 1, SubJobIndex: 0}: job.BeginnerId, // findings.md gms_v83 case 1 (Explorer); gms_v79 case 1
		{RaceIndex: 2, SubJobIndex: 0}: job.LegendId,   // findings.md gms_v83 case 2 (Aran, class symbol present); gms_v79 case 2 (Aran-family dialog by geometry, class unverified -- FR-21 freezes the mapping regardless of label confidence)
		// raceIndex 3 does not exist: both switches have only three arms (default case).
	}

	// race4Carousel: gms_v84, gms_v87, gms_v92. Four-arm CLogin::Update-equivalent switch.
	// Ordinals 2/3 are class-unverified on gms_v92 (which of Aran/the fourth-race slot is
	// which was not established within budget); FR-21 freezes the mapping anyway.
	race4Carousel = Carousel{
		{RaceIndex: 0, SubJobIndex: 0}: job.NoblesseId, // findings.md gms_v84 0x609e9f, gms_v87 0x62c5c8, gms_v92 0x5d5cad case 0 (Cygnus)
		{RaceIndex: 1, SubJobIndex: 0}: job.BeginnerId, // case 1 (Explorer)
		{RaceIndex: 2, SubJobIndex: 0}: job.LegendId,   // case 2 (Aran on v84/v87; Aran-or-Evan, class unverified, on v92)
		{RaceIndex: 3, SubJobIndex: 0}: job.EvanId,     // case 3 (fourth race/Evan slot on v84/v87; class unverified on v92)
		// gms_v92 (1,1): sub-job is transmitted live, but the race-select button handler
		// that would set it was not located within budget -- neither confirmed nor denied,
		// so no entry (findings.md gms_v92 notes).
	}

	// race5Carousel: gms_v95. Five-arm CLogin::Update jump table (0x5df54f), the only
	// version with Resistance/Citizen and a live sub-job field.
	race5Carousel = Carousel{
		{RaceIndex: 0, SubJobIndex: 0}: job.CitizenId,  // findings.md gms_v95 CLogin::Update 0x5dee90 case 0 -> CUINewCharNameSelectRes; job id 3000 is human-supplied, not IDA-derived (findings.md "New job constants required")
		{RaceIndex: 1, SubJobIndex: 0}: job.BeginnerId, // case 1 -> CUINewCharNameSelectNormal (Explorer)
		{RaceIndex: 1, SubJobIndex: 1}: job.BeginnerId, // CUINewCharRaceSelect::SelectRaceButton 0x5f4d60 button 0 -- Explorer race with sub-job marker 1 (Dual Blade); no distinct creation job id exists on any version examined
		{RaceIndex: 2, SubJobIndex: 0}: job.NoblesseId, // case 2 -> CUINewCharNameSelectCygnus
		{RaceIndex: 3, SubJobIndex: 0}: job.LegendId,   // case 3 -> CUINewCharNameSelectAran
		{RaceIndex: 4, SubJobIndex: 0}: job.EvanId,     // case 4 -> CUINewCharNameSelectEvan
	}

	// race4JmsCarousel: gms_jms_185. CLogin::Update (0x66c17f) branches on the same
	// four-ordinal shape as race4Carousel, but every class identity is unverified and no
	// job id can be established for any of them -- findings.md explicitly warns not to
	// reuse the race4 class labels here without checking. No entries: every ordinal on this
	// carousel is a genuinely unestablished mapping (findings.md "jobId: null" slots), not a
	// guess.
	race4JmsCarousel = Carousel{}
)

// carouselFor selects the carousel for the tenant's client version. The chain is ordered
// most-specific-first and uses only the tenant.Model version predicates (task-283 FR-2);
// a raw `> N` comparison here is a review failure.
func carouselFor(t tenant.Model) Carousel {
	switch {
	case t.IsRegion("JMS") && t.MajorAtLeast(185):
		return race4JmsCarousel
	case t.IsRegion("GMS") && t.MajorAtLeast(95):
		return race5Carousel
	case t.IsRegion("GMS") && t.MajorInRange(84, 92):
		return race4Carousel
	case t.IsRegion("GMS") && t.MajorInRange(79, 83):
		return race3Carousel
	case t.IsRegion("GMS") && t.MajorInRange(48, 72):
		return noRaceCarousel
	default:
		// gms_12 has no IDA export and cannot be verified (findings.md, gms_12); its lone
		// seeded slot is (1,0) -> Explorer, which is present in every candidate mapping.
		return unverifiedCarousel
	}
}

// FromIndex maps a client-sent race ordinal to the beginner job it creates for this
// tenant's client version.
//
// ok=false means the tenant's client could not have sent this slot. The caller MUST
// reject; there is deliberately no default branch and no fallback to job.BeginnerId,
// because coercing an unknown ordinal is the bug task-283 exists to fix (FR-1).
func FromIndex(t tenant.Model, raceIndex uint32, subJobIndex uint32) (job.Id, bool) {
	id, ok := carouselFor(t)[Slot{RaceIndex: raceIndex, SubJobIndex: subJobIndex}]
	return id, ok
}
