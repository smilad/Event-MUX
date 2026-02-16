package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// message adapts a kafka.Message to core.Message.
// It holds a reference to the reader for offset management.
type message struct {
	raw    kafka.Message
	reader *kafka.Reader
	ctx    context.Context
}

func (m *message) Key() []byte   { return m.raw.Key }
func (m *message) Value() []byte { return m.raw.Value }

func (m *message) Headers() map[string]string {
	h := make(map[string]string, len(m.raw.Headers))
	for _, kh := range m.raw.Headers {
		h[kh.Key] = string(kh.Value)
	}
	return h
}

// Ack commits the offset for this message.
func (m *message) Ack() error {
	if err := m.reader.CommitMessages(m.ctx, m.raw); err != nil {
		return fmt.Errorf("eventmux/kafka: commit offset: %w", err)
	}
	return nil
}

// Nack is a no-op for Kafka. Not committing the offset causes the message
// to be redelivered on the next consumer group rebalance or restart.
func (m *message) Nack() error {
	return nil
}
