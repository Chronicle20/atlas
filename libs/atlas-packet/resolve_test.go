package atlas_packet

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestResolveCodeValid(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			"LEVEL_UP":  float64(0),
			"SKILL_USE": float64(1),
		},
	}
	assert.Equal(t, byte(0), ResolveCode(l, options, "operations", "LEVEL_UP"))
	assert.Equal(t, byte(1), ResolveCode(l, options, "operations", "SKILL_USE"))
}

func TestResolveCodeMissingProperty(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{}
	assert.Equal(t, byte(99), ResolveCode(l, options, "operations", "LEVEL_UP"))
}

func TestResolveCodeMissingKey(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"operations": map[string]interface{}{},
	}
	assert.Equal(t, byte(99), ResolveCode(l, options, "operations", "MISSING"))
}

func TestResolveCodeWrongType(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"operations": "not a map",
	}
	assert.Equal(t, byte(99), ResolveCode(l, options, "operations", "LEVEL_UP"))
}

func TestResolveCodeHexString(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			"QUEST_RECORD":    "0x01",
			"SYSTEM_MESSAGE":  "0x09",
			"QUEST_RECORD_EX": "0x0A",
		},
	}
	assert.Equal(t, byte(0x01), ResolveCode(l, options, "operations", "QUEST_RECORD"))
	assert.Equal(t, byte(0x09), ResolveCode(l, options, "operations", "SYSTEM_MESSAGE"))
	assert.Equal(t, byte(0x0A), ResolveCode(l, options, "operations", "QUEST_RECORD_EX"))
}

func TestResolveCodeDecimalString(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			"SHOW": "3",
		},
	}
	assert.Equal(t, byte(3), ResolveCode(l, options, "operations", "SHOW"))
}

func TestResolveCodeUnparseableString(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			"LEVEL_UP": "not-a-number",
		},
	}
	assert.Equal(t, byte(99), ResolveCode(l, options, "operations", "LEVEL_UP"))
}

func TestResolveCodeUnsupportedType(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			"LEVEL_UP": true,
		},
	}
	assert.Equal(t, byte(99), ResolveCode(l, options, "operations", "LEVEL_UP"))
}

func TestResolveNameValid(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"messageType": map[string]interface{}{
			"SAY":      float64(0),
			"ASK_MENU": float64(4),
		},
	}
	name, ok := ResolveName(l, options, "messageType", 4)
	assert.True(t, ok)
	assert.Equal(t, "ASK_MENU", name)

	name, ok = ResolveName(l, options, "messageType", 0)
	assert.True(t, ok)
	assert.Equal(t, "SAY", name)
}

func TestResolveNameHexString(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"messageType": map[string]interface{}{
			"ASK_MENU": "0x04",
		},
	}
	name, ok := ResolveName(l, options, "messageType", 4)
	assert.True(t, ok)
	assert.Equal(t, "ASK_MENU", name)
}

func TestResolveNameMiss(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"messageType": map[string]interface{}{
			"ASK_MENU": float64(4),
		},
	}
	_, ok := ResolveName(l, options, "messageType", 7)
	assert.False(t, ok)
}

func TestResolveNameMissingProperty(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	_, ok := ResolveName(l, map[string]interface{}{}, "messageType", 0)
	assert.False(t, ok)
}

func TestResolveNameWrongType(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"messageType": "not a map",
	}
	_, ok := ResolveName(l, options, "messageType", 0)
	assert.False(t, ok)
}

func TestResolveCodeResolveNameRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"messageType": map[string]interface{}{
			"ASK_MENU":       float64(4),
			"ASK_AVATAR":     float64(7),
			"ASK_SLIDE_MENU": float64(14),
		},
	}
	for _, key := range []string{"ASK_MENU", "ASK_AVATAR", "ASK_SLIDE_MENU"} {
		code := ResolveCode(l, options, "messageType", key)
		name, ok := ResolveName(l, options, "messageType", code)
		assert.True(t, ok)
		assert.Equal(t, key, name)
	}
}

