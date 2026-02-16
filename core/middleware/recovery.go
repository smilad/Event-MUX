package middleware

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"github.com/miladsoleymani/eventmux/core"
)

// Recovery returns middleware that recovers from panics in handlers,
// logs the stack trace, and returns the panic as an error.
func Recovery() core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(ctx context.Context, msg core.Message) (err error) {
			defer func() {
				if r := recover(); r != nil {
					buf := make([]byte, 4096)
					n := runtime.Stack(buf, false)
					log.Printf("[EventMux] PANIC recovered: %v\n%s", r, buf[:n])
					err = fmt.Errorf("eventmux: panic recovered: %v", r)
				}
			}()
			return next(ctx, msg)
		}
	}
}
