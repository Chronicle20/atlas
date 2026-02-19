package storage

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestCache(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitNpcContextCache(client)
}

func TestPut_And_Get(t *testing.T) {
	setupTestCache(t)

	GetNpcContextCache().Put(12345, 9001, 30*time.Minute)

	npcId, ok := GetNpcContextCache().Get(12345)
	assert.True(t, ok)
	assert.Equal(t, uint32(9001), npcId)
}

func TestGet_NotFound(t *testing.T) {
	setupTestCache(t)

	_, ok := GetNpcContextCache().Get(99999)
	assert.False(t, ok)
}

func TestRemove(t *testing.T) {
	setupTestCache(t)

	GetNpcContextCache().Put(12345, 9001, 30*time.Minute)
	GetNpcContextCache().Remove(12345)

	_, ok := GetNpcContextCache().Get(12345)
	assert.False(t, ok)
}

func TestRemove_NonExistent(t *testing.T) {
	setupTestCache(t)

	// Should not panic
	GetNpcContextCache().Remove(99999)
}

func TestPut_Overwrite(t *testing.T) {
	setupTestCache(t)

	GetNpcContextCache().Put(12345, 9001, 30*time.Minute)
	GetNpcContextCache().Put(12345, 9002, 30*time.Minute)

	npcId, ok := GetNpcContextCache().Get(12345)
	assert.True(t, ok)
	assert.Equal(t, uint32(9002), npcId)
}