func TestResolveCode16(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"petSkill": map[string]interface{}{
			"consumeHP":    "0x20",
			"autoSpeaking": "0x100",
			"asNumber":     float64(64),
			"asDecimal":    "64",
			"overflow":     float64(70000),
			"bad":          "zzz",
		},
		"notMap": "not a map",
	}

	if v, ok := ResolveCode16(l, options, "petSkill", "consumeHP"); !ok || v != 0x20 {
		t.Errorf("consumeHP = %#x,%v; want 0x20,true", v, ok)
	}
	if v, ok := ResolveCode16(l, options, "petSkill", "autoSpeaking"); !ok || v != 0x100 {
		t.Errorf("autoSpeaking = %#x,%v; want 0x100,true", v, ok)
	}
	if v, ok := ResolveCode16(l, options, "petSkill", "asNumber"); !ok || v != 64 {
		t.Errorf("asNumber = %d,%v; want 64,true", v, ok)
	}
	if v, ok := ResolveCode16(l, options, "petSkill", "asDecimal"); !ok || v != 64 {
		t.Errorf("asDecimal = %d,%v; want 64,true", v, ok)
	}
	// soft misses: absent key, absent property, unparseable value, out-of-range float64, non-map property
	if _, ok := ResolveCode16(l, options, "petSkill", "recall"); ok {
		t.Error("absent key resolved ok=true, want false")
	}
	if _, ok := ResolveCode16(l, options, "nope", "consumeHP"); ok {
		t.Error("absent property resolved ok=true, want false")
	}
	if _, ok := ResolveCode16(l, options, "petSkill", "bad"); ok {
		t.Error("unparseable value resolved ok=true, want false")
	}
	if _, ok := ResolveCode16(l, options, "petSkill", "overflow"); ok {
		t.Error("out-of-range float64 resolved ok=true, want false")
	}
	if _, ok := ResolveCode16(l, options, "notMap", "someKey"); ok {
		t.Error("non-map property resolved ok=true, want false")
	}
}

func TestResolveValueValid(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	options := map[string]interface{}{
		"skills": map[string]interface{}{
			"BATTLESHIP_HP_GAUGE": float64(5221999),
		},
		"vehicles": map[string]interface{}{
			"CORSAIR_BATTLESHIP": "0x1D7AE0", // 1932000 — see R-10; 0x1D7B60 is 1932128, NOT 1932000
		},
	}
	v, ok := ResolveValue(l, options, "skills", "BATTLESHIP_HP_GAUGE")
	assert.True(t, ok)
	assert.Equal(t, uint32(5221999), v)
	v, ok = ResolveValue(l, options, "vehicles", "CORSAIR_BATTLESHIP")
	assert.True(t, ok)
	assert.Equal(t, uint32(1932000), v)
}

func TestResolveValueMisses(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	cases := []struct {
		name    string
		options map[string]interface{}
	}{
		{"missing property", map[string]interface{}{}},
		{"property not a map", map[string]interface{}{"skills": "nope"}},
		{"missing key", map[string]interface{}{"skills": map[string]interface{}{}}},
		{"unparseable string", map[string]interface{}{"skills": map[string]interface{}{"BATTLESHIP_HP_GAUGE": "zz"}}},
		{"unsupported type", map[string]interface{}{"skills": map[string]interface{}{"BATTLESHIP_HP_GAUGE": true}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, ok := ResolveValue(l, tc.options, "skills", "BATTLESHIP_HP_GAUGE")
			assert.False(t, ok)
			assert.Equal(t, uint32(0), v)
		})
	}
}

// TestCodeConfigured pins the arm-presence predicate that lets a caller skip a
// write for a dispatcher arm the client version does not have, instead of
// sending ResolveCode's 99 sentinel. Note the zero-valued case: a mode byte of
// 0 is a legitimate arm, so presence must be decided by key membership and
// never by the value being non-zero.
func TestCodeConfigured(t *testing.T) {
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			"LEVEL_UP":  float64(0),
			"SKILL_USE": float64(1),
			"AS_STRING": "0x15",
		},
		"notAMap": float64(3),
	}

	assert.True(t, CodeConfigured(options, "operations", "SKILL_USE"))
	assert.True(t, CodeConfigured(options, "operations", "LEVEL_UP"), "a zero mode byte is still a configured arm")
	assert.True(t, CodeConfigured(options, "operations", "AS_STRING"), "presence is key membership, independent of value type")

	assert.False(t, CodeConfigured(options, "operations", "MISSING"))
	assert.False(t, CodeConfigured(options, "missingProperty", "SKILL_USE"))
	assert.False(t, CodeConfigured(options, "notAMap", "SKILL_USE"))
	assert.False(t, CodeConfigured(map[string]interface{}{}, "operations", "SKILL_USE"))
	assert.False(t, CodeConfigured(nil, "operations", "SKILL_USE"))
}
