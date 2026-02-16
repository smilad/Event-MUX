package middleware

import (
	"fmt"
	"log"
	"runtime"

	"github.com/miladsoleymani/eventmux/core"
)

// Recovery returns middleware that recovers from panics in handlers,
// logs the stack trace, and returns the panic as an error.
func Recovery() core.MiddlewareFunc {
	return func(next core.HandlerFunc) core.HandlerFunc {
		return func(c core.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					buf := make([]byte, 4096)
					n := runtime.Stack(buf, false)
					log.Printf("[EventMux] PANIC recovered: %v\n%s", r, buf[:n])
					err = fmt.Errorf("eventmux: panic recovered: %v", r)
				}
			}()
			return next(c)
		}
	}
}
