package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSeedFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

const testRegistryV83 = `- op: LOGIN_STATUS
  direction: clientbound
  opcode: 1
  fname: CLogin::OnCheckPasswordResult
  provenance: csv-import
- op: LOGIN_PASSWORD
  direction: serverbound
  opcode: 1
  fname: CLogin::SendCheckPasswordPacket
  provenance: csv-import
- op: NO_FNAME
  direction: clientbound
  opcode: 2
  fname: ""
  provenance: csv-import
`

// testTemplate83 deliberately matches the REAL seed templates' compact
// single-line "services" array convention (measured: all 2,859 socket entries
// across all 11 real files write "services" on one line, e.g. ["login"] or
// ["login","channel"] - never one element per line). Using that convention
// here rather than an expanded, encoding/json-friendly one is what makes
// TestSeedFName_ByteStableNoOpWrite an honest proof of round-trip fidelity
// instead of a tautology against whatever json.MarshalIndent happens to emit.
const testTemplate83 = `{
  "region": "GMS",
  "majorVersion": 83,
  "minorVersion": 1,
  "usesPin": false,
  "socket": {
    "handlers": [
      {
        "opCode": "0x01",
        "validator": "NoOpValidator",
        "handler": "LoginHandle",
        "services": ["login","channel"]
      }
    ],
    "writers": [
      {
        "opCode": "0x01",
        "writer": "AuthSuccess",
        "options": {},
        "services": ["login"]
      },
      {
        "opCode": "0x02",
        "writer": "ServerLoad",
        "services": ["login"]
      }
    ]
  },
  "characters": {
    "templates": [],
    "presets": []
  },
  "npcs": [],
  "worlds": [
    {
      "name": "Scania"
    }
  ],
  "cashShop": {
    "commodities": {}
  }
}
`

// setupSeedFName lays down a registry and a set of templates in a temp dir and
// returns (registryDir, templateDir).
func setupSeedFName(t *testing.T, registries map[string]string, templates map[string]string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	regDir := filepath.Join(dir, "registry")
	tplDir := filepath.Join(dir, "templates")
	for stem, body := range registries {
		writeSeedFile(t, filepath.Join(regDir, stem+".yaml"), body)
	}
	for stem, body := range templates {
		writeSeedFile(t, filepath.Join(tplDir, "template_"+stem+".json"), body)
	}
	return regDir, tplDir
}

func socketOf(t *testing.T, path string) (handlers, writers []map[string]json.RawMessage) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		t.Fatalf("reparse %s: %v", path, err)
	}
	var sock struct {
		Handlers []map[string]json.RawMessage `json:"handlers"`
		Writers  []map[string]json.RawMessage `json:"writers"`
	}
	if err := json.Unmarshal(top["socket"], &sock); err != nil {
		t.Fatalf("parse socket in %s: %v", path, err)
	}
	return sock.Handlers, sock.Writers
}

func TestSeedFName_ResolvesBothDirections(t *testing.T) {
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": testRegistryV83},
		map[string]string{"gms_83_1": testTemplate83})

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}

	h, w := socketOf(t, filepath.Join(tplDir, "template_gms_83_1.json"))
	if got := string(h[0]["fname"]); got != `"CLogin::SendCheckPasswordPacket"` {
		t.Errorf("handler joined the wrong direction: fname = %s", got)
	}
	if got := string(w[0]["fname"]); got != `"CLogin::OnCheckPasswordResult"` {
		t.Errorf("writer joined the wrong direction: fname = %s", got)
	}
	if _, present := w[1]["fname"]; present {
		t.Errorf("a registry row with an empty fname produced a key: %v", w[1])
	}
}

func TestSeedFName_PreservesUnmodelledTopLevelKeysVerbatim(t *testing.T) {
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": testRegistryV83},
		map[string]string{"gms_83_1": testTemplate83})

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}

	b, _ := os.ReadFile(filepath.Join(tplDir, "template_gms_83_1.json"))
	var before, after map[string]any
	if err := json.Unmarshal([]byte(testTemplate83), &before); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if err := json.Unmarshal(b, &after); err != nil {
		t.Fatalf("parse output: %v", err)
	}
	for _, k := range []string{"region", "majorVersion", "minorVersion", "usesPin", "characters", "npcs", "worlds", "cashShop"} {
		bj, _ := json.Marshal(before[k])
		aj, _ := json.Marshal(after[k])
		if string(bj) != string(aj) {
			t.Errorf("top-level %q changed:\n before: %s\n after : %s", k, bj, aj)
		}
	}
}

