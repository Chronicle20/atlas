package invite

import (
	"atlas-invites/invite/mock"
	"atlas-invites/kafka/message"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewProcessor(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	p := NewProcessor(l, ctx)

	assert.NotNil(t, p)
}

func TestNewProcessor_ExtractsTenant(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	p := NewProcessor(l, ctx)
	impl := p.(*ProcessorImpl)

	assert.Equal(t, ten, impl.t)
}

func TestNewProcessor_PanicsOnMissingTenant(t *testing.T) {
	setupTestRegistry(t)
	ctx := context.Background() // No tenant in context
	l := setupTestLogger(t)

	assert.Panics(t, func() {
		NewProcessor(l, ctx)
	})
}

func TestProcessor_GetByCharacterId_Empty(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	p := NewProcessor(l, ctx)

	result, err := p.GetByCharacterId(9999)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestProcessor_GetByCharacterId_ReturnsInvites(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	// Create some invites directly in registry
	GetRegistry().Create(ctx, 1001, 1, 2001, "BUDDY", 5001)
	GetRegistry().Create(ctx, 1002, 1, 2001, "PARTY", 5002)

	p := NewProcessor(l, ctx)

	result, err := p.GetByCharacterId(2001)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestProcessor_ByCharacterIdProvider(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	GetRegistry().Create(ctx, 1001, 1, 2001, "BUDDY", 5001)

	p := NewProcessor(l, ctx)
	provider := p.ByCharacterIdProvider(2001)

	result, err := provider()

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestProcessor_Create(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	p := NewProcessor(l, ctx)
	mb := message.NewBuffer()
	transactionId := uuid.New()

	m, err := p.Create(mb)(5001)(1)("BUDDY")(1001)(2001)(transactionId)

	assert.NoError(t, err)
	assert.NotZero(t, m.Id())
	assert.Equal(t, uint32(1001), m.OriginatorId())
	assert.Equal(t, uint32(2001), m.TargetId())
	assert.Equal(t, "BUDDY", m.Type())
	assert.Equal(t, uint32(5001), m.ReferenceId())

	// Verify message was buffered
	messages := mb.GetAll()
	assert.NotEmpty(t, messages)
}

func TestProcessor_CreateAndEmit(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	mockProducer := mock.NewProducerMock()

	// Create processor with mock producer
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   ten,
		p:   mockProducer.Provider(),
	}
	transactionId := uuid.New()

	m, err := p.CreateAndEmit(5001, 1, "BUDDY", 1001, 2001, transactionId)

	assert.NoError(t, err)
	assert.NotZero(t, m.Id())
	assert.Equal(t, uint32(1001), m.OriginatorId())
	assert.Equal(t, uint32(2001), m.TargetId())

	// Verify message was emitted
	assert.Greater(t, mockProducer.MessageCount(), 0)
}

func TestProcessor_Accept(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	// Create an invite first
	created := GetRegistry().Create(ctx, 1001, 1, 2001, "BUDDY", 5001)

	p := NewProcessor(l, ctx)
	mb := message.NewBuffer()
	transactionId := uuid.New()

	m, err := p.Accept(mb)(5001)(1)("BUDDY")(2001)(transactionId)

	assert.NoError(t, err)
	assert.Equal(t, created.Id(), m.Id())

	// Verify invite was deleted from registry
	_, err = GetRegistry().GetByReference(ctx, 2001, "BUDDY", 5001)
	assert.Error(t, err)

	// Verify message was buffered
	messages := mb.GetAll()
	assert.NotEmpty(t, messages)
}

func TestProcessor_Accept_NotFound(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	p := NewProcessor(l, ctx)
	mb := message.NewBuffer()
	transactionId := uuid.New()

	_, err := p.Accept(mb)(9999)(1)("BUDDY")(2001)(transactionId)

	assert.Error(t, err)
}

func TestProcessor_AcceptAndEmit(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	mockProducer := mock.NewProducerMock()

	// Create an invite first
	created := GetRegistry().Create(ctx, 1001, 1, 2001, "BUDDY", 5001)

	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   ten,
		p:   mockProducer.Provider(),
	}
	transactionId := uuid.New()

	m, err := p.AcceptAndEmit(5001, 1, "BUDDY", 2001, transactionId)

	assert.NoError(t, err)
	assert.Equal(t, created.Id(), m.Id())
	assert.Greater(t, mockProducer.MessageCount(), 0)
}

func TestProcessor_Reject(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	// Create an invite first
	created := GetRegistry().Create(ctx, 1001, 1, 2001, "BUDDY", 5001)

	p := NewProcessor(l, ctx)
	mb := message.NewBuffer()
	transactionId := uuid.New()

	m, err := p.Reject(mb)(1001)(1)("BUDDY")(2001)(transactionId)

	assert.NoError(t, err)
	assert.Equal(t, created.Id(), m.Id())

	// Verify invite was deleted from registry
	_, err = GetRegistry().GetByOriginator(ctx, 2001, "BUDDY", 1001)
	assert.Error(t, err)

	// Verify message was buffered
	messages := mb.GetAll()
	assert.NotEmpty(t, messages)
}

func TestProcessor_Reject_NotFound(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)

	p := NewProcessor(l, ctx)
	mb := message.NewBuffer()
	transactionId := uuid.New()

	_, err := p.Reject(mb)(9999)(1)("BUDDY")(2001)(transactionId)

	assert.Error(t, err)
}

func TestProcessor_RejectAndEmit(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	mockProducer := mock.NewProducerMock()

	// Create an invite first
	created := GetRegistry().Create(ctx, 1001, 1, 2001, "BUDDY", 5001)

	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   ten,
		p:   mockProducer.Provider(),
	}
	transactionId := uuid.New()

	m, err := p.RejectAndEmit(1001, 1, "BUDDY", 2001, transactionId)

	assert.NoError(t, err)
	assert.Equal(t, created.Id(), m.Id())
	assert.Greater(t, mockProducer.MessageCount(), 0)
}

func TestProcessor_Create_MultipleInviteTypes(t *testing.T) {
	inviteTypes := []string{"BUDDY", "PARTY", "GUILD", "MESSENGER", "FAMILY", "TRADE"}

	for _, inviteType := range inviteTypes {
		t.Run(inviteType, func(t *testing.T) {
			setupTestRegistry(t)
			ten := setupTestTenant(t)
			ctx := setupTestContext(t, ten)
			l := setupTestLogger(t)

			p := NewProcessor(l, ctx)
			mb := message.NewBuffer()
			transactionId := uuid.New()

			m, err := p.Create(mb)(5001)(1)(inviteType)(1001)(2001)(transactionId)

			assert.NoError(t, err)
			assert.Equal(t, inviteType, m.Type())
		})
	}
}

func TestProcessor_TenantIsolation(t *testing.T) {
	setupTestRegistry(t)
	ten1 := setupTestTenant(t)
	ten2 := setupTestTenant(t)
	ctx1 := setupTestContext(t, ten1)
	ctx2 := setupTestContext(t, ten2)
	l := setupTestLogger(t)

	// Create invite in tenant 1
	p1 := NewProcessor(l, ctx1)
	mb := message.NewBuffer()
	p1.Create(mb)(5001)(1)("BUDDY")(1001)(2001)(uuid.New())

	// Tenant 2 should not see tenant 1's invites
	p2 := NewProcessor(l, ctx2)
	results, err := p2.GetByCharacterId(2001)

	assert.NoError(t, err)
	assert.Empty(t, results)
}
