package nats

import (
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
)

// message adapts a JetStream message to core.Message.
type message struct {
	msg jetstream.Msg
}

func (m *message) Key() []byte   { return []byte(m.msg.Subject()) }
func (m *message) Value() []byte { return m.msg.Data() }

func (m *message) Headers() map[string]string {
	raw := m.msg.Headers()
	h := make(map[string]string, len(raw))
	for k, v := range raw {
		if len(v) > 0 {
			h[k] = v[0]
		}
	}
	return h
}

// Ack acknowledges the message, marking it as processed.
func (m *message) Ack() error {
	if err := m.msg.Ack(); err != nil {
		return fmt.Errorf("eventmux/nats: ack: %w", err)
	}
	return nil
}

// Nack signals that the message could not be processed.
// The server will redeliver it according to the consumer's MaxDeliver setting.
func (m *message) Nack() error {
	if err := m.msg.Nak(); err != nil {
		return fmt.Errorf("eventmux/nats: nack: %w", err)
	}
	return nil
}
