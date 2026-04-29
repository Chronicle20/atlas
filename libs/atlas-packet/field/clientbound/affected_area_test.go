package clientbound

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestAffectedAreaCreated_EncodeShape(t *testing.T) {
	mistId := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	w := NewAffectedAreaCreated(mistId, 0xCAFE, 100, 200, -50, -30, 50, 30, 10000, 12345)

	require.Equal(t, AffectedAreaCreatedWriter, w.Operation())

	enc := w.Encode(logrus.New(), context.Background())
	require.NotNil(t, enc)

	out := enc(map[string]interface{}{})
	require.NotEmpty(t, out, "encoded packet must be non-empty")
}

func TestAffectedAreaRemoved_EncodeShape(t *testing.T) {
	mistId := uuid.MustParse("00000000-0000-0000-0000-00000000000b")
	w := NewAffectedAreaRemoved(mistId, 0xCAFE)

	require.Equal(t, AffectedAreaRemovedWriter, w.Operation())

	enc := w.Encode(logrus.New(), context.Background())
	require.NotNil(t, enc)

	out := enc(map[string]interface{}{})
	require.NotEmpty(t, out)
}

func TestAffectedAreaCreated_Getters(t *testing.T) {
	mistId := uuid.New()
	w := NewAffectedAreaCreated(mistId, 7, 11, 22, -1, -2, 3, 4, 555, 9)
	require.Equal(t, mistId, w.MistId())
	require.Equal(t, uint32(7), w.OwnerId())
	require.Equal(t, int16(11), w.OriginX())
	require.Equal(t, int16(22), w.OriginY())
	require.Equal(t, int16(-1), w.LtX())
	require.Equal(t, int16(-2), w.LtY())
	require.Equal(t, int16(3), w.RbX())
	require.Equal(t, int16(4), w.RbY())
	require.Equal(t, int64(555), w.Duration())
	require.Equal(t, uint32(9), w.SkillLevel())
}
