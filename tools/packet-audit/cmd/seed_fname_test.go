package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// realTemplateDir / realRegistryDir point at the actual, committed seed
// templates and op registries from tools/packet-audit/cmd/ - three levels up
// reaches the worktree root, the same pattern gatecheck_test.go and
// family_cap_test.go already use. Tests using these NEVER pass --template-dir
// pointing at these paths directly to runSeedFName with --write; they always
// copy into a t.TempDir() first, so the real seed files are read-only.
func realTemplateDir() string {
	return filepath.Join("..", "..", "..", "services", "atlas-configurations", "seed-data", "templates")
}

func realAtlasPacketDir() string {
	return filepath.Join("..", "..", "..", "libs", "atlas-packet")
}

func realRegistryDir() string {
	return filepath.Join("..", "..", "..", "docs", "packets", "registry")
}

// copyRealTemplatesToTemp copies every real seed template listed in
// seedFNameVersions into a fresh temp dir and returns that dir plus the
// stems actually found on disk. It never writes into realTemplateDir().
func copyRealTemplatesToTemp(t *testing.T) (tplDir string, stems []string) {
	t.Helper()
	tplDir = t.TempDir()
	for _, v := range seedFNameVersions {
		src := filepath.Join(realTemplateDir(), "template_"+v.Template+".json")
		b, err := os.ReadFile(src)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatalf("read real template %s: %v", src, err)
		}
		writeSeedFile(t, filepath.Join(tplDir, "template_"+v.Template+".json"), string(b))
		stems = append(stems, v.Template)
	}
	if len(stems) == 0 {
		t.Fatal("no real seed templates found - realTemplateDir() path may be wrong")
	}
	return tplDir, stems
}

// assertSemanticFidelity parses origPath and newPath as generic JSON and
// asserts they are deeply equal once any "fname" key is stripped from both
// sides. "fname" is stripped symmetrically - not just from newPath - because
// origPath is read live from the real, committed seed templates, which (as of
// task-194) already carry fname for 2845 of 2859 entries; a re-run
// regenerating an already-populated entry must not be flagged as drift.
//
// IMPORTANT SCOPE NOTE: because origPath now already carries fname for most
// entries, this helper - via its only caller,
// TestSeedFName_RealTemplatesSemanticFidelity - predominantly exercises the
// "re-resolve and overwrite an already-present fname" path, not the "insert a
// brand-new fname into a virgin entry" path, for those 2845 entries. Only the
// 14 still-unresolved entries still exercise fresh insertion through this
// helper. It does NOT prove that inserting fname into a virgin (fname-less)
// entry is safe across the full corpus - that is
// TestSeedFName_RealTemplatesInsertionCoverage's job, which synthesizes a
// virgin corpus in memory by recursively stripping fname from the real
// templates and asserts fidelity plus reproducibility against all 2859
// entries. What this helper alone still proves, unconditionally: no
// addition, removal, or value change beyond fname is permitted anywhere else
// - any top-level section (characters, worlds, cashShop, ...), any entry
// field, any options map - it fails with the exact key path if one occurs.
func assertSemanticFidelity(t *testing.T, label, origPath, newPath string) {
	t.Helper()
	origB, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("%s: read original: %v", label, err)
	}
	newB, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("%s: read generated: %v", label, err)
	}
	var orig, generated map[string]any
	if err := json.Unmarshal(origB, &orig); err != nil {
		t.Fatalf("%s: parse original: %v", label, err)
	}
	if err := json.Unmarshal(newB, &generated); err != nil {
		t.Fatalf("%s: parse generated: %v", label, err)
	}

	for _, k := range []string{"region", "majorVersion", "minorVersion", "usesPin", "characters", "npcs", "worlds", "cashShop"} {
		if !reflect.DeepEqual(orig[k], generated[k]) {
			t.Errorf("%s: top-level %q changed:\n before: %#v\n after : %#v", label, k, orig[k], generated[k])
		}
	}

	origSock, _ := orig["socket"].(map[string]any)
	genSock, _ := generated["socket"].(map[string]any)
	for _, group := range []string{"handlers", "writers"} {
		origEntries, _ := origSock[group].([]any)
		genEntries, _ := genSock[group].([]any)
		if len(origEntries) != len(genEntries) {
			t.Fatalf("%s: socket.%s length changed: %d -> %d", label, group, len(origEntries), len(genEntries))
		}
		for i := range origEntries {
			oe, _ := origEntries[i].(map[string]any)
			ge, _ := genEntries[i].(map[string]any)
			oeCopy := make(map[string]any, len(oe))
			for k, v := range oe {
				oeCopy[k] = v
			}
			delete(oeCopy, "fname")
			geCopy := make(map[string]any, len(ge))
			for k, v := range ge {
				geCopy[k] = v
			}
			delete(geCopy, "fname") // the one key this generator is allowed to add or change
			if !reflect.DeepEqual(oeCopy, geCopy) {
				t.Errorf("%s: socket.%s[%d] changed beyond fname:\n before: %#v\n after : %#v (fname stripped from both)",
					label, group, i, oeCopy, geCopy)
			}
		}
	}
}

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

