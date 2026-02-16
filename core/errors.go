package core

import "errors"

var (
	// ErrBrokerClosed is returned when operations are attempted on a closed broker.
	ErrBrokerClosed = errors.New("eventmux: broker is closed")

	// ErrNoHandler is returned when no handler matches the incoming topic.
	ErrNoHandler = errors.New("eventmux: no handler registered for topic")

	// ErrAlreadyStarted is returned when Start is called on a running router.
	ErrAlreadyStarted = errors.New("eventmux: router already started")

	// ErrNoBroker is returned when a router is created without a broker.
	ErrNoBroker = errors.New("eventmux: broker is nil")
)
