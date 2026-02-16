package middleware

import (
	"context"
	"log"
	"time"

	"github.com/miladsoleymani/eventmux/core"
)

// Logging returns middleware that logs message processing duration and errors.
func Logging() core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(ctx context.Context, msg core.Message) error {
			start := time.Now()
			err := next(ctx, msg)
			elapsed := time.Since(start)

			if err != nil {
				log.Printf("[EventMux] ERROR key=%s elapsed=%s err=%v", string(msg.Key()), elapsed, err)
			} else {
				log.Printf("[EventMux] OK    key=%s elapsed=%s", string(msg.Key()), elapsed)
			}
			return err
		}
	}
}
