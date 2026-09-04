package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// checkIdentity asserts every reactor id seen in any version directory is
// present in every version directory with an identical SHA-256. seen maps
// version dir (e.g. "gms/83_1") -> reactor id -> file digest.
func checkIdentity(seen map[string]map[string]string) []string {
	var errs []string

	versions := make([]string, 0, len(seen))
	for v := range seen {
		versions = append(versions, v)
	}
	sort.Strings(versions)

	ids := map[string]struct{}{}
	for _, v := range versions {
		for id := range seen[v] {
			ids[id] = struct{}{}
		}
	}
	allIds := make([]string, 0, len(ids))
	for id := range ids {
		allIds = append(allIds, id)
	}
	sort.Strings(allIds)

	for _, id := range allIds {
		var missing []string
		digests := map[string][]string{}
		for _, v := range versions {
			d, ok := seen[v][id]
			if !ok {
				missing = append(missing, v)
				continue
			}
			digests[d] = append(digests[d], v)
		}
		if len(missing) > 0 {
			errs = append(errs, fmt.Sprintf("reactor %s: missing from %s", id, strings.Join(missing, ", ")))
		}
		if len(digests) > 1 {
			var groups []string
			for d, vs := range digests {
				groups = append(groups, fmt.Sprintf("%s=[%s]", d[:12], strings.Join(vs, " ")))
			}
			sort.Strings(groups)
			errs = append(errs, fmt.Sprintf("reactor %s: copies differ: %s", id, strings.Join(groups, " ")))
		}
	}
	return errs
}

// digest is the file's SHA-256, hex encoded.
func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// versionKey turns "<root>/gms/83_1/reactor-actions/reactors/reactor-2001.json"
// into "gms/83_1".
func versionKey(root, path string) (string, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", false
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 5 {
		return "", false
	}
	return parts[0] + "/" + parts[1], true
}
