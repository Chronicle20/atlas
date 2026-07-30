// audit_validate_test.go is the task-187 Task-1 structural validator
// (originally docs/tasks/task-187-version-aware-id-semantics/audit/validate.go,
// a standalone `go run` script), ported into the gen module as a real Go
// test per that script's own header comment: "Task 2 is expected to port
// this logic into a real Go test in the generator module once it exists."
//
// It checks the two machine-readable audit deliverables Task 1 produced:
//
//   - divergences.csv (region,major,minor,domain,wireId,identityName,evidence):
//     every row must have a non-empty evidence citation and non-empty wireId.
//   - availability.csv (region,major,minor,domain,identityName,released,meymink):
//     every row must have a non-empty meymink citation, a non-empty
//     identityName, and released must be exactly "true" or "false".
//
// Both files: region must be gms or jms, (major,minor) must be a member of
// the provisioned set from deploy/k8s/base/versions.json, and domain must
// be skill or job.
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// auditDir is the audit folder's path relative to this test's package
// directory (libs/atlas-constants/gen).
const auditDir = "../../../docs/tasks/task-187-version-aware-id-semantics/audit"

// provisioned is the exact (region,major,minor) set from
// deploy/k8s/base/versions.json. Kept as a literal here (not read from the
// JSON file) so this validator has no dependency beyond the standard
// library; if versions.json changes, update this list too.
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

// expectedFields is the exact column count for both divergences.csv and
// availability.csv (region,major,minor,domain,+2 more = 7). Setting
// csv.Reader.FieldsPerRecord to this value (rather than the -1
// "no enforcement" sentinel) makes a short or long row a clean CSV parse
// error surfaced through readCSV's err return -- not a silent len(record)
// index-out-of-range panic downstream.
const expectedFields = 7

func readCSV(path string) (header []string, rows [][]string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.FieldsPerRecord = expectedFields
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

func checkKeyCols(file string, cols [4]int, rowNum int, r []string) []string {
	var errs []string
	regionIdx, majorIdx, minorIdx, domainIdx := cols[0], cols[1], cols[2], cols[3]

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
		errs = append(errs, checkKeyCols(path, [4]int{regionI, majorI, minorI, domainI}, rowNum, r)...)
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
		errs = append(errs, checkKeyCols(path, [4]int{regionI, majorI, minorI, domainI}, rowNum, r)...)
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

func TestAuditCSVs_Valid(t *testing.T) {
	divPath := filepath.Join(auditDir, "divergences.csv")
	availPath := filepath.Join(auditDir, "availability.csv")

	nDiv, divErrs := validateDivergences(divPath)
	nAvail, availErrs := validateAvailability(availPath)

	for _, e := range divErrs {
		t.Error(e)
	}
	for _, e := range availErrs {
		t.Error(e)
	}
	if t.Failed() {
		t.Fatalf("%d error(s) across %d divergence rows, %d availability rows", len(divErrs)+len(availErrs), nDiv, nAvail)
	}

	t.Logf("OK: %d divergence rows, %d availability rows, all cited", nDiv, nAvail)
}
