package workers

import (
	"atlas-data/skill"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestJobStats_CountsImagesNumericAndWritten(t *testing.T) {
	var s jobStats
	rf := s.Wrap(func(path string) (int, error) {
		if filepath.Base(path) == "MobSkill.img.xml" {
			return 0, nil
		}
		if strings.Contains(path, "Dragon") {
			return 0, nil // FR-1.1: numeric image, no skill node, no document
		}
		return 1, nil
	})

	require.NoError(t, rf(filepath.Join("Skill.wz", "112.img.xml")))
	require.NoError(t, rf(filepath.Join("Skill.wz", "MobSkill.img.xml")))
	require.NoError(t, rf(filepath.Join("Skill.wz", "Dragon", "2200.img.xml")))

	require.Equal(t, 3, s.images)
	require.Equal(t, 2, s.numeric)
	require.Equal(t, 1, s.written)
}

func TestJobStats_PropagatesErrorAndCountsNoDocument(t *testing.T) {
	var s jobStats
	rf := s.Wrap(func(path string) (int, error) { return 0, errors.New("boom") })

	require.Error(t, rf("112.img.xml"))
	require.Equal(t, 1, s.images)
	require.Equal(t, 1, s.numeric)
	require.Equal(t, 0, s.written)
}

func TestJobStats_LogReportsSkipped(t *testing.T) {
	l, hook := test.NewNullLogger()
	s := jobStats{images: 98, numeric: 88, written: 78}
	s.Log(l)

	require.Len(t, hook.Entries, 1)
	require.Equal(t, logrus.InfoLevel, hook.Entries[0].Level)
	require.Contains(t, hook.Entries[0].Message, "images=98")
	require.Contains(t, hook.Entries[0].Message, "numeric=88")
	require.Contains(t, hook.Entries[0].Message, "written=78")
	require.Contains(t, hook.Entries[0].Message, "skipped=10")
}

func TestJobStats_LogWarnsOnZeroDocuments(t *testing.T) {
	l, hook := test.NewNullLogger()
	s := jobStats{images: 3, numeric: 0, written: 0}
	s.Log(l)

	require.Len(t, hook.Entries, 2)
	require.Equal(t, logrus.InfoLevel, hook.Entries[0].Level)
	require.Equal(t, logrus.WarnLevel, hook.Entries[1].Level)
}

// TestSkillWorker_SummaryEmittedOnWalkError pins the exact composition Skill.Run
// uses around the SKILL registration walk: skillStats.Log(l) is deferred
// immediately after the accumulator is declared, so it still fires when
// registerAllInDirectory returns a walk-level error (corrupt job image, I/O
// failure) rather than being skipped by an early return. This matches
// data/processor.go's WorkerSkill branch, which logs unconditionally.
func TestSkillWorker_SummaryEmittedOnWalkError(t *testing.T) {
	l, hook := test.NewNullLogger()

	var walkErr error
	func() {
		var skillStats skill.StatsAccumulator
		defer skillStats.Log(l)
		walkErr = registerAllInDirectory(l, context.Background(), filepath.Join(t.TempDir(), "does-not-exist"), skillStats.Wrap(func(path string) (skill.Stats, error) {
			return skill.Stats{Processed: 1}, nil
		}))
	}()
	require.Error(t, walkErr, "a missing Skill.wz directory must be a walk-level error")

	var summary bool
	for _, entry := range hook.AllEntries() {
		if entry.Level == logrus.InfoLevel && strings.Contains(entry.Message, "skills: processed=") {
			summary = true
		}
	}
	require.True(t, summary, "the run summary must be logged even when the walk itself failed")
}
