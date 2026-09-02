package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestTemplatesRouteImitatedNpcData proves the ImitatedNPCData and RemoveNPC
// writers (task-251, tasks 5 and 6) are wired into every seed template's
// socket.writers[] at the exact opcode the matching
// docs/packets/registry/<version>.yaml entry evidences for that version —
// or absent where no opcode is evidenced. A wiring mistake here is a silent
// mis-encode: the codec exists and compiles, but the server never invokes it
// (writer missing) or invokes it at the wrong opcode (wrong hex).
func TestTemplatesRouteImitatedNpcData(t *testing.T) {
	type entry struct {
		OpCode string `json:"opCode"`
		Writer string `json:"writer"`
	}
	type doc struct {
		Socket struct {
			Writers []entry `json:"writers"`
		} `json:"socket"`
	}

	// opCode == "" means the writer must be absent from the template.
	cases := []struct {
		file            string
		imitatedNpcData string
		removeNpc       string
	}{
		{"template_gms_12_1.json", "", ""},
		{"template_gms_48_1.json", "", "0xB2"},
		{"template_gms_61_1.json", "0x4E", "0xC3"},
		{"template_gms_72_1.json", "0x4E", "0xE4"},
		{"template_gms_79_1.json", "0x4E", "0xEC"},
		{"template_gms_83_1.json", "0x51", "0x102"},
		{"template_gms_84_1.json", "0x53", "0x109"},
		{"template_gms_87_1.json", "0x53", "0x113"},
		{"template_gms_92_1.json", "0x56", "0x130"},
		{"template_gms_95_1.json", "0x54", "0x138"},
		{"template_jms_185_1.json", "0x55", "0x117"},
	}

	dir := filepath.Join("..", "..", "seed-data", "templates")

	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join(dir, c.file))
			if err != nil {
				t.Fatalf("read %s: %v", c.file, err)
			}
			var d doc
			if err := json.Unmarshal(b, &d); err != nil {
				t.Fatalf("parse %s: %v", c.file, err)
			}

			byWriter := map[string][]string{}
			for _, w := range d.Socket.Writers {
				if w.Writer == "ImitatedNPCData" || w.Writer == "RemoveNPC" {
					byWriter[w.Writer] = append(byWriter[w.Writer], w.OpCode)
				}
			}

			assertOpCode(t, c.file, "ImitatedNPCData", byWriter["ImitatedNPCData"], c.imitatedNpcData)
			assertOpCode(t, c.file, "RemoveNPC", byWriter["RemoveNPC"], c.removeNpc)
		})
	}
}

func assertOpCode(t *testing.T, file, writer string, got []string, want string) {
	t.Helper()
	if len(got) > 1 {
		t.Errorf("%s: writer %q registered %d times, at %v", file, writer, len(got), got)
		return
	}
	var gotOpCode string
	if len(got) == 1 {
		gotOpCode = got[0]
	}
	if gotOpCode != want {
		if want == "" {
			t.Errorf("%s: writer %q present at opCode %q, want absent", file, writer, gotOpCode)
			return
		}
		if gotOpCode == "" {
			t.Errorf("%s: writer %q absent, want opCode %q", file, writer, want)
			return
		}
		t.Errorf("%s: writer %q opCode = %q, want %q", file, writer, gotOpCode, want)
	}
}
