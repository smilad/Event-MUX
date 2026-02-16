package middleware

import (
	"log"
	"time"

	"github.com/miladsoleymani/eventmux/core"
)

// Logging returns middleware that logs message processing duration and errors.
func Logging() core.MiddlewareFunc {
	return func(next core.HandlerFunc) core.HandlerFunc {
		return func(c core.Context) error {
			start := time.Now()
			err := next(c)
			elapsed := time.Since(start)

			if err != nil {
				log.Printf("[EventMux] ERROR topic=%s key=%s elapsed=%s err=%v",
					c.Topic(), string(c.Key()), elapsed, err)
			} else {
				log.Printf("[EventMux] OK    topic=%s key=%s elapsed=%s",
					c.Topic(), string(c.Key()), elapsed)
			}
			return err
		}
	}
}
