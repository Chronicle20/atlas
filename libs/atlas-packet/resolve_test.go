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