// testTemplate83 writes "services" compactly on one line, matching the real
// seed templates' own convention (measured: all 2,859 socket entries across
// all 11 real files write "services" that way, e.g. ["login"] or
// ["login","channel"] - never one element per line). That convention is NOT
// preserved by this tool's write path - see the seedDoc doc-comment in
// seed_fname.go - so any assertion here about the exact bytes of a "services"
// array is deliberately avoided; the fixture just documents what the real
// corpus looks like going in.
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

// TestSeedFName_FailsLoudlyOnUnknownSocketKey covers the guard's third level:
// a key on the "socket" object itself, sibling to "handlers"/"writers"
// (e.g. a future "somethingNew" section), rather than at the top level of the
// document or inside one handler/writer entry. seedSocket only models
// "handlers" and "writers", so without this check such a key would be
// silently dropped by writeSeedTemplate's re-marshal - exactly the failure
// mode the two other guards exist to prevent.
func TestSeedFName_FailsLoudlyOnUnknownSocketKey(t *testing.T) {
	surprising := strings.Replace(testTemplate83, `"socket": {`,
		"\"socket\": {\n    \"surpriseSocketKey\": {\"a\": 1},", 1)
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": testRegistryV83},
		map[string]string{"gms_83_1": surprising})

	var stderr bytes.Buffer
	code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr)
	if code == 0 {
		t.Fatal("exit = 0, want non-zero on an unmodelled socket key")
	}
	if !strings.Contains(stderr.String(), "surpriseSocketKey") {
		t.Errorf("stderr did not name the offending key:\n%s", stderr.String())
	}
}

// TestSeedFName_PreservesUnsupportedSocketSection pins the one socket key
// besides "handlers"/"writers" that atlas-configurations really does model:
// "unsupported" (socket.UnsupportedRestModel - the audited-and-absent list,
// two arrays of implementation names). seed-fname does not resolve fnames for
// it, so it must survive a --write round-trip verbatim rather than tripping
// the unknown-socket-key guard.
func TestSeedFName_PreservesUnsupportedSocketSection(t *testing.T) {
	withUnsupported := strings.Replace(testTemplate83, "\n  },\n  \"characters\"",
		",\n    \"unsupported\": {\n      \"handlers\": [\"MapleLifeCheckNameHandle\"],\n"+
			"      \"writers\": [\"MapleLifeResult\", \"MapleLifeError\"]\n    }\n  },\n  \"characters\"", 1)
	if withUnsupported == testTemplate83 {
		t.Fatal("test fixture did not gain an unsupported section")
	}
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": testRegistryV83},
		map[string]string{"gms_83_1": withUnsupported})
	path := filepath.Join(tplDir, "template_gms_83_1.json")

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got seedDoc
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Socket.Unsupported.Handlers) != 1 || got.Socket.Unsupported.Handlers[0] != "MapleLifeCheckNameHandle" {
		t.Errorf("unsupported.handlers = %v, want [MapleLifeCheckNameHandle]", got.Socket.Unsupported.Handlers)
	}
	want := []string{"MapleLifeResult", "MapleLifeError"}
	if !reflect.DeepEqual(got.Socket.Unsupported.Writers, want) {
		t.Errorf("unsupported.writers = %v, want %v", got.Socket.Unsupported.Writers, want)
	}
}