func TestSeedFName_FailsLoudlyOnUnknownTopLevelKey(t *testing.T) {
	surprising := strings.Replace(testTemplate83, `"usesPin": false,`,
		"\"usesPin\": false,\n  \"surpriseKey\": {\"a\": 1},", 1)
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": testRegistryV83},
		map[string]string{"gms_83_1": surprising})

	var stderr bytes.Buffer
	code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr)
	if code == 0 {
		t.Fatal("exit = 0, want non-zero on an unmodelled top-level key")
	}
	if !strings.Contains(stderr.String(), "surpriseKey") {
		t.Errorf("stderr did not name the offending key:\n%s", stderr.String())
	}
}

func TestSeedFName_FailsLoudlyOnUnknownEntryKey(t *testing.T) {
	surprising := strings.Replace(testTemplate83, `"handler": "LoginHandle",`,
		"\"handler\": \"LoginHandle\",\n        \"surpriseEntryKey\": 1,", 1)
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": testRegistryV83},
		map[string]string{"gms_83_1": surprising})

	var stderr bytes.Buffer
	code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr)
	if code == 0 {
		t.Fatal("exit = 0, want non-zero on an unmodelled socket-entry key")
	}
	if !strings.Contains(stderr.String(), "surpriseEntryKey") {
		t.Errorf("stderr did not name the offending key:\n%s", stderr.String())
	}
}

func TestSeedFName_WithoutWriteLeavesFilesUntouched(t *testing.T) {
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": testRegistryV83},
		map[string]string{"gms_83_1": testTemplate83})
	path := filepath.Join(tplDir, "template_gms_83_1.json")

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}
	b, _ := os.ReadFile(path)
	if string(b) != testTemplate83 {
		t.Errorf("dry run modified the file:\n%s", b)
	}
}

func TestSeedFName_AmbiguityPicksLexicographicallyFirstOp(t *testing.T) {
	const reg = `- op: STORAGE
  direction: clientbound
  opcode: 242
  fname: CTrunkDlg::OnPacket
  provenance: manual
- op: RPS_GAME
  direction: clientbound
  opcode: 242
  fname: CRPSGameDlg::OnPacket
  provenance: manual
`
	const tpl = `{
  "region": "GMS",
  "majorVersion": 61,
  "minorVersion": 1,
  "usesPin": false,
  "socket": {
    "handlers": [],
    "writers": [
      {
        "opCode": "0xF2",
        "writer": "MiniRoom",
        "services": [
          "channel"
        ]
      }
    ]
  },
  "characters": {
    "templates": [],
    "presets": []
  },
  "npcs": [],
  "worlds": [],
  "cashShop": {
    "commodities": {}
  }
}
`
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v61": reg},
		map[string]string{"gms_61_1": tpl})

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}
	_, w := socketOf(t, filepath.Join(tplDir, "template_gms_61_1.json"))
	if got := string(w[0]["fname"]); got != `"CRPSGameDlg::OnPacket"` {
		t.Errorf("tie-break did not pick RPS_GAME (lexicographically first op): fname = %s", got)
	}
	if !strings.Contains(stderr.String(), "ambiguous") {
		t.Errorf("tie-break was not logged to stderr:\n%s", stderr.String())
	}
}

// gms_92_1 and gms_12_1 have no registry of their own. They resolve by
// implementation name against adjacent versions, which is valid because the
// implementation name is the definition identity within a direction. LoginHandle
// sits at a DIFFERENT opcode in the borrower, so only an impl-name match works.
func TestSeedFName_FallsBackToAdjacentVersionByImplName(t *testing.T) {
	const tpl92 = `{
  "region": "GMS",
  "majorVersion": 92,
  "minorVersion": 1,
  "usesPin": false,
  "socket": {
    "handlers": [
      {
        "opCode": "0x7F",
        "validator": "NoOpValidator",
        "handler": "LoginHandle",
        "services": [
          "login"
        ]
      }
    ],
    "writers": []
  },
  "characters": {
    "templates": [],
    "presets": []
  },
  "npcs": [],
  "worlds": [],
  "cashShop": {
    "commodities": {}
  }
}
`
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v87": testRegistryV83},
		map[string]string{"gms_87_1": testTemplate83, "gms_92_1": tpl92})

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}
	h, _ := socketOf(t, filepath.Join(tplDir, "template_gms_92_1.json"))
	if got := string(h[0]["fname"]); got != `"CLogin::SendCheckPasswordPacket"` {
		t.Errorf("adjacent-version impl-name fallback did not resolve: fname = %s", got)
	}
}

