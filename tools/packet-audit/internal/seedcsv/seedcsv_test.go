package seedcsv

import (
	"path/filepath"
	"testing"
)

func TestLoadClientbound(t *testing.T) {
	rows, err := Load(filepath.Join("testdata", "clientbound_excerpt.csv"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}

	ls := rows[0]
	if ls.Op != "LOGIN_STATUS" || ls.FName != "CLogin::OnCheckPasswordResult" {
		t.Errorf("row0 = %+v", ls)
	}
	// Index-column quirk: first version pair uses the named Index column.
	v12 := ls.Versions["GMS:12"]
	if !v12.Present || v12.Opcode != 0x001 {
		t.Errorf("v12 = %+v", v12)
	}
	// Present with opcode 0x000 but non-empty index.
	v83 := ls.Versions["GMS:83"]
	if !v83.Present || v83.Opcode != 0x000 {
		t.Errorf("v83 = %+v (presence must come from index cell)", v83)
	}
	// Absent: empty index + 0x000.
	v48 := ls.Versions["GMS:48"]
	if v48.Present {
		t.Errorf("v48 should be absent: %+v", v48)
	}

	// ACCOUNT_INFO absent in JMS185.
	ai := rows[2]
	if ai.Versions["JMS:185"].Present {
		t.Errorf("ACCOUNT_INFO JMS:185 should be absent")
	}
}

func TestLoadServerboundQuirks(t *testing.T) {
	rows, err := Load(filepath.Join("testdata", "serverbound_excerpt.csv"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Line numbers are set to physical CSV record numbers (header=1, first data row=2).
	lp := rows[0]
	if lp.Line != 2 {
		t.Errorf("LOGIN_PASSWORD Line = %d, want 2", lp.Line)
	}
	// Empty FName kept.
	gl := rows[1]
	if gl.Op != "GUEST_LOGIN" || gl.FName != "" {
		t.Errorf("row1 = %+v", gl)
	}
	if gl.Line != 3 {
		t.Errorf("GUEST_LOGIN Line = %d, want 3", gl.Line)
	}
	if !gl.Versions["GMS:83"].Present || gl.Versions["GMS:83"].Opcode != 0x002 {
		t.Errorf("GUEST_LOGIN v83 = %+v", gl.Versions["GMS:83"])
	}
	// Multiline FName splits into fname + alts.
	sr := rows[2]
	if sr.FName != "CLogin::Init" || len(sr.FNameAlts) != 1 || sr.FNameAlts[0] != "CLogin::ChangeStepImmediate" {
		t.Errorf("multiline fname = %q alts=%v", sr.FName, sr.FNameAlts)
	}
	// SERVERLIST_REREQUEST is a multi-line CSV record but still one data record (rowNum=2).
	if sr.Line != 4 {
		t.Errorf("SERVERLIST_REREQUEST Line = %d, want 4", sr.Line)
	}
	// n/a placeholder row: Op="n/a", FName="n/a". Line is the next record = 5.
	na := rows[3]
	if na.Op != "n/a" || na.FName != "n/a" {
		t.Errorf("n/a row = %+v", na)
	}
	if na.Line != 5 {
		t.Errorf("n/a row Line = %d, want 5", na.Line)
	}
	if !na.Versions["GMS:83"].Present {
		t.Errorf("n/a row GMS:83 should be present")
	}
}

func TestLoadBadOpcodeFailsLoudly(t *testing.T) {
	_, err := LoadFromString("Op,FName,Index,GMS v83\nX,CFoo::Bar,1,zzz\n")
	if err == nil {
		t.Fatal("expected loud failure with row number")
	}
}
