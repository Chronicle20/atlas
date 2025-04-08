package asset

import (
	"atlas-inventory/kafka/message/asset"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func CreatedEventStatusProvider(assetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(assetId))
	value := &asset.StatusEvent[asset.CreatedStatusEventBody]{
		AssetId: assetId,
		Type:    asset.StatusEventTypeCreated,
		Body:    asset.CreatedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func DeletedEventStatusProvider(assetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(assetId))
	value := &asset.StatusEvent[asset.DeletedStatusEventBody]{
		AssetId: assetId,
		Type:    asset.StatusEventTypeDeleted,
		Body:    asset.DeletedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
