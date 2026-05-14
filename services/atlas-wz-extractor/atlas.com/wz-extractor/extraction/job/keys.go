package job

import "fmt"

// Key namespace prefix. Kept private to the package so the layout cannot drift.
const namespace = "wz-extractor"

func jobKey(jobId string) string {
	return fmt.Sprintf("%s:job:%s", namespace, jobId)
}

func unitsKey(jobId string) string {
	return fmt.Sprintf("%s:job:%s:units", namespace, jobId)
}

// LockKey composes the tenant-lock key. Exported so the lock package and the
// dispatcher can both reference it without re-deriving the format.
func LockKey(tenantId, region string, major, minor uint16) string {
	return fmt.Sprintf("%s:tenant-lock:%s:%s:%d.%d", namespace, tenantId, region, major, minor)
}
