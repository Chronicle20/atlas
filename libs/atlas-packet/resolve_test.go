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
