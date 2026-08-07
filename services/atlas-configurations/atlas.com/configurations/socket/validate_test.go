package socket

import (
	"testing"

	opcodes "github.com/Chronicle20/atlas/libs/atlas-opcodes"
)

func TestParseOpCode(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		want   int
		wantOk bool
	}{
		{"two digit lower prefix", "0x2a", 42, true},
		{"two digit upper hex", "0x2A", 42, true},
		{"upper X prefix", "0X2A", 42, true},
		{"single digit", "0x9", 9, true},
		{"three digit padded", "0x0A5", 165, true},
		{"four digit", "0xFFFF", 65535, true},
		{"five digits rejected", "0x10000", 0, false},
		{"missing prefix rejected", "2A", 0, false},
		{"decimal rejected", "42", 0, false},
		{"empty rejected", "", 0, false},
		{"non hex rejected", "0xZZ", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseOpCode(tt.raw)
			if ok != tt.wantOk {
				t.Fatalf("ParseOpCode(%q) ok = %v, want %v", tt.raw, ok, tt.wantOk)
			}
			if ok && got != tt.want {
				t.Errorf("ParseOpCode(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestValidate_AcceptsCleanInput(t *testing.T) {
	in := Input{
		Handlers: []Binding{
			{Name: "LoginHandle", OpCode: "0x01", Validator: "NoOpValidator", Services: []string{opcodes.ServiceLogin}},
			{Name: "NoOpHandler", OpCode: "0x17", Validator: "LoggedInValidator", Services: []string{opcodes.ServiceChannel}},
			{Name: "NoOpHandler", OpCode: "0x19", Validator: "LoggedInValidator", Services: []string{opcodes.ServiceChannel}},
		},
		Writers: []Binding{
			{Name: "AuthLoginFailed", OpCode: "0x01", Services: []string{opcodes.ServiceLogin}},
			{Name: "AuthPermanentBan", OpCode: "0x01", Services: []string{opcodes.ServiceLogin}},
			{Name: "MiniRoom", OpCode: "0x0A5", Services: []string{opcodes.ServiceChannel}},
			{Name: "MiniRoom", OpCode: "0xA8", Services: []string{opcodes.ServiceChannel}},
		},
		UnsupportedHandlers: []string{"GuestLoginHandle"},
	}
	if got := Validate(in); len(got) != 0 {
		t.Fatalf("Validate() returned %d issues on clean input: %+v", len(got), got)
	}
}

func TestValidate_Rules(t *testing.T) {
	tests := []struct {
		name     string
		in       Input
		wantPath string
		wantMsg  string
	}{
		{
			name: "FR-11.1 duplicate name and opcode in one collection",
			in: Input{Handlers: []Binding{
				{Name: "MiniRoom", OpCode: "0xB8", Validator: "LoggedInValidator", Services: []string{opcodes.ServiceChannel}},
				{Name: "MiniRoom", OpCode: "0x0B8", Validator: "LoggedInValidator", Services: []string{opcodes.ServiceChannel}},
			}},
			wantPath: "socket.handlers[1].opCode",
			wantMsg:  `"MiniRoom" is already bound to opcode 0xB8`,
		},
		{
			name: "FR-11.2 malformed opcode",
			in: Input{Writers: []Binding{
				{Name: "AuthSuccess", OpCode: "B8", Services: []string{opcodes.ServiceLogin}},
			}},
			wantPath: "socket.writers[0].opCode",
			wantMsg:  `opCode "B8" must match 0x followed by 1-4 hex digits`,
		},
		{
			name: "FR-11.3 name both defined and unsupported",
			in: Input{
				Handlers:            []Binding{{Name: "LoginHandle", OpCode: "0x01", Validator: "NoOpValidator", Services: []string{opcodes.ServiceLogin}}},
				UnsupportedHandlers: []string{"LoginHandle"},
			},
			wantPath: "socket.unsupported.handlers[0]",
			wantMsg:  `"LoginHandle" is marked unsupported but is also defined in socket.handlers`,
		},
		{
			name: "FR-11.4 empty handler validator",
			in: Input{Handlers: []Binding{
				{Name: "LoginHandle", OpCode: "0x01", Validator: "", Services: []string{opcodes.ServiceLogin}},
			}},
			wantPath: "socket.handlers[0].validator",
			wantMsg:  `validator is required for handler "LoginHandle"`,
		},
		{
			name: "FR-11.4 whitespace-only handler validator",
			in: Input{Handlers: []Binding{
				{Name: "LoginHandle", OpCode: "0x01", Validator: "  ", Services: []string{opcodes.ServiceLogin}},
			}},
			wantPath: "socket.handlers[0].validator",
			wantMsg:  `validator is required for handler "LoginHandle"`,
		},
		{
			name: "FR-11.5 unknown service",
			in: Input{Writers: []Binding{
				{Name: "AuthSuccess", OpCode: "0x00", Services: []string{"drops"}},
			}},
			wantPath: "socket.writers[0].services[0]",
			wantMsg:  `unknown service "drops"; expected one of login, channel`,
		},
		{
			name:     "duplicate unsupported name",
			in:       Input{UnsupportedWriters: []string{"MonsterCarnival", "MonsterCarnival"}},
			wantPath: "socket.unsupported.writers[1]",
			wantMsg:  `"MonsterCarnival" is listed more than once`,
		},
		{
			name: "empty definition name",
			in: Input{Writers: []Binding{
				{Name: "", OpCode: "0x00", Services: []string{opcodes.ServiceLogin}},
			}},
			wantPath: "socket.writers[0].writer",
			wantMsg:  "definition name is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Validate(tt.in)
			for _, iss := range got {
				if iss.Path == tt.wantPath && iss.Message == tt.wantMsg {
					return
				}
			}
			t.Errorf("Validate() = %+v\nwant an issue at %q with message %q", got, tt.wantPath, tt.wantMsg)
		})
	}
}

// FR-11.6: several writers legitimately share one opcode. gms_12_1 has
// AuthPermanentBan and AuthLoginFailed both at 0x01. This must never fail.
func TestValidate_DuplicateOpCodeAcrossNamesIsLegal(t *testing.T) {
	in := Input{Writers: []Binding{
		{Name: "AuthPermanentBan", OpCode: "0x01", Services: []string{opcodes.ServiceLogin}},
		{Name: "AuthLoginFailed", OpCode: "0x01", Services: []string{opcodes.ServiceLogin}},
	}}
	if got := Validate(in); len(got) != 0 {
		t.Fatalf("Validate() rejected a legal shared opcode: %+v", got)
	}
}

// Writers carry no validator; an empty one must never be reported for them.
func TestValidate_WritersNeedNoValidator(t *testing.T) {
	in := Input{Writers: []Binding{
		{Name: "AuthSuccess", OpCode: "0x00", Validator: "", Services: []string{opcodes.ServiceLogin}},
	}}
	if got := Validate(in); len(got) != 0 {
		t.Fatalf("Validate() required a validator on a writer: %+v", got)
	}
}

// An entry with no services applies to every service (libs/atlas-opcodes
// appliesToService treats an empty list as universal), so it is legal.
func TestValidate_EmptyServicesIsLegal(t *testing.T) {
	in := Input{Writers: []Binding{{Name: "AuthSuccess", OpCode: "0x00"}}}
	if got := Validate(in); len(got) != 0 {
		t.Fatalf("Validate() rejected an untagged entry: %+v", got)
	}
}
