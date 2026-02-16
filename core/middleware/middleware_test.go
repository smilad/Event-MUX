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

func TestLogging(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer log.SetOutput(nil)

	handler := middleware.Logging()(func(ctx context.Context, msg core.Message) error {
		return nil
	})

	msg := &mock.Message{K: []byte("test-key"), V: []byte("val")}
	if err := handler(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "OK") {
		t.Errorf("expected OK log, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "test-key") {
		t.Errorf("expected key in log, got: %s", buf.String())
	}
}

func TestLogging_Error(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer log.SetOutput(nil)

	handler := middleware.Logging()(func(ctx context.Context, msg core.Message) error {
		return errors.New("boom")
	})

	msg := &mock.Message{K: []byte("k"), V: []byte("v")}
	handler(context.Background(), msg)

	if !strings.Contains(buf.String(), "ERROR") {
		t.Errorf("expected ERROR log, got: %s", buf.String())
	}
}

func TestRecovery(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := middleware.Recovery()(func(ctx context.Context, msg core.Message) error {
		panic("test panic")
	})

	msg := &mock.Message{K: []byte("k"), V: []byte("v")}
	err := handler(context.Background(), msg)
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

	handler := middleware.Recovery()(func(ctx context.Context, msg core.Message) error {
		return nil
	})

	msg := &mock.Message{K: []byte("k"), V: []byte("v")}
	if err := handler(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