// TestSeedFName_ByteStableNoOpWrite is the strongest byte-stability proof: a
// --write run where the registry has zero opcode overlap with the template
// resolves nothing, and the re-marshalled file must still equal the original
// byte-for-byte. This is what guarantees json.MarshalIndent's field order and
// escaping match the existing seed-file formatting exactly, independent of the
// dry-run short-circuit exercised by TestSeedFName_WithoutWriteLeavesFilesUntouched.
func TestSeedFName_ByteStableNoOpWrite(t *testing.T) {
	const regNoOverlap = `- op: UNRELATED
  direction: clientbound
  opcode: 999
  fname: CSomething::OnUnrelated
  provenance: manual
`
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": regNoOverlap},
		map[string]string{"gms_83_1": testTemplate83})
	path := filepath.Join(tplDir, "template_gms_83_1.json")

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(b) != testTemplate83 {
		t.Errorf("a no-match --write run was not byte-stable:\n--- got ---\n%s\n--- want ---\n%s", b, testTemplate83)
	}
}

// TestSeedFName_JoinsOneAndThreeDigitOpcodes proves opcode parsing is purely
// numeric and does not assume a fixed hex-digit width: real templates carry
// both "0x9" (template_jms_185_1.json) and "0x0A5" (template_gms_84_1.json)
// style opcodes.
func TestSeedFName_JoinsOneAndThreeDigitOpcodes(t *testing.T) {
	const reg = `- op: SHORT_OP
  direction: serverbound
  opcode: 9
  fname: CShort::SendOp
  provenance: manual
- op: LONG_OP
  direction: clientbound
  opcode: 165
  fname: CLong::OnOp
  provenance: manual
`
	const tpl = `{
  "region": "GMS",
  "majorVersion": 84,
  "minorVersion": 1,
  "usesPin": false,
  "socket": {
    "handlers": [
      {
        "opCode": "0x9",
        "validator": "NoOpValidator",
        "handler": "ShortHandle",
        "services": [
          "login"
        ]
      }
    ],
    "writers": [
      {
        "opCode": "0x0A5",
        "writer": "LongWrite",
        "services": [
          "login"
        ]
      }
    ]
  },
  "characters": {
    "templates": [],
    "presets": []
  },
  "npcs": [],
  "worlds": [],
  "cashShop": {
    "commodities": {}
  }
}
`
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v84": reg},
		map[string]string{"gms_84_1": tpl})

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}
	h, w := socketOf(t, filepath.Join(tplDir, "template_gms_84_1.json"))
	if got := string(h[0]["fname"]); got != `"CShort::SendOp"` {
		t.Errorf("1-digit opcode 0x9 did not join: fname = %s", got)
	}
	if got := string(w[0]["fname"]); got != `"CLong::OnOp"` {
		t.Errorf("3-digit opcode 0x0A5 did not join: fname = %s", got)
	}
}

