package consumer

import (
	"context"

	"github.com/segmentio/kafka-go"
)

// Group is the subset of *kafka.ConsumerGroup this package depends on.
// Defining it as an interface is what lets the assignment-awareness tests
// script generations without a broker (task-209 design §6).
type Group interface {
	// Next blocks until the previous generation has ended and the next one
	// is ready. It returns ErrGroupClosed after Close, or the ctx error.
	Next(ctx context.Context) (Generation, error)
	Close() error
}

// Generation is the subset of *kafka.Generation this package depends on.
//
// IMPORTANT: a function passed to Start ENDS THE GENERATION when it returns
// (kafka-go consumergroup.go:387-405). Every recoverable error must therefore
// be handled inside the function; returning is reserved for "the generation
// is already ending". That inversion — recover in place, never by rejoining —
// is the point of the migration (design §1.1).
type Generation interface {
	ID() int32
	Assignments() map[string][]kafka.PartitionAssignment
	Start(fn func(ctx context.Context))
	CommitOffsets(offsets map[string]map[int]int64) error
}

// GroupProducer builds a Group from a kafka-go consumer-group config.
type GroupProducer func(cfg kafka.ConsumerGroupConfig) (Group, error)

// PartitionReaderProducer builds a positioned, non-group reader for a single
// partition. The offset is a producer argument rather than a KafkaReader
// method so the existing KafkaReader seam stays frozen (FR-3.5): every mock
// written against KafkaReader works unchanged against the new fetch loop.
type PartitionReaderProducer func(cfg kafka.ReaderConfig, offset int64) KafkaReader

//goland:noinspection GoUnusedExportedFunction
func ConfigGroupProducer(gp GroupProducer) ManagerConfig {
	return func(m *Manager) {
		m.gp = gp
	}
}

//goland:noinspection GoUnusedExportedFunction
func ConfigPartitionReaderProducer(prp PartitionReaderProducer) ManagerConfig {
	return func(m *Manager) {
		m.prp = prp
	}
}

// defaultGroupProducer wraps kafka.NewConsumerGroup.
func defaultGroupProducer(cfg kafka.ConsumerGroupConfig) (Group, error) {
	cg, err := kafka.NewConsumerGroup(cfg)
	if err != nil {
		return nil, err
	}
	return &kafkaGroup{cg: cg}, nil
}

// defaultPartitionReaderProducer builds a plain (non-group) reader and
// positions it. SetOffset only fails on a closed reader or a group reader
// (kafka-go reader.go SetOffset); this reader is neither, and a no-op when
// offset already equals the reader's default FirstOffset.
func defaultPartitionReaderProducer(cfg kafka.ReaderConfig, offset int64) KafkaReader {
	r := kafka.NewReader(cfg)
	_ = r.SetOffset(offset)
	return r
}

type kafkaGroup struct {
	cg *kafka.ConsumerGroup
}

func (g *kafkaGroup) Next(ctx context.Context) (Generation, error) {
	gen, err := g.cg.Next(ctx)
	if err != nil {
		return nil, err
	}
	return &kafkaGeneration{gen: gen}, nil
}

func (g *kafkaGroup) Close() error { return g.cg.Close() }

type kafkaGeneration struct {
	gen *kafka.Generation
}

func (g *kafkaGeneration) ID() int32 { return g.gen.ID }

func (g *kafkaGeneration) Assignments() map[string][]kafka.PartitionAssignment {
	return g.gen.Assignments
}

func (g *kafkaGeneration) Start(fn func(ctx context.Context)) { g.gen.Start(fn) }

func (g *kafkaGeneration) CommitOffsets(offsets map[string]map[int]int64) error {
	return g.gen.CommitOffsets(offsets)
}

// groupConfig builds the consumer-group config for this Consumer. It is a
// deliberate 1:1 mirror of what kafka-go derives internally from today's
// ReaderConfig (reader.go:717-733): same group ID, same single-topic
// subscription, same StartOffset, WatchPartitionChanges left false.
func (c *Consumer) groupConfig() kafka.ConsumerGroupConfig {
	return kafka.ConsumerGroupConfig{
		ID:                    c.groupId,
		Brokers:               append([]string(nil), c.brokers...),
		Topics:                []string{c.topic},
		StartOffset:           c.startOffset,
		WatchPartitionChanges: false,
	}
}

// partitionReaderConfig builds the config for one assigned partition's
// reader. GroupID is deliberately absent: group membership belongs to the
// ConsumerGroup, and kafka-go rejects a reader with both Partition and
// GroupID set (reader.go:545-547).
func (c *Consumer) partitionReaderConfig(partition int) kafka.ReaderConfig {
	return kafka.ReaderConfig{
		Brokers:   append([]string(nil), c.brokers...),
		Topic:     c.topic,
		Partition: partition,
		MaxWait:   c.maxWait,
	}
}
