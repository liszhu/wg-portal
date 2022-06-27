package internal

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// SignalAwareContext returns a context that gets closed once a given signal is retrieved.
// By default, the following signals are handled: syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP
func SignalAwareContext(ctx context.Context, sig ...os.Signal) context.Context {
	c := make(chan os.Signal, 1)
	if len(sig) == 0 {
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	} else {

		signal.Notify(c, sig...)
	}
	signalCtx, cancel := context.WithCancel(ctx)

	// Attach signal handlers to context
	go func() {
		select {
		case <-ctx.Done():
			// normal shutdown, quit go routine
		case <-c:
			cancel() // cancel the context
		}

		// cleanup
		signal.Stop(c)
		close(c)
	}()

	return signalCtx
}

// AssertNoError panics if the given error is not nil.
func AssertNoError(err error) {
	if err != nil {
		panic(err)
	}
}

// ByteCountSI returns the byte count as string, see: https://yourbasic.org/golang/formatting-byte-size-to-human-readable-format/
func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
