// Package lock provides leader-election semantics on top of a single Redis
// instance.
//
// CORRECTNESS BOUNDARY (single-Redis split-brain caveat):
// During a Redis primary→replica failover the lease key is replicated
// asynchronously. For 1–5 seconds two pods can each believe they hold the
// lease. Use this library only for workloads whose downstream consumers
// already tolerate at-least-once delivery (Atlas sweep tasks emitting Kafka
// events qualify; financial transactions and exclusive resource claims do
// not). Multi-Redis Redlock is out of scope.
package lock
