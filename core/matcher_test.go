package core

import "testing"

func TestDefaultMatcher(t *testing.T) {
	m := DefaultMatcher{}

	tests := []struct {
		pattern string
		topic   string
		want    bool
	}{
		// Exact match
		{"orders.created", "orders.created", true},
		{"orders.created", "orders.updated", false},
		{"orders", "orders", true},

		// Single-level wildcard
		{"orders.*", "orders.created", true},
		{"orders.*", "orders.updated", true},
		{"orders.*", "orders.us.created", false},
		{"*.created", "orders.created", true},
		{"*.created", "payments.created", true},

		// Multi-level wildcard
		{"orders.#", "orders.created", true},
		{"orders.#", "orders.us.created", true},
		{"orders.#", "orders.us.east.created", true},
		{"#", "anything", true},
		{"#", "a.b.c", true},

		// Combined
		{"orders.*.#", "orders.us.created", true},
		{"orders.*.#", "orders.us.east.created", true},

		// Edge cases
		{"orders.created", "orders", false},
		{"orders", "orders.created", false},
		{"orders.*", "orders", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"→"+tt.topic, func(t *testing.T) {
			got := m.Match(tt.pattern, tt.topic)
			if got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.topic, got, tt.want)
			}
		})
	}
}
