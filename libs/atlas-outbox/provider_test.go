package outbox_test

import (
	"errors"
	"testing"

	kafkaproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// localProvider mirrors every service's kafka/producer.Provider named type;
// assignment here is the compile-time proof EmitProvider satisfies it.
type localProvider func(token string) kafkaproducer.MessageProducer

func TestEmitProvider_EnqueuesThroughEmitShapedLoop(t *testing.T) {
	db := bridgeDb(t)
	ctx := tenantCtx(t)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var p localProvider = outbox.EmitProvider(logrus.New(), ctx, tx)
		// The message.Emit flush loop shape: p(token)(FixedProvider(msgs)).
		return p("EVENT_TOPIC_X")(model.FixedProvider([]kafka.Message{
			{Key: []byte("k"), Value: []byte("v")},
		}))
	}))

	var ents []outbox.Entity
	require.NoError(t, db.Find(&ents).Error)
	require.Len(t, ents, 1)
	require.Equal(t, "EVENT_TOPIC_X", ents[0].Topic)
	require.Equal(t, []byte("k"), ents[0].MessageKey)
}

func TestEmitProvider_ProviderErrorPropagates(t *testing.T) {
	db := bridgeDb(t)
	boom := errors.New("boom")
	err := db.Transaction(func(tx *gorm.DB) error {
		p := outbox.EmitProvider(logrus.New(), tenantCtx(t), tx)
		return p("T")(model.ErrorProvider[[]kafka.Message](boom))
	})
	require.ErrorIs(t, err, boom)
}

func TestEmitProvider_EnqueueErrorFailsTransaction(t *testing.T) {
	db := bridgeDb(t)
	err := db.Transaction(func(tx *gorm.DB) error {
		p := outbox.EmitProvider(logrus.New(), tenantCtx(t), tx)
		return p("T")(model.FixedProvider([]kafka.Message{{Key: nil, Value: []byte("v")}}))
	})
	require.Error(t, err)
	var count int64
	require.NoError(t, db.Model(&outbox.Entity{}).Count(&count).Error)
	require.Zero(t, count)
}
