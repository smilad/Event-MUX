package middleware_test

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/miladsoleymani/eventmux/core"
	"github.com/miladsoleymani/eventmux/core/middleware"
	"github.com/miladsoleymani/eventmux/internal/mock"
)

// newTestContext creates a core.Context from a mock message for testing.
func newTestContext(msg *mock.Message) core.Context {
	return core.NewContext(
		context.Background(),
		msg,
		"test.topic",
		mock.NewBroker(),
		core.JSONBinder{},
	)
}

func TestLogging(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer log.SetOutput(nil)

	handler := middleware.Logging()(func(c core.Context) error {
		return nil
	})

	c := newTestContext(&mock.Message{K: []byte("test-key"), V: []byte("val")})
	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("expected OK log, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "test-key") {
		t.Errorf("expected key in log, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "test.topic") {
		t.Errorf("expected topic in log, got: %s", buf.String())
	}
}

func TestLogging_Error(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer log.SetOutput(nil)

	handler := middleware.Logging()(func(c core.Context) error {
		return errors.New("boom")
	})

	c := newTestContext(&mock.Message{K: []byte("k"), V: []byte("v")})
	handler(c)

	if !strings.Contains(buf.String(), "ERROR") {
		t.Errorf("expected ERROR log, got: %s", buf.String())
	}
}

func TestRecovery(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := middleware.Recovery()(func(c core.Context) error {
		panic("test panic")
	})

	c := newTestContext(&mock.Message{K: []byte("k"), V: []byte("v")})
	err := handler(c)
	if err == nil {
		t.Fatal("expected error from recovered panic")
	}
	if !strings.Contains(err.Error(), "panic recovered") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRecovery_NoPanic(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := middleware.Recovery()(func(c core.Context) error {
		return nil
	})

	c := newTestContext(&mock.Message{K: []byte("k"), V: []byte("v")})
	if err := handler(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
