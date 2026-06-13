package evidence

import (
	"path/filepath"
	"testing"
)

func TestLoadRecord(t *testing.T) {
	r, err := LoadRecord(filepath.Join("testdata", "buddy.clientbound.Invite.yaml"))
	if err != nil {
		t.Fatalf("LoadRecord: %v", err)
	}
	if r.Packet != "buddy/clientbound/Invite" || r.Version != "gms_v83" {
		t.Errorf("record = %+v", r)
	}
	if r.Category != "OPAQUE" || r.IDA.Function != "CWvsContext::OnFriendResult" {
		t.Errorf("record = %+v", r)
	}
	if len(r.Verifies) != 1 {
		t.Errorf("verifies = %v", r.Verifies)
	}
}

func TestLoadRecordRejectsBadCategory(t *testing.T) {
	_, err := loadRecordBytes([]byte(
		"packet: a/b/C\ndirection: clientbound\nversion: gms_v83\ncategory: BOGUS\nida:\n  function: F\n  address: 0x1\n  decompile_sha256: aa\n"), "x.yaml")
	if err == nil {
		t.Fatal("expected category validation error")
	}
}

func TestFunctionHashStableAndDriftSensitive(t *testing.T) {
	h1, err := FunctionHash(filepath.Join("testdata", "export_mini.json"), "CLogin::OnFoo")
	if err != nil {
		t.Fatalf("FunctionHash: %v", err)
	}
	h2, _ := FunctionHash(filepath.Join("testdata", "export_mini.json"), "CLogin::OnFoo")
	if h1 != h2 {
		t.Error("hash not stable")
	}
	hBar, _ := FunctionHash(filepath.Join("testdata", "export_mini.json"), "CLogin::OnBar")
	if h1 == hBar {
		t.Error("different functions must hash differently")
	}
	if _, err := FunctionHash(filepath.Join("testdata", "export_mini.json"), "CLogin::Missing"); err == nil {
		t.Error("missing function must error (citation unresolvable, design §13)")
	}
}