// TestSeedFName_PreservesRealWorldFormattingQuirks reproduces, in miniature,
// the two real-file conventions that broke a naive decode-then-
// json.MarshalIndent round-trip (found by diffing an actual --write run
// against a scratch copy of the real templates): (1) entries that carry both
// "services" and "options" always order services FIRST in the source, the
// opposite of this file's struct-tag-free splice model's neutral order; (2)
// gms_12_1.json and gms_92_1.json inline the first array element's opening
// brace onto the array's own line ("handlers": [{ ... instead of
// "handlers": [\n  { ...). Both must survive completely untouched on an
// entry that resolves nothing, and the "services"-before-"options" order must
// survive even on an entry that DOES get an fname spliced in.
func TestSeedFName_PreservesRealWorldFormattingQuirks(t *testing.T) {
	const reg = `- op: LOGIN_STATUS
  direction: serverbound
  opcode: 1
  fname: CLogin::SendCheckPasswordPacket
  provenance: manual
`
	// opCode 0x02 (CharacterMoveHandle) has no registry match and must be
	// byte-for-byte untouched, options/services order and all. opCode 0x01
	// (LoginHandle) resolves and must gain "fname" without disturbing the
	// inlined "[{" array-open style.
	const tpl = `{
  "region": "GMS",
  "majorVersion": 83,
  "minorVersion": 1,
  "usesPin": false,
  "socket": {
    "handlers": [{
        "opCode": "0x01",
        "validator": "NoOpValidator",
        "handler": "LoginHandle",
        "services": ["login"]
      },
      {
        "opCode": "0x02",
        "validator": "LoggedInValidator",
        "handler": "CharacterMoveHandle",
        "services": ["channel"],
        "options": {
          "types": [{
              "Name": "NORMAL"
            }
          ]
        }
      }
    ],
    "writers": []
  },
  "characters": {
    "templates": [],
    "presets": []
  },
  "npcs": [],
  "worlds": [],
  "cashShop": {
    "commodities": {}
  }
}
`
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": reg},
		map[string]string{"gms_83_1": tpl})
	path := filepath.Join(tplDir, "template_gms_83_1.json")

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	got := string(b)

	// The untouched second entry, options-block-and-all, must be byte-for-byte
	// identical to the source, including the inlined "[{" on options.types and
	// the services-before-options field order.
	const untouchedTail = `      {
        "opCode": "0x02",
        "validator": "LoggedInValidator",
        "handler": "CharacterMoveHandle",
        "services": ["channel"],
        "options": {
          "types": [{
              "Name": "NORMAL"
            }
          ]
        }
      }
    ],`
	if !strings.Contains(got, untouchedTail) {
		t.Errorf("unresolved entry's formatting (services-before-options, inlined types array) was disturbed:\n%s", got)
	}

	// The resolved first entry must gain "fname" right after "handler", and the
	// array's inlined "[{" opening must be undisturbed.
	const wantHead = `  "socket": {
    "handlers": [{
        "opCode": "0x01",
        "validator": "NoOpValidator",
        "handler": "LoginHandle",
        "fname": "CLogin::SendCheckPasswordPacket",
        "services": ["login"]
      },`
	if !strings.Contains(got, wantHead) {
		t.Errorf("resolved entry's fname insertion or inlined array-open formatting was wrong:\n%s", got)
	}
}

// TestSeedFName_ReRunIsIdempotent proves the "re-runnable" design goal: running
// --write a second time over output that already carries the correct fname
// values changes nothing on disk (the file's mtime-independent bytes are
// identical), because spliceEntry treats NewFName == ExistingFName as a no-op
// rather than re-inserting or duplicating the key.
func TestSeedFName_ReRunIsIdempotent(t *testing.T) {
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": testRegistryV83},
		map[string]string{"gms_83_1": testTemplate83})
	path := filepath.Join(tplDir, "template_gms_83_1.json")

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("first run: exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}
	firstPass, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after first run: %v", err)
	}

	stderr.Reset()
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("second run: exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}
	secondPass, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after second run: %v", err)
	}

	if string(firstPass) != string(secondPass) {
		t.Errorf("re-running --write changed an already-resolved file:\n--- first ---\n%s\n--- second ---\n%s", firstPass, secondPass)
	}
}

// TestSeedFName_UpdatesExistingFNameInPlace covers the companion re-run case:
// the registry's resolved value for an opcode has changed since a prior run
// (e.g. a wire-divergence correction like the v84 ITC_OPERATION fixes), so the
// entry already has a DIFFERENT "fname" on disk. spliceEntry must replace the
// old value in place rather than leave it stale or insert a duplicate key.
func TestSeedFName_UpdatesExistingFNameInPlace(t *testing.T) {
	const reg = `- op: LOGIN_STATUS
  direction: serverbound
  opcode: 1
  fname: CLogin::CorrectedSendPacket
  provenance: manual
`
	const tpl = `{
  "region": "GMS",
  "majorVersion": 83,
  "minorVersion": 1,
  "usesPin": false,
  "socket": {
    "handlers": [
      {
        "opCode": "0x01",
        "validator": "NoOpValidator",
        "handler": "LoginHandle",
        "fname": "CLogin::StaleSendPacket",
        "services": ["login"]
      }
    ],
    "writers": []
  },
  "characters": {
    "templates": [],
    "presets": []
  },
  "npcs": [],
  "worlds": [],
  "cashShop": {
    "commodities": {}
  }
}
`
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": reg},
		map[string]string{"gms_83_1": tpl})

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}
	h, _ := socketOf(t, filepath.Join(tplDir, "template_gms_83_1.json"))
	if got := string(h[0]["fname"]); got != `"CLogin::CorrectedSendPacket"` {
		t.Errorf("stale existing fname was not updated in place: fname = %s", got)
	}
	b, _ := os.ReadFile(filepath.Join(tplDir, "template_gms_83_1.json"))
	if strings.Count(string(b), `"fname"`) != 1 {
		t.Errorf("expected exactly one fname key after an in-place update, got:\n%s", b)
	}
}
