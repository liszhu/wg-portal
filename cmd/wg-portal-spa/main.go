package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := signalAwareContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	logrus.Infof("Starting web portal...")

	noAuth := func(c *gin.Context) {}
	webService, err := NewServer(noAuth)
	if err != nil {
		panic(err)
	}

	webService.Run(ctx, ":5000")

	// wait until context gets cancelled
	<-ctx.Done()

	logrus.Infof("Stopped web portal")

}

// signalAwareContext returns a context that gets closed once a given signal is retrieved.
// By default, the following signals are handled: syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP
func signalAwareContext(ctx context.Context, sig ...os.Signal) context.Context {
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
