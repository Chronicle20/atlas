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
			"LEVEL_UP": float64(0),
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
