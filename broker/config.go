package broker

// Config holds broker-agnostic configuration.
// Broker plugins extract the fields they need.
type Config struct {
	// Brokers is a list of broker addresses (e.g., "localhost:9092").
	Brokers []string

	// Topic is the default topic or queue name.
	Topic string

	// Group is the consumer group ID.
	Group string

	// Extra holds plugin-specific configuration.
	Extra map[string]any
}
