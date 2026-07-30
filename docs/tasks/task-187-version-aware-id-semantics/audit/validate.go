// Command validate is the task-187 Task-1 structural validator (Step 6).
//
// It is a standalone `go run` script rather than a _test.go file because
// the generator module referenced by the task-1 brief ("gen/audit_validate_test.go
// once the generator module exists") does not exist yet -- Task 1 runs
// first. Task 2 is expected to port this logic into a real Go test in the
// generator module once it exists.
//
// It checks the two machine-readable audit deliverables:
//
//   - divergences.csv (region,major,minor,domain,wireId,identityName,evidence):
//     every row must have a non-empty evidence citation.
//   - availability.csv (region,major,minor,domain,identityName,released,meymink):
//     every row must have a non-empty meymink citation, and released must
//     be exactly "true" or "false".
//
// Both files: region must be gms or jms, (major,minor) must be a member of
// the provisioned set from deploy/k8s/base/versions.json, and domain must
// be skill or job.
//
// Usage (from the repo root):
//
//	go run ./docs/tasks/task-187-version-aware-id-semantics/audit/validate.go
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
)

// provisioned is the exact (region,major,minor) set from
// deploy/k8s/base/versions.json. Kept as a literal here (not read from the
// JSON file) so this validator has zero dependencies and can run as a bare
// `go run` script; if versions.json changes, update this list too.
var provisioned = map[[3]string]bool{
	{"gms", "12", "1"}:  true,
	{"gms", "48", "1"}:  true,
	{"gms", "61", "1"}:  true,
	{"gms", "72", "1"}:  true,
	{"gms", "79", "1"}:  true,
	{"gms", "83", "1"}:  true,
	{"gms", "84", "1"}:  true,
	{"gms", "87", "1"}:  true,
	{"gms", "92", "1"}:  true,
	{"gms", "95", "1"}:  true,
	{"jms", "185", "1"}: true,
}

var (
	validRegions = map[string]bool{"gms": true, "jms": true}
	validDomains = map[string]bool{"skill": true, "job": true}
)

func readCSV(path string) (header []string, rows [][]string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	all, err := r.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(all) == 0 {
		return nil, nil, fmt.Errorf("%s: empty file", path)
	}
	return all[0], all[1:], nil
}

func col(header []string, name string) int {
	for i, h := range header {
		if h == name {
			return i
		}
	}
	return -1
}

func checkKeyCols(file string, header []string, row [4]int, rowNum int, r []string) []string {
	var errs []string
	regionIdx, majorIdx, minorIdx, domainIdx := row[0], row[1], row[2], row[3]

	region := r[regionIdx]
	if !validRegions[region] {
		errs = append(errs, fmt.Sprintf("%s row %d: region %q not in {gms,jms}", file, rowNum, region))
	}
	major := r[majorIdx]
	minor := r[minorIdx]
	if !provisioned[[3]string{region, major, minor}] {
		errs = append(errs, fmt.Sprintf("%s row %d: (region=%s,major=%s,minor=%s) not in the provisioned set", file, rowNum, region, major, minor))
	}
	domain := r[domainIdx]
	if !validDomains[domain] {
		errs = append(errs, fmt.Sprintf("%s row %d: domain %q not in {skill,job}", file, rowNum, domain))
	}
	return errs
}

func validateDivergences(path string) (int, []string) {
	header, rows, err := readCSV(path)
	if err != nil {
		return 0, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	regionI, majorI, minorI, domainI := col(header, "region"), col(header, "major"), col(header, "minor"), col(header, "domain")
	wireIDI := col(header, "wireId")
	evidenceI := col(header, "evidence")
	if regionI < 0 || majorI < 0 || minorI < 0 || domainI < 0 || wireIDI < 0 || evidenceI < 0 {
		return 0, []string{fmt.Sprintf("%s: header missing required column(s): %v", path, header)}
	}

	var errs []string
	for i, r := range rows {
		rowNum := i + 2 // 1-indexed + header line
		errs = append(errs, checkKeyCols(path, header, [4]int{regionI, majorI, minorI, domainI}, rowNum, r)...)
		if r[evidenceI] == "" {
			errs = append(errs, fmt.Sprintf("%s row %d: empty evidence", path, rowNum))
		}
		if r[wireIDI] == "" {
			errs = append(errs, fmt.Sprintf("%s row %d: empty wireId", path, rowNum))
		}
	}
	return len(rows), errs
}

func validateAvailability(path string) (int, []string) {
	header, rows, err := readCSV(path)
	if err != nil {
		return 0, []string{fmt.Sprintf("%s: %v", path, err)}
	}
	regionI, majorI, minorI, domainI := col(header, "region"), col(header, "major"), col(header, "minor"), col(header, "domain")
	releasedI := col(header, "released")
	meyminkI := col(header, "meymink")
	identityI := col(header, "identityName")
	if regionI < 0 || majorI < 0 || minorI < 0 || domainI < 0 || releasedI < 0 || meyminkI < 0 || identityI < 0 {
		return 0, []string{fmt.Sprintf("%s: header missing required column(s): %v", path, header)}
	}

	var errs []string
	for i, r := range rows {
		rowNum := i + 2
		errs = append(errs, checkKeyCols(path, header, [4]int{regionI, majorI, minorI, domainI}, rowNum, r)...)
		if r[meyminkI] == "" {
			errs = append(errs, fmt.Sprintf("%s row %d: empty meymink citation", path, rowNum))
		}
		if r[identityI] == "" {
			errs = append(errs, fmt.Sprintf("%s row %d: empty identityName", path, rowNum))
		}
		if r[releasedI] != "true" && r[releasedI] != "false" {
			errs = append(errs, fmt.Sprintf("%s row %d: released must be \"true\" or \"false\", got %q", path, rowNum, r[releasedI]))
		}
	}
	return len(rows), errs
}

// repoRootRelDir is the audit folder's path relative to the repo root, for
// the case where this script is invoked as
// `go run ./docs/.../audit/validate.go` from the repo root (cwd = repo
// root, so the bare "divergences.csv" won't resolve there).
const repoRootRelDir = "docs/tasks/task-187-version-aware-id-semantics/audit"

func resolveDir() string {
	candidates := []string{".", repoRootRelDir}
	if len(os.Args) > 1 {
		candidates = append([]string{os.Args[1]}, candidates...)
	}
	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "divergences.csv")); err == nil {
			return dir
		}
	}
	return "." // let the CSV-open error below report the real problem
}

func main() {
	dir := resolveDir()
	divPath := filepath.Join(dir, "divergences.csv")
	availPath := filepath.Join(dir, "availability.csv")

	nDiv, divErrs := validateDivergences(divPath)
	nAvail, availErrs := validateAvailability(availPath)

	allErrs := append(divErrs, availErrs...)
	if len(allErrs) > 0 {
		for _, e := range allErrs {
			fmt.Fprintln(os.Stderr, "FAIL:", e)
		}
		fmt.Fprintf(os.Stderr, "FAIL: %d error(s) across %d divergence rows, %d availability rows\n", len(allErrs), nDiv, nAvail)
		os.Exit(1)
	}

	fmt.Printf("OK: %d divergence rows, %d availability rows, all cited\n", nDiv, nAvail)
}
