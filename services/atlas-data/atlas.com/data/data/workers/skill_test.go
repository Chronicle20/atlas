package workers

import (
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestCountingRegister_SumsWrittenDocuments(t *testing.T) {
	total := 0
	rf := countingRegister(&total, func(path string) (int, error) {
		if path == "MobSkill.img.xml" {
			return 0, nil
		}
		return 1, nil
	})

	require.NoError(t, rf("112.img.xml"))
	require.NoError(t, rf("MobSkill.img.xml"))
	require.NoError(t, rf("100.img.xml"))
	require.Equal(t, 2, total)
}

func TestCountingRegister_PropagatesErrorAndAddsNothing(t *testing.T) {
	total := 0
	rf := countingRegister(&total, func(path string) (int, error) {
		return 0, errors.New("boom")
	})

	require.Error(t, rf("112.img.xml"))
	require.Equal(t, 0, total)
}

func TestLogJobDocCount_WarnsOnZero(t *testing.T) {
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	logJobDocCount(l, 0)

	require.Len(t, hook.Entries, 2)
	require.Equal(t, logrus.InfoLevel, hook.Entries[0].Level)
	require.Equal(t, logrus.WarnLevel, hook.Entries[1].Level)
}

func TestLogJobDocCount_NoWarnWhenDocumentsWritten(t *testing.T) {
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	logJobDocCount(l, 82)

	require.Len(t, hook.Entries, 1)
	require.Equal(t, logrus.InfoLevel, hook.Entries[0].Level)
	require.Contains(t, hook.Entries[0].Message, "written=82")
}
