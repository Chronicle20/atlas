package mobskill

import (
	"atlas-data/xml"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestReadLevel_DurationIsMilliseconds pins the ONE seconds→milliseconds
// conversion in the system. WZ MobSkill.img authors `time` in seconds; every
// consumer downstream of this reader (atlas-monsters executeDebuff,
// buildMistCreateBody, executeStatBuff; atlas-maps mist tick) forwards the
// value verbatim as milliseconds. Mirrors skill/reader.go's convention
// (task-054). Do not add a second conversion anywhere downstream.
func TestReadLevel_DurationIsMilliseconds(t *testing.T) {
	node := xml.Node{
		Name: "2",
		IntegerNodes: []xml.IntegerNode{
			{Name: "time", Value: "15"},
		},
	}

	m := readLevel(logrus.New(), 126, 2, "126", node)

	if m.Duration != 15000 {
		t.Errorf("Duration: got %d, want 15000 (15s authored in WZ, emitted as ms)", m.Duration)
	}
}
