package core

import "strings"

// TopicMatcher determines whether a subscription pattern matches a given topic.
type TopicMatcher interface {
	Match(pattern string, topic string) bool
}

// DefaultMatcher supports exact matching, single-level wildcard (*),
// and multi-level wildcard (#).
//
// Examples:
//
//	"orders.created" matches "orders.created"       (exact)
//	"orders.*"       matches "orders.created"       (single-level)
//	"orders.*"       does NOT match "orders.us.created"
//	"payments.#"     matches "payments.us.created"  (multi-level)
//	"payments.#"     matches "payments.created"
type DefaultMatcher struct{}

func (DefaultMatcher) Match(pattern, topic string) bool {
	patParts := strings.Split(pattern, ".")
	topParts := strings.Split(topic, ".")

	pi, ti := 0, 0
	for pi < len(patParts) && ti < len(topParts) {
		switch patParts[pi] {
		case "#":
			// # at the end matches all remaining levels
			if pi == len(patParts)-1 {
				return true
			}
			// # in the middle: try all remaining positions
			pi++
			for ti <= len(topParts) {
				if matchFrom(patParts, pi, topParts, ti) {
					return true
				}
				ti++
			}
			return false
		case "*":
			// matches exactly one level — just advance both
			pi++
			ti++
		default:
			if patParts[pi] != topParts[ti] {
				return false
			}
			pi++
			ti++
		}
	}

	// Both must be fully consumed
	return pi == len(patParts) && ti == len(topParts)
}

func matchFrom(pat []string, pi int, top []string, ti int) bool {
	for pi < len(pat) && ti < len(top) {
		switch pat[pi] {
		case "#":
			if pi == len(pat)-1 {
				return true
			}
			pi++
			for ti <= len(top) {
				if matchFrom(pat, pi, top, ti) {
					return true
				}
				ti++
			}
			return false
		case "*":
			pi++
			ti++
		default:
			if pat[pi] != top[ti] {
				return false
			}
			pi++
			ti++
		}
	}
	return pi == len(pat) && ti == len(top)
}
