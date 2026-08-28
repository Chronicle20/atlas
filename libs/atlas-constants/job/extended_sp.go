package job

import "strings"

// UsesExtendedSP reports whether GW_CharacterStat carries the variable-length
// extended-SP block (a Decode1 count followed by count x (Decode1
// masterLevelIdx, Decode1 sp)) in place of the single SP Decode2.
//
// Because the divergence is not length-prefixed at the point of the branch,
// answering it differently than the client shifts every field after SP.
//
// Client reads, per version:
//
//	GMS <= 83                    no extended-SP path at all (Evan launched at v84)
//	GMS v84  inline @0x4e9da4    job/100 == 22 || job == 2001
//	GMS v87  inline @0x501e9c    job/100 == 22 || job == 2001
//	GMS v92  inline @0x4f50f4 / @0x4f5100 / @0x4f510f
//	                             job/1000 == 3 || job/100 == 22 || job == 2001
//	GMS v95  is_extendsp_job @0x4f1e30   same as v92
//	JMS 185  sub_5163A2 @0x5163a2, called @0x50eda2   same as v92
//
// region is matched case-insensitively, the same normalisation constants.For
// applies (constants/for.go:40). The arms are monotone in major within a
// region, so an unprovisioned major takes the nearest lower provisioned arm.
//
// The job/1000 == 3 arm is modelled although no Atlas job identity reaches it
// today (the only 3xx identities are Bowman 300 .. Marksman 322, all < 1000
// after the divide); a future Resistance bring-up must inherit the client's
// rule, not rediscover it. A Dual Blade (43x) is deliberately NOT an
// extended-SP job: 430/1000 == 0, so it keeps the plain SP short.
func UsesExtendedSP(jobId Id, region string, major uint16) bool {
	if strings.ToUpper(region) == "GMS" {
		if major < 84 {
			return false
		}
		if major < 92 {
			return jobId/100 == 22 || jobId == 2001
		}
	}
	return jobId/1000 == 3 || jobId/100 == 22 || jobId == 2001
}
