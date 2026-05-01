package characterrender

import "testing"

func TestParseRenderPath(t *testing.T) {
	got, err := ParseRenderPath(map[string]string{
		"tenant":  "ec876921-aaaa-bbbb-cccc-deadbeef0000",
		"region":  "GMS",
		"version": "83.1",
		"hash":    "abcdef1234567890",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Tenant != "ec876921-aaaa-bbbb-cccc-deadbeef0000" {
		t.Fatalf("tenant: %s", got.Tenant)
	}
	if got.Region != "GMS" {
		t.Fatalf("region: %s", got.Region)
	}
	if got.MajorVersion != 83 || got.MinorVersion != 1 {
		t.Fatalf("version: %d.%d", got.MajorVersion, got.MinorVersion)
	}
	if got.Hash != "abcdef1234567890" {
		t.Fatalf("hash: %s", got.Hash)
	}
}

func TestParseRenderPathRejectsBadHash(t *testing.T) {
	_, err := ParseRenderPath(map[string]string{
		"tenant": "t", "region": "GMS", "version": "83.1", "hash": "ZZZZ",
	})
	if err == nil {
		t.Fatal("expected error on bad hash")
	}
}

func TestParseRenderPathRejectsBadVersion(t *testing.T) {
	_, err := ParseRenderPath(map[string]string{
		"tenant": "t", "region": "GMS", "version": "abc", "hash": "abcdef1234567890",
	})
	if err == nil {
		t.Fatal("expected error on bad version")
	}
}

func TestParseRenderPathRejectsTraversalTenant(t *testing.T) {
	_, err := ParseRenderPath(map[string]string{
		"tenant": "..", "region": "GMS", "version": "83.1", "hash": "abcdef1234567890",
	})
	if err == nil {
		t.Fatal("expected error on traversal tenant")
	}
}

func TestParseRenderPathRejectsBadRegion(t *testing.T) {
	_, err := ParseRenderPath(map[string]string{
		"tenant": "tenant-a", "region": "gms!", "version": "83.1", "hash": "abcdef1234567890",
	})
	if err == nil {
		t.Fatal("expected error on bad region")
	}
}