// TestSeedFName_OmitsAbsentUnsupportedSection is the other half of the
// round-trip contract: a template with no "unsupported" key must not grow an
// empty one. Normalizing absent -> {"handlers":[],"writers":[]} is
// atlas-configurations' job at its REST boundary, not the seed generator's.
func TestSeedFName_OmitsAbsentUnsupportedSection(t *testing.T) {
	regDir, tplDir := setupSeedFName(t,
		map[string]string{"gms_v83": testRegistryV83},
		map[string]string{"gms_83_1": testTemplate83})
	path := filepath.Join(tplDir, "template_gms_83_1.json")

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", regDir, "--template-dir", tplDir}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "unsupported") {
		t.Errorf("write invented an unsupported section:\n%s", b)
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

// TestSeedFName_AmbiguityPrefersBoundImplementationOverLexicographicOrder is
// the task-19/task-207 fix-up regression test: jms opcode 167 (0xA7) carries
// two real senders in the registry (CASHSHOP_SURPRISE -> fname
// CCashShop::SendChangeMaplePoint, and CASH_ITEM_GACHAPON_BUTTON -> fname
// CUICashItemGachapon::OnButtonClicked, task-3's human-approved dual-row
// ruling) and CASHSHOP_SURPRISE sorts lexicographically first ('S' 0x53 <
// '_' 0x5F). The template's committed `handler` field is
// "CashItemGachaponHandle" - the Operation() constant
// cash/serverbound/item_gachapon_button.go's CashItemGachaponButton struct
// actually returns - which is CUICashItemGachapon::OnButtonClicked's fname,
// NOT the lexicographically-first op's. The bound-implementation tie-break
// must win over the lexicographic fallback here. This test fails against the
// pre-fix tie-break (which always picks the lexicographically-first op and
// would resolve to CCashShop::SendChangeMaplePoint instead).
func TestSeedFName_AmbiguityPrefersBoundImplementationOverLexicographicOrder(t *testing.T) {
	const reg = `- op: CASHSHOP_SURPRISE
  direction: serverbound
  opcode: 167
  fname: CCashShop::SendChangeMaplePoint
  provenance: manual
- op: CASH_ITEM_GACHAPON_BUTTON
  direction: serverbound
  opcode: 167
  fname: CUICashItemGachapon::OnButtonClicked
  provenance: manual
`
	const tpl = `{
  "region": "JMS",
  "majorVersion": 185,
  "minorVersion": 1,
  "usesPin": false,
  "socket": {
    "handlers": [
      {
        "opCode": "0xA7",
        "validator": "LoggedInValidator",
        "handler": "CashItemGachaponHandle",
        "services": [
          "channel"
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
		map[string]string{"jms_v185": reg},
		map[string]string{"jms_185_1": tpl})

	var stderr bytes.Buffer
	code := runSeedFName([]string{
		"--write", "--registry-dir", regDir, "--template-dir", tplDir,
		"--atlas-packet", realAtlasPacketDir(),
	}, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}
	h, _ := socketOf(t, filepath.Join(tplDir, "template_jms_185_1.json"))
	if got := string(h[0]["fname"]); got != `"CUICashItemGachapon::OnButtonClicked"` {
		t.Errorf("tie-break did not pick the bound implementation (CashItemGachaponHandle -> "+
			"CUICashItemGachapon::OnButtonClicked): fname = %s", got)
	}
	if !strings.Contains(stderr.String(), "ambiguous") || !strings.Contains(stderr.String(), "bound implementation") {
		t.Errorf("tie-break decision was not logged as bound-implementation-driven:\n%s", stderr.String())
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

// TestSeedFName_NormalizesRealWorldFormattingQuirks documents (and pins) what
// happens to the two real-file conventions that make a naive byte comparison
// against the source fail: (1) entries that carry both "services" and
// "options" always order services FIRST in the source, but seedEntry's fixed
// struct field order writes Options before Services; (2) gms_12_1.json and
// gms_92_1.json inline the first array element's opening brace onto the
// array's own line ("handlers": [{ ... instead of "handlers": [\n  { ...).
// json.MarshalIndent normalizes BOTH away - on the whole document, not just
// the entries that changed - which is the accepted tradeoff (see the seedDoc
// doc-comment in seed_fname.go): a simpler, more robust generator over a
// minimal diff. This test's job is to prove that normalization is
// deterministic and that no VALUE is lost in the process, on both an entry
// that resolves an fname and one that does not.
func TestSeedFName_NormalizesRealWorldFormattingQuirks(t *testing.T) {
	const reg = `- op: LOGIN_STATUS
  direction: serverbound
  opcode: 1
  fname: CLogin::SendCheckPasswordPacket
  provenance: manual
`
	// opCode 0x02 (CharacterMoveHandle) has no registry match; opCode 0x01
	// (LoginHandle) resolves. Both start from the real inlined-"[{"/
	// services-before-options source style.
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

	// The inlined "[{" is gone: MarshalIndent's standard one-element-per-line
	// array style is now used uniformly, including for the untouched entry.
	if strings.Contains(got, "[{") {
		t.Errorf("expected the inlined \"[{\" array-open style to be normalized away, still present:\n%s", got)
	}

	// The untouched second entry's VALUES survive even though its formatting
	// (and options-before-services key order) changed.
	h, _ := socketOf(t, path)
	if len(h) != 2 {
		t.Fatalf("expected 2 handler entries, got %d", len(h))
	}
	var opts map[string]any
	if err := json.Unmarshal(h[1]["options"], &opts); err != nil {
		t.Fatalf("parse normalized options: %v", err)
	}
	types, _ := opts["types"].([]any)
	if len(types) != 1 {
		t.Fatalf("options.types value was lost or corrupted by normalization: %v", opts)
	}
	first, _ := types[0].(map[string]any)
	if first["Name"] != "NORMAL" {
		t.Errorf("options.types[0].Name was lost or corrupted by normalization: %v", first)
	}
	var services []string
	if err := json.Unmarshal(h[1]["services"], &services); err != nil {
		t.Fatalf("parse normalized services: %v", err)
	}
	if len(services) != 1 || services[0] != "channel" {
		t.Errorf("services value was lost or corrupted by normalization: %v", services)
	}
	if _, present := h[1]["fname"]; present {
		t.Errorf("unresolved entry gained an fname key it should not have: %v", h[1])
	}

	// The resolved first entry gained the correct fname.
	if got := string(h[0]["fname"]); got != `"CLogin::SendCheckPasswordPacket"` {
		t.Errorf("resolved entry's fname is wrong: fname = %s", got)
	}
}

// TestSeedFName_ReRunIsIdempotent proves the "re-runnable" design goal: running
// --write a second time over output that already carries the correct fname
// values changes nothing on disk. The first run normalizes formatting AND
// resolves fname; loadSeedTemplate on the second run reads that already-
// resolved value back into seedEntry.FName, applyDirect resolves the SAME
// value again, and json.MarshalIndent is a deterministic function of the
// (unchanged) seedDoc - so the second pass's bytes equal the first pass's.
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
// entry already has a DIFFERENT "fname" on disk. Because FName is a single
// struct field, applyDirect overwriting it and re-marshalling naturally
// replaces the old value - there is no separate splice/insert path that could
// leave the stale value in place or duplicate the key.
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

// TestSeedFName_RealTemplatesSemanticFidelity runs --write against a temp
// copy of all 11 REAL seed templates, joined against the REAL registries, and
// asserts every parsed template is deeply equal to its original except for
// "fname" keys - nothing in characters, worlds, cashShop, options maps,
// opcode spellings, or service lists may appear, disappear, or change value.
// The real seed templates themselves are only ever read, never written (the
// --template-dir passed to runSeedFName is always the temp copy).
//
// Since task-194 backfilled fname into the real committed templates, the
// "original" side read here (realTemplateDir()) already carries fname for
// 2845 of 2859 entries. This test therefore mostly proves fidelity of the
// "re-resolve and overwrite an already-present fname" path, and is
// materially redundant with TestSeedFName_RealTemplatesIdempotentSecondRun
// for those entries. It does NOT give full-corpus coverage of the "insert
// fname into a virgin entry" path - see
// TestSeedFName_RealTemplatesInsertionCoverage for that guarantee.
func TestSeedFName_RealTemplatesSemanticFidelity(t *testing.T) {
	tplDir, stems := copyRealTemplatesToTemp(t)

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", realRegistryDir(), "--template-dir", tplDir, "--atlas-packet", realAtlasPacketDir()}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}

	for _, stem := range stems {
		origPath := filepath.Join(realTemplateDir(), "template_"+stem+".json")
		newPath := filepath.Join(tplDir, "template_"+stem+".json")
		assertSemanticFidelity(t, stem, origPath, newPath)
	}
}

// stripFnameRecursive returns a deep copy of v with every map key named
// "fname" removed, at any nesting depth. v must be the result of
// json.Unmarshal into `any` (so only map[string]any, []any, and JSON scalars
// are expected).
func stripFnameRecursive(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			if k == "fname" {
				continue
			}
			out[k] = stripFnameRecursive(vv)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, vv := range t {
			out[i] = stripFnameRecursive(vv)
		}
		return out
	default:
		return v
	}
}

// copyRealTemplatesVirginToTemp reconstructs the pre-task-194 "virgin"
// corpus - the fname-less input that the real templates derive from - by
// reading every real seed template and recursively stripping every "fname"
// key, then writing the result into a fresh temp dir. It never touches the
// real files. It returns the temp dir, the stems found, and a map of
// stem -> parsed-and-stripped struct so callers can assert fidelity without
// re-parsing.
func copyRealTemplatesVirginToTemp(t *testing.T) (tplDir string, stems []string, virgin map[string]map[string]any) {
	t.Helper()
	tplDir = t.TempDir()
	virgin = make(map[string]map[string]any)
	for _, v := range seedFNameVersions {
		src := filepath.Join(realTemplateDir(), "template_"+v.Template+".json")
		b, err := os.ReadFile(src)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatalf("read real template %s: %v", src, err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(b, &parsed); err != nil {
			t.Fatalf("parse real template %s: %v", src, err)
		}
		stripped, ok := stripFnameRecursive(parsed).(map[string]any)
		if !ok {
			t.Fatalf("strip fname from real template %s: unexpected top-level type", src)
		}
		out, err := json.MarshalIndent(stripped, "", "  ")
		if err != nil {
			t.Fatalf("marshal virgin template %s: %v", src, err)
		}
		writeSeedFile(t, filepath.Join(tplDir, "template_"+v.Template+".json"), string(out))
		stems = append(stems, v.Template)
		virgin[v.Template] = stripped
	}
	if len(stems) == 0 {
		t.Fatal("no real seed templates found - realTemplateDir() path may be wrong")
	}
	return tplDir, stems, virgin
}

// assertFnamesMatch compares only the fname value of every socket
// handlers/writers entry, index for index, between real (parsed from the
// committed on-disk template) and generated (parsed from a fresh --write run
// over a synthesized virgin copy of that same template). Entries where
// neither side resolved a fname (nil == nil, the 14 still-unresolved
// entries) compare equal.
func assertFnamesMatch(t *testing.T, label string, real, generated map[string]any) {
	t.Helper()
	realSock, _ := real["socket"].(map[string]any)
	genSock, _ := generated["socket"].(map[string]any)
	for _, group := range []string{"handlers", "writers"} {
		realEntries, _ := realSock[group].([]any)
		genEntries, _ := genSock[group].([]any)
		if len(realEntries) != len(genEntries) {
			t.Fatalf("%s: socket.%s length mismatch between committed and regenerated-from-virgin: %d vs %d",
				label, group, len(realEntries), len(genEntries))
		}
		for i := range realEntries {
			re, _ := realEntries[i].(map[string]any)
			ge, _ := genEntries[i].(map[string]any)
			rf := re["fname"]
			gf := ge["fname"]
			if !reflect.DeepEqual(rf, gf) {
				t.Errorf("%s: socket.%s[%d] fname mismatch: committed=%#v regenerated-from-virgin=%#v",
					label, group, i, rf, gf)
			}
		}
	}
}

// TestSeedFName_RealTemplatesInsertionCoverage restores full-corpus coverage
// of the "insert fname into a virgin entry" path that
// TestSeedFName_RealTemplatesSemanticFidelity lost once task-194 backfilled
// fname into the real committed templates (see that test's doc comment). It
// synthesizes a virgin (fname-less) copy of all 11 real templates in memory -
// via copyRealTemplatesVirginToTemp, which recursively strips every "fname"
// key from the real files without ever writing to them - runs --write
// against that virgin copy joined with the REAL registries, and asserts two
// things for every one of the 2859 entries:
//
//  1. Semantic fidelity: the generated output, with fname stripped back out,
//     is deeply equal to the virgin input. Nothing but fname may appear,
//     disappear, or change.
//  2. Insertion completeness (the stronger claim): the fname value the
//     generator inserts into each virgin entry is identical, entry for
//     entry, to the fname value already committed in the real template (or
//     absent from both, for the 14 still-unresolved entries). This proves
//     the committed corpus is reproducible from a virgin corpus plus the
//     current registries - not a one-off, hand-editable artifact that could
//     silently drift from what --write would actually produce.
func TestSeedFName_RealTemplatesInsertionCoverage(t *testing.T) {
	tplDir, stems, virgin := copyRealTemplatesVirginToTemp(t)

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", realRegistryDir(), "--template-dir", tplDir, "--atlas-packet", realAtlasPacketDir()}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}

	for _, stem := range stems {
		newPath := filepath.Join(tplDir, "template_"+stem+".json")
		newB, err := os.ReadFile(newPath)
		if err != nil {
			t.Fatalf("%s: read generated: %v", stem, err)
		}
		var generated map[string]any
		if err := json.Unmarshal(newB, &generated); err != nil {
			t.Fatalf("%s: parse generated: %v", stem, err)
		}

		genStripped, ok := stripFnameRecursive(generated).(map[string]any)
		if !ok {
			t.Fatalf("%s: strip fname from generated: unexpected top-level type", stem)
		}
		if !reflect.DeepEqual(virgin[stem], genStripped) {
			t.Errorf("%s: generated output (fname stripped) differs from the virgin input - the write touched something beyond fname", stem)
		}

		realPath := filepath.Join(realTemplateDir(), "template_"+stem+".json")
		realB, err := os.ReadFile(realPath)
		if err != nil {
			t.Fatalf("%s: read real committed template: %v", stem, err)
		}
		var real map[string]any
		if err := json.Unmarshal(realB, &real); err != nil {
			t.Fatalf("%s: parse real committed template: %v", stem, err)
		}
		assertFnamesMatch(t, stem, real, generated)
	}
}

// TestSeedFName_RealTemplatesIdempotentSecondRun runs --write twice over a
// temp copy of all 11 real templates against the real registries: the first
// pass normalizes formatting and resolves fname, the second pass must be a
// complete no-op (byte-identical output), proving the tool converges rather
// than drifting or duplicating fields on repeated runs - the "stays
// re-runnable when the next client-version bring-up adds a registry"
// requirement from the design.
func TestSeedFName_RealTemplatesIdempotentSecondRun(t *testing.T) {
	tplDir, stems := copyRealTemplatesToTemp(t)

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", realRegistryDir(), "--template-dir", tplDir, "--atlas-packet", realAtlasPacketDir()}, &stderr); code != 0 {
		t.Fatalf("first run: exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}
	firstPass := make(map[string][]byte, len(stems))
	for _, stem := range stems {
		b, err := os.ReadFile(filepath.Join(tplDir, "template_"+stem+".json"))
		if err != nil {
			t.Fatalf("%s: read after first run: %v", stem, err)
		}
		firstPass[stem] = b
	}

	stderr.Reset()
	if code := runSeedFName([]string{"--write", "--registry-dir", realRegistryDir(), "--template-dir", tplDir, "--atlas-packet", realAtlasPacketDir()}, &stderr); code != 0 {
		t.Fatalf("second run: exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}
	for _, stem := range stems {
		b, err := os.ReadFile(filepath.Join(tplDir, "template_"+stem+".json"))
		if err != nil {
			t.Fatalf("%s: read after second run: %v", stem, err)
		}
		if string(b) != string(firstPass[stem]) {
			t.Errorf("%s: second --write run was not a no-op - output changed after the file was already normalized", stem)
		}
	}
}

// TestSeedFName_PreservesOriginalOpcodeSpelling proves reformatting never
// rewrites a VALUE, specifically the opCode string itself: the real corpus's
// two extremes are "0x9" (one hex digit, template_jms_185_1.json) and "0x0A5"
// (three hex digits with a leading zero, template_gms_84_1.json). Both must
// survive --write character-for-character, not canonicalized to "0x09" or
// "0xA5" or any other spelling - opcodes are parsed numerically for matching
// but never rewritten.
func TestSeedFName_PreservesOriginalOpcodeSpelling(t *testing.T) {
	tplDir := t.TempDir()
	for _, stem := range []string{"jms_185_1", "gms_84_1"} {
		src := filepath.Join(realTemplateDir(), "template_"+stem+".json")
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read real template %s: %v", src, err)
		}
		writeSeedFile(t, filepath.Join(tplDir, "template_"+stem+".json"), string(b))
	}

	var stderr bytes.Buffer
	if code := runSeedFName([]string{"--write", "--registry-dir", realRegistryDir(), "--template-dir", tplDir, "--atlas-packet", realAtlasPacketDir()}, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0. stderr:\n%s", code, stderr.String())
	}

	assertHasOpCode := func(stem, wantOpCode string) {
		h, w := socketOf(t, filepath.Join(tplDir, "template_"+stem+".json"))
		for _, e := range append(h, w...) {
			var oc string
			if err := json.Unmarshal(e["opCode"], &oc); err != nil {
				t.Fatalf("%s: parse opCode: %v", stem, err)
			}
			if oc == wantOpCode {
				return
			}
		}
		t.Errorf("%s: opCode %q did not survive --write unchanged", stem, wantOpCode)
	}
	assertHasOpCode("jms_185_1", "0x9")
	assertHasOpCode("gms_84_1", "0x0A5")
}
