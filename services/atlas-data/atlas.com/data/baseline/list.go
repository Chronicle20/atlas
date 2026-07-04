package baseline

import (
	"strconv"
	"strings"
)

// parseDumpKey extracts (region, major, minor) from a canonical dump key of
// the exact shape DumpKey produces:
// baseline/regions/<region>/versions/<major>.<minor>/documents.dump.
// Keys that do not parse are the caller's cue to skip-and-warn, never fail.
func parseDumpKey(key string) (string, int, int, bool) {
	parts := strings.Split(key, "/")
	if len(parts) != 6 || parts[0] != "baseline" || parts[1] != "regions" ||
		parts[3] != "versions" || parts[5] != "documents.dump" {
		return "", 0, 0, false
	}
	region := parts[2]
	if region == "" {
		return "", 0, 0, false
	}
	ver := parts[4]
	dot := strings.LastIndex(ver, ".")
	if dot <= 0 || dot == len(ver)-1 {
		return "", 0, 0, false
	}
	major, err := strconv.Atoi(ver[:dot])
	if err != nil || major < 0 {
		return "", 0, 0, false
	}
	minor, err := strconv.Atoi(ver[dot+1:])
	if err != nil || minor < 0 {
		return "", 0, 0, false
	}
	return region, major, minor, true
}
