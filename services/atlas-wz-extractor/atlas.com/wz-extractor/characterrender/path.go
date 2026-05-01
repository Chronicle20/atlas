package characterrender

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var hashPattern = regexp.MustCompile(`^[a-f0-9]{16}$`)

// tenantPattern allows UUID-shaped or short slug tenants (alphanumeric and hyphens, 1–64 chars).
var tenantPattern = regexp.MustCompile(`^[a-zA-Z0-9-]{1,64}$`)

// regionPattern allows 2–8 uppercase ASCII letters (e.g. GMS, JMS, KMS).
var regionPattern = regexp.MustCompile(`^[A-Z]{2,8}$`)

// RenderPath is the parsed path component of a render request.
type RenderPath struct {
	Tenant       string
	Region       string
	MajorVersion uint16
	MinorVersion uint16
	Hash         string
}

// ParseRenderPath validates the gorilla/mux path vars produced by the route
// `/render/{tenant}/{region}/{version}/{hash}.png`. Hash must be 16 lowercase
// hex chars; version must be `MAJOR.MINOR` integers.
func ParseRenderPath(vars map[string]string) (RenderPath, error) {
	tenant := vars["tenant"]
	region := vars["region"]
	version := vars["version"]
	hash := vars["hash"]
	if tenant == "" || region == "" || version == "" || hash == "" {
		return RenderPath{}, fmt.Errorf("missing path component")
	}
	if strings.Contains(tenant, "..") || strings.Contains(tenant, "/") {
		return RenderPath{}, fmt.Errorf("invalid tenant %q: path traversal rejected", tenant)
	}
	if !tenantPattern.MatchString(tenant) {
		return RenderPath{}, fmt.Errorf("invalid tenant %q: must match [a-zA-Z0-9-]{1,64}", tenant)
	}
	if strings.Contains(region, "..") || strings.Contains(region, "/") {
		return RenderPath{}, fmt.Errorf("invalid region %q: path traversal rejected", region)
	}
	if !regionPattern.MatchString(region) {
		return RenderPath{}, fmt.Errorf("invalid region %q: must match [A-Z]{2,8}", region)
	}
	if !hashPattern.MatchString(hash) {
		return RenderPath{}, fmt.Errorf("invalid hash %q", hash)
	}
	parts := strings.SplitN(version, ".", 2)
	if len(parts) != 2 {
		return RenderPath{}, fmt.Errorf("invalid version %q", version)
	}
	major, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return RenderPath{}, fmt.Errorf("invalid major version: %w", err)
	}
	minor, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return RenderPath{}, fmt.Errorf("invalid minor version: %w", err)
	}
	return RenderPath{
		Tenant:       tenant,
		Region:       region,
		MajorVersion: uint16(major),
		MinorVersion: uint16(minor),
		Hash:         hash,
	}, nil
}
