package mock

// Message is a simple core.Message implementation for testing.
type Message struct {
	K       []byte
	V       []byte
	H       map[string]string
	Acked   bool
	Nacked  bool
	AckErr  error
	NackErr error
}

func (m *Message) Key() []byte              { return m.K }
func (m *Message) Value() []byte            { return m.V }
func (m *Message) Headers() map[string]string { return m.H }

func (m *Message) Ack() error {
	m.Acked = true
	return m.AckErr
}

func (m *Message) Nack() error {
	m.Nacked = true
	return m.NackErr
}
