package expression

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestNewRevertTask(t *testing.T) {
	logger, _ := test.NewNullLogger()
	interval := 100 * time.Millisecond

	task := NewRevertTask(logger, interval)

	assert.NotNil(t, task)
}

func TestRevertTask_SleepTime(t *testing.T) {
	logger, _ := test.NewNullLogger()
	interval := 250 * time.Millisecond

	task := NewRevertTask(logger, interval)

	assert.Equal(t, interval, task.SleepTime())
}

func TestRevertTask_SleepTime_DifferentIntervals(t *testing.T) {
	logger, _ := test.NewNullLogger()

	testCases := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		5 * time.Second,
	}

	for _, interval := range testCases {
		task := NewRevertTask(logger, interval)
		assert.Equal(t, interval, task.SleepTime(), "SleepTime should return %v", interval)
	}
}

func TestRevertTask_Run_NoExpiredExpressions(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()

	logger, _ := test.NewNullLogger()
	task := NewRevertTask(logger, 100*time.Millisecond)

	ten := setupTestTenant(t)

	// Add a non-expired expression
	f := field.NewBuilder(0, 1, 100000000).Build()
	r.add(ten, 1000, f, 5)

	// Run should not panic and expression should still exist
	task.Run()

	// Verify expression still exists (not expired yet)
	_, found := r.get(ten, 1000)
	assert.True(t, found)
}

func TestRevertTask_Run_WithExpiredExpressions(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()

	logger, _ := test.NewNullLogger()
	task := NewRevertTask(logger, 100*time.Millisecond)

	ten := setupTestTenant(t)

	// Add an expression
	f := field.NewBuilder(0, 1, 100000000).Build()
	r.add(ten, 1000, f, 5)

	// Manually expire it
	r.lock.Lock()
	r.tenantLock[ten].Lock()
	if m, ok := r.expressionReg[ten][1000]; ok {
		expired := Model{
			tenant:      m.tenant,
			characterId: m.characterId,
			field:       m.field,
			expression:  m.expression,
			expiration:  time.Now().Add(-1 * time.Second),
		}
		r.expressionReg[ten][1000] = expired
	}
	r.tenantLock[ten].Unlock()
	r.lock.Unlock()

	// Run should process expired expression
	task.Run()

	// Verify expression was removed
	_, found := r.get(ten, 1000)
	assert.False(t, found)
}

func TestRevertTask_Run_MixedExpiredAndNonExpired(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()

	logger, _ := test.NewNullLogger()
	task := NewRevertTask(logger, 100*time.Millisecond)

	ten := setupTestTenant(t)

	// Add expressions
	f := field.NewBuilder(0, 1, 100000000).Build()
	r.add(ten, 1000, f, 5)
	r.add(ten, 2000, f, 10)

	// Manually expire only one
	r.lock.Lock()
	r.tenantLock[ten].Lock()
	if m, ok := r.expressionReg[ten][1000]; ok {
		expired := Model{
			tenant:      m.tenant,
			characterId: m.characterId,
			field:       m.field,
			expression:  m.expression,
			expiration:  time.Now().Add(-1 * time.Second),
		}
		r.expressionReg[ten][1000] = expired
	}
	r.tenantLock[ten].Unlock()
	r.lock.Unlock()

	// Run
	task.Run()

	// Verify expired was removed
	_, found1 := r.get(ten, 1000)
	assert.False(t, found1)

	// Verify non-expired still exists
	_, found2 := r.get(ten, 2000)
	assert.True(t, found2)
}
