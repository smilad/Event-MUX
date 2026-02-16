package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// message adapts an amqp.Delivery to core.Message.
type message struct {
	delivery amqp.Delivery
	requeue  bool
}

func (m *message) Key() []byte   { return []byte(m.delivery.RoutingKey) }
func (m *message) Value() []byte { return m.delivery.Body }

func (m *message) Headers() map[string]string {
	h := make(map[string]string, len(m.delivery.Headers))
	for k, v := range m.delivery.Headers {
		if s, ok := v.(string); ok {
			h[k] = s
		} else {
			h[k] = fmt.Sprintf("%v", v)
		}
	}
	return h
}

// Ack acknowledges the message, removing it from the queue.
func (m *message) Ack() error {
	if err := m.delivery.Ack(false); err != nil {
		return fmt.Errorf("eventmux/rabbitmq: ack: %w", err)
	}
	return nil
}

// Nack negatively acknowledges the message. If requeue is enabled,
// the message is returned to the queue for redelivery.
func (m *message) Nack() error {
	if err := m.delivery.Nack(false, m.requeue); err != nil {
		return fmt.Errorf("eventmux/rabbitmq: nack: %w", err)
	}
	return nil
}
